# nimo SRM 供应商与采购管理系统 PRD

> 版本: v1.0 | 作者: Lyra | 日期: 2026-02-09
> 状态: 待评审

---

## 1. 背景与目标

### 1.1 背景

nimo智能眼镜产品研发涉及大量零部件（结构件、电子元器件、光学组件、包装件等），从EVT到MP每个阶段都需要打样、验证、迭代。当前打样和采购流程依赖人工协调（飞书/Excel），存在以下痛点：

- 打样进度难以追踪（50+零件分散在不同供应商）
- 供应商信息散落在个人文档中
- 询价比价无标准流程
- 来料检验结果未系统化记录
- PLM的BOM变更无法自动通知采购侧

### 1.2 目标

建设轻量级SRM系统，实现：
1. **供应商全生命周期管理**（准入→合作→评价→淘汰）
2. **研发打样全流程在线化**（需求→寻源→下单→到货→检验）
3. **与PLM无缝衔接**（BOM发布自动触发采购需求，检验结果回写PLM）
4. **为ERP量产采购打基础**（供应商和价格数据复用）

### 1.3 用户角色

| 角色 | 职责 |
|---|---|
| 采购员 | 寻源、询价、下单、跟踪到货 |
| 研发工程师 | 发起打样需求、参与来料检验 |
| 品质工程师(IQC) | 来料检验、出具检验报告 |
| 项目经理 | 查看打样进度、阶段评审 |
| 管理层 | 供应商审批、采购审批、数据看板 |

---

## 2. 系统架构

### 2.1 技术栈（复用PLM架构）

```
前端: React + Ant Design + Vite（独立SPA，部署到 web/srm/）
后端: Pure Go (Gin + GORM)
数据库: PostgreSQL（与PLM共享实例，独立schema或表前缀）
认证: 飞书SSO（复用PLM的飞书登录）
通知: 飞书消息卡片
```

### 2.2 服务架构

```
                    ┌──────────────┐
                    │   Nginx/Go   │
                    │  静态文件服务  │
                    └──────┬───────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
         web/plm/     web/srm/     web/erp/
         (PLM前端)    (SRM前端)    (ERP前端)
              │            │            │
              └────────────┼────────────┘
                           │
                    ┌──────┴───────┐
                    │   Go服务      │
                    │ /api/v1/plm  │
                    │ /api/v1/srm  │
                    │ /api/v1/erp  │
                    └──────┬───────┘
                           │
                    ┌──────┴───────┐
                    │  PostgreSQL   │
                    │  共享物料库    │
                    │  PLM表 + SRM表│
                    └──────────────┘
```

**关键决策：单体服务 vs 微服务**

推荐**单体服务**（一个Go二进制），理由：
- 团队20人，不需要微服务复杂度
- 共享物料库直接内存调用，无需API/消息队列
- 部署运维简单（一台服务器）
- 后期如果需要拆分，按Go package拆即可

### 2.3 与PLM的集成方式

由于是同一个Go服务，PLM和SRM之间通过**内部Service调用**集成（非HTTP/消息队列）：

```go
// PLM侧：BOM审批通过后
func (s *ApprovalService) Approve(...) {
    // ... 审批通过 ...
    // 通知SRM创建采购需求
    if s.srmSvc != nil {
        go s.srmSvc.CreatePRFromBOM(ctx, projectID, bomID, userID)
    }
}

// SRM侧：检验完成后回写PLM
func (s *InspectionService) Complete(...) {
    // ... 检验完成 ...
    // 回写PLM物料验证状态
    if s.plmSvc != nil {
        s.plmSvc.UpdateMaterialValidation(ctx, materialID, result)
    }
}
```

---

## 3. 功能模块设计

### 3.1 模块总览

