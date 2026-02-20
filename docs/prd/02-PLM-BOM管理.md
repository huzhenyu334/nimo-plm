# 02 - PLM BOM管理

> 版本: v1.0 | 日期: 2026-02-19 | 编制: PM Agent
> 涵盖：BOM管理、物料体系、CMF配色、SKU管理、多类型BOM控件

---

## 一、功能概述

BOM（Bill of Materials）管理是PLM的核心数据模块，管理智能眼镜从设计到量产的物料清单。

### 1.1 已实现功能

| 功能 | 状态 | 页面 |
|------|------|------|
| BOM列表（按项目） | ✅ 已实现 | /bom-management |
| BOM详情（树形/平铺） | ✅ 已实现 | /bom-management/:projectId |
| 项目BOM管理（项目详情Tab） | ✅ 已实现 | /projects/:id |
| 物料选型库 | ✅ 已实现 | /materials |
| CMF配色管理 | ✅ 已实现 | 项目详情CMF Tab |
| BOM审批流程 | ✅ 已实现 | 审批管理 |
| PADS .rep文件导入 | ✅ 已实现 | 电子BOM导入 |

---

## 二、BOM架构设计

### 2.1 BOM类型

当前系统以**项目BOM**为核心，按BOM模板类型区分不同的BOM：

| BOM类型 | 模板 | 说明 | 使用阶段 |
|--------|------|------|---------|
| 电子BOM | electronic | EDA工具导出的电子元器件清单 | EVT起 |
| 结构BOM | structural | 结构件/外壳/光学件清单 | EVT起 |
| 包装BOM | packaging | 包材/印刷品/配件清单 | DVT起 |

### 2.2 智能眼镜典型BOM结构

```
nimo Air V2.0（整机）
├── 电子BOM
│   ├── 元器件（52种，PADS导入）
│   │   ├── EL-IC-000001  高通XR芯片 x1
│   │   ├── EL-CAP-000001 100nF 0402电容 x30
│   │   ├── EL-RES-000001 10KΩ 0402电阻 x25
│   │   └── ...
│   ├── PCB裸板（手动添加）
│   │   └── EL-PCB-000001 主板PCB 4层 FR4 x1
│   └── 贴片服务（手动添加）
│       └── SMT贴片加工 x1
│
├── 结构BOM
│   ├── ME-HSG-000001 前框组件 x1
│   ├── ME-HSG-000003 右镜腿组件 x1
│   ├── ME-HSG-000005 左镜腿组件 x1
│   ├── OP-WGD-000001 光波导模组 x2
│   └── ME-FST-000001 螺丝M1.2 x8
│
└── 包装BOM
    ├── PK-BOX-000001 产品彩盒 x1 (🌐 多语言)
    ├── PK-INS-000001 说明书 x1 (🌐 多语言)
    ├── PK-BAG-000001 绒布袋 x1
    └── PK-TRY-000001 吸塑内衬 x1
```

---

## 三、数据模型

### 3.1 项目BOM项表 (project_bom_items)

| 字段 | 类型 | 必填 | 说明 |
|-----|------|-----|------|
| id | UUID | 是 | 主键 |
| project_id | UUID | 是 | 关联项目 |
| bom_template_id | UUID | 否 | 关联BOM模板 |
| parent_item_id | UUID | 否 | 父级物料（空为顶层） |
| level | INTEGER | 是 | BOM层级(0-n) |
| seq_no | INTEGER | 是 | 序号 |
| material_id | UUID | 否 | 关联物料主数据 |
| material_code | VARCHAR(64) | 否 | 物料编码 |
| name | VARCHAR(200) | 是 | 物料名称 |
| quantity | DECIMAL(12,4) | 是 | 用量 |
| unit | VARCHAR(20) | 是 | 单位 |
| unit_price | DECIMAL(12,4) | 否 | 单价 |
| specification | TEXT | 否 | 规格描述 |
| supplier_id | VARCHAR(32) | 否 | 供应商（关联suppliers表） |
| item_type | VARCHAR(20) | 是 | component/pcb/service/material |
| is_appearance_part | BOOLEAN | 否 | 是否外观件 |
| sampling_ready | BOOLEAN | 否 | 是否可进入SRM打样 |
| **电子料专用** | | | |
| designator | VARCHAR(500) | 否 | 位号（R1,R2,...） |
| package | VARCHAR(50) | 否 | 封装 |
| manufacturer_id | VARCHAR(32) | 否 | 制造商（关联suppliers表，category=manufacturer） |
| mpn | VARCHAR(200) | 否 | 制造商料号 |
| **PCB专用** | | | |
| pcb_layers | INT | 否 | 层数 |
| pcb_thickness | VARCHAR(20) | 否 | 板厚 |
| pcb_material | VARCHAR(50) | 否 | 板材 |
| pcb_dimensions | VARCHAR(50) | 否 | 尺寸 |
| pcb_surface_finish | VARCHAR(50) | 否 | 表面工艺 |
| **服务专用** | | | |
| service_type | VARCHAR(50) | 否 | 加工类型（smt等） |
| process_requirements | TEXT | 否 | 工艺要求 |
| **包装专用** | | | |
| category | VARCHAR(20) | 否 | box/insert/print/accessory/label |
| print_process | VARCHAR(50) | 否 | 印刷工艺 |
| surface_finish_pkg | VARCHAR(50) | 否 | 表面处理 |
| is_multilang | BOOLEAN | 否 | 是否多语言件 |
| packing_qty | INT | 否 | 装箱数量 |
| **通用** | | | |
| drawing_2d | VARCHAR(500) | 否 | 2D图纸URL |
| drawing_3d | VARCHAR(500) | 否 | 3D图纸URL |
| attachments | JSONB | 否 | 附件列表 |
| remark | TEXT | 否 | 备注 |

