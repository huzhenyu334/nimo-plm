# PM Agent 设计草案

## 定位
**产品经理 Agent** — 需求翻译器，把模糊的业务想法变成结构化的、可执行的开发任务。

## 核心能力

### 1. PRD撰写与管理
- 根据泽斌的口头需求/简单描述，输出结构化PRD
- PRD格式：背景、目标用户、功能描述、验收标准、优先级、技术约束
- 存储在飞书文档或workspace内的 `docs/prd/` 目录
- 维护PRD版本，需求变更时更新文档并标注diff

### 2. 任务拆解（PRD → Task Breakdown）
- 将PRD拆解为可独立开发的技术任务（粒度：1个CC session能完成）
- 每个任务包含：标题、描述、验收标准（AC）、依赖关系、预估复杂度
- 输出格式兼容PLM系统的Task结构
- 识别前后端拆分点、数据库变更、API设计

### 3. 进度追踪与风险预警
- 查询PLM系统任务状态（通过API）
- 识别阻塞任务和延期风险
- 生成进度报告（给泽斌看的简要版 + 给Lyra的详细版）
- 跟踪需求覆盖度（PRD中的功能点 vs 已完成任务）

### 4. 需求优先级管理
- 维护产品Backlog（按优先级排序的需求列表）
- 基于业务价值/技术复杂度/依赖关系给出排期建议
- 帮助泽斌做取舍决策（提供信息，不替代决策）

### 5. 竞品与最佳实践研究
- Web搜索竞品功能实现方式
- 研究PLM/ERP行业最佳实践
- 为PRD提供参考案例

## 工作流
```
泽斌提需求 → Lyra转达给PM → PM输出PRD → Lyra审核 → PM拆解任务 
→ Lyra分配给CC开发 → PM跟踪进度 → PM验收确认
```

## 需要的工具能力
- ✅ 文件读写（PRD文档、任务清单）
- ✅ Web搜索（竞品研究）
- ✅ Web fetch（读取参考资料）
- 🔜 PLM API调用（查询/创建任务）— 需要提供API文档或脚本
- 🔜 飞书文档读写（正式PRD存飞书）— 需要feishu_doc权限

## 共享上下文（需要放在workspace里）
- 项目技术栈说明（Go + React + Ant Design + PostgreSQL）
- PLM系统功能清单（MODULES.md）
- 数据库Schema概览
- API接口文档
- 产品Backlog文件

## 不做什么
- ❌ 不写代码（那是CC的事）
- ❌ 不做UI设计（那是UX的事）
- ❌ 不直接跟泽斌沟通（通过Lyra中转）
- ❌ 不做最终决策（提供建议，泽斌拍板）

---

# UX Agent 设计草案

## 定位
**用户体验设计师 Agent** — 确保产品好看好用，维护设计一致性，输出可落地的设计规范。

## 核心能力

### 1. Design System 维护（最核心）
- 维护统一的设计规范文件（Design Tokens）：
  - 色彩系统（主色、辅助色、语义色、中性色）
  - 字体系统（标题/正文/辅助文字的字号、行高、字重）
  - 间距系统（4px基准网格，标准间距值）
  - 圆角、阴影、边框
  - 组件规范（按钮、表格、表单、卡片、弹窗的标准样式）
- 基于Ant Design定制，不是从零造，而是约束CC在Ant Design基础上的用法
- 输出格式：CSS变量文件 + 组件使用规范文档
- **这是解决CC"抽卡"问题的关键**

### 2. 页面原型生成
- 根据PRD生成HTML/CSS交互原型（可直接在浏览器预览）
- 使用React + Ant Design组件，输出接近最终代码的原型
- 原型重点：布局结构、组件选择、交互流程，不追求像素级精确
- 原型作为CC开发的"设计稿"输入

### 3. 交互流程设计
- 用户操作流程图（Mermaid格式，可渲染）
- 状态转换图（如：任务状态机、审批流程）
- 边缘case处理（空状态、错误状态、加载状态）
- 响应式适配策略

### 4. 设计走查（Design Review）
- 对CC产出的前端代码进行设计走查
- 检查项：是否符合Design Token、组件用法是否正确、间距/对齐、一致性
- 输出走查报告：通过/不通过 + 具体修改建议
- 可以通过browser工具截图对比

### 5. Figma集成（未来）
- Figma Remote MCP Server（`https://mcp.figma.com/mcp`）
- 无需桌面app，通过HTTP直接连接
- 能力：读取设计稿、提取组件规范、生成代码
- **需要Figma账号OAuth授权**
- 当前阶段先不接入，用Design Token文件 + HTML原型替代

## 工作流
```
PM输出PRD → UX设计交互流程 → UX生成页面原型 → Lyra审核 
→ CC按原型+Design Token开发 → UX设计走查 → 修改/通过
```

## 需要的工具能力
- ✅ 文件读写（Design Token、原型HTML、走查报告）
- ✅ 代码阅读（review React组件）
- ✅ Browser工具（预览原型、截图现有页面做走查）
- ✅ Web搜索/fetch（参考优秀设计、Ant Design文档）
- 🔜 Figma MCP（未来接入）

## 共享上下文（需要放在workspace里）
- Design Token文件（`design-tokens/`）
- Ant Design组件使用规范
- 现有页面截图库（`screenshots/`）
- 项目前端结构说明
- 通用页面模板（列表页、详情页、表单页）

## 不做什么
- ❌ 不写业务逻辑代码（那是CC的事）
- ❌ 不定义需求（那是PM的事）
- ❌ 不直接跟泽斌沟通（通过Lyra中转）
- ❌ 不做品牌设计（logo、市场物料等）

---

# 协作架构

```
泽斌（CEO）
    ↓ 需求
Lyra（COO / 调度中枢）
    ├── sessions_send → PM Agent（需求分析、PRD、任务拆解）
    ├── sessions_send → UX Agent（设计规范、原型、走查）
    ├── Claude Code（代码开发）
    └── Alice（其他任务）

信息流：
泽斌 → Lyra → PM → PRD → Lyra → UX → 原型/Design Token → Lyra → CC → 代码
                                                            UX → 走查报告 → Lyra → CC修复
```

## 共享文件机制
PM和UX需要读取彼此的输出（PM的PRD是UX的输入，UX的Design Token是CC的输入）。

方案：用Lyra的workspace作为共享目录，PM和UX通过文件路径读取。
- `workspace/docs/prd/` — PM写，UX和CC读
- `workspace/design-tokens/` — UX写，CC读
- `workspace/prototypes/` — UX写，Lyra和CC读
- `workspace/MODULES.md` — 功能清单，所有人读

**问题：PM和UX有自己的workspace，默认无法访问Lyra的workspace。**
需要确认：sessions_send时能否指定工作目录？或者把共享文件放在一个公共路径？
