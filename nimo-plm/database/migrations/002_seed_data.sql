-- ============================================================
-- nimo PLM System Seed Data
-- Version: 1.0.0
-- Created: 2026-02-05
-- ============================================================

-- ============================================================
-- 测试用户数据
-- ============================================================

-- 插入管理员用户
INSERT INTO users (id, username, name, email, status) VALUES
    ('u_admin', 'admin', '系统管理员', 'admin@bitfantasy.com', 'active');

-- 为管理员分配角色
INSERT INTO user_roles (user_id, role_id) VALUES
    ('u_admin', 'role_plm_admin');

-- ============================================================
-- 项目任务模板数据
-- ============================================================

-- 项目任务模板表(用于创建项目时自动生成任务)
CREATE TABLE IF NOT EXISTS task_templates (
    id              VARCHAR(32) PRIMARY KEY DEFAULT REPLACE(uuid_generate_v4()::TEXT, '-', ''),
    template_name   VARCHAR(64) NOT NULL,                   -- 模板名称 (standard/fast_track)
    phase           VARCHAR(16) NOT NULL,                   -- evt/dvt/pvt/mp
    
    task_code       VARCHAR(64),                            -- 任务编码模板
    task_name       VARCHAR(256) NOT NULL,                  -- 任务名称
    task_type       VARCHAR(32) NOT NULL DEFAULT 'task',    -- task/milestone/deliverable
    
    parent_code     VARCHAR(64),                            -- 父任务编码
    sequence        INT NOT NULL DEFAULT 0,                 -- 顺序
    level           INT NOT NULL DEFAULT 0,                 -- 层级
    
    default_duration_days INT,                              -- 默认工期(天)
    description     TEXT,
    
    is_required     BOOLEAN NOT NULL DEFAULT TRUE,          -- 是否必须
    
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 标准研发模板 - EVT阶段
INSERT INTO task_templates (template_name, phase, task_code, task_name, task_type, level, sequence, default_duration_days, description) VALUES
-- EVT阶段
('standard', 'evt', 'EVT', 'EVT阶段', 'milestone', 0, 1, 45, '工程验证测试阶段'),
('standard', 'evt', 'EVT-1', '需求分析', 'task', 1, 1, 5, '收集和分析产品需求'),
('standard', 'evt', 'EVT-1-1', '市场需求调研', 'task', 2, 1, 3, NULL),
('standard', 'evt', 'EVT-1-2', '技术可行性分析', 'task', 2, 2, 2, NULL),
('standard', 'evt', 'EVT-2', '概念设计', 'task', 1, 2, 10, '产品概念设计'),
('standard', 'evt', 'EVT-2-1', 'ID设计', 'task', 2, 1, 5, '工业设计'),
('standard', 'evt', 'EVT-2-2', '结构概念设计', 'task', 2, 2, 5, NULL),
('standard', 'evt', 'EVT-2-3', '电子方案选型', 'task', 2, 3, 5, NULL),
('standard', 'evt', 'EVT-3', '原理验证', 'task', 1, 3, 15, '关键技术原理验证'),
('standard', 'evt', 'EVT-3-1', '光学方案验证', 'task', 2, 1, 7, NULL),
('standard', 'evt', 'EVT-3-2', '显示方案验证', 'task', 2, 2, 7, NULL),
('standard', 'evt', 'EVT-3-3', '通信方案验证', 'task', 2, 3, 5, NULL),
('standard', 'evt', 'EVT-4', 'EVT样机制作', 'task', 1, 4, 10, '制作EVT验证样机'),
('standard', 'evt', 'EVT-5', 'EVT测试验证', 'task', 1, 5, 5, 'EVT样机测试'),
('standard', 'evt', 'EVT-5-1', '功能测试', 'task', 2, 1, 3, NULL),
('standard', 'evt', 'EVT-5-2', '性能测试', 'task', 2, 2, 2, NULL),
('standard', 'evt', 'EVT-EXIT', 'EVT阶段评审', 'deliverable', 1, 6, 1, 'EVT阶段准出评审'),

-- DVT阶段
('standard', 'dvt', 'DVT', 'DVT阶段', 'milestone', 0, 1, 60, '设计验证测试阶段'),
('standard', 'dvt', 'DVT-1', '详细设计', 'task', 1, 1, 15, '产品详细设计'),
('standard', 'dvt', 'DVT-1-1', '结构详细设计', 'task', 2, 1, 7, NULL),
('standard', 'dvt', 'DVT-1-2', '电子详细设计', 'task', 2, 2, 7, NULL),
('standard', 'dvt', 'DVT-1-3', '软件架构设计', 'task', 2, 3, 5, NULL),
('standard', 'dvt', 'DVT-2', '模具开发', 'task', 1, 2, 25, '结构件模具开发'),
('standard', 'dvt', 'DVT-2-1', '模具设计', 'task', 2, 1, 5, NULL),
('standard', 'dvt', 'DVT-2-2', '模具制作', 'task', 2, 2, 15, NULL),
('standard', 'dvt', 'DVT-2-3', '模具试模', 'task', 2, 3, 5, NULL),
('standard', 'dvt', 'DVT-3', 'PCBA开发', 'task', 1, 3, 20, '电路板开发'),
('standard', 'dvt', 'DVT-3-1', 'PCB Layout', 'task', 2, 1, 5, NULL),
('standard', 'dvt', 'DVT-3-2', 'PCB打样', 'task', 2, 2, 7, NULL),
('standard', 'dvt', 'DVT-3-3', 'PCBA焊接调试', 'task', 2, 3, 8, NULL),
('standard', 'dvt', 'DVT-4', '软件开发', 'task', 1, 4, 30, '嵌入式软件开发'),
('standard', 'dvt', 'DVT-4-1', 'BSP开发', 'task', 2, 1, 10, NULL),
('standard', 'dvt', 'DVT-4-2', '应用软件开发', 'task', 2, 2, 15, NULL),
('standard', 'dvt', 'DVT-4-3', '软件集成测试', 'task', 2, 3, 5, NULL),
('standard', 'dvt', 'DVT-5', 'DVT样机组装', 'task', 1, 5, 5, 'DVT样机组装'),
('standard', 'dvt', 'DVT-6', 'DVT测试验证', 'task', 1, 6, 10, 'DVT全面测试'),
('standard', 'dvt', 'DVT-6-1', '功能测试', 'task', 2, 1, 3, NULL),
('standard', 'dvt', 'DVT-6-2', '性能测试', 'task', 2, 2, 3, NULL),
('standard', 'dvt', 'DVT-6-3', '可靠性测试', 'task', 2, 3, 4, NULL),
('standard', 'dvt', 'DVT-EXIT', 'DVT阶段评审', 'deliverable', 1, 7, 1, 'DVT阶段准出评审'),

-- PVT阶段
('standard', 'pvt', 'PVT', 'PVT阶段', 'milestone', 0, 1, 45, '生产验证测试阶段'),
('standard', 'pvt', 'PVT-1', '工艺验证', 'task', 1, 1, 15, '生产工艺验证'),
('standard', 'pvt', 'PVT-1-1', '组装工艺验证', 'task', 2, 1, 5, NULL),
('standard', 'pvt', 'PVT-1-2', '测试工艺验证', 'task', 2, 2, 5, NULL),
('standard', 'pvt', 'PVT-1-3', '包装工艺验证', 'task', 2, 3, 5, NULL),
('standard', 'pvt', 'PVT-2', '小批量试产', 'task', 1, 2, 10, '小批量试产验证'),
('standard', 'pvt', 'PVT-2-1', '试产准备', 'task', 2, 1, 2, NULL),
('standard', 'pvt', 'PVT-2-2', '试产执行', 'task', 2, 2, 5, NULL),
('standard', 'pvt', 'PVT-2-3', '试产总结', 'task', 2, 3, 3, NULL),
('standard', 'pvt', 'PVT-3', '认证测试', 'task', 1, 3, 20, '产品认证测试'),
('standard', 'pvt', 'PVT-3-1', 'FCC认证', 'task', 2, 1, 15, NULL),
('standard', 'pvt', 'PVT-3-2', 'CE认证', 'task', 2, 2, 15, NULL),
('standard', 'pvt', 'PVT-3-3', 'CCC认证', 'task', 2, 3, 20, NULL),
('standard', 'pvt', 'PVT-4', '量产准备', 'task', 1, 4, 5, '量产准备工作'),
('standard', 'pvt', 'PVT-4-1', '供应商确认', 'task', 2, 1, 2, NULL),
('standard', 'pvt', 'PVT-4-2', '产能规划', 'task', 2, 2, 2, NULL),
('standard', 'pvt', 'PVT-4-3', '文档定版', 'task', 2, 3, 1, NULL),
('standard', 'pvt', 'PVT-EXIT', 'PVT阶段评审', 'deliverable', 1, 5, 1, 'PVT阶段准出评审'),

-- MP阶段
('standard', 'mp', 'MP', 'MP阶段', 'milestone', 0, 1, 30, '量产阶段'),
('standard', 'mp', 'MP-1', '量产爬坡', 'task', 1, 1, 15, '产能爬坡'),
('standard', 'mp', 'MP-1-1', '首批量产', 'task', 2, 1, 5, NULL),
('standard', 'mp', 'MP-1-2', '产能提升', 'task', 2, 2, 10, NULL),
('standard', 'mp', 'MP-2', '品质监控', 'task', 1, 2, 30, '持续品质监控'),
('standard', 'mp', 'MP-2-1', '来料检验', 'task', 2, 1, 30, NULL),
('standard', 'mp', 'MP-2-2', '过程检验', 'task', 2, 2, 30, NULL),
('standard', 'mp', 'MP-2-3', '出货检验', 'task', 2, 3, 30, NULL),
('standard', 'mp', 'MP-3', '持续改进', 'task', 1, 3, 30, '持续优化改进'),
('standard', 'mp', 'MP-EXIT', 'MP阶段评审', 'deliverable', 1, 4, 1, 'MP阶段评审');

-- ============================================================
-- 示例产品数据
-- ============================================================

-- 示例产品
INSERT INTO products (id, code, name, category_id, status, description, specs, created_by) VALUES
    ('prd_demo_001', 'PRD-NIMO-AIR-001', 'NIMO Air', 'cat_platform', 'active', 
     'NIMO Air 智能眼镜平台，超轻量设计，单色显示屏',
     '{"weight": "38g", "display": "Micro-OLED", "resolution": "640x400", "battery": "250mAh", "bluetooth": "5.2"}',
     'u_admin'),
    ('prd_demo_002', 'PRD-NIMO-FRAME-001', 'NIMO Air 镜框 - 商务款', 'cat_frame', 'active',
     'NIMO Air配套镜框，商务风格设计',
     '{"style": "business", "material": "TR90", "color": ["black", "brown"]}',
     'u_admin');

-- ============================================================
-- 示例物料数据
-- ============================================================

INSERT INTO materials (id, code, name, category_id, unit, description, specs, standard_cost, created_by) VALUES
    -- 电子元器件
    ('mat_e_001', 'MAT-E-001', 'Micro-OLED显示屏', 'mcat_electronic', 'pcs', 
     '0.23寸单色Micro-OLED显示屏，640x400分辨率',
     '{"size": "0.23inch", "resolution": "640x400", "type": "Micro-OLED", "color": "green"}',
     85.00, 'u_admin'),
    ('mat_e_002', 'MAT-E-002', '蓝牙模块BT5.2', 'mcat_electronic', 'pcs',
     '蓝牙5.2低功耗模块',
     '{"version": "5.2", "power": "low"}',
     12.50, 'u_admin'),
    ('mat_e_003', 'MAT-E-003', '锂电池250mAh', 'mcat_electronic', 'pcs',
     '250mAh软包锂电池',
     '{"capacity": "250mAh", "voltage": "3.7V"}',
     15.00, 'u_admin'),
    ('mat_e_004', 'MAT-E-004', '主控芯片', 'mcat_electronic', 'pcs',
     'ARM Cortex-M4主控芯片',
     '{"core": "ARM Cortex-M4", "frequency": "120MHz"}',
     8.00, 'u_admin'),
    ('mat_e_005', 'MAT-E-005', 'MEMS麦克风', 'mcat_electronic', 'pcs',
     '数字MEMS麦克风',
     '{"type": "digital", "snr": "65dB"}',
     3.50, 'u_admin'),
    ('mat_e_006', 'MAT-E-006', '触摸传感器', 'mcat_electronic', 'pcs',
     '电容式触摸传感器',
     '{"type": "capacitive"}',
     2.00, 'u_admin'),
    
    -- 结构件
    ('mat_m_001', 'MAT-M-001', '镜腿壳体-左', 'mcat_mechanical', 'pcs',
     '镜腿外壳，左侧',
     '{"material": "PC+ABS", "process": "injection"}',
     5.00, 'u_admin'),
    ('mat_m_002', 'MAT-M-002', '镜腿壳体-右', 'mcat_mechanical', 'pcs',
     '镜腿外壳，右侧',
     '{"material": "PC+ABS", "process": "injection"}',
     5.00, 'u_admin'),
    ('mat_m_003', 'MAT-M-003', '铰链组件', 'mcat_mechanical', 'set',
     '弹簧铰链组件',
     '{"material": "stainless steel", "type": "spring"}',
     8.00, 'u_admin'),
    ('mat_m_004', 'MAT-M-004', '鼻托', 'mcat_mechanical', 'pcs',
     '硅胶鼻托',
     '{"material": "silicone"}',
     1.50, 'u_admin'),
    
    -- 光学器件
    ('mat_o_001', 'MAT-O-001', '光波导镜片', 'mcat_optical', 'pcs',
     'AR光波导镜片',
     '{"type": "waveguide", "fov": "30°"}',
     120.00, 'u_admin'),
    
    -- 包材
    ('mat_p_001', 'MAT-P-001', '包装盒', 'mcat_packaging', 'pcs',
     '产品包装盒',
     '{"size": "180x90x50mm"}',
     8.00, 'u_admin'),
    ('mat_p_002', 'MAT-P-002', '眼镜盒', 'mcat_packaging', 'pcs',
     '眼镜收纳盒',
     '{"material": "PU leather"}',
     15.00, 'u_admin'),
    ('mat_p_003', 'MAT-P-003', '充电线', 'mcat_packaging', 'pcs',
     'USB-C充电线',
     '{"length": "1m", "type": "USB-C"}',
     5.00, 'u_admin');

-- ============================================================
-- 示例BOM数据
-- ============================================================

-- NIMO Air BOM
INSERT INTO bom_headers (id, product_id, version, status, description, total_items, created_by, released_by, released_at) VALUES
    ('bom_air_v1', 'prd_demo_001', '1.0', 'released', 'NIMO Air 正式量产BOM', 13, 'u_admin', 'u_admin', NOW());

-- 更新产品的当前BOM版本
UPDATE products SET current_bom_version = '1.0' WHERE id = 'prd_demo_001';

-- BOM行项 - 顶级物料
INSERT INTO bom_items (id, bom_header_id, material_id, level, sequence, quantity, unit, position, unit_cost, extended_cost) VALUES
    -- 电子模块
    ('bi_001', 'bom_air_v1', 'mat_e_001', 0, 1, 1, 'pcs', 'E1', 85.00, 85.00),
    ('bi_002', 'bom_air_v1', 'mat_e_002', 0, 2, 1, 'pcs', 'E2', 12.50, 12.50),
    ('bi_003', 'bom_air_v1', 'mat_e_003', 0, 3, 1, 'pcs', 'E3', 15.00, 15.00),
    ('bi_004', 'bom_air_v1', 'mat_e_004', 0, 4, 1, 'pcs', 'E4', 8.00, 8.00),
    ('bi_005', 'bom_air_v1', 'mat_e_005', 0, 5, 2, 'pcs', 'E5', 3.50, 7.00),
    ('bi_006', 'bom_air_v1', 'mat_e_006', 0, 6, 1, 'pcs', 'E6', 2.00, 2.00),
    -- 结构件
    ('bi_007', 'bom_air_v1', 'mat_m_001', 0, 7, 1, 'pcs', 'M1', 5.00, 5.00),
    ('bi_008', 'bom_air_v1', 'mat_m_002', 0, 8, 1, 'pcs', 'M2', 5.00, 5.00),
    ('bi_009', 'bom_air_v1', 'mat_m_003', 0, 9, 2, 'set', 'M3', 8.00, 16.00),
    ('bi_010', 'bom_air_v1', 'mat_m_004', 0, 10, 2, 'pcs', 'M4', 1.50, 3.00),
    -- 光学器件
    ('bi_011', 'bom_air_v1', 'mat_o_001', 0, 11, 1, 'pcs', 'O1', 120.00, 120.00),
    -- 包材
    ('bi_012', 'bom_air_v1', 'mat_p_001', 0, 12, 1, 'pcs', 'P1', 8.00, 8.00),
    ('bi_013', 'bom_air_v1', 'mat_p_002', 0, 13, 1, 'pcs', 'P2', 15.00, 15.00);

-- 更新BOM总成本
UPDATE bom_headers SET total_cost = 301.50 WHERE id = 'bom_air_v1';

-- ============================================================
-- 示例项目数据
-- ============================================================

INSERT INTO projects (id, code, name, product_id, status, current_phase, owner_id, planned_start, planned_end, progress, created_by) VALUES
    ('proj_demo_001', 'PROJ-2026-001', 'NIMO Air 2代研发项目', 'prd_demo_001', 'dvt', 'dvt', 'u_admin', 
     '2026-01-15', '2026-07-15', 35, 'u_admin');

-- 项目阶段
INSERT INTO project_phases (id, project_id, phase, name, status, sequence, planned_start, planned_end) VALUES
    ('phase_001', 'proj_demo_001', 'evt', 'EVT阶段', 'completed', 1, '2026-01-15', '2026-02-28'),
    ('phase_002', 'proj_demo_001', 'dvt', 'DVT阶段', 'in_progress', 2, '2026-03-01', '2026-04-30'),
    ('phase_003', 'proj_demo_001', 'pvt', 'PVT阶段', 'pending', 3, '2026-05-01', '2026-06-15'),
    ('phase_004', 'proj_demo_001', 'mp', 'MP阶段', 'pending', 4, '2026-06-16', '2026-07-15');

-- 示例任务
INSERT INTO tasks (id, project_id, phase_id, code, name, task_type, status, priority, assignee_id, planned_start, planned_end, progress, level, sequence, created_by) VALUES
    ('task_001', 'proj_demo_001', 'phase_002', 'DVT-1', '详细设计', 'task', 'completed', 'high', 'u_admin', '2026-03-01', '2026-03-15', 100, 0, 1, 'u_admin'),
    ('task_002', 'proj_demo_001', 'phase_002', 'DVT-2', '模具开发', 'task', 'in_progress', 'high', 'u_admin', '2026-03-10', '2026-04-05', 60, 0, 2, 'u_admin'),
    ('task_003', 'proj_demo_001', 'phase_002', 'DVT-3', 'PCBA开发', 'task', 'in_progress', 'high', 'u_admin', '2026-03-10', '2026-03-30', 80, 0, 3, 'u_admin'),
    ('task_004', 'proj_demo_001', 'phase_002', 'DVT-4', '软件开发', 'task', 'in_progress', 'medium', 'u_admin', '2026-03-01', '2026-04-15', 45, 0, 4, 'u_admin'),
    ('task_005', 'proj_demo_001', 'phase_002', 'DVT-5', 'DVT样机组装', 'task', 'pending', 'medium', 'u_admin', '2026-04-05', '2026-04-10', 0, 0, 5, 'u_admin');

-- ============================================================
-- 完成
-- ============================================================

SELECT 'Seed data inserted successfully!' AS result;
