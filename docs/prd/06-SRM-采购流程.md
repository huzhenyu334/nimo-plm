# 06 - SRM 采购流程

> 版本: v1.0 | 日期: 2026-02-19 | 编制: PM Agent
> 涵盖：采购项目、采购需求、采购订单、来料检验、多轮打样、交期管控

---

## 一、功能概述

SRM采购流程覆盖从BOM审批到来料验收的全链路，核心理念是**文档驱动**（非任务驱动）。

### 1.1 已实现功能

| 功能 | 状态 | 页面 |
|------|------|------|
| 采购总览仪表盘 | ✅ 已实现 | /srm |
| 采购需求管理 | ✅ 已实现 | /srm/purchase-requests |
| 采购订单管理 | ✅ 已实现 | /srm/purchase-orders |
| 来料检验 | ✅ 已实现 | /srm/inspections |
| PLM→SRM自动推送 | ✅ 已实现 | BOM审批→创建PR |

### 1.2 设计原则

**文档驱动（非任务驱动）**：每种单据（PR/PO/GR/IQC）是独立文档，有自己的状态机，文档之间通过引用关联。

| 维度 | 任务驱动（PLM模式） | 文档驱动（SRM模式） |
|------|-------------------|-------------------|
| 用户思维 | "完成任务" | "处理这张单" |
| 异常处理 | 需动态插入任务 | 直接创建新文档 |
| 多轮打样 | 同一任务回退 | 创建Round 2的PO |
| 并行 | 需配置并行节点 | 天然并行 |

---

## 二、核心流程

### 2.1 采购全流程

```
PLM BOM审批通过
    ↓
SRM自动创建采购需求(PR) + 零件清单
    ↓
采购员寻源：选供应商 → 询价比价
    ↓
创建采购订单(PO) → 审批 → 发送供应商
    ↓
供应商发货 → 收货(GR)
    ↓
来料检验(IQC) → 通过/不通过
    ├── 通过 → 入库 → 零件完成
    └── 不通过 → 8D改进 → 重打(Round N+1)
    ↓
所有零件通过 → 采购项目完成 → 进度回写PLM
```

### 2.2 文档关系

```
采购需求(PR) 1:N PR行项
PR行项 1:N 采购订单(PO)  ← 多轮打样
PO 1:N 收货记录(GR)
GR 1:1 检验记录(IQC)
IQC(不通过) 1:1 8D改进单
```

---

## 三、采购需求（PR）

### 3.1 数据来源

- **自动创建**：PLM BOM审批通过时，系统自动创建PR + 零件清单
- **手动创建**：采购员手动创建（补充物料、紧急需求等）

### 3.2 PR状态流转

```
draft → pending → approved → sourcing → completed
```

### 3.3 编码规则

格式：`PR-{年}-{4位流水}`，如 `PR-2026-0001`

---

## 四、采购订单（PO）

### 4.1 PO状态流转

| 状态 | 说明 | 允许操作 |
|-----|------|---------|
| DRAFT | 新建未提交 | 编辑、删除、提交 |
| PENDING | 已提交待审批 | 审批、驳回 |
| APPROVED | 审批通过 | 发送供应商 |
| SENT | 已发送 | 收货 |
| PARTIAL | 部分到货 | 继续收货 |
| RECEIVED | 全部到货 | 关闭 |
| CLOSED | 完结 | — |
| CANCELLED | 取消 | — |

### 4.2 多轮打样支持

PO增加round字段，支持多轮打样追踪：

```
零件A: 外壳(上)
├── Round 1: PO-001(供应商X) → GR-001 → IQC-001(不通过) → 8D-001
├── Round 2: PO-005(供应商X, 修模后) → GR-003 → IQC-004(不通过) → 8D-002
└── Round 3: PO-009(供应商Y, 换供应商) → GR-006 → IQC-007(通过 ✓)
```

### 4.3 PO关键字段

