# PRD: ACP v2.1 — Agent模板管理 + 稳定性修复

## 背景

ACP当前状态：后端API基本完整（Agent/Task/Workflow/Queue/Lessons），前端有Dashboard/Agent详情/任务/工作流页面，Plugin有14个MCP tool。但存在以下问题：

1. **没有稳定可用的版本** — 工作流引擎有各种edge case未处理
2. **Agent配置文件直接编辑，无版本管理** — 改一行规则就可能让agent行为完全不同，但没有变更记录、无法回滚
3. **巡检质量不可控** — agent可以偷懒（如UX agent用grep代码替代浏览器巡检），缺乏系统级约束

## 目标

**交付一个能稳定跑工作流的ACP版本**，同时加入Agent模板管理能力。

## 核心功能

### 一、Agent角色模板管理（NEW）

#### 1.1 设计理念

管理的不是agent实例，而是**角色模板**。模板定义了一个角色的全部能力基因。

```
角色模板（如"产品经理 v1.2"）
    ├── agents.md      — 角色定义、规则、红线
    ├── tools.md       — 工具使用方法、项目上下文
    ├── soul.md        — 性格、沟通风格
    ├── user.md        — 服务对象信息
    └── skills         — 可用skill列表（软约束）

实例部署：
    workspace-pm/ → 基于 "产品经理 v1.2" 模板部署
    workspace-ux/ → 基于 "UX设计师 v2.0" 模板部署
```

#### 1.2 数据模型

```go
// AgentTemplate 角色模板
type AgentTemplate struct {
    ID          string    `gorm:"primaryKey" json:"id"`
    Slug        string    `gorm:"uniqueIndex;not null" json:"slug"`       // e.g. "product-manager"
    Name        string    `gorm:"not null" json:"name"`                   // e.g. "产品经理"
    Description string    `gorm:"default:''" json:"description"`
    Version     string    `gorm:"not null" json:"version"`                // semver: "1.0.0"
    Status      string    `gorm:"not null;default:'draft'" json:"status"` // draft/active/deprecated
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

// AgentTemplateFile 模板文件
type AgentTemplateFile struct {
    ID         string    `gorm:"primaryKey" json:"id"`
    TemplateID string    `gorm:"index;not null" json:"template_id"`
    FileName   string    `gorm:"not null" json:"file_name"`    // "AGENTS.md", "TOOLS.md", etc.
    Content    string    `gorm:"type:text;not null" json:"content"`
    CreatedAt  time.Time `json:"created_at"`
}

// AgentTemplateVersion 模板版本历史
type AgentTemplateVersion struct {
    ID         string    `gorm:"primaryKey" json:"id"`
    TemplateID string    `gorm:"index;not null" json:"template_id"`
    Version    string    `gorm:"not null" json:"version"`
    Comment    string    `gorm:"default:''" json:"comment"`     // 变更说明
    ChangedBy  string    `gorm:"not null" json:"changed_by"`    // lyra/zebin/system
    Diff       string    `gorm:"type:text" json:"diff"`         // 变更内容diff
    CreatedAt  time.Time `json:"created_at"`
}

// AgentTemplateFile belongs to version
// 每个version包含该版本下所有文件的完整快照

// AgentInstance 实例（agent与模板的绑定关系）
type AgentInstance struct {
    ID           string    `gorm:"primaryKey" json:"id"`
    AgentID      string    `gorm:"uniqueIndex;not null" json:"agent_id"`   // "pm", "ux", etc.
    TemplateID   string    `gorm:"index;not null" json:"template_id"`
    TemplateVer  string    `gorm:"not null" json:"template_version"`
    WorkspacePath string   `gorm:"not null" json:"workspace_path"`
    SyncStatus   string    `gorm:"default:'synced'" json:"sync_status"`    // synced/outdated/pending
    LastSyncAt   *time.Time `json:"last_sync_at"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}
