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

	// 手动添加新列（AutoMigrate 会触发 FK 级联问题，所以用原始 SQL）
	migrationSQL := []string{
		"ALTER TABLE template_tasks ADD COLUMN IF NOT EXISTS auto_create_feishu_task boolean DEFAULT false",
		"ALTER TABLE template_tasks ADD COLUMN IF NOT EXISTS feishu_approval_code varchar(100) DEFAULT ''",
		"ALTER TABLE tasks ADD COLUMN IF NOT EXISTS auto_create_feishu_task boolean DEFAULT false",
		"ALTER TABLE tasks ADD COLUMN IF NOT EXISTS feishu_approval_code varchar(100) DEFAULT ''",

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
	}
	for _, sql := range migrationSQL {
		if err := db.Exec(sql).Error; err != nil {
			zapLogger.Warn("Migration SQL warning (may already exist)", zap.String("sql", sql), zap.Error(err))
		}
	}
	zapLogger.Info("Database migration completed")

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

	// 注册路由
	registerRoutes(router, handlers, services, cfg)

	// 创建HTTP服务器
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
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

func registerRoutes(r *gin.Engine, h *handler.Handlers, svc *service.Services, cfg *config.Config) {
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

	// 静态文件服务 - 前端
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
		c.File("./web/plm/index.html")
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
				projects.GET("/:id/boms", h.ProjectBOM.ListBOMs)
				projects.POST("/:id/boms", h.ProjectBOM.CreateBOM)
				projects.GET("/:id/boms/:bomId", h.ProjectBOM.GetBOM)
				projects.PUT("/:id/boms/:bomId", h.ProjectBOM.UpdateBOM)
				projects.POST("/:id/boms/:bomId/submit", h.ProjectBOM.SubmitBOM)
				projects.POST("/:id/boms/:bomId/approve", h.ProjectBOM.ApproveBOM)
				projects.POST("/:id/boms/:bomId/reject", h.ProjectBOM.RejectBOM)
				projects.POST("/:id/boms/:bomId/freeze", h.ProjectBOM.FreezeBOM)
				projects.POST("/:id/boms/:bomId/items", h.ProjectBOM.AddItem)
				projects.POST("/:id/boms/:bomId/items/batch", h.ProjectBOM.BatchAddItems)
				projects.DELETE("/:id/boms/:bomId/items/:itemId", h.ProjectBOM.DeleteItem)

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

			// 文件上传
			authorized.POST("/upload", h.Upload.Upload)

			// ECN管理
			ecns := authorized.Group("/ecns")
			{
				ecns.GET("", h.ECN.List)
				ecns.POST("", h.ECN.Create)
				ecns.GET("/:id", h.ECN.Get)
				ecns.PUT("/:id", h.ECN.Update)
				ecns.POST("/:id/submit", h.ECN.Submit)
				ecns.POST("/:id/approve", h.ECN.Approve)
				ecns.POST("/:id/reject", h.ECN.Reject)
				ecns.POST("/:id/implement", h.ECN.Implement)
				ecns.GET("/:id/affected-items", h.ECN.ListAffectedItems)
				ecns.POST("/:id/affected-items", h.ECN.AddAffectedItem)
				ecns.DELETE("/:id/affected-items/:itemId", h.ECN.RemoveAffectedItem)
				ecns.GET("/:id/approvals", h.ECN.ListApprovals)
				ecns.POST("/:id/approvers", h.ECN.AddApprover)
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
				templates.GET("/:id/versions", h.Template.ListVersions)

				// V7: 模板任务表单
				templates.GET("/:id/task-forms", h.TaskForm.GetTemplateTaskForms)
				templates.POST("/:id/task-forms", h.TaskForm.SaveTemplateTaskForm)
			}

			// 从模板创建项目
			authorized.POST("/projects/create-from-template", h.Template.CreateProjectFromTemplate)
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