```
SRM系统
├── M1: 供应商管理
│   ├── 供应商档案（基本信息、资质、联系人）
│   ├── 供应商准入审批
│   ├── 供应商分类（结构件/电子/光学/包装）
│   └── 供应商评价
│
├── M2: 采购需求管理
│   ├── 采购需求(PR)（从PLM BOM自动生成或手动创建）
│   ├── 需求审批
│   └── 需求合并
│
├── M3: 寻源与询价
│   ├── 询价单(RFQ)（发送给供应商）
│   ├── 报价管理（供应商回复报价）
│   ├── 比价分析
│   └── 定源决策
│
├── M4: 采购订单
│   ├── 采购单(PO)
│   ├── PO审批
│   ├── 到货跟踪
│   └── 收货确认
│
├── M5: 来料检验(IQC)
│   ├── 检验任务（自动生成）
│   ├── 检验记录
│   ├── 检验报告
│   └── 结果回写PLM
│
└── M6: 看板与报表
    ├── 打样进度看板（按项目/阶段）
    ├── 供应商绩效看板
    └── 采购统计
```

### 3.2 M1: 供应商管理

#### 3.2.1 供应商档案

**数据模型：**

```sql
-- 供应商主表
CREATE TABLE srm_suppliers (
    id              VARCHAR(32) PRIMARY KEY,
    code            VARCHAR(32) UNIQUE NOT NULL,    -- 供应商编码 SUP-001
    name            VARCHAR(200) NOT NULL,          -- 公司名称
    short_name      VARCHAR(50),                    -- 简称
    category        VARCHAR(50) NOT NULL,           -- 分类: structural/electronic/optical/packaging/other
    level           VARCHAR(20) DEFAULT 'potential', -- 等级: potential/qualified/preferred/strategic
    status          VARCHAR(20) DEFAULT 'pending',  -- 状态: pending/active/suspended/blacklisted
    
    -- 基本信息
    country         VARCHAR(50),
    province        VARCHAR(50),
    city            VARCHAR(50),
    address         VARCHAR(500),
    website         VARCHAR(200),
    
    -- 业务信息
    business_scope  TEXT,                           -- 经营范围/主营产品
    annual_revenue  DECIMAL(15,2),                  -- 年营业额（万元）
    employee_count  INT,                            -- 员工人数
    factory_area    DECIMAL(10,2),                  -- 厂房面积（㎡）
    certifications  JSONB,                          -- 资质证书 [{name, expiry_date, file_url}]
    
    -- 付款信息
    bank_name       VARCHAR(200),
    bank_account    VARCHAR(50),
    tax_id          VARCHAR(50),                    -- 税号
    payment_terms   VARCHAR(100),                   -- 付款条件（如NET30）
    
    -- 管理信息
    created_by      VARCHAR(32) REFERENCES users(id),
    approved_by     VARCHAR(32) REFERENCES users(id),
    approved_at     TIMESTAMP,
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW(),
    notes           TEXT
);

-- 供应商联系人
CREATE TABLE srm_supplier_contacts (
    id              VARCHAR(32) PRIMARY KEY,
    supplier_id     VARCHAR(32) REFERENCES srm_suppliers(id),
    name            VARCHAR(100) NOT NULL,
    title           VARCHAR(100),                   -- 职务
    phone           VARCHAR(50),
    email           VARCHAR(200),
    wechat          VARCHAR(100),
    is_primary      BOOLEAN DEFAULT false,          -- 主联系人
    created_at      TIMESTAMP DEFAULT NOW()
);

-- 供应商可供物料（供应商能供应哪些品类）
CREATE TABLE srm_supplier_materials (
    id              VARCHAR(32) PRIMARY KEY,
    supplier_id     VARCHAR(32) REFERENCES srm_suppliers(id),
    category_id     VARCHAR(32),                    -- 关联PLM物料分类
    material_id     VARCHAR(32),                    -- 关联PLM具体物料（可选）
    lead_time_days  INT,                            -- 交期（天）
    moq             INT,                            -- 最小起订量
    unit_price      DECIMAL(12,4),                  -- 单价
    currency        VARCHAR(10) DEFAULT 'CNY',
    notes           TEXT,
    created_at      TIMESTAMP DEFAULT NOW()
);
```

