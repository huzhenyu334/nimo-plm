# 03 - PLM ECN变更管理

> 版本: v1.0 | 日期: 2026-02-19 | 编制: PM Agent
> 涵盖：ECN流程、品类差异化字段、执行跟踪、审批流

---

## 一、功能概述

ECN（Engineering Change Notice）工程变更管理，管理产品从设计到量产过程中的所有变更，确保变更可控、可追溯。

### 1.1 已实现功能

| 功能 | 状态 | 页面 |
|------|------|------|
| ECN列表页 | ✅ 已实现 | /ecn |
| ECN详情页（独立页面） | ✅ 已实现 | /ecn/:id |
| ECN创建/编辑（分步表单） | ✅ 已实现 | /ecn/new, /ecn/:id/edit |
| 受影响项管理 | ✅ 已实现 | ECN详情Tab |
| 审批流程 | ✅ 已实现 | 飞书集成 |

---

## 二、核心流程

### 2.1 状态流转

```
草稿(DRAFT) ──提交──▶ 待审批(SUBMITTED) ──审批通过──▶ 执行中(APPROVED) ──全部完成──▶ 已关闭(CLOSED)
                          │                           │
                          ▼                           ▼
                       驳回(REJECTED)              部分完成（可查进度）
                       → 退回修改(DRAFT)
```

**单阶段设计**：直接创建ECN（不需要ECR→ECN两级流转），适合小团队快速迭代。

### 2.2 ECN类型

| 类型 | 说明 | 审批级别 |
|-----|------|---------|
| 设计变更 | 电路/结构/光学设计修改 | 部门负责人 + PM |
| 物料变更 | 替换物料、更换供应商 | 部门负责人 + PM |
| 工艺变更 | 加工工艺/表面处理调整 | 部门负责人 |
| 规格变更 | 产品规格参数调整 | PM + 技术总监 |
| 文档变更 | 图纸/规格书更新 | 部门负责人 |

### 2.3 紧急程度

| 级别 | 审批规则 |
|------|---------|
| 常规 | 多部门会签（串行或并行） |
| 紧急 | 简化审批（仅PM确认） |
| 特急 | PM确认 + 事后补签 |

---

## 三、数据模型

### 3.1 ECN主表 (engineering_changes)

| 字段 | 类型 | 必填 | 说明 |
|-----|------|-----|------|
| id | UUID | 是 | 主键 |
| ecn_code | VARCHAR(50) | 是 | ECN编号（ECN-YYYY-NNNN） |
| title | VARCHAR(200) | 是 | 变更标题 |
| type | ENUM | 是 | 变更类型 |
| urgency | ENUM | 是 | 常规/紧急/特急 |
| product_id | UUID | 否 | 关联产品 |
| project_id | UUID | 否 | 关联项目 |
| status | ENUM | 是 | ECN状态 |
| reason | TEXT | 是 | 变更原因（富文本） |
| description | TEXT | 是 | 变更描述 |
| technical_plan | TEXT | 否 | 技术方案（富文本） |
| impact_analysis | TEXT | 否 | 影响分析 |
| cost_impact | DECIMAL(12,2) | 否 | 成本影响 |
| schedule_impact | INTEGER | 否 | 进度影响（天） |
| planned_date | DATE | 否 | 计划实施日期 |
| completion_rate | INTEGER | 否 | 执行完成百分比 |
| approval_mode | VARCHAR(16) | 否 | serial(串行) / parallel(并行) |
| requested_by | VARCHAR(64) | 是 | 发起人 |
| effective_date | DATE | 否 | 生效日期 |
| created_at | TIMESTAMP | 是 | 创建时间 |
| updated_at | TIMESTAMP | 是 | 更新时间 |

### 3.2 受影响项表 (ecn_affected_items)

| 字段 | 类型 | 说明 |
|-----|------|------|
| id | UUID | 主键 |
| ecn_id | UUID | 关联ECN |
| material_id | UUID | 关联物料 |
| material_code | VARCHAR(64) | 物料编号 |
| material_name | VARCHAR(256) | 物料名称 |
| change_description | TEXT | 变更描述 |
| before_value | JSONB | 变更前参数快照 |
| after_value | JSONB | 变更后参数 |
| cost_impact | DECIMAL(12,4) | 单价变化 |
| inventory_action | VARCHAR(20) | 用完为止/立即切换/报废/返工 |
| affected_bom_ids | JSONB | 引用该物料的BOM ID列表 |
| category_fields | JSONB | 品类专用变更字段（见第四节） |

