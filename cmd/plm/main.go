package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/bitfantasy/nimo/internal/config"
	"github.com/bitfantasy/nimo/internal/middleware"
	"github.com/bitfantasy/nimo/internal/plm/entity"
	"github.com/bitfantasy/nimo/internal/plm/handler"
	"github.com/bitfantasy/nimo/internal/plm/repository"
	"github.com/bitfantasy/nimo/internal/plm/service"
	"github.com/bitfantasy/nimo/internal/shared/engine"
	"github.com/bitfantasy/nimo/internal/shared/feishu"
	srmentity "github.com/bitfantasy/nimo/internal/srm/entity"
	srmhandler "github.com/bitfantasy/nimo/internal/srm/handler"
	srmrepo "github.com/bitfantasy/nimo/internal/srm/repository"
	srmsvc "github.com/bitfantasy/nimo/internal/srm/service"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found, using environment variables")
	}

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化日志
	zapLogger, err := initLogger(cfg.Log)
	if err != nil {
		log.Fatalf("Failed to init logger: %v", err)
	}
	defer zapLogger.Sync()

	zapLogger.Info("Starting nimo-plm service",
		zap.String("version", Version),
		zap.String("build_time", BuildTime),
	)

	// 初始化数据库
	db, err := initDatabase(cfg.Database)
	if err != nil {
		zapLogger.Fatal("Failed to connect to database", zap.Error(err))
	}

	// AutoMigrate for task form tables
	if err := db.AutoMigrate(
		&entity.TaskForm{},
		&entity.TaskFormSubmission{},
		&entity.TemplateTaskForm{},
	); err != nil {
		zapLogger.Warn("AutoMigrate task form tables warning", zap.Error(err))
	}

	// V13: project_boms 增加 task_id 列
	db.Exec("ALTER TABLE project_boms ADD COLUMN IF NOT EXISTS task_id VARCHAR(32)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_project_boms_task_id ON project_boms(task_id)")

	// AutoMigrate for CMF tables
	if err := db.AutoMigrate(
		&entity.CMFDesign{},
		&entity.CMFDrawing{},
	); err != nil {
		zapLogger.Warn("AutoMigrate CMF tables warning", zap.Error(err))
	}
	// Drop old CMF placeholder tables if they exist
	db.Exec("DROP TABLE IF EXISTS cmf_items")
	db.Exec("DROP TABLE IF EXISTS cmf_specs")

	// AutoMigrate for SKU tables
	if err := db.AutoMigrate(
		&entity.ProductSKU{},
		&entity.SKUCMFConfig{},
		&entity.SKUBOMItem{},
	); err != nil {
		zapLogger.Warn("AutoMigrate SKU tables warning", zap.Error(err))
	}

	// AutoMigrate for PartDrawing table (V15: 图纸版本管理)
	if err := db.AutoMigrate(&entity.PartDrawing{}); err != nil {
		zapLogger.Warn("AutoMigrate PartDrawing table warning", zap.Error(err))
	}
	// V16: CMF变体表
	if err := db.AutoMigrate(&entity.BOMItemCMFVariant{}); err != nil {
		zapLogger.Warn("AutoMigrate BOMItemCMFVariant table warning", zap.Error(err))
	}
	// V16: sampling_ready字段
	db.Exec("ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS sampling_ready BOOLEAN DEFAULT false")

	// V14: ProjectBOMItem增加is_variant字段
	db.Exec("ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS is_variant BOOLEAN DEFAULT false")

	// V17: 语言变体表
	if err := db.AutoMigrate(&entity.BOMItemLangVariant{}); err != nil {
		zapLogger.Warn("AutoMigrate BOMItemLangVariant table warning", zap.Error(err))
	}

	// V16: STP缩略图URL字段
	db.Exec("ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS thumbnail_url VARCHAR(512)")

	// V19: BOM供应商关联字段
	db.Exec("ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS supplier_id VARCHAR(32)")
	db.Exec("ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS manufacturer_id VARCHAR(32)")
	db.Exec("ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS mpn VARCHAR(128)")

	// V18: BOM ECN功能
	if err := db.AutoMigrate(&entity.BOMDraft{}, &entity.BOMECN{}); err != nil {
		zapLogger.Warn("AutoMigrate BOM ECN tables warning", zap.Error(err))
	}
	// 扩展BOM status支持新状态
	db.Exec("ALTER TABLE project_boms DROP CONSTRAINT IF EXISTS project_boms_status_check")
	db.Exec("ALTER TABLE project_boms ADD CONSTRAINT project_boms_status_check CHECK (status IN ('draft', 'submitted', 'approved', 'rejected', 'released', 'frozen', 'obsolete', 'editing', 'ecn_pending'))")

	// 清理旧的唯一索引（EmployeeNo 允许为空，不再需要唯一约束）
	// 清理所有可能的 employee_no 唯一约束/索引
	db.Exec("ALTER TABLE users DROP CONSTRAINT IF EXISTS users_employee_no_key")
	db.Exec("DROP INDEX IF EXISTS idx_users_employee_no")
	db.Exec("DROP INDEX IF EXISTS users_employee_no_key")

	// 手动添加新列（AutoMigrate 会触发 FK 级联问题，所以用原始 SQL）
	migrationSQL := []string{
		"ALTER TABLE template_tasks ADD COLUMN IF NOT EXISTS auto_create_feishu_task boolean DEFAULT false",
		"ALTER TABLE template_tasks ADD COLUMN IF NOT EXISTS feishu_approval_code varchar(100) DEFAULT ''",
		"ALTER TABLE template_tasks ADD COLUMN IF NOT EXISTS is_locked boolean DEFAULT false",
		"ALTER TABLE template_tasks DROP CONSTRAINT IF EXISTS template_tasks_task_type_check",
		"ALTER TABLE template_tasks ADD CONSTRAINT template_tasks_task_type_check CHECK (task_type IN ('MILESTONE', 'TASK', 'SUBTASK', 'srm_procurement'))",
		"ALTER TABLE tasks ADD COLUMN IF NOT EXISTS auto_create_feishu_task boolean DEFAULT false",
		"ALTER TABLE tasks ADD COLUMN IF NOT EXISTS feishu_approval_code varchar(100) DEFAULT ''",
		"ALTER TABLE tasks ADD COLUMN IF NOT EXISTS is_locked boolean DEFAULT false",
		"ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_task_type_check",
		"ALTER TABLE tasks ADD CONSTRAINT tasks_task_type_check CHECK (task_type IN ('MILESTONE', 'TASK', 'SUBTASK', 'srm_procurement'))",

		// 状态机引擎表 (Phase 1)
		`CREATE TABLE IF NOT EXISTS state_machine_definitions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(100) NOT NULL UNIQUE,
			description TEXT,
			initial_state VARCHAR(50) NOT NULL,
			states JSONB,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS state_transitions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			machine_id UUID NOT NULL REFERENCES state_machine_definitions(id) ON DELETE CASCADE,
			from_state VARCHAR(50) NOT NULL,
			to_state VARCHAR(50) NOT NULL,
			event VARCHAR(100) NOT NULL,
			condition JSONB,
			actions JSONB,
			priority INT DEFAULT 0,
			description TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS state_transition_logs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			entity_type VARCHAR(50) NOT NULL,
			entity_id UUID NOT NULL,
			from_state VARCHAR(50),
			to_state VARCHAR(50) NOT NULL,
			event VARCHAR(100) NOT NULL,
			event_data JSONB,
			triggered_by VARCHAR(64),
			triggered_by_type VARCHAR(20),
			actions_executed JSONB,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS entity_states (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			entity_type VARCHAR(50) NOT NULL,
			entity_id UUID NOT NULL,
			current_state VARCHAR(50) NOT NULL,
			machine_id UUID REFERENCES state_machine_definitions(id),
			updated_at TIMESTAMP DEFAULT NOW(),
			UNIQUE(entity_type, entity_id)
		)`,

		// Phase 3: 工作流集成表
		`CREATE TABLE IF NOT EXISTS template_phase_roles (
			id VARCHAR(36) PRIMARY KEY,
			template_id VARCHAR(36) NOT NULL,
			phase VARCHAR(20) NOT NULL,
			role_code VARCHAR(50) NOT NULL,
			role_name VARCHAR(100) NOT NULL,
			is_required BOOLEAN DEFAULT true,
			trigger_task_code VARCHAR(50),
			UNIQUE(template_id, phase, role_code)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_template_phase_roles_template ON template_phase_roles(template_id)`,

		`CREATE TABLE IF NOT EXISTS template_task_outcomes (
			id VARCHAR(36) PRIMARY KEY,
			template_id VARCHAR(36) NOT NULL,
			task_code VARCHAR(50) NOT NULL,
			outcome_code VARCHAR(50) NOT NULL,
			outcome_name VARCHAR(100) NOT NULL,
			outcome_type VARCHAR(20) NOT NULL DEFAULT 'pass',
			rollback_to_task_code VARCHAR(50),
			rollback_cascade BOOLEAN DEFAULT false,
			sort_order INT DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_template_task_outcomes_template ON template_task_outcomes(template_id)`,

		`CREATE TABLE IF NOT EXISTS project_role_assignments (
			id VARCHAR(36) PRIMARY KEY,
			project_id VARCHAR(32) NOT NULL,
			phase VARCHAR(20) NOT NULL,
			role_code VARCHAR(50) NOT NULL,
			user_id VARCHAR(32) NOT NULL,
			feishu_user_id VARCHAR(64),
			assigned_by VARCHAR(32) NOT NULL,
			assigned_at TIMESTAMP DEFAULT NOW(),
			UNIQUE(project_id, phase, role_code)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_project_role_assignments_project ON project_role_assignments(project_id)`,

		`CREATE TABLE IF NOT EXISTS task_action_logs (
			id VARCHAR(36) PRIMARY KEY,
			project_id VARCHAR(32) NOT NULL,
			task_id VARCHAR(32) NOT NULL,
			action VARCHAR(50) NOT NULL,
			from_status VARCHAR(20),
			to_status VARCHAR(20) NOT NULL,
			operator_id VARCHAR(64) NOT NULL,
			operator_type VARCHAR(20) DEFAULT 'user',
			event_data JSONB,
			comment TEXT,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_task_action_logs_project ON task_action_logs(project_id)`,
		`CREATE INDEX IF NOT EXISTS idx_task_action_logs_task ON task_action_logs(task_id)`,

		// V4: PLM 自建审批表
		`CREATE TABLE IF NOT EXISTS approval_requests (
			id VARCHAR(36) PRIMARY KEY,
			project_id VARCHAR(32) NOT NULL,
			task_id VARCHAR(32) NOT NULL,
			title VARCHAR(200) NOT NULL,
			description TEXT,
			type VARCHAR(50) NOT NULL DEFAULT 'task_review',
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			form_data JSONB,
			result VARCHAR(20),
			result_comment TEXT,
			requested_by VARCHAR(32) NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_approval_requests_status ON approval_requests(status)`,
		`CREATE INDEX IF NOT EXISTS idx_approval_requests_task ON approval_requests(task_id)`,
		`CREATE TABLE IF NOT EXISTS approval_reviewers (
			id VARCHAR(36) PRIMARY KEY,
			approval_id VARCHAR(36) NOT NULL,
			user_id VARCHAR(32) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			comment TEXT,
			decided_at TIMESTAMP,
			sequence INT DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_approval_reviewers_approval ON approval_reviewers(approval_id)`,
		`CREATE INDEX IF NOT EXISTS idx_approval_reviewers_user ON approval_reviewers(user_id)`,

		// V5: 审批定义（模板管理）
		`CREATE TABLE IF NOT EXISTS approval_definitions (
			id VARCHAR(36) PRIMARY KEY,
			code VARCHAR(50) NOT NULL UNIQUE,
			name VARCHAR(200) NOT NULL,
			description TEXT,
			icon VARCHAR(50) DEFAULT 'approval',
			group_name VARCHAR(50) NOT NULL DEFAULT '其他',
			form_schema JSONB NOT NULL DEFAULT '[]',
			flow_schema JSONB NOT NULL DEFAULT '{"nodes":[]}',
			visibility VARCHAR(200) DEFAULT '全员',
			status VARCHAR(20) NOT NULL DEFAULT 'draft',
			admin_user_id VARCHAR(32),
			sort_order INT DEFAULT 0,
			created_by VARCHAR(32) NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS approval_groups (
			id VARCHAR(36) PRIMARY KEY,
			name VARCHAR(50) NOT NULL UNIQUE,
			sort_order INT DEFAULT 0,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		// Alter existing approval_definitions table to add new columns & fix old constraints
		`ALTER TABLE approval_definitions ALTER COLUMN approval_type DROP NOT NULL`,
		`ALTER TABLE approval_definitions ALTER COLUMN approval_type SET DEFAULT ''`,
		`ALTER TABLE approval_definitions DROP CONSTRAINT IF EXISTS approval_definitions_approval_type_check`,
		`CREATE TABLE IF NOT EXISTS template_task_dependencies (
			id VARCHAR(36) PRIMARY KEY,
			template_id VARCHAR(36) NOT NULL,
			task_code VARCHAR(50) NOT NULL,
			depends_on_task_code VARCHAR(50) NOT NULL,
			dependency_type VARCHAR(10) DEFAULT 'FS',
			lag_days INT DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ttd_template ON template_task_dependencies(template_id)`,
		`ALTER TABLE approval_definitions ADD COLUMN IF NOT EXISTS description TEXT`,
		`ALTER TABLE approval_definitions ADD COLUMN IF NOT EXISTS icon VARCHAR(50) DEFAULT 'approval'`,
		`ALTER TABLE approval_definitions ADD COLUMN IF NOT EXISTS group_name VARCHAR(50) NOT NULL DEFAULT '其他'`,
		`ALTER TABLE approval_definitions ADD COLUMN IF NOT EXISTS form_schema JSONB NOT NULL DEFAULT '[]'`,
		`ALTER TABLE approval_definitions ADD COLUMN IF NOT EXISTS flow_schema JSONB NOT NULL DEFAULT '{"nodes":[]}'`,
		`ALTER TABLE approval_definitions ADD COLUMN IF NOT EXISTS visibility VARCHAR(200) DEFAULT '全员'`,
		`ALTER TABLE approval_definitions ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'draft'`,
		`ALTER TABLE approval_definitions ADD COLUMN IF NOT EXISTS admin_user_id VARCHAR(32)`,
		`ALTER TABLE approval_definitions ADD COLUMN IF NOT EXISTS sort_order INT DEFAULT 0`,
		`ALTER TABLE approval_definitions ADD COLUMN IF NOT EXISTS created_by VARCHAR(32) DEFAULT ''`,
		`INSERT INTO approval_groups (id, name, sort_order) VALUES
			(gen_random_uuid(), '研发', 1),
			(gen_random_uuid(), '供应链', 2),
			(gen_random_uuid(), '财务', 3),
			(gen_random_uuid(), '人事', 4),
			(gen_random_uuid(), '行政', 5)
		ON CONFLICT (name) DO NOTHING`,
		// 给已有的 approval_requests 表增加字段
		`ALTER TABLE approval_requests ADD COLUMN IF NOT EXISTS definition_id VARCHAR(36)`,
		`ALTER TABLE approval_requests ADD COLUMN IF NOT EXISTS code VARCHAR(50)`,
		`ALTER TABLE approval_requests ADD COLUMN IF NOT EXISTS current_node INT DEFAULT 0`,
		`ALTER TABLE approval_requests ADD COLUMN IF NOT EXISTS flow_snapshot JSONB`,
		// 给已有的 approval_reviewers 表增加字段
		`ALTER TABLE approval_reviewers ADD COLUMN IF NOT EXISTS node_index INT DEFAULT 0`,
		`ALTER TABLE approval_reviewers ADD COLUMN IF NOT EXISTS node_name VARCHAR(100)`,
		`ALTER TABLE approval_reviewers ADD COLUMN IF NOT EXISTS review_type VARCHAR(20) DEFAULT 'approve'`,

		// V8: 任务角色分配字段
		`ALTER TABLE tasks ADD COLUMN IF NOT EXISTS default_assignee_role VARCHAR(50) DEFAULT ''`,

		// V10: project_bom_items 增加大厂BOM标准字段
		`ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS parent_item_id VARCHAR(32) REFERENCES project_bom_items(id)`,
		`ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS level INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS procurement_type VARCHAR(16) NOT NULL DEFAULT 'buy'`,
		`ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS extended_cost NUMERIC(15,4)`,
		`ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS supplier_pn VARCHAR(64)`,
		`ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS moq INTEGER`,
		`ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS approved_vendors JSONB`,
		`ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS lifecycle_status VARCHAR(16) DEFAULT 'active'`,

		// V9: 任务角色表（区别于权限角色 roles 表）
		`CREATE TABLE IF NOT EXISTS task_roles (
			id VARCHAR(36) PRIMARY KEY,
			code VARCHAR(50) NOT NULL UNIQUE,
			name VARCHAR(100) NOT NULL,
			is_system BOOLEAN NOT NULL DEFAULT false,
			sort_order INT NOT NULL DEFAULT 0,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,

		// V10: 智能路由 (Phase 4)
		`CREATE TABLE IF NOT EXISTS routing_rules (
			id VARCHAR(36) PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			entity_type VARCHAR(50) NOT NULL,
			event VARCHAR(100) NOT NULL,
			conditions JSONB NOT NULL,
			channel VARCHAR(20) NOT NULL,
			priority INT DEFAULT 0,
			action_config JSONB,
			enabled BOOLEAN DEFAULT true,
			description TEXT,
			created_by VARCHAR(32),
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_routing_rules_entity_event ON routing_rules(entity_type, event)`,
		`CREATE TABLE IF NOT EXISTS routing_logs (
			id VARCHAR(36) PRIMARY KEY,
			rule_id VARCHAR(36),
			rule_name VARCHAR(100),
			entity_type VARCHAR(50) NOT NULL,
			entity_id VARCHAR(50),
			event VARCHAR(100) NOT NULL,
			channel VARCHAR(20) NOT NULL,
			context JSONB,
			reason TEXT,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_routing_logs_entity ON routing_logs(entity_type, event)`,

		// V12: 结构BOM专属字段
		`ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS material_type VARCHAR(64)`,
		`ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS color VARCHAR(64)`,
		`ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS surface_treatment VARCHAR(128)`,
		`ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS process_type VARCHAR(32)`,
		`ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS drawing_no VARCHAR(64)`,
		`ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS drawing2d_file_id VARCHAR(32)`,
		`ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS drawing2d_file_name VARCHAR(256)`,
		`ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS drawing3d_file_id VARCHAR(32)`,
		`ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS drawing3d_file_name VARCHAR(256)`,
		`ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS weight_grams NUMERIC(10,2)`,
		`ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS target_price NUMERIC(15,4)`,
		`ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS tooling_estimate NUMERIC(15,2)`,
		`ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS cost_notes TEXT`,
		`ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS is_appearance_part BOOLEAN DEFAULT false`,
		`ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS assembly_method VARCHAR(32)`,
		`ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS tolerance_grade VARCHAR(32)`,

		// V11: BOM发布快照表（ERP对接）
		`CREATE TABLE IF NOT EXISTS bom_releases (
			id VARCHAR(36) PRIMARY KEY,
			bom_id VARCHAR(32) NOT NULL,
			project_id VARCHAR(32) NOT NULL,
			bom_type VARCHAR(16) NOT NULL,
			version VARCHAR(16) NOT NULL,
			snapshot_json JSONB NOT NULL,
			status VARCHAR(16) NOT NULL DEFAULT 'pending',
			synced_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_bom_releases_status ON bom_releases(status)`,
		`CREATE INDEX IF NOT EXISTS idx_bom_releases_bom ON bom_releases(bom_id)`,

		// CMF变体新字段（AutoMigrate因FK级联问题可能跳过）
		"ALTER TABLE bom_item_cmf_variants ADD COLUMN IF NOT EXISTS process_drawing_type VARCHAR(50)",
		"ALTER TABLE bom_item_cmf_variants ADD COLUMN IF NOT EXISTS process_drawings JSONB DEFAULT '[]'",

		// V17: PBOM字段 on project_bom_items
		"ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS print_process VARCHAR(50)",
		"ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS surface_finish_pkg VARCHAR(50)",
		"ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS design_file_id VARCHAR(100)",
		"ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS design_file_name VARCHAR(200)",
		"ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS die_cut_file_id VARCHAR(100)",
		"ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS die_cut_file_name VARCHAR(200)",
		"ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS is_multilang BOOLEAN DEFAULT false",
		"ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS packing_qty INT",

		// V17: 语言变体表 (兜底，AutoMigrate因FK可能跳过)
		`CREATE TABLE IF NOT EXISTS bom_item_lang_variants (
			id VARCHAR(32) PRIMARY KEY,
			bom_item_id VARCHAR(32) NOT NULL REFERENCES project_bom_items(id) ON DELETE CASCADE,
			variant_index INT NOT NULL DEFAULT 1,
			material_code VARCHAR(50),
			language_code VARCHAR(10) NOT NULL,
			language_name VARCHAR(50) NOT NULL,
			design_file_id VARCHAR(100),
			design_file_name VARCHAR(200),
			design_file_url VARCHAR(500),
			notes TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			UNIQUE(bom_item_id, variant_index)
		)`,

		// ECN redesign: 新增字段和表
		"ALTER TABLE ecns ADD COLUMN IF NOT EXISTS technical_plan TEXT",
		"ALTER TABLE ecns ADD COLUMN IF NOT EXISTS planned_date TIMESTAMP",
		"ALTER TABLE ecns ADD COLUMN IF NOT EXISTS completion_rate INT DEFAULT 0",
		"ALTER TABLE ecns ADD COLUMN IF NOT EXISTS approval_mode VARCHAR(16) DEFAULT 'serial'",
		"ALTER TABLE ecns ADD COLUMN IF NOT EXISTS sop_impact JSONB",
		"ALTER TABLE ecn_affected_items ADD COLUMN IF NOT EXISTS material_code VARCHAR(64)",
		"ALTER TABLE ecn_affected_items ADD COLUMN IF NOT EXISTS material_name VARCHAR(256)",
		"ALTER TABLE ecn_affected_items ADD COLUMN IF NOT EXISTS affected_bom_ids JSONB",
		`CREATE TABLE IF NOT EXISTS ecn_tasks (
			id VARCHAR(32) PRIMARY KEY,
			ecn_id VARCHAR(32) NOT NULL,
			type VARCHAR(32) NOT NULL,
			title VARCHAR(256) NOT NULL,
			description TEXT,
			assignee_id VARCHAR(32),
			due_date TIMESTAMP,
			status VARCHAR(16) NOT NULL DEFAULT 'pending',
			completed_at TIMESTAMP,
			completed_by VARCHAR(32),
			metadata JSONB,
			sort_order INT NOT NULL DEFAULT 0,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		"CREATE INDEX IF NOT EXISTS idx_ecn_tasks_ecn_id ON ecn_tasks(ecn_id)",
		`CREATE TABLE IF NOT EXISTS ecn_histories (
			id VARCHAR(32) PRIMARY KEY,
			ecn_id VARCHAR(32) NOT NULL,
			action VARCHAR(32) NOT NULL,
			user_id VARCHAR(32) NOT NULL,
			detail JSONB,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		"CREATE INDEX IF NOT EXISTS idx_ecn_histories_ecn_id ON ecn_histories(ecn_id)",

		// EBOM专用字段 — GORM AutoMigrate may skip these on FK tables
		"ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS item_type VARCHAR(20) DEFAULT 'component'",
		"ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS designator VARCHAR(500)",
		"ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS package VARCHAR(50)",
		"ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS pcb_layers INT",
		"ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS pcb_thickness VARCHAR(20)",
		"ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS pcb_material VARCHAR(50)",
		"ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS pcb_dimensions VARCHAR(50)",
		"ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS pcb_surface_finish VARCHAR(50)",
		"ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS service_type VARCHAR(50)",
		"ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS process_requirements TEXT",
		"ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS attachments JSONB DEFAULT '[]'",
		// PBOM language_code field
		"ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS language_code VARCHAR(20)",
		// Update bom_type CHECK constraint to include PBOM and MBOM
		"ALTER TABLE project_boms DROP CONSTRAINT IF EXISTS ck_bom_type",
		"ALTER TABLE project_boms ADD CONSTRAINT ck_bom_type CHECK (bom_type IN ('EBOM','SBOM','PBOM','MBOM','OBOM','FWBOM'))",

		// V18: 三级BOM重构 — 新增列
		"ALTER TABLE project_boms ADD COLUMN IF NOT EXISTS source_bom_id VARCHAR(32)",
		"ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS category VARCHAR(32) DEFAULT '结构件'",
		"ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS sub_category VARCHAR(64)",
		"ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS extended_attrs JSONB DEFAULT '{}'",
		"ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS process_step_id VARCHAR(32)",
		"ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS scrap_rate NUMERIC(5,2)",
		"ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS effective_date DATE",
		"ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS expire_date DATE",
		"ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS is_alternative BOOLEAN DEFAULT false",
		"ALTER TABLE project_bom_items ADD COLUMN IF NOT EXISTS alternative_for VARCHAR(32)",

		// V18: 属性模板表
		`CREATE TABLE IF NOT EXISTS category_attr_templates (
			id VARCHAR(32) PRIMARY KEY,
			category VARCHAR(32) NOT NULL,
			sub_category VARCHAR(64) NOT NULL,
			field_key VARCHAR(64) NOT NULL,
			field_name VARCHAR(64) NOT NULL,
			field_type VARCHAR(16) NOT NULL DEFAULT 'text',
			unit VARCHAR(16),
			required BOOLEAN DEFAULT false,
			options JSONB,
			validation JSONB,
			default_value VARCHAR(128),
			sort_order INT DEFAULT 0,
			show_in_table BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			UNIQUE(category, sub_category, field_key)
		)`,

		// V18: 工艺路线表
		`CREATE TABLE IF NOT EXISTS process_routes (
			id VARCHAR(32) PRIMARY KEY,
			project_id VARCHAR(32) NOT NULL,
			bom_id VARCHAR(32),
			name VARCHAR(128) NOT NULL,
			description TEXT,
			version VARCHAR(16) DEFAULT 'v1.0',
			status VARCHAR(16) DEFAULT 'draft',
			total_steps INT DEFAULT 0,
			created_by VARCHAR(32),
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		"CREATE INDEX IF NOT EXISTS idx_process_routes_project ON process_routes(project_id)",
		"CREATE INDEX IF NOT EXISTS idx_process_routes_bom ON process_routes(bom_id)",
		"ALTER TABLE process_routes ADD COLUMN IF NOT EXISTS total_steps INT DEFAULT 0",

		// V18: 工序表
		`CREATE TABLE IF NOT EXISTS process_steps (
			id VARCHAR(32) PRIMARY KEY,
			route_id VARCHAR(32) NOT NULL REFERENCES process_routes(id) ON DELETE CASCADE,
			step_number INT NOT NULL,
			name VARCHAR(128) NOT NULL,
			description TEXT,
			equipment VARCHAR(128),
			cycle_time_seconds INT,
			setup_time_minutes INT,
			quality_checks TEXT,
			sort_order INT DEFAULT 0,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		"CREATE INDEX IF NOT EXISTS idx_process_steps_route ON process_steps(route_id)",

		// V18: 工序物料表
		`CREATE TABLE IF NOT EXISTS process_step_materials (
			id VARCHAR(32) PRIMARY KEY,
			step_id VARCHAR(32) NOT NULL REFERENCES process_steps(id) ON DELETE CASCADE,
			material_id VARCHAR(32),
			bom_item_id VARCHAR(32),
			name VARCHAR(128) NOT NULL,
			category VARCHAR(32) NOT NULL DEFAULT 'material',
			quantity NUMERIC(15,4) DEFAULT 1,
			unit VARCHAR(16) DEFAULT 'pcs',
			notes TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		"CREATE INDEX IF NOT EXISTS idx_process_step_materials_step ON process_step_materials(step_id)",
		"ALTER TABLE process_step_materials ADD COLUMN IF NOT EXISTS category VARCHAR(32) NOT NULL DEFAULT 'material'",

		// V18: Migrate is_appearance_part and is_variant from fixed columns to extended_attrs JSONB
		`UPDATE project_bom_items SET extended_attrs = COALESCE(extended_attrs, '{}'::jsonb) || jsonb_build_object('is_appearance_part', COALESCE(is_appearance_part, false)) WHERE is_appearance_part = true AND (extended_attrs IS NULL OR NOT (extended_attrs ? 'is_appearance_part'))`,
		`UPDATE project_bom_items SET extended_attrs = COALESCE(extended_attrs, '{}'::jsonb) || jsonb_build_object('is_variant', COALESCE(is_variant, false)) WHERE is_variant = true AND (extended_attrs IS NULL OR NOT (extended_attrs ? 'is_variant'))`,

		// V19: Migrate specification, reference, manufacturer, manufacturer_pn, supplier_pn, lead_time_days, drawing_no, is_critical to extended_attrs
		`UPDATE project_bom_items SET extended_attrs = COALESCE(extended_attrs, '{}'::jsonb) || jsonb_build_object('specification', specification) WHERE specification IS NOT NULL AND specification != '' AND (extended_attrs IS NULL OR NOT (extended_attrs ? 'specification'))`,
		`UPDATE project_bom_items SET extended_attrs = COALESCE(extended_attrs, '{}'::jsonb) || jsonb_build_object('reference', reference) WHERE reference IS NOT NULL AND reference != '' AND (extended_attrs IS NULL OR NOT (extended_attrs ? 'reference'))`,
		`UPDATE project_bom_items SET extended_attrs = COALESCE(extended_attrs, '{}'::jsonb) || jsonb_build_object('manufacturer', manufacturer) WHERE manufacturer IS NOT NULL AND manufacturer != '' AND (extended_attrs IS NULL OR NOT (extended_attrs ? 'manufacturer'))`,
		`UPDATE project_bom_items SET extended_attrs = COALESCE(extended_attrs, '{}'::jsonb) || jsonb_build_object('manufacturer_pn', manufacturer_pn) WHERE manufacturer_pn IS NOT NULL AND manufacturer_pn != '' AND (extended_attrs IS NULL OR NOT (extended_attrs ? 'manufacturer_pn'))`,
		`UPDATE project_bom_items SET extended_attrs = COALESCE(extended_attrs, '{}'::jsonb) || jsonb_build_object('supplier_pn', supplier_pn) WHERE supplier_pn IS NOT NULL AND supplier_pn != '' AND (extended_attrs IS NULL OR NOT (extended_attrs ? 'supplier_pn'))`,
		`UPDATE project_bom_items SET extended_attrs = COALESCE(extended_attrs, '{}'::jsonb) || jsonb_build_object('lead_time_days', lead_time_days) WHERE lead_time_days IS NOT NULL AND (extended_attrs IS NULL OR NOT (extended_attrs ? 'lead_time_days'))`,
		`UPDATE project_bom_items SET extended_attrs = COALESCE(extended_attrs, '{}'::jsonb) || jsonb_build_object('drawing_no', drawing_no) WHERE drawing_no IS NOT NULL AND drawing_no != '' AND (extended_attrs IS NULL OR NOT (extended_attrs ? 'drawing_no'))`,
		`UPDATE project_bom_items SET extended_attrs = COALESCE(extended_attrs, '{}'::jsonb) || jsonb_build_object('is_critical', true) WHERE is_critical = true AND (extended_attrs IS NULL OR NOT (extended_attrs ? 'is_critical'))`,

		// V19: Clear old seed templates so new comprehensive seeds get applied
		`DELETE FROM category_attr_templates`,

		// V20: Add bom_type column to category_attr_templates for EBOM/PBOM grouping
		"ALTER TABLE category_attr_templates ADD COLUMN IF NOT EXISTS bom_type VARCHAR(16) NOT NULL DEFAULT 'EBOM'",
		// V20: Re-clear templates for updated seeds with bom_type + drawing fields
		`DELETE FROM category_attr_templates`,

		// V21: BOM版本控制字段
		"ALTER TABLE project_boms ADD COLUMN IF NOT EXISTS version_major INT DEFAULT 0",
		"ALTER TABLE project_boms ADD COLUMN IF NOT EXISTS version_minor INT DEFAULT 0",
		"ALTER TABLE project_boms ADD COLUMN IF NOT EXISTS released_at TIMESTAMP",
		"ALTER TABLE project_boms ADD COLUMN IF NOT EXISTS released_by VARCHAR(32)",
		"ALTER TABLE project_boms ADD COLUMN IF NOT EXISTS source_version VARCHAR(20) DEFAULT ''",
		"ALTER TABLE project_boms ADD COLUMN IF NOT EXISTS release_note TEXT DEFAULT ''",
		// Backfill existing 'published' status to 'released'
		"UPDATE project_boms SET status = 'released' WHERE status = 'published'",
		// Backfill version_major/version_minor from existing version strings (e.g. 'v1.0')
		`UPDATE project_boms SET version_major = 1, version_minor = 0 WHERE version = 'v1.0' AND version_major = 0`,

		// V22: Merge structural sub-categories housing+internal → structural_part
		`UPDATE project_bom_items SET sub_category = 'structural_part' WHERE sub_category IN ('housing', 'internal') AND category = 'structural'`,
		`UPDATE category_attr_templates SET sub_category = 'structural_part' WHERE sub_category IN ('housing', 'internal') AND category = 'structural'`,
		// Delete templates to re-seed with merged structural_part fields
		`DELETE FROM category_attr_templates`,
	}
	for _, sql := range migrationSQL {
		if err := db.Exec(sql).Error; err != nil {
			zapLogger.Warn("Migration SQL warning (may already exist)", zap.String("sql", sql), zap.Error(err))
		}
	}
	zapLogger.Info("Database migration completed")

	// Seed: 默认PLM角色
	roleSeeds := []struct{ Code, Name string }{
		{"hw_engineer", "硬件工程师"},
		{"sw_engineer", "软件工程师"},
		{"me_engineer", "结构工程师"},
		{"qa_engineer", "质量工程师"},
		{"pm", "项目经理"},
		{"id_designer", "工业设计师"},
		{"te_engineer", "测试工程师"},
		{"supply_chain", "供应链"},
	}
	for _, rs := range roleSeeds {
		db.Exec(`INSERT INTO roles (id, code, name, status, is_system, created_at, updated_at)
			VALUES (gen_random_uuid(), ?, ?, 'active', true, NOW(), NOW())
			ON CONFLICT (code) DO NOTHING`, rs.Code, rs.Name)
	}

	// Seed: 预设任务角色（用于模板任务分配）
	taskRoleSeeds := []struct{ Code, Name string; Sort int }{
		{"pm", "项目经理", 1},
		{"hw_engineer", "硬件工程师", 2},
		{"sw_engineer", "软件工程师", 3},
		{"struct_engineer", "结构工程师", 4},
		{"test_engineer", "测试工程师", 5},
		{"qa_engineer", "品质工程师", 6},
		{"procurement", "采购", 7},
		{"id_engineer", "ID工程师", 8},
	}
	for _, rs := range taskRoleSeeds {
		db.Exec(`INSERT INTO task_roles (id, code, name, is_system, sort_order, created_at, updated_at)
			VALUES (gen_random_uuid(), ?, ?, true, ?, NOW(), NOW())
			ON CONFLICT (code) DO NOTHING`, rs.Code, rs.Name, rs.Sort)
	}

	// 初始化Redis
	rdb := initRedis(cfg.Redis)

	// 初始化依赖
	repos := repository.NewRepositories(db)
	services := service.NewServices(repos, rdb, cfg)

	// 初始化状态机引擎 (Phase 3)
	stateEngine := engine.NewEngine(db, nil)
	plmTaskMachine := engine.NewPLMTaskMachine()
	if err := stateEngine.RegisterMachine(plmTaskMachine); err != nil {
		zapLogger.Warn("Failed to register PLM task state machine", zap.Error(err))
	}

	// 初始化飞书客户端 (Phase 3 — 工作流用)
	var feishuWorkflowClient *feishu.FeishuClient
	feishuAppID := cfg.Feishu.AppID
	feishuAppSecret := cfg.Feishu.AppSecret
	if envID := os.Getenv("FEISHU_APP_ID"); envID != "" {
		feishuAppID = envID
	}
	if envSecret := os.Getenv("FEISHU_APP_SECRET"); envSecret != "" {
		feishuAppSecret = envSecret
	}
	if feishuAppID != "" && feishuAppSecret != "" {
		feishuWorkflowClient = feishu.NewClient(feishuAppID, feishuAppSecret)
		zapLogger.Info("Feishu workflow client initialized")
	}

	// 初始化工作流服务 (Phase 3)
	workflowSvc := service.NewWorkflowService(db, stateEngine, feishuWorkflowClient, repos.Project, repos.Task)

	// 初始化审批服务 (V4)
	approvalSvc := service.NewApprovalService(db, feishuWorkflowClient)

	// 初始化通讯录同步服务 (V4)
	contactSyncSvc := service.NewContactSyncService(db, feishuWorkflowClient)

	handlers := handler.NewHandlers(services, repos, cfg, workflowSvc)

	// V4: 设置审批和管理员处理器
	handlers.Approval = handler.NewApprovalHandler(approvalSvc)
	handlers.Admin = handler.NewAdminHandler(contactSyncSvc)

	// V5: 审批定义服务
	approvalDefSvc := service.NewApprovalDefinitionService(db, feishuWorkflowClient, approvalSvc)
	handlers.ApprovalDef = handler.NewApprovalDefinitionHandler(approvalDefSvc)

	// V8: 角色管理 + 注入审批服务到项目服务
	handlers.Role = handler.NewRoleHandler(db, feishuWorkflowClient)
	services.Project.SetApprovalService(approvalSvc)
	services.Project.SetBOMService(services.ProjectBOM)
	services.Project.SetFeishuClient(feishuWorkflowClient, repos.User)
	approvalSvc.SetProjectService(services.Project)
	services.Template.SetProjectService(services.Project)

	// V9: 智能路由 (Phase 4)
	routingSvc := service.NewRoutingService(db)
	handlers.Routing = handler.NewRoutingHandler(routingSvc)
	workflowSvc.SetRoutingService(routingSvc)
	workflowSvc.SetTaskFormRepo(repos.TaskForm)
	workflowSvc.SetBOMRepo(repos.ProjectBOM)

	// Backfill: 为已有BOM items自动创建缺失的物料
	services.ProjectBOM.BackfillMaterials(context.Background())

	// Seed: BOM属性模板
	services.ProjectBOM.SeedDefaultTemplates(context.Background())

	// V13: CMF控件 (now initialized in NewHandlers via service)

	// === SRM模块初始化 ===
	// AutoMigrate SRM实体
	if err := db.AutoMigrate(
		&srmentity.Supplier{},
		&srmentity.SupplierContact{},
		&srmentity.SupplierMaterial{},
		&srmentity.PurchaseRequest{},
		&srmentity.PRItem{},
		&srmentity.PurchaseOrder{},
		&srmentity.POItem{},
		&srmentity.Inspection{},
		&srmentity.CorrectiveAction{},
		&srmentity.Settlement{},
		&srmentity.SettlementDispute{},
		&srmentity.SRMProject{},
		&srmentity.ActivityLog{},
		&srmentity.DelayRequest{},
		&srmentity.SupplierEvaluation{},
		&srmentity.Equipment{},
		&srmentity.RFQ{},
		&srmentity.RFQQuote{},
		&srmentity.SamplingRequest{},
		&srmentity.InspectionItem{},
		&srmentity.InventoryRecord{},
		&srmentity.InventoryTransaction{},
	); err != nil {
		zapLogger.Warn("AutoMigrate SRM tables warning", zap.Error(err))
	}

	// V14: PRItem增加物料分类/展示增强/治具字段
	v14SQL := []string{
		"ALTER TABLE srm_pr_items ADD COLUMN IF NOT EXISTS source_bom_type VARCHAR(20)",
		"ALTER TABLE srm_pr_items ADD COLUMN IF NOT EXISTS material_group VARCHAR(20)",
		"ALTER TABLE srm_pr_items ADD COLUMN IF NOT EXISTS image_url VARCHAR(500)",
		"ALTER TABLE srm_pr_items ADD COLUMN IF NOT EXISTS process_type VARCHAR(100)",
		"ALTER TABLE srm_pr_items ADD COLUMN IF NOT EXISTS tooling_cost DECIMAL(12,2)",
		"ALTER TABLE srm_pr_items ADD COLUMN IF NOT EXISTS tooling_status VARCHAR(20)",
		"ALTER TABLE srm_pr_items ADD COLUMN IF NOT EXISTS jig_phase VARCHAR(20)",
		"ALTER TABLE srm_pr_items ADD COLUMN IF NOT EXISTS jig_progress INT DEFAULT 0",
	}
	for _, sql := range v14SQL {
		if err := db.Exec(sql).Error; err != nil {
			zapLogger.Warn("V14 migration warning", zap.String("sql", sql), zap.Error(err))
		}
	}
	// V15: RFQ询价单 + PR物料状态扩展
	v15SQL := []string{
		`CREATE TABLE IF NOT EXISTS srm_rfqs (
			id VARCHAR(32) PRIMARY KEY,
			code VARCHAR(32) NOT NULL UNIQUE,
			srm_project_id VARCHAR(32) NOT NULL,
			pr_item_id VARCHAR(32) NOT NULL,
			material_name VARCHAR(200),
			status VARCHAR(20) DEFAULT 'draft',
			deadline TIMESTAMP,
			created_by VARCHAR(32),
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_srm_rfqs_project ON srm_rfqs(srm_project_id)`,
		`CREATE INDEX IF NOT EXISTS idx_srm_rfqs_pr_item ON srm_rfqs(pr_item_id)`,
		`CREATE TABLE IF NOT EXISTS srm_rfq_quotes (
			id VARCHAR(32) PRIMARY KEY,
			rfq_id VARCHAR(32) NOT NULL,
			supplier_id VARCHAR(32) NOT NULL,
			supplier_name VARCHAR(200),
			unit_price DECIMAL(12,4),
			currency VARCHAR(10) DEFAULT 'CNY',
			moq INT,
			lead_time_days INT,
			tooling_cost DECIMAL(12,2),
			sample_cost DECIMAL(12,2),
			validity VARCHAR(50),
			notes TEXT,
			is_selected BOOLEAN DEFAULT false,
			quoted_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_srm_rfq_quotes_rfq ON srm_rfq_quotes(rfq_id)`,
		"ALTER TABLE srm_pr_items ADD COLUMN IF NOT EXISTS rfq_id VARCHAR(32)",
	}
	for _, sql := range v15SQL {
		if err := db.Exec(sql).Error; err != nil {
			zapLogger.Warn("V15 migration warning", zap.String("sql", sql), zap.Error(err))
		}
	}
	// V16: POItem增加BOM行项关联字段
	v16SQL := []string{
		"ALTER TABLE srm_po_items ADD COLUMN IF NOT EXISTS bom_item_id VARCHAR(32)",
	}
	for _, sql := range v16SQL {
		if err := db.Exec(sql).Error; err != nil {
			zapLogger.Warn("V16 migration warning", zap.String("sql", sql), zap.Error(err))
		}
	}
	// V17: Inspection增强字段 + 库存表
	v17SQL := []string{
		"ALTER TABLE srm_inspections ADD COLUMN IF NOT EXISTS overall_result VARCHAR(20)",
		"ALTER TABLE srm_inspections ADD COLUMN IF NOT EXISTS inspector VARCHAR(100)",
		"ALTER TABLE srm_inspections ADD COLUMN IF NOT EXISTS inspection_date TIMESTAMP",
	}
	for _, sql := range v17SQL {
		if err := db.Exec(sql).Error; err != nil {
			zapLogger.Warn("V17 migration warning", zap.String("sql", sql), zap.Error(err))
		}
	}
	zapLogger.Info("SRM database migration completed (including V17)")

	// SRM仓库和服务
	srmRepos := srmrepo.NewRepositories(db)
	srmSupplierSvc := srmsvc.NewSupplierService(srmRepos.Supplier)
	srmProcurementSvc := srmsvc.NewProcurementService(srmRepos.PR, srmRepos.PO, db)
	srmInspectionSvc := srmsvc.NewInspectionService(srmRepos.Inspection, srmRepos.PR)
	srmInventorySvc := srmsvc.NewInventoryService(srmRepos.Inventory)
	srmInspectionSvc.SetPORepo(srmRepos.PO)
	srmInspectionSvc.SetInventoryService(srmInventorySvc)
	srmDashboardSvc := srmsvc.NewDashboardService(db)
	srmProjectSvc := srmsvc.NewSRMProjectService(srmRepos.Project, srmRepos.PR, srmRepos.ActivityLog, srmRepos.DelayRequest, db)
	srmSettlementSvc := srmsvc.NewSettlementService(srmRepos.Settlement)
	srmCorrectiveActionSvc := srmsvc.NewCorrectiveActionService(srmRepos.CorrectiveAction, srmRepos.Inspection)
	srmEvaluationSvc := srmsvc.NewEvaluationService(srmRepos.Evaluation)
	srmEvaluationSvc.SetSupplierRepo(srmRepos.Supplier)
	srmEquipmentSvc := srmsvc.NewEquipmentService(srmRepos.Equipment)
	srmRFQSvc := srmsvc.NewRFQService(srmRepos.RFQ, srmRepos.PO, srmRepos.PR, srmRepos.ActivityLog, db)
	srmPRItemSvc := srmsvc.NewPRItemService(srmRepos.PR, srmRepos.Project, srmRepos.ActivityLog, db)
	srmSamplingSvc := srmsvc.NewSamplingService(srmRepos.Sampling, srmRepos.PR, srmRepos.Supplier, srmRepos.ActivityLog, db)
	srmHandlers := srmhandler.NewHandlers(srmSupplierSvc, srmProcurementSvc, srmInspectionSvc, srmDashboardSvc, srmRepos.PO, srmProjectSvc, srmSettlementSvc, srmCorrectiveActionSvc, srmEvaluationSvc, srmEquipmentSvc, srmRFQSvc, srmPRItemSvc, srmSamplingSvc)
	srmHandlers.Inventory = srmhandler.NewInventoryHandler(srmInventorySvc)

	// SRM→飞书：注入飞书客户端到SRM各服务
	if feishuWorkflowClient != nil {
		srmProcurementSvc.SetFeishuClient(feishuWorkflowClient)
		srmInspectionSvc.SetFeishuClient(feishuWorkflowClient)
		srmCorrectiveActionSvc.SetFeishuClient(feishuWorkflowClient)
		srmSamplingSvc.SetFeishuClient(feishuWorkflowClient)
	}

	// 工作流→SRM集成：采购控件自动创建PR
	workflowSvc.SetSRMProcurementService(srmProcurementSvc)

	// 设置Gin模式
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建路由
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.Logger(zapLogger))
	router.Use(middleware.CORS())
	router.Use(middleware.RequestID())
	router.Use(gzip.Gzip(gzip.DefaultCompression))

	// 注册路由
	registerRoutes(router, handlers, services, cfg, srmHandlers)

	// 创建HTTP服务器
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: 0, // Disable for SSE long-lived connections
	}

	// 启动服务器
	go func() {
		zapLogger.Info("Server starting", zap.Int("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zapLogger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	zapLogger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		zapLogger.Error("Server forced to shutdown", zap.Error(err))
	}

	zapLogger.Info("Server exited")
}

func initLogger(cfg config.LogConfig) (*zap.Logger, error) {
	var zapCfg zap.Config

	if cfg.Format == "json" {
		zapCfg = zap.NewProductionConfig()
	} else {
		zapCfg = zap.NewDevelopmentConfig()
	}

	switch cfg.Level {
	case "debug":
		zapCfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		zapCfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		zapCfg.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		zapCfg.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	}

	return zapCfg.Build()
}

func initDatabase(cfg config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	return db, nil
}

func initRedis(cfg config.RedisConfig) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})
}

func registerRoutes(r *gin.Engine, h *handler.Handlers, svc *service.Services, cfg *config.Config, srmH *srmhandler.Handlers) {
	// 健康检查
	r.GET("/health/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/health/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 版本信息
	r.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"version":    Version,
			"build_time": BuildTime,
		})
	})

	// 静态文件服务 - 上传文件
	r.Static("/uploads", "./uploads")

	// 静态文件服务 - 前端 (hashed filenames → immutable cache)
	r.Use(func(c *gin.Context) {
		if len(c.Request.URL.Path) > 8 && c.Request.URL.Path[:8] == "/assets/" {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		}
		c.Next()
	})
	r.Static("/assets", "./web/plm/assets")
	r.StaticFile("/logo.svg", "./web/plm/logo.svg")
	r.StaticFile("/vite.svg", "./web/plm/vite.svg")

	// SPA 路由回退 - 所有非 API 路由返回 index.html
	r.NoRoute(func(c *gin.Context) {
		// 如果是 API 请求，返回 404
		if len(c.Request.URL.Path) > 4 && c.Request.URL.Path[:5] == "/api/" {
			c.JSON(http.StatusNotFound, gin.H{"code": 40400, "message": "Not found"})
			return
		}
		indexData, err := os.ReadFile("./web/plm/index.html")
		if err != nil {
			c.String(http.StatusInternalServerError, "index.html not found")
			return
		}
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexData)
	})

	// API v1
	v1 := r.Group("/api/v1")
	{
		// 认证 (无需登录)
		auth := v1.Group("/auth")
		{
			auth.GET("/feishu/login", h.Auth.FeishuLogin)
			auth.GET("/feishu/callback", h.Auth.FeishuCallback)
			auth.POST("/refresh", h.Auth.RefreshToken)
		}

		// Webhook路由（无需认证，飞书回调使用）
		webhooks := v1.Group("/webhooks")
		{
			webhooks.POST("/feishu/approval", handleFeishuApprovalWebhook)
			webhooks.POST("/feishu/event", handleFeishuEventVerification)
		}

		// SSE 实时推送（需要认证，支持 query param token）
		sseGroup := v1.Group("/sse")
		sseGroup.Use(middleware.JWTAuth(cfg.JWT.Secret))
		{
			sseGroup.GET("/events", h.SSE.Stream)
		}

		// 需要认证的接口
		authorized := v1.Group("")
		authorized.Use(middleware.JWTAuth(cfg.JWT.Secret))
		{
			// 当前用户
			authorized.GET("/auth/me", h.Auth.GetCurrentUser)
			authorized.POST("/auth/logout", h.Auth.Logout)

			// 用户管理
			users := authorized.Group("/users")
			{
				users.GET("", h.User.List)
				users.GET("/search", h.User.Search)
				users.GET("/:id", h.User.Get)
			}

			// V4: 管理员操作
			admin := authorized.Group("/admin")
			{
				admin.POST("/sync-contacts", h.Admin.SyncContacts)
			}

			// V4: 审批
			approvals := authorized.Group("/approvals")
			{
				approvals.POST("", h.Approval.Create)
				approvals.GET("", h.Approval.List)
				approvals.GET("/:id", h.Approval.Get)
				approvals.POST("/:id/approve", h.Approval.Approve)
				approvals.POST("/:id/reject", h.Approval.Reject)
			}

			// V5: 审批定义管理
			approvalDefs := authorized.Group("/approval-definitions")
			{
				approvalDefs.GET("", h.ApprovalDef.ListDefinitions)
				approvalDefs.POST("", h.ApprovalDef.CreateDefinition)
				approvalDefs.GET("/:id", h.ApprovalDef.GetDefinition)
				approvalDefs.PUT("/:id", h.ApprovalDef.UpdateDefinition)
				approvalDefs.DELETE("/:id", h.ApprovalDef.DeleteDefinition)
				approvalDefs.POST("/:id/publish", h.ApprovalDef.PublishDefinition)
				approvalDefs.POST("/:id/unpublish", h.ApprovalDef.UnpublishDefinition)
				approvalDefs.POST("/:id/submit", h.ApprovalDef.SubmitInstance)
			}

			// V5: 审批分组管理
			approvalGroups := authorized.Group("/approval-groups")
			{
				approvalGroups.GET("", h.ApprovalDef.ListGroups)
				approvalGroups.POST("", h.ApprovalDef.CreateGroup)
				approvalGroups.DELETE("/:id", h.ApprovalDef.DeleteGroup)
			}

			// V8: 角色管理
			roles := authorized.Group("/roles")
			{
				roles.GET("", h.Role.List)
				roles.POST("", h.Role.Create)
				roles.GET("/:id", h.Role.Get)
				roles.PUT("/:id", h.Role.Update)
				roles.DELETE("/:id", h.Role.Delete)
				roles.GET("/:id/members", h.Role.ListMembers)
				roles.POST("/:id/members", h.Role.AddMembers)
				roles.DELETE("/:id/members", h.Role.RemoveMembers)
			}

			// 部门树（角色成员选择用）
			authorized.GET("/departments", h.Role.ListDepartments)

			// V9: 智能路由
			if h.Routing != nil {
				routingRules := authorized.Group("/routing-rules")
				{
					routingRules.GET("", h.Routing.ListRules)
					routingRules.POST("", h.Routing.CreateRule)
					routingRules.POST("/test", h.Routing.TestRoute)
					routingRules.GET("/:id", h.Routing.GetRule)
					routingRules.PUT("/:id", h.Routing.UpdateRule)
					routingRules.DELETE("/:id", h.Routing.DeleteRule)
				}
				authorized.GET("/routing-logs", h.Routing.ListLogs)
			}

			// Phase 2: BOM导入模板下载
			authorized.GET("/bom-template", h.ProjectBOM.DownloadTemplate)
			authorized.GET("/bom-items/search", h.ProjectBOM.SearchItems)
			authorized.GET("/bom-items/search-paginated", h.ProjectBOM.SearchItemsPaginated)
			authorized.GET("/bom-items/global", h.ProjectBOM.GlobalSearch)

			// V18: 属性模板管理
			bomTemplates := authorized.Group("/bom-attr-templates")
			{
				bomTemplates.GET("", h.ProjectBOM.ListTemplates)
				bomTemplates.POST("", h.ProjectBOM.CreateTemplate)
				bomTemplates.PUT("/:id", h.ProjectBOM.UpdateTemplate)
				bomTemplates.DELETE("/:id", h.ProjectBOM.DeleteTemplate)
				bomTemplates.POST("/seed", h.ProjectBOM.SeedTemplates)
			}

			// BOM解析预览（不保存）
			authorized.POST("/bom/parse", h.ProjectBOM.ParseBOM)

			// Phase 3: BOM版本对比
			authorized.GET("/bom-compare", h.ProjectBOM.CompareBOMs)

			// Phase 4: ERP对接
			erp := authorized.Group("/erp")
			{
				erp.GET("/bom-releases", h.ProjectBOM.ListBOMReleases)
				erp.POST("/bom-releases/:id/ack", h.ProjectBOM.AckBOMRelease)
			}

			// 任务角色（用于模板任务分配）
			authorized.GET("/task-roles", h.Role.ListTaskRoles)

			// 飞书角色（部门列表）
			authorized.GET("/feishu/roles", h.Role.ListFeishuRoles)

			// 产品管理
			products := authorized.Group("/products")
			{
				products.GET("", h.Product.List)
				products.POST("", h.Product.Create)
				products.GET("/:id", h.Product.Get)
				products.PUT("/:id", h.Product.Update)
				products.DELETE("/:id", h.Product.Delete)
				products.POST("/:id/release", h.Product.Release)

				// BOM
				products.GET("/:id/bom", h.BOM.Get)
				products.GET("/:id/bom/versions", h.BOM.ListVersions)
				products.POST("/:id/bom/items", h.BOM.AddItem)
				products.PUT("/:id/bom/items/:itemId", h.BOM.UpdateItem)
				products.DELETE("/:id/bom/items/:itemId", h.BOM.DeleteItem)
				products.POST("/:id/bom/release", h.BOM.Release)
				products.GET("/:id/bom/compare", h.BOM.Compare)
			}

			// 产品类别
			authorized.GET("/product-categories", h.Product.ListCategories)

			// 物料管理
			materials := authorized.Group("/materials")
			{
				materials.GET("", h.Material.List)
				materials.POST("", h.Material.Create)
				materials.GET("/:id", h.Material.Get)
				materials.PUT("/:id", h.Material.Update)
			}

			// 物料类别
			authorized.GET("/material-categories", h.Material.ListCategories)

			// 项目管理
			projects := authorized.Group("/projects")
			{
				projects.GET("", h.Project.ListProjects)
				projects.POST("", h.Project.CreateProject)
				projects.GET("/:id", h.Project.GetProject)
				projects.PUT("/:id", h.Project.UpdateProject)
				projects.DELETE("/:id", h.Project.DeleteProject)
				projects.PUT("/:id/status", h.Project.UpdateProjectStatus)

				// 项目级角色分配
				projects.POST("/:id/assign-roles", h.Project.AssignRoles)

				// 项目阶段
				projects.GET("/:id/phases", h.Project.ListPhases)
				projects.PUT("/:id/phases/:phaseId/status", h.Project.UpdatePhaseStatus)

				// 项目任务
				projects.GET("/:id/tasks", h.Project.ListTasks)
				projects.POST("/:id/tasks", h.Project.CreateTask)
				projects.GET("/:id/tasks/:taskId", h.Project.GetTask)
				projects.PUT("/:id/tasks/:taskId", h.Project.UpdateTask)
				projects.DELETE("/:id/tasks/:taskId", h.Project.DeleteTask)
				projects.PUT("/:id/tasks/:taskId/status", h.Project.UpdateTaskStatus)
				projects.GET("/:id/tasks/:taskId/subtasks", h.Project.ListSubTasks)
				projects.GET("/:id/tasks/:taskId/comments", h.Project.ListTaskComments)
				projects.POST("/:id/tasks/:taskId/comments", h.Project.AddTaskComment)
				projects.GET("/:id/tasks/:taskId/dependencies", h.Project.ListTaskDependencies)
				projects.POST("/:id/tasks/:taskId/dependencies", h.Project.AddTaskDependency)
				projects.DELETE("/:id/tasks/:taskId/dependencies/:depId", h.Project.RemoveTaskDependency)
				projects.GET("/:id/overdue-tasks", h.Project.GetOverdueTasks)

				// V6: 任务表单
				projects.GET("/:id/tasks/:taskId/form", h.TaskForm.GetTaskForm)
				projects.PUT("/:id/tasks/:taskId/form", h.TaskForm.UpsertTaskForm)
				projects.GET("/:id/tasks/:taskId/form/submission", h.TaskForm.GetFormSubmission)

				// V6: 任务确认/驳回
				projects.POST("/:id/tasks/:taskId/confirm", h.Project.ConfirmTask)
				projects.POST("/:id/tasks/:taskId/reject", h.Project.RejectTask)

				// V2: 项目BOM管理
				projects.GET("/:id/bom-permissions", h.ProjectBOM.GetBOMPermissions)
				projects.GET("/:id/boms", h.ProjectBOM.ListBOMs)
				projects.POST("/:id/boms", h.ProjectBOM.CreateBOM)
				projects.GET("/:id/boms/:bomId", h.ProjectBOM.GetBOM)
				projects.PUT("/:id/boms/:bomId", h.ProjectBOM.UpdateBOM)
				projects.DELETE("/:id/boms/:bomId", h.ProjectBOM.DeleteBOM)
				projects.POST("/:id/boms/:bomId/submit", h.ProjectBOM.SubmitBOM)
				projects.POST("/:id/boms/:bomId/approve", h.ProjectBOM.ApproveBOM)
				projects.POST("/:id/boms/:bomId/reject", h.ProjectBOM.RejectBOM)
				projects.POST("/:id/boms/:bomId/freeze", h.ProjectBOM.FreezeBOM)
				projects.POST("/:id/boms/:bomId/items", h.ProjectBOM.AddItem)
				projects.POST("/:id/boms/:bomId/items/batch", h.ProjectBOM.BatchAddItems)
				projects.PUT("/:id/boms/:bomId/items/:itemId", h.ProjectBOM.UpdateItem)
				projects.DELETE("/:id/boms/:bomId/items/:itemId", h.ProjectBOM.DeleteItem)
				projects.POST("/:id/boms/:bomId/reorder", h.ProjectBOM.ReorderItems)
				// Phase 2: Excel导入导出
				projects.GET("/:id/boms/:bomId/export", h.ProjectBOM.ExportBOM)
				projects.POST("/:id/boms/:bomId/import", h.ProjectBOM.ImportBOM)
				// 版本发布
				projects.POST("/:id/boms/:bomId/release", h.ProjectBOM.ReleaseBOM)
				projects.POST("/:id/boms/create-from", h.ProjectBOM.CreateFromBOM)
				// Phase 3: EBOM→MBOM/PBOM转换
				projects.POST("/:id/boms/:bomId/convert-to-mbom", h.ProjectBOM.ConvertToMBOM)
				projects.POST("/:id/boms/:bomId/convert-to-pbom", h.ProjectBOM.ConvertToPBOM)
				// BOM分类树
				projects.GET("/:id/boms/:bomId/category-tree", h.ProjectBOM.GetCategoryTree)
				// 工艺路线
				projects.GET("/:id/routes", h.ProjectBOM.ListRoutes)
				projects.POST("/:id/boms/:bomId/routes", h.ProjectBOM.CreateRoute)
				projects.GET("/:id/routes/:routeId", h.ProjectBOM.GetRoute)
				projects.PUT("/:id/routes/:routeId", h.ProjectBOM.UpdateRoute)
				projects.POST("/:id/routes/:routeId/steps", h.ProjectBOM.CreateStep)
				projects.PUT("/:id/routes/:routeId/steps/:stepId", h.ProjectBOM.UpdateStep)
				projects.DELETE("/:id/routes/:routeId/steps/:stepId", h.ProjectBOM.DeleteStep)
				projects.POST("/:id/routes/:routeId/steps/:stepId/materials", h.ProjectBOM.CreateStepMaterial)
				projects.DELETE("/:id/routes/:routeId/steps/:stepId/materials/:materialId", h.ProjectBOM.DeleteStepMaterial)

				// V13: CMF编制
				projects.GET("/:id/tasks/:taskId/cmf/appearance-parts", h.CMF.GetAppearanceParts)
				projects.GET("/:id/tasks/:taskId/cmf/designs", h.CMF.ListDesigns)
				projects.POST("/:id/tasks/:taskId/cmf/designs", h.CMF.CreateDesign)
				projects.PUT("/:id/tasks/:taskId/cmf/designs/:designId", h.CMF.UpdateDesign)
				projects.DELETE("/:id/tasks/:taskId/cmf/designs/:designId", h.CMF.DeleteDesign)
				projects.GET("/:id/cmf/designs", h.CMF.ListDesignsByProject)

				// V15: 图纸版本管理
				projects.GET("/:id/bom-items/:itemId/drawings", h.PartDrawing.ListDrawings)
				projects.POST("/:id/bom-items/:itemId/drawings", h.PartDrawing.UploadDrawing)
				projects.DELETE("/:id/bom-items/:itemId/drawings/:drawingId", h.PartDrawing.DeleteDrawing)
				projects.GET("/:id/bom-items/:itemId/drawings/:drawingId/download", h.PartDrawing.DownloadDrawing)
				projects.GET("/:id/boms/:bomId/drawings", h.PartDrawing.ListDrawingsByBOM)

				// V14: SKU管理
				projects.GET("/:id/skus", h.SKU.ListSKUs)
				projects.POST("/:id/skus", h.SKU.CreateSKU)
				projects.PUT("/:id/skus/:skuId", h.SKU.UpdateSKU)
				projects.DELETE("/:id/skus/:skuId", h.SKU.DeleteSKU)
				projects.GET("/:id/skus/:skuId/cmf", h.SKU.GetCMFConfigs)
				projects.PUT("/:id/skus/:skuId/cmf", h.SKU.BatchSaveCMFConfigs)
				projects.GET("/:id/skus/:skuId/bom-items", h.SKU.GetBOMItems)
				projects.PUT("/:id/skus/:skuId/bom-items", h.SKU.BatchSaveBOMItems)
				projects.GET("/:id/skus/:skuId/full-bom", h.SKU.GetFullBOM)

				// V16: CMF变体管理
				projects.GET("/:id/bom-items/:itemId/cmf-variants", h.CMFVariant.ListVariants)
				projects.POST("/:id/bom-items/:itemId/cmf-variants", h.CMFVariant.CreateVariant)
				projects.PUT("/:id/cmf-variants/:variantId", h.CMFVariant.UpdateVariant)
				projects.DELETE("/:id/cmf-variants/:variantId", h.CMFVariant.DeleteVariant)
				projects.GET("/:id/appearance-parts", h.CMFVariant.GetAppearanceParts)
				projects.GET("/:id/srm/items", h.CMFVariant.GetSRMItems)

				// V17: 语言变体
				projects.GET("/:id/bom-items/:itemId/lang-variants", h.LangVariant.ListVariants)
				projects.POST("/:id/bom-items/:itemId/lang-variants", h.LangVariant.CreateVariant)
				projects.PUT("/:id/lang-variants/:variantId", h.LangVariant.UpdateVariant)
				projects.DELETE("/:id/lang-variants/:variantId", h.LangVariant.DeleteVariant)
				projects.GET("/:id/multilang-parts", h.LangVariant.GetMultilangParts)

				// V18: BOM ECN - 草稿和ECN管理
				projects.POST("/:id/boms/:bomId/edit", h.BOMECN.StartEditing)
				projects.POST("/:id/boms/:bomId/draft", h.BOMECN.SaveDraft)
				projects.GET("/:id/boms/:bomId/draft", h.BOMECN.GetDraft)
				projects.DELETE("/:id/boms/:bomId/draft", h.BOMECN.DiscardDraft)
				projects.POST("/:id/boms/:bomId/ecn", h.BOMECN.SubmitECN)

				// V2: 交付物管理
				projects.GET("/:id/deliverables", h.Deliverable.ListByProject)
				projects.GET("/:id/phases/:phaseId/deliverables", h.Deliverable.ListByPhase)
				projects.PUT("/:id/deliverables/:deliverableId", h.Deliverable.Update)

				// V3: 工作流操作
				if h.Workflow != nil {
					projects.POST("/:id/tasks/:taskId/assign", h.Workflow.AssignTask)
					projects.POST("/:id/tasks/:taskId/start", h.Workflow.StartTask)
					projects.POST("/:id/tasks/:taskId/complete", h.Workflow.CompleteTask)
					projects.POST("/:id/tasks/:taskId/review", h.Workflow.SubmitReview)
					projects.POST("/:id/phases/:phase/assign-roles", h.Workflow.AssignPhaseRoles)
					projects.GET("/:id/tasks/:taskId/history", h.Workflow.GetTaskHistory)
				}
			}

			// V2: 代号管理
			authorized.GET("/codenames", h.Codename.List)

			// 我的任务
			authorized.GET("/my/tasks", h.Project.GetMyTasks)
			authorized.POST("/my/tasks/:taskId/complete", h.Project.CompleteMyTask)
			authorized.PUT("/my/tasks/:taskId/form-draft", h.TaskForm.SaveFormDraft)
			authorized.GET("/my/tasks/:taskId/form-draft", h.TaskForm.GetFormDraft)

			// 文件上传
			authorized.POST("/upload", h.Upload.Upload)
			authorized.GET("/files/:fileId/3d", h.Upload.Get3DModel)

			// CMF图纸管理
			authorized.POST("/cmf-designs/:designId/drawings", h.CMF.AddDrawing)
			authorized.DELETE("/cmf-designs/:designId/drawings/:drawingId", h.CMF.RemoveDrawing)

			// V18: BOM ECN管理
			bomEcns := authorized.Group("/bom-ecn")
			{
				bomEcns.GET("", h.BOMECN.ListECNs)
				bomEcns.GET("/:id", h.BOMECN.GetECN)
				bomEcns.POST("/:id/approve", h.BOMECN.ApproveECN)
				bomEcns.POST("/:id/reject", h.BOMECN.RejectECN)
			}

			// ECN管理
			ecns := authorized.Group("/ecns")
			{
				ecns.GET("", h.ECN.List)
				ecns.POST("", h.ECN.Create)
				ecns.GET("/stats", h.ECN.GetStats)
				ecns.GET("/my-pending", h.ECN.ListMyPending)
				ecns.GET("/:id", h.ECN.Get)
				ecns.PUT("/:id", h.ECN.Update)
				ecns.POST("/:id/submit", h.ECN.Submit)
				ecns.POST("/:id/approve", h.ECN.Approve)
				ecns.POST("/:id/reject", h.ECN.Reject)
				ecns.POST("/:id/implement", h.ECN.Implement)
				ecns.GET("/:id/affected-items", h.ECN.ListAffectedItems)
				ecns.POST("/:id/affected-items", h.ECN.AddAffectedItem)
				ecns.PUT("/:id/affected-items/:itemId", h.ECN.UpdateAffectedItem)
				ecns.DELETE("/:id/affected-items/:itemId", h.ECN.RemoveAffectedItem)
				ecns.GET("/:id/approvals", h.ECN.ListApprovals)
				ecns.POST("/:id/approvers", h.ECN.AddApprover)
				ecns.GET("/:id/tasks", h.ECN.ListTasks)
				ecns.POST("/:id/tasks", h.ECN.CreateTask)
				ecns.PUT("/:id/tasks/:taskId", h.ECN.UpdateTask)
				ecns.POST("/:id/apply-bom-changes", h.ECN.ApplyBOMChanges)
				ecns.GET("/:id/history", h.ECN.ListHistory)
			}

			// 文档管理
			documents := authorized.Group("/documents")
			{
				documents.GET("", h.Document.List)
				documents.POST("", h.Document.Upload)
				documents.GET("/:id", h.Document.Get)
				documents.PUT("/:id", h.Document.Update)
				documents.DELETE("/:id", h.Document.Delete)
				documents.GET("/:id/download", h.Document.Download)
				documents.POST("/:id/release", h.Document.Release)
				documents.POST("/:id/obsolete", h.Document.Obsolete)
				documents.GET("/:id/versions", h.Document.ListVersions)
				documents.POST("/:id/versions", h.Document.UploadNewVersion)
				documents.GET("/:id/versions/:versionId/download", h.Document.DownloadVersion)
			}

			// 文档分类
			authorized.GET("/document-categories", h.Document.ListCategories)

			// 模板管理
			templates := authorized.Group("/templates")
			{
				templates.GET("", h.Template.List)
				templates.POST("", h.Template.Create)
				templates.GET("/:id", h.Template.Get)
				templates.PUT("/:id", h.Template.Update)
				templates.DELETE("/:id", h.Template.Delete)
				templates.POST("/:id/duplicate", h.Template.Duplicate)
				templates.POST("/:id/tasks", h.Template.CreateTask)
				templates.PUT("/:id/tasks/:taskCode", h.Template.UpdateTask)
				templates.DELETE("/:id/tasks/:taskCode", h.Template.DeleteTask)
				templates.PUT("/:id/tasks/batch", h.Template.BatchSaveTasks)
				templates.POST("/:id/publish", h.Template.Publish)
				templates.POST("/:id/upgrade", h.Template.UpgradeVersion)
				templates.POST("/:id/revert", h.Template.Revert)
				templates.GET("/:id/versions", h.Template.ListVersions)

				// V7: 模板任务表单
				templates.GET("/:id/task-forms", h.TaskForm.GetTemplateTaskForms)
				templates.POST("/:id/task-forms", h.TaskForm.SaveTemplateTaskForm)
			}

			// 从模板创建项目
			authorized.POST("/projects/create-from-template", h.Template.CreateProjectFromTemplate)

			// === SRM模块路由 ===
			srmGroup := authorized.Group("/srm")
			{
				// 供应商管理
				suppliers := srmGroup.Group("/suppliers")
				{
					suppliers.GET("", srmH.Supplier.ListSuppliers)
					suppliers.POST("", srmH.Supplier.CreateSupplier)
					suppliers.GET("/:id", srmH.Supplier.GetSupplier)
					suppliers.PUT("/:id", srmH.Supplier.UpdateSupplier)
					suppliers.DELETE("/:id", srmH.Supplier.DeleteSupplier)
					suppliers.GET("/:id/contacts", srmH.Supplier.ListContacts)
					suppliers.POST("/:id/contacts", srmH.Supplier.CreateContact)
					suppliers.DELETE("/:id/contacts/:contactId", srmH.Supplier.DeleteContact)
				}

				// 采购需求
				prs := srmGroup.Group("/purchase-requests")
				{
					prs.GET("", srmH.PR.ListPRs)
					prs.POST("", srmH.PR.CreatePR)
					prs.POST("/from-bom", srmH.PR.CreatePRFromBOM)
					prs.GET("/:id", srmH.PR.GetPR)
					prs.PUT("/:id", srmH.PR.UpdatePR)
					prs.POST("/:id/approve", srmH.PR.ApprovePR)
					prs.PUT("/:id/items/:itemId/assign-supplier", srmH.PR.AssignSupplierToItem)
					prs.POST("/:id/generate-pos", srmH.PR.GeneratePOs)
				}

				// 采购订单
				pos := srmGroup.Group("/purchase-orders")
				{
					pos.GET("", srmH.PO.ListPOs)
					pos.GET("/export", srmH.PO.ExportPOs)
					pos.POST("", srmH.PO.CreatePO)
					pos.POST("/from-bom", srmH.PO.GenerateFromBOM)
					pos.GET("/:id", srmH.PO.GetPO)
					pos.PUT("/:id", srmH.PO.UpdatePO)
					pos.POST("/:id/submit", srmH.PO.SubmitPO)
					pos.POST("/:id/approve", srmH.PO.ApprovePO)
					pos.POST("/:id/items/:itemId/receive", srmH.PO.ReceiveItem)
					pos.DELETE("/:id", srmH.PO.DeletePO)
				}

				// 来料检验
				inspections := srmGroup.Group("/inspections")
				{
					inspections.GET("", srmH.Inspection.ListInspections)
					inspections.POST("", srmH.Inspection.CreateInspection)
					inspections.POST("/from-po", srmH.Inspection.CreateFromPO)
					inspections.GET("/:id", srmH.Inspection.GetInspection)
					inspections.PUT("/:id", srmH.Inspection.UpdateInspection)
					inspections.POST("/:id/complete", srmH.Inspection.CompleteInspection)
				}

				// 库存管理
				inventory := srmGroup.Group("/inventory")
				{
					inventory.GET("", srmH.Inventory.ListInventory)
					inventory.GET("/:id/transactions", srmH.Inventory.GetTransactions)
					inventory.POST("/in", srmH.Inventory.StockIn)
					inventory.POST("/out", srmH.Inventory.StockOut)
					inventory.POST("/adjust", srmH.Inventory.StockAdjust)
				}

				// 看板
				srmGroup.GET("/dashboard/sampling-progress", srmH.Dashboard.GetSamplingProgress)

				// 采购项目
				projects := srmGroup.Group("/projects")
				{
					projects.GET("", srmH.Project.ListProjects)
					projects.POST("", srmH.Project.CreateProject)
					projects.GET("/:id", srmH.Project.GetProject)
					projects.PUT("/:id", srmH.Project.UpdateProject)
					projects.GET("/:id/progress", srmH.Project.GetProjectProgress)
					projects.GET("/:id/progress-by-group", srmH.Project.GetProjectProgressByGroup)
					projects.GET("/:id/items-by-group", srmH.Project.GetItemsByGroup)
					projects.GET("/:id/activities", srmH.Project.ListActivityLogs)
				}

				// 通用设备
				equipment := srmGroup.Group("/equipment")
				{
					equipment.GET("", srmH.Equipment.List)
					equipment.POST("", srmH.Equipment.Create)
					equipment.GET("/:id", srmH.Equipment.Get)
					equipment.PUT("/:id", srmH.Equipment.Update)
					equipment.DELETE("/:id", srmH.Equipment.Delete)
				}

				// 延期审批
				delays := srmGroup.Group("/delay-requests")
				{
					delays.GET("", srmH.Project.ListDelayRequests)
					delays.POST("", srmH.Project.CreateDelayRequest)
					delays.GET("/:id", srmH.Project.GetDelayRequest)
					delays.POST("/:id/approve", srmH.Project.ApproveDelayRequest)
					delays.POST("/:id/reject", srmH.Project.RejectDelayRequest)
				}

				// 对账结算
				settlements := srmGroup.Group("/settlements")
				{
					settlements.GET("", srmH.Settlement.ListSettlements)
					settlements.GET("/export", srmH.Settlement.ExportSettlements)
					settlements.POST("", srmH.Settlement.CreateSettlement)
					settlements.POST("/generate", srmH.Settlement.GenerateSettlement)
					settlements.GET("/:id", srmH.Settlement.GetSettlement)
					settlements.PUT("/:id", srmH.Settlement.UpdateSettlement)
					settlements.DELETE("/:id", srmH.Settlement.DeleteSettlement)
					settlements.POST("/:id/confirm-buyer", srmH.Settlement.ConfirmByBuyer)
					settlements.POST("/:id/confirm-supplier", srmH.Settlement.ConfirmBySupplier)
					settlements.POST("/:id/disputes", srmH.Settlement.AddDispute)
					settlements.PUT("/:id/disputes/:disputeId", srmH.Settlement.UpdateDispute)
				}

				// 8D改进
				cas := srmGroup.Group("/corrective-actions")
				{
					cas.GET("", srmH.CorrectiveAction.ListCorrectiveActions)
					cas.POST("", srmH.CorrectiveAction.CreateCorrectiveAction)
					cas.GET("/:id", srmH.CorrectiveAction.GetCorrectiveAction)
					cas.PUT("/:id", srmH.CorrectiveAction.UpdateCorrectiveAction)
					cas.POST("/:id/respond", srmH.CorrectiveAction.SupplierRespond)
					cas.POST("/:id/verify", srmH.CorrectiveAction.Verify)
					cas.POST("/:id/close", srmH.CorrectiveAction.Close)
				}

				// 供应商评价
				evals := srmGroup.Group("/evaluations")
				{
					evals.GET("", srmH.Evaluation.ListEvaluations)
					evals.POST("", srmH.Evaluation.CreateEvaluation)
					evals.POST("/auto-generate", srmH.Evaluation.AutoGenerate)
					evals.GET("/supplier/:supplierId", srmH.Evaluation.GetSupplierHistory)
					evals.GET("/:id", srmH.Evaluation.GetEvaluation)
					evals.PUT("/:id", srmH.Evaluation.UpdateEvaluation)
					evals.POST("/:id/submit", srmH.Evaluation.Submit)
					evals.POST("/:id/approve", srmH.Evaluation.Approve)
				}

				// 询价单 RFQ
				rfqs := srmGroup.Group("/rfq")
				{
					rfqs.GET("", srmH.RFQ.ListRFQs)
					rfqs.POST("", srmH.RFQ.CreateRFQ)
					rfqs.GET("/:id", srmH.RFQ.GetRFQ)
					rfqs.POST("/:id/quotes", srmH.RFQ.AddQuote)
					rfqs.PUT("/:id/quotes/:quoteId", srmH.RFQ.UpdateQuote)
					rfqs.POST("/:id/quotes/:quoteId/select", srmH.RFQ.SelectQuote)
					rfqs.POST("/:id/convert-to-po", srmH.RFQ.ConvertToPO)
					rfqs.GET("/:id/comparison", srmH.RFQ.GetComparison)
				}

				// PR物料状态
				srmGroup.PUT("/pr-items/:id/status", srmH.PRItem.UpdatePRItemStatus)

				// 打样管理
				srmGroup.POST("/pr-items/:itemId/sampling", srmH.Sampling.CreateSamplingRequest)
				srmGroup.GET("/pr-items/:itemId/sampling", srmH.Sampling.ListSamplingRequests)
				srmGroup.PUT("/sampling/:id/status", srmH.Sampling.UpdateSamplingStatus)
				srmGroup.POST("/sampling/:id/request-verify", srmH.Sampling.RequestVerify)
				srmGroup.POST("/sampling/verify-callback", srmH.Sampling.VerifyCallback)

				// 操作日志
				srmGroup.GET("/activities/:entityType/:entityId", srmH.Project.ListEntityActivityLogs)
			}
		}
	}
}

// =============================================================================
// 飞书Webhook处理函数
// 暂时只做日志记录，真正的业务处理在Phase 3实现
// =============================================================================

// handleFeishuApprovalWebhook 处理飞书审批回调事件
func handleFeishuApprovalWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("[Feishu Webhook] 读取请求体失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "msg": "读取请求体失败"})
		return
	}

	// 检查是否为URL验证事件
	if feishu.IsVerificationEvent(body) {
		challenge, err := feishu.HandleVerification(body)
		if err != nil {
			log.Printf("[Feishu Webhook] URL验证失败: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"code": -1, "msg": "URL验证失败"})
			return
		}
		log.Printf("[Feishu Webhook] URL验证成功")
		c.JSON(http.StatusOK, gin.H{"challenge": challenge})
		return
	}

	// 解析审批事件
	event, err := feishu.HandleApprovalEvent(body)
	if err != nil {
		log.Printf("[Feishu Webhook] 解析审批事件失败: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 0}) // 返回成功避免飞书重试
		return
	}

	// 记录审批事件日志（Phase 3 将在此处实现业务处理）
	log.Printf("[Feishu Webhook] 审批事件: approval_code=%s, instance_code=%s, status=%s, open_id=%s",
		event.ApprovalCode, event.InstanceCode, event.Status, event.OpenID)

	c.JSON(http.StatusOK, gin.H{"code": 0})
}

// handleFeishuEventVerification 处理飞书事件订阅URL验证
func handleFeishuEventVerification(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("[Feishu Webhook] 读取请求体失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "msg": "读取请求体失败"})
		return
	}

	// 获取事件类型
	eventType := feishu.GetEventType(body)
	log.Printf("[Feishu Webhook] 收到事件: type=%s", eventType)

	// 处理URL验证
	if feishu.IsVerificationEvent(body) {
		challenge, err := feishu.HandleVerification(body)
		if err != nil {
			log.Printf("[Feishu Webhook] URL验证失败: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"code": -1, "msg": "URL验证失败"})
			return
		}
		log.Printf("[Feishu Webhook] URL验证成功, challenge=%s", challenge)
		c.JSON(http.StatusOK, gin.H{"challenge": challenge})
		return
	}

	// 其他事件暂时只记录日志（Phase 3 将在此处扩展）
	log.Printf("[Feishu Webhook] 收到未处理的事件: %s, body=%s", eventType, string(body))
	c.JSON(http.StatusOK, gin.H{"code": 0})
}