#### 3.2.2 供应商准入流程

```
录入供应商信息 → 提交准入申请 → 管理层审批 → 成为合格供应商
                                    ↓ 驳回
                              补充资料/淘汰
```

准入审批复用PLM的审批引擎（ApprovalService），审批类型为 `supplier_qualification`。

#### 3.2.3 供应商分类与等级

**分类**（对应物料大类）：
- 结构件供应商（CNC/模具/注塑/钣金）
- 电子元器件供应商（被动/主动/连接器/PCB）
- 光学组件供应商（镜片/棱镜/波导）
- 包装件供应商
- 辅料供应商

**等级**（动态调整）：
- 潜在供应商（Potential）— 刚录入，未审核
- 合格供应商（Qualified）— 通过准入审批
- 优选供应商（Preferred）— 绩效优秀，优先选用
- 战略供应商（Strategic）— 核心供应商，长期合作

### 3.3 M2: 采购需求管理

#### 3.3.1 采购需求(PR)

采购需求有两种来源：

**来源A：PLM BOM自动生成**

当PLM中BOM审批通过（或阶段评审通过），系统自动为BOM中的零件生成采购需求。

```sql
-- 采购需求单
CREATE TABLE srm_purchase_requests (
    id              VARCHAR(32) PRIMARY KEY,
    pr_code         VARCHAR(32) UNIQUE NOT NULL,    -- PR编码 PR-2026-001
    title           VARCHAR(200) NOT NULL,
    type            VARCHAR(20) NOT NULL,           -- sample(打样) / production(量产)
    priority        VARCHAR(20) DEFAULT 'normal',   -- urgent/high/normal/low
    status          VARCHAR(20) DEFAULT 'draft',    -- draft/pending/approved/sourcing/completed/cancelled
    
    -- 关联
    project_id      VARCHAR(32),                    -- 关联PLM项目
    bom_id          VARCHAR(32),                    -- 关联PLM BOM
    phase           VARCHAR(20),                    -- EVT/DVT/PVT/MP
    
    -- 需求信息
    required_date   DATE,                           -- 需求日期
    
    -- 管理
    requested_by    VARCHAR(32) REFERENCES users(id),
    approved_by     VARCHAR(32) REFERENCES users(id),
    approved_at     TIMESTAMP,
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW(),
    notes           TEXT
);

-- 采购需求行项
CREATE TABLE srm_pr_items (
    id              VARCHAR(32) PRIMARY KEY,
    pr_id           VARCHAR(32) REFERENCES srm_purchase_requests(id),
    
    -- 物料信息（从PLM同步）
    material_id     VARCHAR(32),                    -- 关联PLM物料
    material_code   VARCHAR(50),
    material_name   VARCHAR(200) NOT NULL,
    specification   VARCHAR(500),
    category        VARCHAR(100),
    
    -- 需求数量
    quantity        DECIMAL(10,2) NOT NULL,
    unit            VARCHAR(20) DEFAULT 'pcs',
    
    -- 采购进度
    status          VARCHAR(20) DEFAULT 'pending',  -- pending/sourcing/ordered/received/inspected/completed
    supplier_id     VARCHAR(32),                    -- 选定供应商
    unit_price      DECIMAL(12,4),
    total_amount    DECIMAL(15,2),
    
    -- 交期
    expected_date   DATE,
    actual_date     DATE,
    
    -- 检验
    inspection_result VARCHAR(20),                  -- passed/failed/conditional
    
    sort_order      INT DEFAULT 0,
    notes           TEXT,
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW()
);
```

**来源B：手动创建**

采购员手动创建PR（非BOM关联的采购，如工具、设备、样品补充等）。