| 字段 | 说明 |
|-----|------|
| round | 轮次（默认1） |
| prev_po_id | 上一轮PO |
| related_8d_id | 关联8D改进单 |
| srm_project_id | 关联采购项目 |

### 4.4 编码规则

格式：`PO-{年}-{4位流水}`，如 `PO-2026-0001`

---

## 五、来料检验（IQC）

### 5.1 检验流程

```
PO到货 → 创建检验单 → IQC工程师检验 → 通过/不通过
                                        │
                               ├── 通过 → 入库
                               └── 不通过 → 自动创建8D → 通知采购员
```

### 5.2 检验结果

| 结果 | 说明 |
|------|------|
| PASS | 检验通过，可入库 |
| FAIL | 检验不通过，触发8D |
| CONDITIONAL | 让步接收（有条件通过） |

### 5.3 编码规则

格式：`IQC-{年}-{4位流水}`，如 `IQC-2026-0001`

---

## 六、采购项目化管理（规划中）

### 6.1 概述

SRM以"采购项目"为核心管理单元，每个采购项目对应PLM的一个阶段备料需求。

### 6.2 PLM↔SRM桥接

```
PLM 研发项目: Meteor智能眼镜
├── EVT阶段
│   ├── ...
│   ├── 🔗 EVT打样采购 ← PLM特殊任务，关联SRM
│   │   └── 进度: 23/50通过(46%) ← 从SRM实时同步
│   └── ...
```

### 6.3 采购项目表 (srm_projects，规划中)

| 字段 | 类型 | 说明 |
|-----|------|------|
| id | VARCHAR(32) | 主键 |
| code | VARCHAR(32) | 编码（SRMP-2026-0001） |
| name | VARCHAR(200) | 名称 |
| type | VARCHAR(20) | sample(打样)/production(量产) |
| phase | VARCHAR(20) | EVT/DVT/PVT/MP |
| status | VARCHAR(20) | active/completed/cancelled |
| plm_project_id | VARCHAR(32) | 关联PLM项目 |
| plm_task_id | VARCHAR(32) | 关联PLM采购任务 |
| plm_bom_id | VARCHAR(32) | 来源BOM |
| total_items | INT | 总零件数 |
| passed_count | INT | 检验通过数 |
| target_date | DATE | 目标完成日期 |

### 6.4 进度同步

PLM查询采购任务进度时，实时从SRM读取：
- 总零件数、通过数 → 计算百分比
- 超期零件列表 → 显示预警

---

## 七、交期管理（规划中）

### 7.1 交期数据来源（三级优先级）

1. 供应商+物料组合交期（最精确）
2. 具体物料交期
3. 物料分类默认交期

### 7.2 交期刚性管控

- 交期定死，延期必须审批
- 距交期3天未下单 → 自动预警（飞书通知）
- 超期 → 采购员必须提交延期申请

### 7.3 延期审批 (srm_delay_requests，规划中)

| 字段 | 类型 | 说明 |
|-----|------|------|
| id | VARCHAR(32) | 主键 |
| srm_project_id | VARCHAR(32) | 采购项目 |
| material_name | VARCHAR(200) | 延期零件 |
| original_days | INT | 原交期 |
| requested_days | INT | 申请延长到 |
| reason_type | VARCHAR(50) | supplier_capacity/design_change/quality_issue/other |
| reason | TEXT | 延期原因 |
| status | VARCHAR(20) | pending/approved/rejected |

---

## 八、数据展现方式（规划中）

### 8.1 看板视图 — 采购员核心工作台

```
┌──────────┬──────────┬──────────┬──────────┬──────────┬──────────┐
│ 待询价(8) │ 已询价(5) │ 已下单(12)│ 已发货(6) │ 检验中(4) │ 已通过(15)│
│ 光学棱镜  │ 主板PCB   │ 外壳R2   │ 排线A    │ 电池     │ 螺丝M1   │
│ 🔴超期!  │ ⏰还剩5天 │ ⏰8天   │ 明天到货  │ 待检     │ ✓通过    │
└──────────┴──────────┴──────────┴──────────┴──────────┴──────────┘
```

