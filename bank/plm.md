# nimo PLM/ERP 系统

> BitFantasy核心内部系统，管理智能眼镜产品开发全流程

## 基本信息
- 技术栈：Go后端 + React前端 + PostgreSQL
- 地址：http://43.134.86.237:8080
- 代码目录：/home/claw/.openclaw/workspace/（Go module根目录）
- 前端源码：/home/claw/.openclaw/workspace/nimo-plm-web/
- 管理员：u_admin（系统管理员）
- API Token：plm-agent-token-2026（.env中PLM_API_TOKEN）
- 代码恢复点：tag v0.9-stable

## 已完成模块
- 项目管理（项目/任务/模板/依赖）
- BOM管理（SBOM/EBOM/PBOM/MBOM + 版本控制）
- ECN变更管理
- CMF配色管理
- 物料库 + 文档管理
- SRM（供应商/采购/检验/对账/8D/评价）
- 审批中心
- 任务表单系统
- 飞书SSO + 通知集成

## 已知问题
- SRM Settlements双重加载bug（新增行第二次加载才出现）
- Dashboard page_size=999临时方案（需服务端统计API）
- 供应商缺少批量导入/导出
- UI风格不一致（CC每次开发"抽卡"问题）→ 方案：Design Token + 共享模板

## 关键技术点
- 通用EditableTable组件 + onChange + debounce自动保存 = 所有可编辑表格标准
- 前端部署：`cd nimo-plm-web && npm run build`，然后`cp -r dist/* ../web/plm/`
- QA自动化：`./qa-tests/run-qa.sh`（40测试/20页面/165秒）