#### 3.3.2 PLM → SRM 自动推送逻辑

```
触发条件：PLM BOM审批通过
    ↓
遍历BOM行项
    ↓
过滤：跳过已有采购需求的物料（防重复）
    ↓
生成PR + PR Items
    ↓
飞书通知采购员："项目XXX的EVT BOM已通过审批，请处理采购需求 PR-2026-001"
```

### 3.4 M3: 寻源与询价

#### 3.4.1 询价单(RFQ)

```sql
-- 询价单
CREATE TABLE srm_rfqs (
    id              VARCHAR(32) PRIMARY KEY,
    rfq_code        VARCHAR(32) UNIQUE NOT NULL,    -- RFQ-2026-001
    pr_id           VARCHAR(32) REFERENCES srm_purchase_requests(id),
    title           VARCHAR(200) NOT NULL,
    status          VARCHAR(20) DEFAULT 'draft',    -- draft/sent/quoted/evaluated/decided
    
    deadline        TIMESTAMP,                      -- 报价截止时间
    created_by      VARCHAR(32) REFERENCES users(id),
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW(),
    notes           TEXT
);

-- 询价单发送记录（发给哪些供应商）
CREATE TABLE srm_rfq_suppliers (
    id              VARCHAR(32) PRIMARY KEY,
    rfq_id          VARCHAR(32) REFERENCES srm_rfqs(id),
    supplier_id     VARCHAR(32) REFERENCES srm_suppliers(id),
    status          VARCHAR(20) DEFAULT 'pending',  -- pending/quoted/declined
    quoted_at       TIMESTAMP,
    notes           TEXT
);

-- 报价明细
CREATE TABLE srm_quotations (
    id              VARCHAR(32) PRIMARY KEY,
    rfq_id          VARCHAR(32) REFERENCES srm_rfqs(id),
    supplier_id     VARCHAR(32) REFERENCES srm_suppliers(id),
    pr_item_id      VARCHAR(32) REFERENCES srm_pr_items(id),
    
    unit_price      DECIMAL(12,4),
    currency        VARCHAR(10) DEFAULT 'CNY',
    lead_time_days  INT,                            -- 交期（天）
    moq             INT,                            -- 最小起订量
    tooling_cost    DECIMAL(12,2),                  -- 模具费/开版费
    sample_cost     DECIMAL(12,2),                  -- 打样费
    
    is_selected     BOOLEAN DEFAULT false,          -- 是否选中（定源）
    notes           TEXT,
    created_at      TIMESTAMP DEFAULT NOW()
);
```

#### 3.4.2 比价与定源

采购员收集报价后，在比价页面对比：
- 各供应商的单价、交期、MOQ、模具费
- 系统自动计算综合成本（单价×数量+模具费分摊）
- 采购员选定供应商，标记 `is_selected = true`
- 如果金额超过阈值，需要审批

### 3.5 M4: 采购订单(PO)

