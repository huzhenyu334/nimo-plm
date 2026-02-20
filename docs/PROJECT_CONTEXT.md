# PROJECT_CONTEXT.md - nimo PLM/ERP 项目上下文

> 所有agent共享此文件。了解项目全貌。

## 公司背景

**BitFantasy** — 智能眼镜品牌 nimo
- 团队约20人
- CEO: 陈泽斌
- AI团队: Lyra(COO) + PM Agent + UX Agent + Alice + Claude Code

## 产品定位

nimo PLM/ERP 是内部工具，服务于公司的产品开发和供应链管理流程。

**核心用户：** BitFantasy内部员工（产品经理、工程师、采购、品质）
**核心场景：** 智能眼镜产品从设计→研发→采购→生产的全生命周期管理

## 产品开发阶段（硬件行业）

| 阶段 | 全称 | 说明 |
|------|------|------|
| EVT | Engineering Verification Test | 工程验证，验证功能可行性 |
| DVT | Design Verification Test | 设计验证，验证设计规格 |
| PVT | Production Verification Test | 生产验证，验证量产可行性 |
| MP | Mass Production | 量产 |

PLM系统的项目管理就是围绕这些阶段展开的。

## 已完成的系统架构

### PLM（产品生命周期管理）
- **项目管理** — 创建项目、分配任务、进度跟踪、甘特图
- **任务管理** — 任务依赖、表单收集、审批流、自动状态机
- **BOM管理** — 物料清单、CMF（颜色/材料/表面处理）、多级BOM
- **ECN变更** — 工程变更通知，关联BOM和物料
- **物料库** — 物料选型、规格管理
- **审批流** — 通用审批引擎，支持飞书集成
- **文档管理** — 文件上传、版本管理
- **角色权限** — RBAC权限控制

### SRM（供应商关系管理）
- **供应商管理** — 供应商档案、资质审核
- **采购流程** — 采购需求→采购订单→来料检验→入库
- **质量管理** — 来料检验、8D纠正改进
- **库存管理** — 入库/出库/盘点
- **财务** — 对账结算
- **评价** — 供应商绩效评价
- **通用设备** — 公司设备资产管理

## 技术决策记录

| 决策 | 选择 | 原因 |
|------|------|------|
| 后端语言 | Go（纯Go） | 泽斌偏好，简化运维，单二进制部署 |
| 前端框架 | React + Ant Design | 企业级UI库成熟度最高 |
| 数据库 | PostgreSQL | 可靠性、JSON支持 |
| 认证 | JWT + 飞书SSO | 公司统一用飞书 |
| 编辑模式 | onChange + debounce自动保存 | BOM/CMF等表格统一体验 |
| 组件库 | ProComponents | 减少样板代码 |

## 已知问题

1. **UI不一致** — CC每次开发风格不同（Design Token正在解决）
2. **SRM对账结算双重加载bug** — 新增行第二次才出现
3. **缺少E2E测试覆盖** — 只有BOM模块有自动化测试

## 开发规范

- Git commit遵循Conventional Commits
- 前端新页面必须参考Design Token和页面模板
- 后端API遵循RESTful规范
- 所有CRUD操作必须有错误处理和反馈
