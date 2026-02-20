# 04 - PLM 文档管理与物料选型库

> 版本: v1.0 | 日期: 2026-02-19 | 编制: PM Agent
> 涵盖：文档管理、物料选型库、物料编码体系、物料分类

---

## 一、文档管理

### 1.1 已实现功能

| 功能 | 状态 | 页面 |
|------|------|------|
| 文档列表 | ✅ 已实现 | /documents |
| 文档上传 | ✅ 已实现 | — |
| 文档分类管理 | ✅ 已实现 | — |
| 文档关联（项目/产品） | ✅ 已实现 | — |
| 文件上传（通用） | ✅ 已实现 | /api/v1/upload |

### 1.2 文档分类

| 分类代码 | 分类名称 | 说明 |
|---------|---------|------|
| DESIGN | 设计文档 | 原理图、PCB、结构图、光学设计 |
| SPEC | 规格书 | 产品规格书、物料规格书 |
| TEST | 测试文档 | 测试用例、测试报告 |
| QUALITY | 质量文档 | 检验标准、质量报告 |
| CERT | 认证文档 | 3C、FCC、CE等认证资料 |
| MFG | 生产文档 | 作业指导书、工艺文件 |
| USER | 用户文档 | 说明书、快速指南 |
| OTHER | 其他 | 会议纪要、培训资料 |

### 1.3 数据模型 (documents)

| 字段 | 类型 | 必填 | 说明 |
|-----|------|-----|------|
| id | UUID | 是 | 主键 |
| doc_code | VARCHAR(50) | 是 | 文档编码 |
| title | VARCHAR(200) | 是 | 文档标题 |
| category | ENUM | 是 | 文档分类 |
| product_id | UUID | 否 | 关联产品 |
| project_id | UUID | 否 | 关联项目 |
| current_version | INTEGER | 是 | 当前版本号 |
| status | ENUM | 是 | 文档状态 |
| confidential_level | ENUM | 是 | 机密级别 |
| file_url | VARCHAR(500) | 否 | 文件URL |
| file_size | BIGINT | 否 | 文件大小 |
| created_by | VARCHAR(64) | 是 | 创建人 |
| created_at | TIMESTAMP | 是 | 创建时间 |
| updated_at | TIMESTAMP | 是 | 更新时间 |

### 1.4 API接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/documents | 上传文档 |
| GET | /api/v1/documents | 查询文档列表 |
| GET | /api/v1/documents/:id | 查询文档详情 |
| GET | /api/v1/documents/:id/download | 下载文档 |
| POST | /api/v1/upload | 通用文件上传 |

### 1.5 文件存储

- 存储路径：`./uploads/YYYY/MM/uuid_filename`
- 通过 `GET /uploads/:path` 提供静态访问
- 当前为本地磁盘存储，未来可迁移至MinIO对象存储

---

## 二、物料选型库

### 2.1 已实现功能

| 功能 | 状态 | 页面 |
|------|------|------|
| 物料列表 | ✅ 已实现 | /materials |
| 物料CRUD | ✅ 已实现 | — |
| 物料分类筛选 | ✅ 已实现 | — |
| PADS .rep导入 | ✅ 已实现 | BOM导入时 |

### 2.2 数据模型 (materials)

| 字段 | 类型 | 必填 | 说明 |
|-----|------|-----|------|
| id | UUID | 是 | 主键 |
| material_code | VARCHAR(50) | 是 | 物料编码（唯一） |
| name | VARCHAR(200) | 是 | 物料名称 |
| name_en | VARCHAR(200) | 否 | 英文名称 |
| category | ENUM | 是 | 一级分类 |
| sub_category | VARCHAR(50) | 否 | 二级分类（子类码） |
| specification | TEXT | 否 | 规格描述 |
| unit | VARCHAR(20) | 是 | 计量单位 |
| standard_cost | DECIMAL(12,4) | 否 | 标准成本 |
| lead_time_days | INTEGER | 否 | 标准采购周期 |
| moq | INTEGER | 否 | 最小起订量 |
| manufacturer_id | VARCHAR(32) | 否 | 制造商（关联suppliers表） |
| mpn | VARCHAR(200) | 否 | 制造商料号 |
| status | ENUM | 是 | ACTIVE/INACTIVE/OBSOLETE |
| created_at | TIMESTAMP | 是 | 创建时间 |
| updated_at | TIMESTAMP | 是 | 更新时间 |

### 2.3 API接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/materials | 查询物料列表 |
| GET | /api/v1/materials/:id | 查询物料详情 |
| POST | /api/v1/materials | 创建物料 |
| PUT | /api/v1/materials/:id | 更新物料 |
| DELETE | /api/v1/materials/:id | 删除物料 |

---

## 三、物料编码体系（规划中）

### 3.1 编码格式

**推荐方案：分类前缀 + 流水号**

格式：`{类别码}-{子类码}-{6位流水号}`

示例：
- `EL-CAP-000001` → 电子-电容-第1号
- `ME-HSG-000001` → 结构-外壳-第1号
- `PK-BOX-000001` → 包材-包装盒-第1号

### 3.2 物料分类体系（二级分类）

