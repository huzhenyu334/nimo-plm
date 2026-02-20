# 05 - SRM 供应商管理

> 版本: v1.0 | 日期: 2026-02-19 | 编制: PM Agent
> 涵盖：供应商档案、供应商评价、8D改进、通用设备

---

## 一、功能概述

供应商管理覆盖供应商全生命周期：准入→合作→评价→淘汰。

### 1.1 已实现功能

| 功能 | 状态 | 页面 |
|------|------|------|
| 供应商档案管理 | ✅ 已实现 | /srm/suppliers |
| 供应商评价 | ✅ 已实现 | /srm/evaluations |
| 8D改进 | ✅ 已实现 | /srm/corrective-actions |
| 通用设备管理 | ✅ 已实现 | /srm/equipment |

---

## 二、供应商档案

### 2.1 数据模型 (suppliers)

| 字段 | 类型 | 必填 | 说明 |
|-----|------|-----|------|
| id | VARCHAR(32) | 是 | 主键 |
| code | VARCHAR(32) | 是 | 供应商编码（SUP-0001） |
| name | VARCHAR(200) | 是 | 供应商名称 |
| short_name | VARCHAR(100) | 否 | 简称 |
| category | ENUM | 是 | electronic/structural/optical/packaging/manufacturer/other |
| status | ENUM | 是 | pending/approved/rejected |
| contact_name | VARCHAR(100) | 否 | 联系人 |
| phone | VARCHAR(20) | 否 | 电话 |
| email | VARCHAR(100) | 否 | 邮箱 |
| website | VARCHAR(200) | 否 | 官网 |
| city | VARCHAR(100) | 否 | 城市 |
| address | VARCHAR(500) | 否 | 详细地址 |
| payment_terms | VARCHAR(100) | 否 | 付款条款 |
| overall_score | DECIMAL(5,2) | 否 | 综合评分 |
| quality_score | DECIMAL(5,2) | 否 | 质量得分 |
| delivery_score | DECIMAL(5,2) | 否 | 交期得分 |
| price_score | DECIMAL(5,2) | 否 | 价格得分 |
| service_score | DECIMAL(5,2) | 否 | 服务得分 |
| created_at | TIMESTAMP | 是 | 创建时间 |
| updated_at | TIMESTAMP | 是 | 更新时间 |

### 2.2 供应商分类

| category值 | 说明 | 角色 |
|-----------|------|------|
| electronic | 电子元器件代理商/分销商 | 供应商（卖） |
| structural | 结构件供应商 | 供应商（做+卖） |
| optical | 光学器件供应商 | 供应商 |
| packaging | 包材供应商 | 供应商 |
| manufacturer | 芯片原厂（TI、村田等） | 制造商（造） |
| other | 其他 | — |

**关键设计**：制造商和供应商共用一张suppliers表，通过category区分。BOM中选择供应商时筛选 category≠manufacturer，选择制造商时筛选 category=manufacturer。

---

## 三、供应商评价

### 3.1 评分维度

| 维度 | 权重 | 计算方式 |
|-----|------|---------|
| 质量得分 | 40% | 100 - (不良率 × 1000) |
| 交期得分 | 30% | 准时交货率 × 100 |
| 价格得分 | 20% | 市场比价得分 |
| 服务得分 | 10% | 响应速度、配合度 |

### 3.2 评级规则

| 等级 | 综合得分 | 说明 |
|------|---------|------|
| A级 | ≥ 90 | 优选供应商 |
| B级 | ≥ 75 | 合格供应商 |
| C级 | ≥ 60 | 需改善 |
| D级 | < 60 | 淘汰候选 |

### 3.3 评价周期

- 定期评价：每季度
- 事件驱动：来料检验不通过时自动触发评分更新

---

## 四、8D改进

### 4.1 概述

8D（Eight Disciplines）是一种系统化的质量问题解决方法。当来料检验不通过时，系统自动创建8D改进单。

### 4.2 8D流程

```
来料检验不通过
    ↓
自动创建8D改进单 → 飞书通知采购员+供应商
    ↓
供应商回复改进措施（或采购员手动录入）
    ↓
采购员决策：
  ├── 同供应商重打 → 创建Round N+1 PO（关联8D）
  ├── 换供应商 → 重新寻源
  └── 让步接收 → 标记conditional，流程继续
    ↓
验证改进效果 → 关闭8D
```

### 4.3 8D编码

格式：`8D-{年}-{4位流水}`，如 `8D-2026-0001`

---

## 五、通用设备管理

### 5.1 概述

管理与采购相关的通用设备信息（如测试设备、检验设备等）。

### 5.2 已实现页面

/srm/equipment — 设备列表CRUD

---

## 六、API接口汇总

### 6.1 供应商接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/srm/suppliers | 创建供应商 |
| GET | /api/v1/srm/suppliers | 查询供应商列表 |
| GET | /api/v1/srm/suppliers/:id | 查询供应商详情 |
| PUT | /api/v1/srm/suppliers/:id | 更新供应商 |
| DELETE | /api/v1/srm/suppliers/:id | 删除供应商 |

### 6.2 评价接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/srm/evaluations | 评价列表 |
| POST | /api/v1/srm/evaluations | 创建评价 |
| PUT | /api/v1/srm/evaluations/:id | 更新评价 |

### 6.3 8D接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/srm/corrective-actions | 8D列表 |
| POST | /api/v1/srm/corrective-actions | 创建8D |
| PUT | /api/v1/srm/corrective-actions/:id | 更新8D |

---

## 七、规划中功能

| 功能 | 优先级 | 说明 |
|------|--------|------|
| 供应商360画像 | P1 | 标签体系、历史记录聚合 |
| 供应商绩效自动评分 | P1 | 基于来料检验和交期数据自动计算 |
| 供应商准入审批 | P2 | 新供应商需走审批流程 |
| 供应商可供物料关联 | P2 | 记录每个供应商可供的物料列表 |
| 供应商合同管理 | P3 | 合同上传和到期提醒 |

---

*本文档基于已有SRM前端页面（Suppliers.tsx, Evaluations, CorrectiveActions）和SRM PRD整理。*
