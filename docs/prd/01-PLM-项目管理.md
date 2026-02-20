# 01 - PLM 项目管理

> 版本: v1.0 | 日期: 2026-02-19 | 编制: PM Agent
> 涵盖：项目管理、任务管理、项目模板、审批流程、甘特图、任务表单

---

## 一、功能概述

项目管理是PLM的核心模块，围绕智能眼镜产品开发的**四阶段流程**（EVT→DVT→PVT→MP）管理项目全生命周期。

### 1.1 已实现功能清单

| 功能 | 状态 | 页面路由 |
|------|------|---------|
| 项目列表 | ✅ 已实现 | /projects |
| 项目详情（多Tab） | ✅ 已实现 | /projects/:id |
| 任务管理（甘特图） | ✅ 已实现 | 项目详情内 |
| 从模板创建项目 | ✅ 已实现 | /templates |
| 审批管理 | ✅ 已实现 | /approvals |
| 我的任务 | ✅ 已实现 | /my-tasks |
| 工作台仪表盘 | ✅ 已实现 | /dashboard |
| 任务表单系统 | ✅ 已实现 | 任务完成时填写 |
| 任务确认/驳回 | ✅ 已实现 | PM操作 |
| 角色管理 | ✅ 已实现 | /roles |

---

## 二、项目数据模型

### 2.1 项目表 (projects)

| 字段 | 类型 | 必填 | 说明 |
|-----|------|-----|------|
| id | UUID | 是 | 主键 |
| project_code | VARCHAR(50) | 是 | 项目编码 |
| name | VARCHAR(200) | 是 | 项目名称 |
| codename | VARCHAR(100) | 否 | 项目代号（如 BlackHole, Meteor） |
| description | TEXT | 否 | 项目描述 |
| product_id | UUID | 否 | 关联产品 |
| current_phase | ENUM | 是 | 当前阶段（EVT/DVT/PVT/MP） |
| status | ENUM | 是 | 项目状态 |
| manager_id | VARCHAR(64) | 是 | 项目经理（飞书ID） |
| planned_start_date | DATE | 是 | 计划开始 |
| planned_end_date | DATE | 是 | 计划结束 |
| actual_start_date | DATE | 否 | 实际开始 |
| actual_end_date | DATE | 否 | 实际结束 |
| progress | INTEGER | 是 | 整体进度(0-100) |
| template_id | UUID | 否 | 来源模板 |
| created_at | TIMESTAMP | 是 | 创建时间 |
| updated_at | TIMESTAMP | 是 | 更新时间 |

### 2.2 任务表 (tasks)

| 字段 | 类型 | 必填 | 说明 |
|-----|------|-----|------|
| id | UUID | 是 | 主键 |
| project_id | UUID | 是 | 关联项目 |
| task_code | VARCHAR(50) | 是 | 任务编码 |
| name | VARCHAR(200) | 是 | 任务名称 |
| description | TEXT | 否 | 任务描述 |
| phase | ENUM | 是 | EVT/DVT/PVT/MP |
| parent_id | UUID | 否 | 父任务ID |
| seq_no | INTEGER | 是 | 序号 |
| assignee_id | VARCHAR(64) | 否 | 负责人（飞书ID） |
| assignee_dept | VARCHAR(100) | 否 | 负责部门 |
| status | ENUM | 是 | 任务状态 |
| priority | ENUM | 是 | LOW/MEDIUM/HIGH/URGENT |
| progress | INTEGER | 是 | 进度(0-100) |
| planned_start | DATE | 是 | 计划开始 |
| planned_end | DATE | 是 | 计划结束 |
| actual_start | DATE | 否 | 实际开始 |
| actual_end | DATE | 否 | 实际结束 |
| duration_days | INTEGER | 是 | 计划工期 |
| is_milestone | BOOLEAN | 是 | 是否里程碑 |
| task_type | VARCHAR(20) | 是 | normal / srm_procurement |
| linked_srm_project_id | VARCHAR(32) | 否 | 关联SRM采购项目（task_type=srm_procurement时） |
| feishu_task_id | VARCHAR(100) | 否 | 飞书任务ID |
| created_at | TIMESTAMP | 是 | 创建时间 |
| updated_at | TIMESTAMP | 是 | 更新时间 |

### 2.3 任务状态流转

```
pending → in_progress → completed → confirmed
              ↑                        |
              └── (驳回，回到进行中) ───┘
```