```
EL 电子元器件
├── EL-RES  电阻          ├── EL-LED  LED/背光
├── EL-CAP  电容          ├── EL-SEN  传感器
├── EL-IND  电感          ├── EL-ANT  天线
├── EL-IC   集成电路      ├── EL-MOD  模组
├── EL-CON  连接器        ├── EL-BAT  电池
├── EL-DIO  二极管/ESD    ├── EL-PCB  PCB
├── EL-TRN  晶体管        └── EL-OTH  其他电子
├── EL-OSC  晶振/时钟

ME 结构件
├── ME-HSG  外壳/壳体     ├── ME-SPG  弹性件
├── ME-LNS  镜片/光学件   ├── ME-THM  散热件
├── ME-FLX  柔性件        ├── ME-DEC  装饰件
├── ME-FST  紧固件        └── ME-OTH  其他结构
├── ME-GSK  密封件

OP 光学器件
├── OP-DIS  显示器件      ├── OP-WGD  光波导
├── OP-PRJ  投影器件      └── OP-OTH  其他光学

PK 包材
├── PK-BOX  包装盒        ├── PK-TRY  托盘/衬垫
├── PK-BAG  袋类          └── PK-LBL  标签
├── PK-INS  插页/说明书

AX 辅料/耗材
├── AX-ADH  胶粘剂        ├── AX-INS  绝缘材料
├── AX-SLD  焊接材料      ├── AX-TLS  工装治具
├── AX-CLN  清洗材料      └── AX-OTH  其他辅料

SW 软件/固件
├── SW-FW   固件          └── SW-LIC  授权/许可
├── SW-APP  应用软件
```

### 3.3 当前分类（5类）→ 目标分类（6大类 + 二级子类）

| 当前 | 目标 |
|------|------|
| mcat_electronic | EL 电子元器件（14个子类） |
| mcat_mechanical | ME 结构件（9个子类） |
| mcat_optical | OP 光学器件（4个子类） |
| mcat_packaging | PK 包材（5个子类） |
| mcat_other | AX 辅料/耗材（6个子类） + SW 软件/固件（3个子类） |

---

## 四、物料判重机制（规划中）

### 4.1 判重逻辑

| 物料类型 | 有MPN？ | 判重方式 |
|----------|---------|----------|
| 电子料（外购件） | ✅ | 制造商 + MPN（全球唯一） |
| 标准件 | ✅ | 制造商 + MPN |
| 结构件（定制件） | ❌ | 不需要判重（天然唯一） |
| 光学件（定制件） | ❌ | 不需要判重 |
| 包材（定制件） | ❌ | 不需要判重 |

### 4.2 数据库约束

```sql
-- 同一制造商下MPN不能重复
CREATE UNIQUE INDEX idx_materials_mpn ON materials(manufacturer_id, mpn) WHERE mpn IS NOT NULL;
```

---

## 五、物料来源与入库方式

| 物料类型 | 来源 | 入库方式 | 状态 |
|----------|------|---------|------|
| 电子元器件 | EDA工具（PADS/Altium） | BOM自动导入 | ✅ 已实现（PADS） |
| 结构件/光学件 | CAD工具（SolidWorks） | Excel导入或手动创建 | 🔜 规划中 |
| 包材/辅料 | 采购/工程手动创建 | 系统手动创建 + Excel批量导入 | 🔜 规划中 |

---

## 六、物料生命周期（规划中）

```
Draft(草稿) → Released(已发布) → Obsolete(淘汰)
```

- **Draft**：工程师自由编辑，项目组内可见
- **Released**：审批后发布，全局可用，修改需走ECN
- **Obsolete**：停产/淘汰，不再用于新设计

> 当前阶段所有物料默认Active状态，生命周期管控待团队适应后再加。

---

## 七、物料架构演进路径

### 当前：项目中心

所有数据挂在项目下，无法跨项目查看物料引用情况。

### 短期（Phase 1）
- 加全局物料查询/BOM查询视图
- PADS导入自动匹配
- 物料编码体系升级

### 中期（Phase 2）
- 物料独立化（脱离项目，成为全局实体）
- 物料生命周期状态管理
- ECN基于物料

### 远期（Phase 3）
- 物料版本管理（Revision链）
- 替代料关系管理（AML）
- Where-Used查询
- 完整物料中心化架构

---

## 八、规划中功能汇总

| 功能 | 优先级 | 说明 |
|------|--------|------|
| 物料编码体系升级 | P0 | 分类前缀+流水号 |
| MPN查重提示 | P0 | 创建/导入时自动去重 |
| PADS导入匹配增强 | P0 | 导入时自动匹配已有物料 |
| 结构件Excel批量导入 | P1 | 标准模板导入 |
| 全局物料搜索视图 | P1 | 跨项目查询 |
| 文档版本管理 | P2 | 同一文档多版本 |
| 物料生命周期 | P2 | Draft→Released→Obsolete |
| 物料独立化 | P3 | 物料脱离项目 |
| Where-Used查询 | P3 | 某物料被哪些BOM引用 |

---

*本文档基于已有代码实体（material.go, document.go）和物料编码体系/物料管理架构设计PRD整理。*