### 3.3 执行任务表 (ecn_tasks)

| 字段 | 类型 | 说明 |
|-----|------|------|
| id | VARCHAR(32) | 主键 |
| ecn_id | VARCHAR(32) | 关联ECN |
| type | VARCHAR(32) | bom_update/drawing_update/supplier_notify/inventory_handle/doc_update/sop_update |
| title | VARCHAR(256) | 任务标题 |
| description | TEXT | 任务描述 |
| assignee_id | VARCHAR(32) | 负责人 |
| due_date | DATE | 截止日期 |
| status | VARCHAR(16) | pending/in_progress/completed/skipped |
| completed_at | TIMESTAMP | 完成时间 |
| sort_order | INT | 排序 |

### 3.4 操作历史表 (ecn_history)

| 字段 | 类型 | 说明 |
|-----|------|------|
| id | VARCHAR(32) | 主键 |
| ecn_id | VARCHAR(32) | 关联ECN |
| action | VARCHAR(32) | created/updated/submitted/approved/rejected/task_completed |
| user_id | VARCHAR(32) | 操作人 |
| detail | JSONB | 变更详情快照 |
| created_at | TIMESTAMP | 操作时间 |

---

## 四、品类差异化变更字段

### 4.1 设计原则

不同品类物料变更时关注的技术参数完全不同。系统根据物料分类自动展示对应的变更字段集。

### 4.2 电子料变更字段

| 字段 | 类型 | 说明 |
|------|------|------|
| 规格变更 | 文本对比 | 如 10μF→22μF |
| 封装变更 | 文本对比 | 如 0805→0603 |
| 制造商变更 | 文本对比 | 品牌替换 |
| MPN变更 | 文本对比 | 制造商料号 |
| 替代料信息 | 多选 | form/fit/function替代 |
| PCB Layout影响 | 单选+备注 | 无影响/需改Layout/兼容 |
| 焊接工艺影响 | 单选 | 无/需调整温度曲线/需换锡膏 |

### 4.3 结构件变更字段

| 字段 | 类型 | 说明 |
|------|------|------|
| 图纸版本 | 文件对比 | 变更前后图纸 |
| 尺寸/公差变更 | 文本对比 | 关键尺寸变化 |
| 材质变更 | 文本对比 | 如 PC→PC+ABS |
| 表面处理变更 | 文本对比 | 如 喷砂→阳极氧化 |
| 模具影响 | 单选+备注 | 无影响/需改模/需新开模 |
| 装配关系影响 | 多选 | 影响哪些相邻零件 |
| CMF影响 | 单选 | 是否影响CMF变体 |

### 4.4 光学件变更字段

| 字段 | 类型 | 说明 |
|------|------|------|
| 光学参数变更 | 文本对比 | 透光率/折射率等 |
| 镀膜工艺变更 | 文本对比 | — |
| 对准精度影响 | 文本 | 是否影响光机对准 |

### 4.5 通用变更字段（所有品类）

| 字段 | 类型 | 说明 |
|------|------|------|
| 变更前值 | JSONB | 变更前参数快照 |
| 变更后值 | JSONB | 变更后参数 |
| 变更描述 | 文本 | 自由描述 |
| 成本影响 | 数字 | 单价变化 |
| 库存处理方式 | 单选 | 用完为止/立即切换/报废/返工 |

### 4.6 跨领域联动提醒

系统根据变更物料品类，**自动提示可能的跨领域影响**：

| 变更品类 | 自动提醒 |
|---------|---------|
| 结构件 | "与以下电子元件有装配关系，请确认" |
| 结构件（外观） | "有CMF变体定义，变更可能影响外观方案" |
| 电子料（封装变更） | "可能影响PCB Layout和SMT程序" |
| 光学件 | "可能影响结构件的对准装配" |

---