```

#### 1.3 API

```
# 模板管理
GET    /api/agent-templates                          → 列出所有模板
POST   /api/agent-templates                          → 创建模板
GET    /api/agent-templates/:id                      → 模板详情（含文件内容）
PUT    /api/agent-templates/:id                      → 更新模板（自动创建新版本）
DELETE /api/agent-templates/:id                      → 删除模板

# 版本管理
GET    /api/agent-templates/:id/versions             → 版本历史
GET    /api/agent-templates/:id/versions/:ver        → 某版本详情
POST   /api/agent-templates/:id/versions/:ver/rollback → 回滚到某版本

# 实例管理
GET    /api/agent-instances                          → 列出所有实例
POST   /api/agent-instances/:agent_id/deploy         → 从模板部署到workspace
GET    /api/agent-instances/:agent_id/diff           → 对比实例vs模板差异
POST   /api/agent-instances/:agent_id/sync           → 同步模板到实例
```

#### 1.4 MCP Tool（Plugin，后续批量添加）

```
acp_template_list        — 列出角色模板
acp_template_get         — 查看模板详情
acp_template_create      — 创建模板（需审批）
acp_template_update      — 更新模板（需审批，自动版本递增）
acp_template_deploy      — 从模板部署到agent workspace
acp_template_diff        — 对比实例与模板差异
acp_template_rollback    — 回滚到历史版本
```

#### 1.5 前端页面

**路由：`/agent-templates`**

- 模板列表：卡片式展示每个角色模板，显示版本号、状态、关联实例数
- 模板详情：左侧文件列表，右侧Markdown编辑器，顶部版本选择器
- 版本对比：diff视图，高亮变更内容
- 部署面板：选择模板版本 → 选择目标agent → 确认部署

#### 1.6 工作流程

```
泽斌/Lyra 发现agent表现问题（如UX偷懒grep）
    ↓
总结经验教训 → acp_add_lesson
    ↓
更新角色模板 → PUT /api/agent-templates/:id
    ↓
系统自动：创建版本记录、生成diff、标记变更说明
    ↓
部署到实例 → POST /api/agent-instances/:agent_id/deploy
    ↓
系统自动：将模板文件写入workspace，记录sync状态
    ↓