### 3.2 BOM状态流转

```
DRAFT(草稿) → REVIEWING(审批中) → APPROVED(已审批) → RELEASED(已发布) → OBSOLETE(已废弃)
                  │
                  v
              审批驳回 → DRAFT
```

- **DRAFT**：可自由编辑
- **REVIEWING**：审批中，不可编辑
- **APPROVED**：已审批，待发布
- **RELEASED**：已发布，不可修改。如需变更需走ECN流程
- **OBSOLETE**：已废弃

---

## 四、电子BOM控件

### 4.1 BOM项分类

电子BOM的每一行有item_type字段：
- `component` — 电子元器件（EDA导出，批量导入）
- `pcb` — PCB裸板（手动添加）
- `service` — 贴片加工服务（手动添加）

### 4.2 前端控件布局

```
[📤 导入Excel] [+ 添加元器件] [+ 添加PCB] [+ 添加贴片服务]

── 元器件 (52项) ──────────────────
# │名称    │数量│位号    │封装  │规格   │制造商│MPN    │单价
1 │电阻10K │ 50│R1-R50 │0402 │10kΩ  │国巨  │RC... │¥0.01
...

── PCB (1项) ──── 浅蓝背景 ────────
# │名称    │数量│层数│板厚  │板材│尺寸    │表面│单价│📎附件

── 贴片服务 (1项) ── 浅黄背景 ────
# │名称    │数量│加工工艺      │钢网要求│单价

── 汇总 ─────────────────────────
元器件: ¥XX  PCB: ¥XX  贴片: ¥XX  PCBA总成本: ¥XX
```

### 4.3 PADS导入增强

导入PADS `.rep`文件时，自动匹配已有物料：

| 匹配状态 | 条件 | 系统行为 |
|---------|------|---------|
| ✅ 匹配 | MPN在物料库中找到 | 自动关联已有物料 |
| ⚠️ 新料 | MPN不为空但库中没有 | 确认后自动创建入库 |
| ❌ 缺失 | MPN为空 | 高亮提示，手动补全 |

**匹配逻辑**：先按 manufacturer + MPN 精确匹配 → 再按 MPN 模糊匹配 → 未匹配标记为新料。

---

## 五、结构BOM

### 5.1 特点

- 无EDA工具自动导出，主要通过手动添加或Excel导入
- 结构件供应商=制造商（一家包到底），无需MPN字段
- 外观件需要标记 `is_appearance_part=true`，关联CMF

### 5.2 添加物料交互

**先搜索已有 → 复用 → 未找到 → 新建入库：**

```
点击 [＋ 添加物料]
  → 搜索弹窗（搜索物料库，支持名称/料号/规格）
  → 搜索结果点击直接添加到BOM
  → 没找到 → [创建新物料] → 选分类+填信息 → 自动分配料号 → 入库 + 添加到BOM
```

---

## 六、包装BOM（PBOM）

### 6.1 BOM项分类

category字段区分：
- `box` — 彩盒/外箱
- `insert` — 内衬/托盘
- `print` — 印刷品（说明书/保修卡）
- `accessory` — 配件（充电线/擦镜布）
- `label` — 标签贴纸

### 6.2 多语言变体

标记为 `is_multilang=true` 的包装项需要创建语言变体：

**bom_item_lang_variants 表**

| 字段 | 类型 | 说明 |
|-----|------|------|
| id | UUID | 主键 |
| bom_item_id | UUID | 关联包装BOM项 |
| variant_index | INT | 变体序号 |
| material_code | VARCHAR(50) | 独立料号 |
| language_code | VARCHAR(10) | zh-CN/en/ja/ko/de等 |
| language_name | VARCHAR(50) | 简体中文/English等 |
| design_file_id | VARCHAR(100) | 设计稿文件 |

---

## 七、CMF配色管理

### 7.1 概述

外观件在进入SRM采购前，需要ID（工业设计师）定义CMF方案：
- **C** = Color（颜色）
- **M** = Material（材质）
- **F** = Finish（表面处理）

一个外观件可以有多个CMF方案（如黑色哑光、白色亮面），每个CMF变体有独立料号。

### 7.2 数据模型 (bom_item_cmf_variants)