| 状态 | 说明 | 操作人 |
|------|------|--------|
| pending | 等待前置任务完成 | 系统自动 |
| in_progress | 执行中 | 系统自动 |
| completed | 工程师提交完成（含表单） | 工程师 |
| confirmed | 项目经理验收通过 | 项目经理 |
| blocked | 被阻塞 | 手动标记 |
| cancelled | 已取消 | 手动标记 |

### 2.4 任务依赖关系表 (task_dependencies)

| 字段 | 类型 | 说明 |
|-----|------|------|
| id | UUID | 主键 |
| task_id | UUID | 当前任务 |
| depends_on_task_id | UUID | 依赖任务 |
| dependency_type | ENUM | FS(完成-开始)/SS/FF/SF |
| lag_days | INTEGER | 延迟天数 |

---

## 三、四阶段项目模板

### 3.1 项目模板系统

系统支持预定义项目模板，包含：
- 阶段划分
- 任务列表（含层级/依赖关系/工期/负责部门）
- 任务表单模板（完成任务时需填写的表单）

从模板创建项目时，自动生成所有任务和表单。

### 3.2 EVT阶段模板（工程验证测试，4-6周）

| 任务编码 | 任务名称 | 负责部门 | 工期 | 前置任务 |
|---------|---------|---------|------|---------|
| EVT-001 | 硬件设计完成 | 硬件研发 | 14天 | - |
| EVT-001-01 | 电路原理图设计 | 硬件研发 | 7天 | - |
| EVT-001-02 | PCB布局设计 | 硬件研发 | 5天 | EVT-001-01 |
| EVT-001-03 | 结构3D设计 | 结构研发 | 10天 | - |
| EVT-001-04 | 光学设计 | 光学研发 | 8天 | - |
| EVT-001-05 | 散热设计验证 | 硬件研发 | 3天 | EVT-001-02, EVT-001-03 |
| EVT-002 | EVT样机制作 | 样机工程 | 10天 | EVT-001 |
| EVT-002-01 | PCB打样 | 样机工程 | 5天 | EVT-001-02 |
| EVT-002-02 | 结构件CNC加工 | 样机工程 | 7天 | EVT-001-03 |
| EVT-002-03 | 光学件采购 | 采购部 | 10天 | EVT-001-04 |
| EVT-002-04 | 样机组装 | 样机工程 | 3天 | EVT-002-01,02,03 |
| EVT-002-05 | 基本功能测试 | 测试部 | 2天 | EVT-002-04 |
| EVT-003 | EVT测试验证 | 测试部 | 7天 | EVT-002 |
| EVT-004 | EVT评审 | 项目管理 | 2天 | EVT-003 |

### 3.3 DVT阶段（设计验证测试，8-12周）
- 设计优化、DVT样机制作（50-100台）
- 全面测试验证、认证准备
- DVT评审

### 3.4 PVT阶段（生产验证测试，6-8周）
- 生产准备、PVT试产（500-1000台）
- 量产验证、市场准备
- PVT评审

### 3.5 MP阶段（批量生产，持续）
- 量产启动、持续改进、市场反馈

---

## 四、任务表单系统

### 4.1 概述

任务可关联自定义表单，工程师完成任务前必须提交表单。这确保每个任务都有明确的交付物。

### 4.2 数据模型

**task_forms 表（表单定义）**

| 字段 | 类型 | 说明 |
|-----|------|------|
| id | VARCHAR(32) | 主键 |
| task_id | VARCHAR(32) | 关联任务（一对一） |
| name | VARCHAR(128) | 表单名称 |
| description | TEXT | 表单说明 |
| fields | JSONB | 表单字段定义 |
| created_by | VARCHAR(32) | 创建人 |

**task_form_submissions 表（提交记录）**

| 字段 | 类型 | 说明 |
|-----|------|------|
| id | VARCHAR(32) | 主键 |
| form_id | VARCHAR(32) | 关联表单 |
| task_id | VARCHAR(32) | 关联任务 |
| data | JSONB | 提交的数据 |
| files | JSONB | 上传文件信息 |
| submitted_by | VARCHAR(32) | 提交人 |
| version | INT | 版本号（支持驳回后重新提交） |

### 4.3 支持的表单字段类型

| 类型 | 说明 |
|------|------|
| text | 单行文本 |
| textarea | 多行文本 |
| number | 数字 |
| select | 单选下拉 |
| multiselect | 多选 |
| date | 日期 |
| file | 文件上传（支持accept/multiple） |
| checkbox | 复选框 |

### 4.4 工作流程