```sql
-- 采购订单
CREATE TABLE srm_purchase_orders (
    id              VARCHAR(32) PRIMARY KEY,
    po_code         VARCHAR(32) UNIQUE NOT NULL,    -- PO-2026-001
    supplier_id     VARCHAR(32) REFERENCES srm_suppliers(id),
    pr_id           VARCHAR(32),                    -- 关联PR
    type            VARCHAR(20) NOT NULL,           -- sample/production
    status          VARCHAR(20) DEFAULT 'draft',    -- draft/approved/sent/partial/received/completed/cancelled
    
    -- 金额
    total_amount    DECIMAL(15,2),
    currency        VARCHAR(10) DEFAULT 'CNY',
    
    -- 交期
    expected_date   DATE,
    actual_date     DATE,
    
    -- 收货与付款
    shipping_address VARCHAR(500),
    payment_terms   VARCHAR(100),
    
    -- 管理
    created_by      VARCHAR(32) REFERENCES users(id),
    approved_by     VARCHAR(32) REFERENCES users(id),
    approved_at     TIMESTAMP,
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW(),
    notes           TEXT
);

-- PO行项
CREATE TABLE srm_po_items (
    id              VARCHAR(32) PRIMARY KEY,
    po_id           VARCHAR(32) REFERENCES srm_purchase_orders(id),
    pr_item_id      VARCHAR(32),                    -- 关联PR行项
    material_id     VARCHAR(32),
    material_code   VARCHAR(50),
    material_name   VARCHAR(200) NOT NULL,
    specification   VARCHAR(500),
    
    quantity        DECIMAL(10,2) NOT NULL,
    unit            VARCHAR(20) DEFAULT 'pcs',
    unit_price      DECIMAL(12,4),
    total_amount    DECIMAL(15,2),
    
    -- 收货
    received_qty    DECIMAL(10,2) DEFAULT 0,
    status          VARCHAR(20) DEFAULT 'pending',  -- pending/shipped/partial/received
    
    sort_order      INT DEFAULT 0,
    notes           TEXT,
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW()
);
```

#### 3.5.1 PO到货跟踪

```
PO发出 → 供应商确认 → 发货（填运单号）→ 到货签收 → 自动创建IQC检验任务
```

到货时自动更新PR行项状态为 `received`。

### 3.6 M5: 来料检验(IQC)

```sql
-- 检验任务
CREATE TABLE srm_inspections (
    id              VARCHAR(32) PRIMARY KEY,
    inspection_code VARCHAR(32) UNIQUE NOT NULL,    -- IQC-2026-001
    po_id           VARCHAR(32) REFERENCES srm_purchase_orders(id),
    po_item_id      VARCHAR(32),
    supplier_id     VARCHAR(32),
    
    material_id     VARCHAR(32),
    material_code   VARCHAR(50),
    material_name   VARCHAR(200),
    
    -- 检验信息
    quantity        DECIMAL(10,2),                  -- 送检数量
    sample_qty      INT,                            -- 抽样数量
    status          VARCHAR(20) DEFAULT 'pending',  -- pending/in_progress/completed
    result          VARCHAR(20),                    -- passed/failed/conditional(让步接收)
    
    -- 检验详情
    inspection_items JSONB,                         -- 检验项 [{name, standard, actual, result}]
    report_url      VARCHAR(500),                   -- 检验报告附件
    
    -- 人员
    inspector_id    VARCHAR(32) REFERENCES users(id),
    inspected_at    TIMESTAMP,
    
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW(),
    notes           TEXT
);
```

#### 3.6.1 检验结果处理

```
检验通过 → 更新PR行项状态为 completed → 回写PLM物料验证状态
检验不通过 → 退货/让步接收
           → 通知研发工程师
           → 如需修改设计 → 触发PLM ECN流程
```

#### 3.6.2 回写PLM

检验完成后，SRM自动回写PLM：
- 更新 `project_bom_items` 的验证状态字段
- 更新 `materials` 表的验证信息
- 关联检验报告到PLM项目文档

### 3.7 M6: 看板与报表

#### 3.7.1 打样进度看板（核心页面）

按项目和阶段展示打样进度：

```
项目: Meteor (EVT阶段)
━━━━━━━━━━━━━━━━━━━━━━━
总零件: 50 | 已下单: 45 | 已到样: 30 | 已检验: 25 | 通过: 23
[██████████████░░░░░░░░] 46%

待处理:
🔴 零件A（光学棱镜）— 供应商未回复，已超期3天
🟡 零件B（外壳）— 模具开模中，预计2周后到样
🟢 零件C~Z — 已到样待检验
```

#### 3.7.2 供应商绩效

- 交期达成率
- 来料合格率
- 报价响应速度
- 综合评分（自动计算）

---

## 4. 前端页面设计

### 4.1 页面列表