| 字段 | 类型 | 说明 |
|-----|------|------|
| id | UUID | 主键 |
| bom_item_id | UUID | 关联结构BOM项 |
| variant_index | INT | 变体序号 |
| material_code | VARCHAR(50) | 独立料号（如 ST-001-BK01） |
| color_name | VARCHAR(100) | 颜色名称（如"星空黑"） |
| color_hex | VARCHAR(7) | 色值 |
| material | VARCHAR(200) | 材质（如 PC/ABS） |
| finish | VARCHAR(200) | 表面处理（如 喷砂阳极氧化） |
| texture | VARCHAR(200) | 纹理 |
| coating | VARCHAR(200) | 涂层 |
| pantone_code | VARCHAR(50) | Pantone色号 |
| reference_image_url | VARCHAR(500) | 参考图片 |
| status | VARCHAR(20) | draft / confirmed |

### 7.3 CMF与SRM联动

- 非外观件：BOM审批通过 → 1个零件 = 1张SRM采购卡片
- 外观件：BOM审批通过 + CMF confirmed → 1个CMF变体 = 1张SRM采购卡片

**零件就绪规则 (sampling_ready)：**
- 非外观件：BOM审批通过时自动 true
- 外观件：BOM审批通过 + 至少1个CMF confirmed时 true
- SRM只处理 sampling_ready=true 的项

### 7.4 料号规则

- 非外观件：`{类别前缀}-{序号}`（如 EL-001）
- 外观件CMF变体：`{基础料号}-{CMF后缀}`（如 ST-001-BK01, ST-001-WH01）

---

## 八、SKU管理

### 8.1 概述

SKU = 一组CMF变体的选择组合（每个外观件选一个CMF）。

例：
- SKU "星空黑版" = 镜腿-BK01 + 前壳-BK01 + 按键-BK01
- SKU "月光白版" = 镜腿-WH01 + 前壳-WH01 + 按键-WH01

### 8.2 SKU创建

创建SKU时选择：
- CMF组合（每个外观件选一个CMF方案）
- 语言版本（包装BOM中多语言件的语言选择）

---

## 九、供应商/制造商字段

### 9.1 设计原则

复用SRM的suppliers表，通过category区分角色：
- 供应商：category = electronic/structural/optical/packaging（从谁那买）
- 制造商：category = manufacturer（谁生产的，如TI、村田）

### 9.2 BOM中的供应商字段

| 字段 | 适用BOM | 说明 |
|------|--------|------|
| supplier_id | 所有类型 | 供应商，Select搜索选择 |
| manufacturer_id | 电子BOM | 制造商（芯片原厂），Select搜索选择 |
| mpn | 电子BOM | 制造商料号 |

- 结构BOM：只有供应商字段（供应商=制造商）
- 电子BOM：供应商 + 制造商 + MPN

---

## 十、API接口汇总

### 10.1 BOM接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/projects/:id/bom | 获取项目BOM列表 |
| POST | /api/v1/projects/:id/bom | 创建BOM项 |
| PUT | /api/v1/projects/:id/bom/:itemId | 更新BOM项 |
| DELETE | /api/v1/projects/:id/bom/:itemId | 删除BOM项 |
| POST | /api/v1/projects/:id/bom/import | 导入BOM（PADS/Excel） |
| POST | /api/v1/projects/:id/bom/submit-approval | 提交BOM审批 |

### 10.2 CMF变体接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/projects/:id/bom-items/:itemId/cmf-variants | 获取CMF变体 |
| POST | /api/v1/projects/:id/bom-items/:itemId/cmf-variants | 新增CMF变体 |
| PUT | /api/v1/projects/:id/cmf-variants/:variantId | 更新CMF变体 |
| DELETE | /api/v1/projects/:id/cmf-variants/:variantId | 删除CMF变体 |
| PUT | /api/v1/projects/:id/cmf-variants/:variantId/confirm | 确认CMF方案 |

### 10.3 语言变体接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/projects/:id/bom-items/:itemId/lang-variants | 获取语言变体 |
| POST | /api/v1/projects/:id/bom-items/:itemId/lang-variants | 新增语言变体 |
| PUT | /api/v1/projects/:id/lang-variants/:variantId | 更新语言变体 |
| DELETE | /api/v1/projects/:id/lang-variants/:variantId | 删除语言变体 |

---

## 十一、规划中功能

| 功能 | 优先级 | 说明 |
|------|--------|------|
| 物料编码体系升级 | P0 | 分类前缀+流水号（EL-CAP-000001） |
| MPN查重 | P0 | 创建/导入时按MPN自动去重 |
| PADS导入匹配增强 | P0 | 导入时自动匹配已有物料 |
| 手动添加物料交互改造 | P1 | 先搜索→复用→未找到→新建入库 |
| SRM委外加工 | P1 | PCBA贴片/表面处理作为service类型 |
| EBOM→MBOM转换 | P2 | 增加工序、损耗率、辅料 |
| BOM版本比较 | P2 | 两版本BOM差异对比 |
| 替代料管理（AML） | P3 | 一个物料对应多个制造商料号 |
| Where-Used查询 | P3 | 某物料被哪些BOM引用 |

---

*本文档基于已有代码实体（project_bom.go, cmf_variant.go, lang_variant.go, material.go等）和现有PRD整理。*