## 五、创建ECN流程（分步表单）

### Step 1 — 基本信息（必填）
- 标题、关联产品、变更类型、紧急程度、变更原因

### Step 2 — 受影响项（可后续补充）
- 搜索选择受影响物料/BOM项
- 系统自动展示"哪些BOM引用了该物料"
- 按物料品类显示不同的变更字段

### Step 3 — 影响范围评估（Checklist）
- ☐ 是否影响装配SOP → 填写影响说明和处理方式
- ☐ 是否影响工装/模具
- ☐ 是否影响测试方案
- ☐ 是否需要供应商配合
- ☐ 是否影响已发货产品

### Step 4 — 附件
- 上传支撑文件（图纸、测试报告、对比图等）

---

## 六、页面设计

### 6.1 ECN列表页 (/ecn)

**顶部统计卡片**：待我审批 | 进行中 | 本月新建 | 本月关闭

**筛选栏**：状态Tab + 搜索框 + 高级筛选（产品、类型、紧急程度、日期、申请人）

**列表项**（卡片/表格可切换）

### 6.2 ECN详情页 (/ecn/:id)

5个Tab：
1. **变更概要** — 基本信息、技术方案、附件
2. **受影响项 & 变更对比** — 物料列表 + 变更前后diff
3. **审批流程** — 审批节点可视化
4. **执行任务** — Checklist风格，支持进度追踪
5. **变更历史** — 时间线

---

## 七、执行跟踪

ECN批准后自动生成执行任务清单：

| 任务类型 | 描述 | 自动化程度 |
|---------|------|-----------|
| BOM更新 | 应用变更到相关BOM | 一键应用 |
| 图纸更新 | 上传新版图纸 | 手动上传 |
| 供应商通知 | 通知受影响供应商 | 自动生成通知草稿 |
| 库存处理 | 报废/返工/用完为止 | 手动决策 |
| 文档更新 | 更新相关技术文档 | 手动 |
| SOP更新 | 更新受影响SOP | 手动（如勾选了SOP影响） |

全部任务完成 → ECN自动关闭。

---

## 八、API接口汇总

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/ecns | ECN列表 |
| GET | /api/v1/ecns/:id | ECN详情 |
| POST | /api/v1/ecns | 创建ECN |
| PUT | /api/v1/ecns/:id | 更新ECN |
| POST | /api/v1/ecns/:id/submit | 提交审批 |
| POST | /api/v1/ecns/:id/approve | 审批通过 |
| POST | /api/v1/ecns/:id/reject | 审批驳回 |
| GET | /api/v1/ecns/stats | 统计数据 |
| GET | /api/v1/ecns/my-pending | 待我审批列表 |
| GET | /api/v1/ecns/:id/impact-analysis | 自动影响分析 |
| POST | /api/v1/ecns/:id/affected-items | 添加受影响项 |
| PUT | /api/v1/ecns/:id/affected-items/:itemId | 更新受影响项 |
| DELETE | /api/v1/ecns/:id/affected-items/:itemId | 移除受影响项 |
| GET | /api/v1/ecns/:id/tasks | 获取执行任务 |
| POST | /api/v1/ecns/:id/tasks | 创建执行任务 |
| PUT | /api/v1/ecns/:id/tasks/:taskId | 更新执行任务状态 |
| POST | /api/v1/ecns/:id/apply-bom-changes | 一键应用BOM变更 |
| GET | /api/v1/ecns/:id/history | 操作历史 |

---

## 九、规划中功能

| 功能 | 优先级 | 说明 |
|------|--------|------|
| 品类差异化变更字段 | P1 | 电子/结构/光学各自的专用字段 |
| 执行任务管理 | P1 | ECN批准后自动生成执行任务 |
| BOM变更一键应用 | P1 | 系统自动修改受影响BOM行项 |
| 跨领域联动提醒 | P1 | 自动提示跨领域影响 |
| 变更统计报表 | P2 | 变更频率、类型分布等 |
| ECN基于物料（非项目） | P3 | 依赖物料独立化架构演进 |

---

*本文档基于已有代码实体（ecn.go, bom_ecn.go）和ecn-redesign.md PRD整理。*