```
SRM系统
├── 📊 首页/看板
│   └── 打样进度总览 + 待办事项
│
├── 🏢 供应商管理
│   ├── 供应商列表（搜索、筛选、分类）
│   ├── 供应商详情（基本信息、联系人、可供物料、历史订单、评分）
│   └── 供应商准入审批
│
├── 📋 采购需求
│   ├── PR列表（按项目/状态筛选）
│   ├── PR详情（行项列表、进度跟踪）
│   └── 创建PR（手动/从BOM生成）
│
├── 💰 询价管理
│   ├── RFQ列表
│   ├── RFQ详情（发送记录、报价汇总）
│   └── 比价分析页
│
├── 📦 采购订单
│   ├── PO列表
│   ├── PO详情（行项、物流、收货）
│   └── 创建PO（从RFQ/PR生成）
│
├── 🔍 来料检验
│   ├── 检验任务列表
│   ├── 检验记录填写
│   └── 检验报告
│
└── ⚙️ 设置
    ├── 检验标准模板
    └── 编码规则配置
```

### 4.2 核心交互流程

**研发打样全流程（最常用）：**

```
Step 1: PLM BOM审批通过 → SRM自动收到采购需求
Step 2: 采购员打开PR → 查看零件清单 → 选择供应商 → 创建询价单
Step 3: 供应商报价 → 采购员比价 → 选定供应商 → 创建PO
Step 4: 供应商发货 → 采购员收货 → 自动创建IQC任务
Step 5: 品质工程师检验 → 填写结果 → 自动回写PLM
Step 6: 项目经理在PLM看板查看：所有零件打样验证进度
```

---

## 5. API设计

### 5.1 路由规划

```
/api/v1/srm/
├── suppliers/                  # 供应商CRUD
│   ├── GET    /               # 列表（支持搜索、分类、等级筛选）
│   ├── POST   /               # 创建
│   ├── GET    /:id            # 详情
│   ├── PUT    /:id            # 更新
│   ├── POST   /:id/approve    # 准入审批
│   └── GET    /:id/contacts   # 联系人列表
│
├── purchase-requests/          # 采购需求
│   ├── GET    /               # 列表
│   ├── POST   /               # 手动创建
│   ├── POST   /from-bom       # 从BOM生成
│   ├── GET    /:id            # 详情（含行项）
│   ├── PUT    /:id            # 更新
│   └── POST   /:id/approve   # 审批
│
├── rfqs/                       # 询价
│   ├── GET    /
│   ├── POST   /
│   ├── GET    /:id
│   ├── POST   /:id/send       # 发送给供应商
│   └── POST   /:id/quote      # 录入报价
│
├── purchase-orders/            # 采购订单
│   ├── GET    /
│   ├── POST   /
│   ├── GET    /:id
│   ├── PUT    /:id
│   ├── POST   /:id/approve
│   ├── POST   /:id/receive    # 收货
│   └── POST   /:id/items/:itemId/receive  # 行项收货
│
├── inspections/                # 来料检验
│   ├── GET    /
│   ├── GET    /:id
│   ├── PUT    /:id            # 填写检验结果
│   └── POST   /:id/complete   # 完成检验
│
└── dashboard/                  # 看板
    ├── GET    /sampling-progress  # 打样进度
    └── GET    /supplier-performance  # 供应商绩效
```

---

## 6. 飞书集成

### 6.1 通知场景

| 事件 | 通知对象 | 通知内容 |
|---|---|---|
| PLM BOM审批通过 | 采购员 | "项目X的BOM已通过审批，请处理采购需求" |
| PR审批通过 | 采购员 | "采购需求PR-001已审批通过，可以开始寻源" |
| 到货签收 | IQC工程师 | "PO-001到货，请安排来料检验" |
| 检验不通过 | 研发工程师+采购员 | "零件X检验不通过，请处理" |
| 交期预警 | 采购员 | "零件X距离需求日期还有3天，供应商尚未发货" |
| 打样全部完成 | 项目经理 | "项目X的EVT打样全部验收通过" |

