-- ============================================================
-- nimo PLM System Database Schema
-- Version: 1.0.0
-- Database: PostgreSQL 16
-- Created: 2026-02-05
-- ============================================================

-- 启用扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";  -- 用于模糊搜索

-- ============================================================
-- 用户与权限模块
-- ============================================================

-- 用户表
CREATE TABLE users (
    id              VARCHAR(32) PRIMARY KEY DEFAULT REPLACE(uuid_generate_v4()::TEXT, '-', ''),
    feishu_user_id  VARCHAR(64) UNIQUE,                     -- 飞书用户ID (ou_xxx)
    feishu_union_id VARCHAR(64),                            -- 飞书union_id
    feishu_open_id  VARCHAR(64),                            -- 飞书open_id
    employee_no     VARCHAR(32) UNIQUE,                     -- 工号
    username        VARCHAR(64) NOT NULL UNIQUE,            -- 用户名
    name            VARCHAR(64) NOT NULL,                   -- 姓名
    email           VARCHAR(128) UNIQUE,                    -- 邮箱
    mobile          VARCHAR(20),                            -- 手机号
    avatar_url      VARCHAR(512),                           -- 头像URL
    department_id   VARCHAR(32),                            -- 部门ID
    status          VARCHAR(16) NOT NULL DEFAULT 'active',  -- active/inactive/locked
    last_login_at   TIMESTAMPTZ,                            -- 最后登录时间
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ                             -- 软删除
);

CREATE INDEX idx_users_feishu_user_id ON users(feishu_user_id);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_deleted_at ON users(deleted_at);