### 8.2 甘特图视图 — 管理层看时间线

每个零件一行，显示标准交期线、实际进度、多轮打样。

### 8.3 角色默认视图

| 角色 | 默认视图 |
|------|---------|
| 采购员 | 看板 |
| 采购经理 | 甘特图 |
| 研发PM | PLM内概览 |
| 管理层 | 仪表盘 |

---

## 九、操作日志（规划中）

### 9.1 通用日志表 (srm_activity_logs)

所有SRM文档共享一张操作日志表：

| 字段 | 类型 | 说明 |
|-----|------|------|
| id | VARCHAR(32) | 主键 |
| entity_type | VARCHAR(50) | project/pr/po/inspection/supplier/8d |
| entity_id | VARCHAR(32) | 关联文档ID |
| action | VARCHAR(50) | 操作类型 |
| from_status | VARCHAR(20) | 变更前状态 |
| to_status | VARCHAR(20) | 变更后状态 |
| content | TEXT | 操作描述/评论 |
| attachments | JSONB | 附件 |
| operator_id | VARCHAR(32) | 操作人 |
| created_at | TIMESTAMP | 操作时间 |

### 9.2 评论功能

每个文档支持添加评论（action='comment'），支持附件。

---

## 十、API接口汇总

### 10.1 采购需求接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/srm/purchase-requests | PR列表 |
| POST | /api/v1/srm/purchase-requests | 创建PR |
| GET | /api/v1/srm/purchase-requests/:id | PR详情 |
| PUT | /api/v1/srm/purchase-requests/:id | 更新PR |

### 10.2 采购订单接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/srm/purchase-orders | PO列表 |
| POST | /api/v1/srm/purchase-orders | 创建PO |
| GET | /api/v1/srm/purchase-orders/:id | PO详情 |
| PUT | /api/v1/srm/purchase-orders/:id | 更新PO |
| POST | /api/v1/srm/purchase-orders/:id/submit | 提交审批 |
| POST | /api/v1/srm/purchase-orders/:id/approve | 审批通过 |
| POST | /api/v1/srm/purchase-orders/:id/send | 发送供应商 |
| POST | /api/v1/srm/purchase-orders/:id/receive | 收货 |

### 10.3 来料检验接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/srm/inspections | 检验列表 |
| POST | /api/v1/srm/inspections | 创建检验记录 |
| PUT | /api/v1/srm/inspections/:id | 更新检验结果 |

---

## 十一、规划中功能

| 功能 | 优先级 | 说明 |
|------|--------|------|
| 采购项目化管理 | P0 | srm_projects表 + PLM桥接 |
| 看板视图 | P0 | 采购员核心工作台 |
| 交期自动计算 | P1 | 物料标准交期 + 回写PLM |
| 延期审批流程 | P1 | 交期刚性管控 |
| 操作日志 | P1 | 全流程审计轨迹 |
| 甘特图视图 | P2 | 管理层时间线视图 |
| 多轮打样追踪 | P2 | round字段 + 文档关联链 |
| 询价比价（RFQ） | P3 | TCO比价模型 |
| SRM委外加工 | P3 | PCBA贴片/表面处理 |

---

## 十二、飞书通知集成

| 事件 | 通知对象 | 内容 |
|------|---------|------|
| BOM审批通过 | 采购员 | "采购项目已创建" |
| 交期预警(≤3天) | 采购员 | "零件X距交期还有3天，尚未下单" |
| 交期超期 | 采购员+主管 | "零件X已超期N天" |
| 到货 | IQC工程师 | "PO-001到货，请安排检验" |
| 检验不通过 | 采购员+研发 | "零件X检验不通过" |
| 打样全部通过 | 研发PM | "项目X的打样全部验收通过" |

---

*本文档基于已有SRM前端页面和SRM PRD v2.0整理。*