下次工作流执行 → agent使用新规则
```

### 二、稳定性修复（P0）

以下是当前ACP需要修的核心问题：

#### 2.1 工作流引擎稳定性

| # | 问题 | 优先级 | 说明 |
|---|------|--------|------|
| 1 | WaitForReply稳定性 | P0 | 需要取最后一条assistant消息+稳定性检测，防止误判 |
| 2 | 步骤超时处理 | P0 | 超时后step状态要正确标记，run状态要正确更新 |
| 3 | 并行步骤执行 | P1 | depends_on为空的步骤应该并行执行 |
| 4 | 工作流取消传播 | P0 | cancel run时要正确取消所有pending/running的step |
| 5 | {{input}}变量替换 | P0 | 已修但需验证：RenderPrompt正确替换所有模板变量 |

#### 2.2 Plugin稳定性

| # | 问题 | 优先级 | 说明 |
|---|------|--------|------|
| 6 | Plugin重连机制 | P0 | ACP重启后plugin应能自动重连 |
| 7 | 错误信息传递 | P1 | Plugin tool调用失败时，错误信息要清晰返回给agent |
| 8 | structured_output校验 | P0 | 422拒绝不符合schema的提交，要返回具体哪个字段不对 |

#### 2.3 前端稳定性

| # | 问题 | 优先级 | 说明 |
|---|------|--------|------|
| 9 | 工作流执行详情实时更新 | P1 | 步骤状态变化时前端要及时刷新 |
| 10 | 任务看板数据准确 | P1 | 确保显示的状态与后端一致 |

## 开发计划

### Sprint 1：引擎稳定性（预计3-4天）

**目标：工作流能稳定跑完一个完整流程不出错**

| 任务 | 描述 | 涉及文件 |
|------|------|---------|
| S1-1 | WaitForReply改造：取最后一条assistant消息、加稳定性检测（连续2次相同内容才算完成）、正确处理compaction消息 | `service/engine.go` |
| S1-2 | 步骤超时+取消传播：超时后正确标记step/run状态，cancel时传播到所有子步骤 | `service/engine.go` |
| S1-3 | {{input}}和{{steps.x.output}}变量替换全面测试+修复 | `service/engine.go` |
| S1-4 | structured_output校验增强：422返回具体错误字段信息 | `handler/workflow_run_handler.go` |
| S1-5 | Plugin错误处理：ACP不可达时的优雅降级 | `plugin/` 目录 |

### Sprint 2：Agent模板管理-后端（预计3-4天）

**目标：模板CRUD + 版本管理 + 部署到workspace**

| 任务 | 描述 | 涉及文件 |
|------|------|---------|
| S2-1 | 数据模型：AgentTemplate + AgentTemplateFile + AgentTemplateVersion + AgentInstance | `entity/entity.go` |
| S2-2 | Repository层：模板CRUD、版本查询、实例管理 | `repository/template_repository.go` |
| S2-3 | Service层：创建模板、更新+自动版本、部署到workspace、diff对比、回滚 | `service/template_service.go` |
| S2-4 | Handler层：REST API全套 | `handler/template_handler.go` |
| S2-5 | 初始化：从现有workspace读取当前agent配置，创建初始模板（v1.0.0 baseline） | `service/template_service.go` |

### Sprint 3：Agent模板管理-前端 + 集成测试（预计3-4天）

**目标：前端可视化管理 + 端到端测试**

| 任务 | 描述 |
|------|------|
| S3-1 | 模板列表页：卡片展示、状态标签、关联实例 |
| S3-2 | 模板详情页：文件编辑器（Markdown）、版本切换 |
| S3-3 | 版本对比页：diff高亮 |
| S3-4 | 部署面板：模板→agent部署操作 |
| S3-5 | 端到端测试：创建模板→更新→部署→验证workspace文件变更 |

### Sprint 4：MCP Tool + 经验闭环（预计2天）

**目标：Lyra可通过MCP管理模板，lessons可关联到模板升级**

| 任务 | 描述 |
|------|------|
| S4-1 | MCP tool注册：template_list/get/create/update/deploy/diff/rollback |
| S4-2 | Lesson→模板关联：lesson记录可标记affected_template_id，形成闭环 |
| S4-3 | Plugin更新+OpenClaw restart |

## 验收标准

### 稳定性
- [ ] 一个包含3+步骤的工作流能稳定跑完（PM分析→CC开发→UX验收）
- [ ] 步骤超时能正确处理，不会卡死
- [ ] cancel工作流后所有步骤正确标记为cancelled
- [ ] structured_output校验失败返回具体错误信息

### Agent模板管理
- [ ] 能创建角色模板，包含AGENTS.md/TOOLS.md/SOUL.md等文件
- [ ] 更新模板时自动创建版本记录，包含diff和变更说明
- [ ] 能从模板部署到agent workspace（文件写入）
- [ ] 能回滚到历史版本
- [ ] 能对比实例当前文件与模板的差异
- [ ] 前端能可视化管理模板、查看版本历史、执行部署
- [ ] Lyra能通过MCP tool操作模板（Sprint 4后）

## 技术约束

- 后端：Go（Gin + GORM）+ SQLite
- 前端：React + Ant Design + TypeScript
- 部署到workspace = 直接写文件到 `/home/claw/.openclaw/workspace-{agent}/`
- 不需要OpenClaw restart（除Sprint 4加MCP tool时）

## 变更记录

| 日期 | 变更 | 原因 |
|------|------|------|
| 2026-02-21 | 初版 | 泽斌提出agent模板管理需求+稳定性优先 |