### 6.2 飞书审批集成

供应商准入和大额PO审批可以对接飞书审批流，复用PLM已有的审批引擎。

---

## 7. 编码规则

| 对象 | 编码格式 | 示例 |
|---|---|---|
| 供应商 | SUP-{4位流水} | SUP-0001 |
| 采购需求 | PR-{年}-{4位流水} | PR-2026-0001 |
| 询价单 | RFQ-{年}-{4位流水} | RFQ-2026-0001 |
| 采购订单 | PO-{年}-{4位流水} | PO-2026-0001 |
| 检验单 | IQC-{年}-{4位流水} | IQC-2026-0001 |

---

## 8. 分阶段实施计划

### Phase 1: 供应商 + 打样采购（2周）
**目标**：解决研发打样核心痛点

- [x] 供应商档案管理（CRUD + 搜索）
- [x] 采购需求管理（手动创建 + 从PLM BOM自动生成）
- [x] 简化PO管理（创建 + 到货确认）
- [x] PLM集成：BOM审批通过 → 自动创建PR
- [x] 打样进度看板

### Phase 2: 询价比价 + 检验（2周）
**目标**：完善采购流程闭环

- [ ] 询价单管理（RFQ）
- [ ] 报价录入与比价分析
- [ ] 来料检验（IQC）
- [ ] 检验结果回写PLM
- [ ] 飞书通知集成

### Phase 3: 高级功能（2周）
**目标**：提升管理效率

- [ ] 供应商准入审批流程
- [ ] 供应商绩效评分
- [ ] 交期预警（自动检测即将超期的PO）
- [ ] 采购审批（金额阈值）
- [ ] 报表导出

### Phase 4: ERP衔接（后续）
**目标**：量产切换

- [ ] 供应商数据同步到ERP
- [ ] 打样价格转量产合同价
- [ ] 量产PO在ERP中管理
- [ ] 库存联动

---

## 9. 代码组织

```
internal/
├── plm/          # 现有PLM模块
├── erp/          # 现有ERP模块（待建）
└── srm/          # 新增SRM模块
    ├── entity/
    │   ├── supplier.go          # 供应商实体
    │   ├── purchase_request.go  # 采购需求实体
    │   ├── rfq.go               # 询价实体
    │   ├── purchase_order.go    # 采购订单实体
    │   └── inspection.go        # 检验实体
    ├── repository/
    │   ├── supplier_repo.go
    │   ├── pr_repo.go
    │   ├── po_repo.go
    │   └── inspection_repo.go
    ├── service/
    │   ├── supplier_service.go
    │   ├── procurement_service.go
    │   ├── inspection_service.go
    │   └── dashboard_service.go
    └── handler/
        ├── supplier_handler.go
        ├── pr_handler.go
        ├── po_handler.go
        ├── inspection_handler.go
        └── dashboard_handler.go

cmd/
├── plm/main.go   # 现有，加载SRM模块
└── srm/main.go   # 可选：独立SRM入口（初期不需要）

web/
├── plm/          # PLM前端
└── srm/          # SRM前端

nimo-srm-web/     # SRM前端源码
```

---

## 10. 风险与注意事项

1. **物料主数据一致性**：SRM和PLM共用 `materials` 表，SRM不创建物料，只引用PLM的物料编码
2. **权限隔离**：采购员只能看SRM，研发只能看PLM（除非同时有两个角色）
3. **并发问题**：BOM审批通过自动创建PR时，需防止重复创建（幂等性）
4. **数据迁移**：现有供应商信息需要从Excel/飞书文档导入
5. **供应商自助**：Phase 1不做供应商门户，报价由采购员手动录入；后续可考虑开放供应商自助报价

---

> 本文档为初版设计，具体实现时可能根据实际情况调整。