```
工程师在"我的任务"页面点击"完成"
    ↓
如果任务有关联表单 → 弹出表单Modal → 填写并提交
如果无表单 → 直接完成
    ↓
任务状态变为 completed
    ↓
项目经理审核 → 确认(confirmed) 或 驳回(回到in_progress)
```

---

## 五、甘特图功能

### 5.1 显示要素

- 任务条：显示时间跨度
- 依赖线：显示任务依赖关系
- 里程碑：菱形标记
- 进度条：任务完成百分比
- 今日线：当前日期标记
- SRM采购任务：实时从SRM读取进度

### 5.2 交互功能

- 拖拽调整任务时间
- 点击展开/折叠子任务
- 缩放时间刻度（日/周/月）
- 筛选（按阶段/负责人/状态）
- SRM采购任务点击可跳转到SRM详情

---

## 六、审批管理

### 6.1 审批场景

| 审批场景 | 审批人规则 |
|---------|-----------|
| BOM审批 | 部门负责人 → PM |
| ECN审批 | 根据ECN类型和紧急程度动态匹配 |
| 阶段评审 | PM + 技术负责人 |
| 文档发布 | 部门负责人 |

### 6.2 审批定义管理

系统支持管理员配置审批模板（/approvals 页面），定义：
- 审批名称和类型
- 审批节点（串行/并行）
- 每个节点的审批人规则

### 6.3 飞书审批集成

- 审批提交时推送飞书审批卡片
- 支持在飞书内一键批准/驳回
- 审批结果自动回写系统

---

## 七、我的任务（/my-tasks）

### 7.1 用户故事

作为工程师，我需要一个集中的个人任务视图，查看分配给我的所有任务并完成它们。

### 7.2 页面设计

- 顶部：状态Tab切换（全部 / 进行中 / 已完成 / 已确认）
- 支持按项目筛选
- 任务列表：任务名、所属项目、截止日期、状态、是否有表单
- 进行中的任务显示"完成"按钮

### 7.3 API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/my/tasks | 我的任务列表（支持status/project_id/page/sort筛选） |
| POST | /api/v1/my/tasks/:taskId/complete | 完成任务（含表单提交） |

---

## 八、工作台仪表盘（/dashboard）

### 8.1 展示内容

- 我的待办任务数
- 待我审批的审批单数
- 进行中的项目概览
- 即将到期的任务
- 最近变更（ECN）

---

## 九、API接口汇总

### 9.1 项目接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/projects | 创建项目 |
| GET | /api/v1/projects | 查询项目列表 |
| GET | /api/v1/projects/:id | 查询项目详情 |
| PUT | /api/v1/projects/:id | 更新项目 |
| GET | /api/v1/projects/:id/tasks | 查询任务列表（甘特图数据） |
| POST | /api/v1/tasks | 创建任务 |
| PUT | /api/v1/tasks/:id | 更新任务 |
| PATCH | /api/v1/tasks/:id/progress | 更新任务进度 |
| POST | /api/v1/projects/:id/tasks/:taskId/confirm | 确认任务 |
| POST | /api/v1/projects/:id/tasks/:taskId/reject | 驳回任务 |

### 9.2 任务表单接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/projects/:id/tasks/:taskId/form | 获取任务表单定义 |
| PUT | /api/v1/projects/:id/tasks/:taskId/form | 创建/更新任务表单 |
| GET | /api/v1/projects/:id/tasks/:taskId/form/submission | 获取表单提交内容 |
| POST | /api/v1/upload | 通用文件上传 |

### 9.3 模板接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/templates | 获取模板列表 |
| GET | /api/v1/templates/:id | 获取模板详情 |
| POST | /api/v1/templates | 创建模板 |
| PUT | /api/v1/templates/:id | 更新模板 |
| GET | /api/v1/templates/:id/task-forms | 获取模板表单定义 |
| POST | /api/v1/templates/:id/task-forms | 创建模板表单 |

### 9.4 审批接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/approvals | 审批列表 |
| POST | /api/v1/approvals/:id/approve | 审批通过 |
| POST | /api/v1/approvals/:id/reject | 审批驳回 |

---

## 十、规划中功能

| 功能 | 优先级 | 说明 |
|------|--------|------|
| SRM采购任务进度实时同步 | P1 | PLM甘特图实时显示SRM采购进度 |
| 飞书任务双向同步 | P2 | PLM任务↔飞书任务状态同步 |
| 资源负载视图 | P2 | 在甘特图中显示人员工作负荷 |
| 关键路径分析 | P2 | 自动计算并高亮关键路径 |

---

*本文档基于已有代码实体（project.go, task_form.go, template.go, approval.go等）和现有PRD整理。*