-- 部门表
CREATE TABLE departments (
    id              VARCHAR(32) PRIMARY KEY DEFAULT REPLACE(uuid_generate_v4()::TEXT, '-', ''),
    feishu_dept_id  VARCHAR(64) UNIQUE,                     -- 飞书部门ID
    name            VARCHAR(128) NOT NULL,                  -- 部门名称
    parent_id       VARCHAR(32) REFERENCES departments(id), -- 上级部门
    path            VARCHAR(512),                           -- 部门路径 (1/2/3)
    level           INT NOT NULL DEFAULT 1,                 -- 层级
    sort_order      INT NOT NULL DEFAULT 0,                 -- 排序
    leader_id       VARCHAR(32) REFERENCES users(id),       -- 部门负责人
    status          VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_departments_parent_id ON departments(parent_id);
CREATE INDEX idx_departments_path ON departments(path);

-- 角色表
CREATE TABLE roles (
    id              VARCHAR(32) PRIMARY KEY DEFAULT REPLACE(uuid_generate_v4()::TEXT, '-', ''),
    code            VARCHAR(64) NOT NULL UNIQUE,            -- 角色编码
    name            VARCHAR(64) NOT NULL,                   -- 角色名称
    description     TEXT,                                   -- 描述
    is_system       BOOLEAN NOT NULL DEFAULT FALSE,         -- 是否系统内置
    status          VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 预置角色
INSERT INTO roles (id, code, name, description, is_system) VALUES
    ('role_plm_admin', 'plm_admin', 'PLM管理员', 'PLM系统管理员，拥有所有权限', TRUE),
    ('role_plm_editor', 'plm_editor', 'PLM编辑', '可以创建和编辑产品、BOM、项目', TRUE),
    ('role_plm_viewer', 'plm_viewer', 'PLM查看', '只读权限', TRUE),
    ('role_project_manager', 'project_manager', '项目经理', '项目管理权限', TRUE),
    ('role_engineer', 'engineer', '工程师', '参与项目任务', TRUE);

-- 权限表
CREATE TABLE permissions (
    id              VARCHAR(32) PRIMARY KEY DEFAULT REPLACE(uuid_generate_v4()::TEXT, '-', ''),
    code            VARCHAR(128) NOT NULL UNIQUE,           -- 权限编码 (module:action)
    name            VARCHAR(64) NOT NULL,                   -- 权限名称
    module          VARCHAR(32) NOT NULL,                   -- 所属模块
    description     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 预置权限
INSERT INTO permissions (id, code, name, module) VALUES
    -- 产品权限
    ('perm_product_view', 'product:view', '查看产品', 'product'),
    ('perm_product_create', 'product:create', '创建产品', 'product'),
    ('perm_product_edit', 'product:edit', '编辑产品', 'product'),
    ('perm_product_delete', 'product:delete', '删除产品', 'product'),
    ('perm_product_release', 'product:release', '发布产品', 'product'),
    -- BOM权限
    ('perm_bom_view', 'bom:view', '查看BOM', 'bom'),
    ('perm_bom_edit', 'bom:edit', '编辑BOM', 'bom'),
    ('perm_bom_release', 'bom:release', '发布BOM', 'bom'),
    -- 项目权限
    ('perm_project_view', 'project:view', '查看项目', 'project'),
    ('perm_project_create', 'project:create', '创建项目', 'project'),
    ('perm_project_edit', 'project:edit', '编辑项目', 'project'),
    ('perm_project_delete', 'project:delete', '删除项目', 'project'),
    -- 任务权限
    ('perm_task_view', 'task:view', '查看任务', 'task'),
    ('perm_task_edit', 'task:edit', '编辑任务', 'task'),
    ('perm_task_assign', 'task:assign', '分配任务', 'task'),
    -- ECN权限
    ('perm_ecn_view', 'ecn:view', '查看ECN', 'ecn'),
    ('perm_ecn_create', 'ecn:create', '创建ECN', 'ecn'),
    ('perm_ecn_approve', 'ecn:approve', '审批ECN', 'ecn'),
    -- 文档权限
    ('perm_doc_view', 'document:view', '查看文档', 'document'),
    ('perm_doc_upload', 'document:upload', '上传文档', 'document'),
    ('perm_doc_delete', 'document:delete', '删除文档', 'document');

-- 角色-权限关联表
CREATE TABLE role_permissions (
    role_id         VARCHAR(32) NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id   VARCHAR(32) NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (role_id, permission_id)
);

-- 预置角色权限
INSERT INTO role_permissions (role_id, permission_id)
SELECT 'role_plm_admin', id FROM permissions;  -- 管理员拥有所有权限

INSERT INTO role_permissions (role_id, permission_id)
SELECT 'role_plm_editor', id FROM permissions 
WHERE code NOT IN ('product:delete', 'project:delete', 'ecn:approve', 'document:delete');

INSERT INTO role_permissions (role_id, permission_id)
SELECT 'role_plm_viewer', id FROM permissions 
WHERE code LIKE '%:view';

-- 用户-角色关联表
CREATE TABLE user_roles (
    user_id         VARCHAR(32) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id         VARCHAR(32) NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, role_id)
);

-- ============================================================
-- 产品模块
-- ============================================================

-- 产品类别表
CREATE TABLE product_categories (
    id              VARCHAR(32) PRIMARY KEY DEFAULT REPLACE(uuid_generate_v4()::TEXT, '-', ''),
    code            VARCHAR(32) NOT NULL UNIQUE,            -- 类别编码
    name            VARCHAR(64) NOT NULL,                   -- 类别名称
    parent_id       VARCHAR(32) REFERENCES product_categories(id),
    path            VARCHAR(256),                           -- 路径
    level           INT NOT NULL DEFAULT 1,
    sort_order      INT NOT NULL DEFAULT 0,
    description     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 预置产品类别
INSERT INTO product_categories (id, code, name, level) VALUES
    ('cat_platform', 'platform', '平台(镜腿)', 1),
    ('cat_frame', 'frame', '镜框', 1),
    ('cat_lens', 'lens', '镜片', 1),
    ('cat_accessory', 'accessory', '配件', 1);

-- 产品表
CREATE TABLE products (
    id              VARCHAR(32) PRIMARY KEY DEFAULT REPLACE(uuid_generate_v4()::TEXT, '-', ''),
    code            VARCHAR(64) NOT NULL UNIQUE,            -- 产品编码 (PRD-NIMO-001)
    name            VARCHAR(128) NOT NULL,                  -- 产品名称
    category_id     VARCHAR(32) NOT NULL REFERENCES product_categories(id),
    status          VARCHAR(16) NOT NULL DEFAULT 'draft',   -- draft/developing/active/discontinued
    description     TEXT,                                   -- 产品描述
    specs           JSONB,                                  -- 规格参数 (灵活扩展)
    thumbnail_url   VARCHAR(512),                           -- 缩略图
    current_bom_version VARCHAR(16),                        -- 当前BOM版本
    created_by      VARCHAR(32) NOT NULL REFERENCES users(id),
    updated_by      VARCHAR(32) REFERENCES users(id),
    released_at     TIMESTAMPTZ,                            -- 发布时间
    discontinued_at TIMESTAMPTZ,                            -- 停产时间
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_products_code ON products(code);
CREATE INDEX idx_products_category_id ON products(category_id);
CREATE INDEX idx_products_status ON products(status);
CREATE INDEX idx_products_name_trgm ON products USING gin(name gin_trgm_ops);
CREATE INDEX idx_products_deleted_at ON products(deleted_at);

-- 产品编码序列
CREATE SEQUENCE product_code_seq START 1;

-- ============================================================
-- 物料模块
-- ============================================================

-- 物料类别表
CREATE TABLE material_categories (
    id              VARCHAR(32) PRIMARY KEY DEFAULT REPLACE(uuid_generate_v4()::TEXT, '-', ''),
    code            VARCHAR(32) NOT NULL UNIQUE,
    name            VARCHAR(64) NOT NULL,
    parent_id       VARCHAR(32) REFERENCES material_categories(id),
    path            VARCHAR(256),
    level           INT NOT NULL DEFAULT 1,
    sort_order      INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 预置物料类别
INSERT INTO material_categories (id, code, name, level) VALUES
    ('mcat_electronic', 'electronic', '电子元器件', 1),
    ('mcat_mechanical', 'mechanical', '结构件', 1),
    ('mcat_optical', 'optical', '光学器件', 1),
    ('mcat_packaging', 'packaging', '包材', 1),
    ('mcat_other', 'other', '其他', 1);

-- 物料表
CREATE TABLE materials (
    id              VARCHAR(32) PRIMARY KEY DEFAULT REPLACE(uuid_generate_v4()::TEXT, '-', ''),
    code            VARCHAR(64) NOT NULL UNIQUE,            -- 物料编码 (MAT-E-001)
    name            VARCHAR(128) NOT NULL,                  -- 物料名称
    category_id     VARCHAR(32) NOT NULL REFERENCES material_categories(id),
    status          VARCHAR(16) NOT NULL DEFAULT 'active',  -- active/inactive/obsolete
    unit            VARCHAR(16) NOT NULL DEFAULT 'pcs',     -- 单位 (pcs/kg/m/set)
    description     TEXT,
    specs           JSONB,                                  -- 规格参数
    
    -- 采购相关
    lead_time_days  INT,                                    -- 采购周期(天)
    min_order_qty   DECIMAL(15,4),                          -- 最小订购量
    safety_stock    DECIMAL(15,4),                          -- 安全库存
    
    -- 成本相关
    standard_cost   DECIMAL(15,4),                          -- 标准成本
    last_cost       DECIMAL(15,4),                          -- 最近采购成本
    currency        VARCHAR(3) DEFAULT 'CNY',               -- 货币
    
    created_by      VARCHAR(32) NOT NULL REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_materials_code ON materials(code);
CREATE INDEX idx_materials_category_id ON materials(category_id);
CREATE INDEX idx_materials_status ON materials(status);
CREATE INDEX idx_materials_name_trgm ON materials USING gin(name gin_trgm_ops);

-- 物料编码序列
CREATE SEQUENCE material_code_seq START 1;

-- ============================================================
-- BOM模块
-- ============================================================

-- BOM头表
CREATE TABLE bom_headers (
    id              VARCHAR(32) PRIMARY KEY DEFAULT REPLACE(uuid_generate_v4()::TEXT, '-', ''),
    product_id      VARCHAR(32) NOT NULL REFERENCES products(id),
    version         VARCHAR(16) NOT NULL,                   -- 版本号 (1.0, 1.1, 2.0)
    status          VARCHAR(16) NOT NULL DEFAULT 'draft',   -- draft/released/obsolete
    description     TEXT,
    
    -- 统计字段
    total_items     INT NOT NULL DEFAULT 0,                 -- 物料总数
    total_cost      DECIMAL(15,4),                          -- 总成本
    max_level       INT NOT NULL DEFAULT 0,                 -- 最大层级
    
    released_by     VARCHAR(32) REFERENCES users(id),
    released_at     TIMESTAMPTZ,
    release_notes   TEXT,
    
    created_by      VARCHAR(32) NOT NULL REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(product_id, version)
);

CREATE INDEX idx_bom_headers_product_id ON bom_headers(product_id);
CREATE INDEX idx_bom_headers_status ON bom_headers(status);

-- BOM行项表
CREATE TABLE bom_items (
    id              VARCHAR(32) PRIMARY KEY DEFAULT REPLACE(uuid_generate_v4()::TEXT, '-', ''),
    bom_header_id   VARCHAR(32) NOT NULL REFERENCES bom_headers(id) ON DELETE CASCADE,
    parent_item_id  VARCHAR(32) REFERENCES bom_items(id),   -- 父级物料(顶级为NULL)
    material_id     VARCHAR(32) NOT NULL REFERENCES materials(id),
    
    level           INT NOT NULL DEFAULT 0,                 -- 层级(0为顶级)
    sequence        INT NOT NULL DEFAULT 0,                 -- 同级排序
    quantity        DECIMAL(15,4) NOT NULL,                 -- 用量
    unit            VARCHAR(16) NOT NULL DEFAULT 'pcs',     -- 单位
    
    position        VARCHAR(32),                            -- 位置编号
    reference       VARCHAR(128),                           -- 参考信息
    notes           TEXT,                                   -- 备注
    
    -- 计算字段
    unit_cost       DECIMAL(15,4),                          -- 单价
    extended_cost   DECIMAL(15,4),                          -- 金额(quantity * unit_cost)
    
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_bom_items_bom_header_id ON bom_items(bom_header_id);
CREATE INDEX idx_bom_items_parent_item_id ON bom_items(parent_item_id);
CREATE INDEX idx_bom_items_material_id ON bom_items(material_id);

-- BOM递归展开视图
CREATE OR REPLACE VIEW v_bom_explosion AS
WITH RECURSIVE bom_tree AS (
    -- 起点：顶级物料
    SELECT 
        bi.id,
        bi.bom_header_id,
        bi.parent_item_id,
        bi.material_id,
        bi.level,
        bi.sequence,
        bi.quantity,
        bi.unit,
        bi.quantity AS accumulated_qty,
        ARRAY[bi.id] AS path,
        1 AS depth
    FROM bom_items bi
    WHERE bi.parent_item_id IS NULL
    
    UNION ALL
    
    -- 递归：子物料
    SELECT 
        bi.id,
        bi.bom_header_id,
        bi.parent_item_id,
        bi.material_id,
        bi.level,
        bi.sequence,
        bi.quantity,
        bi.unit,
        bt.accumulated_qty * bi.quantity AS accumulated_qty,
        bt.path || bi.id AS path,
        bt.depth + 1 AS depth
    FROM bom_items bi
    INNER JOIN bom_tree bt ON bi.parent_item_id = bt.id
    WHERE bt.depth < 10  -- 防止无限递归
)
SELECT * FROM bom_tree;

-- ============================================================
-- 项目模块
-- ============================================================

-- 项目表
CREATE TABLE projects (
    id              VARCHAR(32) PRIMARY KEY DEFAULT REPLACE(uuid_generate_v4()::TEXT, '-', ''),
    code            VARCHAR(64) NOT NULL UNIQUE,            -- 项目编码 (PROJ-2026-001)
    name            VARCHAR(128) NOT NULL,                  -- 项目名称
    product_id      VARCHAR(32) REFERENCES products(id),    -- 关联产品
    
    status          VARCHAR(16) NOT NULL DEFAULT 'planning', -- planning/evt/dvt/pvt/mp/completed/cancelled
    current_phase   VARCHAR(16) DEFAULT 'evt',              -- 当前阶段
    
    description     TEXT,
    owner_id        VARCHAR(32) NOT NULL REFERENCES users(id), -- 项目负责人
    
    -- 计划时间
    planned_start   DATE,
    planned_end     DATE,
    actual_start    DATE,
    actual_end      DATE,
    
    -- 进度
    progress        INT NOT NULL DEFAULT 0,                 -- 整体进度(0-100)
    
    -- 飞书集成
    feishu_project_key VARCHAR(64),                         -- 飞书项目Key
    
    created_by      VARCHAR(32) NOT NULL REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_projects_code ON projects(code);
CREATE INDEX idx_projects_product_id ON projects(product_id);
CREATE INDEX idx_projects_status ON projects(status);
CREATE INDEX idx_projects_owner_id ON projects(owner_id);

-- 项目编码序列
CREATE SEQUENCE project_code_seq START 1;

-- 项目阶段表
CREATE TABLE project_phases (
    id              VARCHAR(32) PRIMARY KEY DEFAULT REPLACE(uuid_generate_v4()::TEXT, '-', ''),
    project_id      VARCHAR(32) NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    phase           VARCHAR(16) NOT NULL,                   -- evt/dvt/pvt/mp
    name            VARCHAR(64) NOT NULL,                   -- 阶段名称
    
    status          VARCHAR(16) NOT NULL DEFAULT 'pending', -- pending/in_progress/completed/skipped
    sequence        INT NOT NULL,                           -- 顺序
    
    planned_start   DATE,
    planned_end     DATE,
    actual_start    DATE,
    actual_end      DATE,
    
    entry_criteria  JSONB,                                  -- 准入条件
    exit_criteria   JSONB,                                  -- 准出条件
    
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(project_id, phase)
);

CREATE INDEX idx_project_phases_project_id ON project_phases(project_id);

-- 任务表
CREATE TABLE tasks (
    id              VARCHAR(32) PRIMARY KEY DEFAULT REPLACE(uuid_generate_v4()::TEXT, '-', ''),
    project_id      VARCHAR(32) NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    phase_id        VARCHAR(32) REFERENCES project_phases(id),
    parent_task_id  VARCHAR(32) REFERENCES tasks(id),       -- 父任务
    
    code            VARCHAR(64),                            -- 任务编码
    name            VARCHAR(256) NOT NULL,                  -- 任务名称
    description     TEXT,
    
    task_type       VARCHAR(32) NOT NULL DEFAULT 'task',    -- task/milestone/deliverable
    status          VARCHAR(16) NOT NULL DEFAULT 'pending', -- pending/in_progress/completed/blocked/cancelled
    priority        VARCHAR(16) NOT NULL DEFAULT 'medium',  -- low/medium/high/critical
    
    -- 执行人
    assignee_id     VARCHAR(32) REFERENCES users(id),
    reviewer_id     VARCHAR(32) REFERENCES users(id),       -- 审核人
    
    -- 时间
    planned_start   DATE,
    planned_end     DATE,
    actual_start    DATE,
    actual_end      DATE,
    due_date        DATE,                                   -- 截止日期
    
    -- 进度
    progress        INT NOT NULL DEFAULT 0,                 -- 0-100
    estimated_hours DECIMAL(8,2),                           -- 预估工时
    actual_hours    DECIMAL(8,2),                           -- 实际工时
    
    -- 飞书集成
    feishu_task_id  VARCHAR(64),                            -- 飞书任务ID
    
    -- 排序和层级
    sequence        INT NOT NULL DEFAULT 0,
    level           INT NOT NULL DEFAULT 0,
    path            VARCHAR(512),                           -- 任务路径
    
    created_by      VARCHAR(32) NOT NULL REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tasks_project_id ON tasks(project_id);
CREATE INDEX idx_tasks_phase_id ON tasks(phase_id);
CREATE INDEX idx_tasks_parent_task_id ON tasks(parent_task_id);
CREATE INDEX idx_tasks_assignee_id ON tasks(assignee_id);
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_due_date ON tasks(due_date);

-- 任务依赖表
CREATE TABLE task_dependencies (
    id                  VARCHAR(32) PRIMARY KEY DEFAULT REPLACE(uuid_generate_v4()::TEXT, '-', ''),
    task_id             VARCHAR(32) NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    depends_on_task_id  VARCHAR(32) NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    dependency_type     VARCHAR(16) NOT NULL DEFAULT 'finish_to_start', -- finish_to_start/start_to_start/finish_to_finish/start_to_finish
    lag_days            INT DEFAULT 0,                      -- 延迟天数
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(task_id, depends_on_task_id)
);

-- 任务评论表
CREATE TABLE task_comments (
    id              VARCHAR(32) PRIMARY KEY DEFAULT REPLACE(uuid_generate_v4()::TEXT, '-', ''),
    task_id         VARCHAR(32) NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_id         VARCHAR(32) NOT NULL REFERENCES users(id),
    content         TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_task_comments_task_id ON task_comments(task_id);

-- 任务附件表
CREATE TABLE task_attachments (
    id              VARCHAR(32) PRIMARY KEY DEFAULT REPLACE(uuid_generate_v4()::TEXT, '-', ''),
    task_id         VARCHAR(32) NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    file_name       VARCHAR(256) NOT NULL,
    file_path       VARCHAR(512) NOT NULL,                  -- MinIO路径
    file_size       BIGINT NOT NULL,
    mime_type       VARCHAR(128),
    uploaded_by     VARCHAR(32) NOT NULL REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_task_attachments_task_id ON task_attachments(task_id);

-- ============================================================
-- ECN变更模块
-- ============================================================

-- ECN表
CREATE TABLE ecns (
    id              VARCHAR(32) PRIMARY KEY DEFAULT REPLACE(uuid_generate_v4()::TEXT, '-', ''),
    code            VARCHAR(64) NOT NULL UNIQUE,            -- ECN编码 (ECN-2026-001)
    title           VARCHAR(256) NOT NULL,                  -- 变更标题
    
    product_id      VARCHAR(32) NOT NULL REFERENCES products(id),
    change_type     VARCHAR(32) NOT NULL,                   -- design/material/process/document
    urgency         VARCHAR(16) NOT NULL DEFAULT 'medium',  -- low/medium/high/critical
    
    status          VARCHAR(16) NOT NULL DEFAULT 'draft',   -- draft/pending/approved/rejected/implemented/cancelled
    
    reason          TEXT NOT NULL,                          -- 变更原因
    description     TEXT,                                   -- 变更描述
    impact_analysis TEXT,                                   -- 影响分析
    
    -- 申请人
    requested_by    VARCHAR(32) NOT NULL REFERENCES users(id),
    requested_at    TIMESTAMPTZ,
    
    -- 审批
    approved_by     VARCHAR(32) REFERENCES users(id),
    approved_at     TIMESTAMPTZ,
    rejection_reason TEXT,
    
    -- 实施
    implemented_by  VARCHAR(32) REFERENCES users(id),
    implemented_at  TIMESTAMPTZ,
    
    -- 飞书审批
    feishu_approval_code VARCHAR(64),                       -- 飞书审批定义Code
    feishu_instance_code VARCHAR(64),                       -- 飞书审批实例Code
    
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ecns_code ON ecns(code);
CREATE INDEX idx_ecns_product_id ON ecns(product_id);
CREATE INDEX idx_ecns_status ON ecns(status);
CREATE INDEX idx_ecns_requested_by ON ecns(requested_by);

-- ECN编码序列
CREATE SEQUENCE ecn_code_seq START 1;

-- ECN受影响项目表
CREATE TABLE ecn_affected_items (
    id              VARCHAR(32) PRIMARY KEY DEFAULT REPLACE(uuid_generate_v4()::TEXT, '-', ''),
    ecn_id          VARCHAR(32) NOT NULL REFERENCES ecns(id) ON DELETE CASCADE,
    item_type       VARCHAR(32) NOT NULL,                   -- bom_item/material/document/drawing
    item_id         VARCHAR(32) NOT NULL,                   -- 关联对象ID
    
    before_value    JSONB,                                  -- 变更前的值
    after_value     JSONB,                                  -- 变更后的值
    change_description TEXT,                                -- 变更说明
    
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ecn_affected_items_ecn_id ON ecn_affected_items(ecn_id);

-- ECN审批记录表
CREATE TABLE ecn_approvals (
    id              VARCHAR(32) PRIMARY KEY DEFAULT REPLACE(uuid_generate_v4()::TEXT, '-', ''),
    ecn_id          VARCHAR(32) NOT NULL REFERENCES ecns(id) ON DELETE CASCADE,
    approver_id     VARCHAR(32) NOT NULL REFERENCES users(id),
    sequence        INT NOT NULL,                           -- 审批顺序
    
    status          VARCHAR(16) NOT NULL DEFAULT 'pending', -- pending/approved/rejected
    decision        VARCHAR(16),                            -- approve/reject
    comment         TEXT,
    decided_at      TIMESTAMPTZ,
    
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ecn_approvals_ecn_id ON ecn_approvals(ecn_id);

-- ============================================================
-- 文档模块
-- ============================================================

-- 文档分类表
CREATE TABLE document_categories (
    id              VARCHAR(32) PRIMARY KEY DEFAULT REPLACE(uuid_generate_v4()::TEXT, '-', ''),
    code            VARCHAR(32) NOT NULL UNIQUE,
    name            VARCHAR(64) NOT NULL,
    parent_id       VARCHAR(32) REFERENCES document_categories(id),
    sort_order      INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 预置文档分类
INSERT INTO document_categories (id, code, name) VALUES
    ('dcat_design', 'design', '设计文档'),
    ('dcat_spec', 'spec', '规格书'),
    ('dcat_drawing', 'drawing', '图纸'),
    ('dcat_test', 'test', '测试报告'),
    ('dcat_quality', 'quality', '品质文档'),
    ('dcat_other', 'other', '其他');

-- 文档表
CREATE TABLE documents (
    id              VARCHAR(32) PRIMARY KEY DEFAULT REPLACE(uuid_generate_v4()::TEXT, '-', ''),
    code            VARCHAR(64) NOT NULL UNIQUE,            -- 文档编号 (DOC-2026-001)
    title           VARCHAR(256) NOT NULL,                  -- 文档标题
    category_id     VARCHAR(32) REFERENCES document_categories(id),
    
    -- 关联对象(多态)
    related_type    VARCHAR(32),                            -- product/project/ecn/material
    related_id      VARCHAR(32),
    
    status          VARCHAR(16) NOT NULL DEFAULT 'draft',   -- draft/released/obsolete
    version         VARCHAR(16) NOT NULL DEFAULT '1.0',
    
    description     TEXT,
    
    -- 文件信息
    file_name       VARCHAR(256) NOT NULL,
    file_path       VARCHAR(512) NOT NULL,                  -- MinIO路径
    file_size       BIGINT NOT NULL,
    mime_type       VARCHAR(128),
    
    -- 飞书云文档
    feishu_doc_token VARCHAR(64),                           -- 飞书文档token
    feishu_doc_url   VARCHAR(512),                          -- 飞书文档URL
    
    uploaded_by     VARCHAR(32) NOT NULL REFERENCES users(id),
    released_by     VARCHAR(32) REFERENCES users(id),
    released_at     TIMESTAMPTZ,
    
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_documents_code ON documents(code);
CREATE INDEX idx_documents_category_id ON documents(category_id);
CREATE INDEX idx_documents_related ON documents(related_type, related_id);
CREATE INDEX idx_documents_status ON documents(status);

-- 文档编码序列
CREATE SEQUENCE document_code_seq START 1;

-- 文档版本历史表
CREATE TABLE document_versions (
    id              VARCHAR(32) PRIMARY KEY DEFAULT REPLACE(uuid_generate_v4()::TEXT, '-', ''),
    document_id     VARCHAR(32) NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    version         VARCHAR(16) NOT NULL,
    
    file_name       VARCHAR(256) NOT NULL,
    file_path       VARCHAR(512) NOT NULL,
    file_size       BIGINT NOT NULL,
    
    change_summary  TEXT,                                   -- 版本变更说明
    
    created_by      VARCHAR(32) NOT NULL REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(document_id, version)
);

CREATE INDEX idx_document_versions_document_id ON document_versions(document_id);

-- ============================================================
-- 系统模块
-- ============================================================

-- 操作日志表
CREATE TABLE operation_logs (
    id              VARCHAR(32) PRIMARY KEY DEFAULT REPLACE(uuid_generate_v4()::TEXT, '-', ''),
    user_id         VARCHAR(32) REFERENCES users(id),
    user_name       VARCHAR(64),
    
    module          VARCHAR(32) NOT NULL,                   -- 模块
    action          VARCHAR(32) NOT NULL,                   -- 操作
    
    target_type     VARCHAR(32),                            -- 操作对象类型
    target_id       VARCHAR(32),                            -- 操作对象ID
    target_name     VARCHAR(256),                           -- 操作对象名称
    
    before_data     JSONB,                                  -- 操作前数据
    after_data      JSONB,                                  -- 操作后数据
    
    ip_address      VARCHAR(64),
    user_agent      TEXT,
    
    status          VARCHAR(16) NOT NULL DEFAULT 'success', -- success/failed
    error_message   TEXT,
    
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_operation_logs_user_id ON operation_logs(user_id);
CREATE INDEX idx_operation_logs_module ON operation_logs(module);
CREATE INDEX idx_operation_logs_target ON operation_logs(target_type, target_id);
CREATE INDEX idx_operation_logs_created_at ON operation_logs(created_at);

-- 系统配置表
CREATE TABLE system_configs (
    id              VARCHAR(32) PRIMARY KEY DEFAULT REPLACE(uuid_generate_v4()::TEXT, '-', ''),
    category        VARCHAR(32) NOT NULL,                   -- 配置分类
    key             VARCHAR(64) NOT NULL,                   -- 配置键
    value           TEXT,                                   -- 配置值
    value_type      VARCHAR(16) NOT NULL DEFAULT 'string',  -- string/number/boolean/json
    description     TEXT,
    
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(category, key)
);

-- 预置系统配置
INSERT INTO system_configs (id, category, key, value, value_type, description) VALUES
    ('cfg_company_name', 'system', 'company_name', 'Bitfantasy', 'string', '公司名称'),
    ('cfg_product_prefix', 'product', 'code_prefix', 'PRD-NIMO', 'string', '产品编码前缀'),
    ('cfg_project_prefix', 'project', 'code_prefix', 'PROJ', 'string', '项目编码前缀'),
    ('cfg_ecn_prefix', 'ecn', 'code_prefix', 'ECN', 'string', 'ECN编码前缀');

-- ============================================================
-- 通知模块
-- ============================================================

-- 通知表
CREATE TABLE notifications (
    id              VARCHAR(32) PRIMARY KEY DEFAULT REPLACE(uuid_generate_v4()::TEXT, '-', ''),
    user_id         VARCHAR(32) NOT NULL REFERENCES users(id),
    
    type            VARCHAR(32) NOT NULL,                   -- task_assigned/task_due/ecn_pending/etc
    title           VARCHAR(256) NOT NULL,
    content         TEXT,
    
    -- 关联对象
    related_type    VARCHAR(32),
    related_id      VARCHAR(32),
    
    is_read         BOOLEAN NOT NULL DEFAULT FALSE,
    read_at         TIMESTAMPTZ,
    
    -- 飞书通知
    feishu_message_id VARCHAR(64),
    
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_user_id ON notifications(user_id);
CREATE INDEX idx_notifications_is_read ON notifications(is_read);
CREATE INDEX idx_notifications_created_at ON notifications(created_at);

-- ============================================================
-- 更新时间触发器
-- ============================================================

CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 为所有需要的表添加触发器
CREATE TRIGGER tr_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER tr_departments_updated_at BEFORE UPDATE ON departments FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER tr_roles_updated_at BEFORE UPDATE ON roles FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER tr_products_updated_at BEFORE UPDATE ON products FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER tr_materials_updated_at BEFORE UPDATE ON materials FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER tr_bom_headers_updated_at BEFORE UPDATE ON bom_headers FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER tr_bom_items_updated_at BEFORE UPDATE ON bom_items FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER tr_projects_updated_at BEFORE UPDATE ON projects FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER tr_project_phases_updated_at BEFORE UPDATE ON project_phases FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER tr_tasks_updated_at BEFORE UPDATE ON tasks FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER tr_ecns_updated_at BEFORE UPDATE ON ecns FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER tr_documents_updated_at BEFORE UPDATE ON documents FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER tr_system_configs_updated_at BEFORE UPDATE ON system_configs FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- ============================================================
-- 完成
-- ============================================================

COMMENT ON TABLE users IS '用户表';
COMMENT ON TABLE departments IS '部门表';
COMMENT ON TABLE roles IS '角色表';
COMMENT ON TABLE permissions IS '权限表';
COMMENT ON TABLE products IS '产品表';
COMMENT ON TABLE materials IS '物料表';
COMMENT ON TABLE bom_headers IS 'BOM头表';
COMMENT ON TABLE bom_items IS 'BOM行项表';
COMMENT ON TABLE projects IS '项目表';
COMMENT ON TABLE project_phases IS '项目阶段表';
COMMENT ON TABLE tasks IS '任务表';
COMMENT ON TABLE ecns IS 'ECN变更表';
COMMENT ON TABLE documents IS '文档表';
