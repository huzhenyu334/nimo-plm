# PRD: Agent Control Panel (ACP) v2

## 1. 产品概述

**Agent Control Panel (ACP)** 是 BitFantasy 内部 AI Agent 团队管理与监控平台，支撑 agent 团队 7×24 不间断自主运作。

### 两套用户界面

| 用户 | 界面 | 用途 |
|------|------|------|
| 泽斌（CEO） | Web 前端页面 | 查看数据、监控进度、审批关键节点 |
| Lyra（COO） | MCP/API 接口 | 调度任务、编排工作流、管理团队 |

两套界面操作同一套数据，Lyra通过MCP高效调度，泽斌打开页面看进度和结果。

### 目标
- **可观测性**：实时查看每个 agent 当前操作
- **自动化**：工作流驱动、任务队列自动调度，agent 自主取活执行
- **可控性**：随时介入、暂停、调整优先级
- **可编程**：Lyra 通过 MCP 协议完全操控 ACP

### 非目标（v2 不做）
- 多租户/权限体系
- Agent 代码热部署
- 第三方 agent 接入

---

## 2. 用户角色

| 角色 | 说明 | 界面 | 权限 |
|------|------|------|------|
| Admin（泽斌） | CEO，全部权限 | Web 前端 | 全部 |
| COO（Lyra） | 调度员，通过 MCP 操作 | MCP 接口 | 全部（除系统设置） |
| Viewer（预留） | 只读查看 | Web 前端 | 只读 |

---

## 3. 系统架构（v2）

```
┌─────────────────────────────────────────────────────┐
│               泽斌（CEO）                             │
│            浏览器 (React SPA)                         │
│  Dashboard │ Agent详情 │ 任务 │ 工作流 │ 任务队列      │
└──────────────────────┬──────────────────────────────┘
                       │ HTTP REST
┌──────────────────────▼──────────────────────────────┐
│             Go HTTP Server (:3001)                    │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────┐  │
│  │ Agent API │ │ Task API │ │ WF API   │ │Queue API│  │
│  └─────┬────┘ └────┬─────┘ └────┬─────┘ └───┬────┘  │
│        │           │            │            │        │
│  ┌─────▼───────────▼────────────▼────────────▼─────┐ │
│  │              Service Layer                       │ │
│  │  ┌────────────────────────────────────────────┐  │ │
│  │  │  Workflow Engine v2                        │  │ │
│  │  │  - DAG调度 + 异步等待agent回复             │  │ │
│  │  │  - 步骤间数据流传递                        │  │ │
│  │  │  - 自动创建Task记录                        │  │ │
│  │  └────────────────────────────────────────────┘  │ │
│  │  ┌────────────────────────────────────────────┐  │ │
│  │  │  Task Queue（任务队列）                     │  │ │
│  │  │  - 优先级排序                               │  │ │
│  │  │  - Agent空闲检测 + 自动分配                 │  │ │
│  │  │  - 负载均衡                                 │  │ │
│  │  └────────────────────────────────────────────┘  │ │
│  └──────────────────┬───────────────────────────────┘ │
│                     │                                  │
│  ┌──────────────────▼──────────────────────────────┐  │
│  │    MCP Server (JSON-RPC over stdio)              │  │
│  │    → Lyra 通过 MCP 协议调用全部能力              │  │
│  └──────────────────┬──────────────────────────────┘  │
│                     │                                  │
│  ┌──────────────────▼───────┐  ┌───────────────────┐  │
│  │  OpenClaw Gateway Client │  │  SQLite (数据持久化)│  │
│  │  (WebSocket)             │  │                    │  │
│  └──────────────────────────┘  └───────────────────┘  │
└───────────────────────┬────────────────────────────────┘
                        │ WebSocket
┌───────────────────────▼───────────────┐
│         OpenClaw Gateway               │
│         (agent sessions)               │
└────────────────────────────────────────┘
```

**v1 → v2 新增层：**
- MCP Server：Lyra 的编程接口
- Task Queue：任务自动调度引擎
- Workflow Engine v2：异步等待回复 + 数据流传递

---

## 4. Task 系统重设计 — P0

### 4.1 设计理念

**Task = 一切工作的原子单位。** 无论是手动创建、工作流自动生成、还是定时任务产生，所有 agent 执行的工作都是一个 Task。

### 4.2 Task 来源

| 来源类型 | 说明 | source_type |
|----------|------|-------------|
| manual | Lyra/泽斌 手动创建 | `manual` |
| workflow | 工作流步骤自动创建 | `workflow` |
| cron | 定时任务触发 | `cron` |
| queue | 任务队列自动分配 | `queue` |

### 4.3 Task 数据模型

```go
type Task struct {
    ID          string     `gorm:"primaryKey" json:"id"`
    Title       string     `gorm:"not null" json:"title"`
    Description string     `gorm:"default:''" json:"description"`   // 任务详细描述
    
    // 输入输出
    Input       string     `gorm:"type:text" json:"input"`          // 发给agent的prompt
    Output      string     `gorm:"type:text" json:"output"`         // agent的回复/产出
    
    // 状态
    Status      string     `gorm:"not null;default:'pending';index" json:"status"`
    Priority    string     `gorm:"not null;default:'P1'" json:"priority"`  // P0/P1/P2
    
    // 执行者
    AgentID     *string    `gorm:"index" json:"agent_id"`           // 指派的agent
    
    // 来源追踪
    SourceType  string     `gorm:"not null;default:'manual'" json:"source_type"`  // manual/workflow/cron/queue
    SourceID    *string    `json:"source_id"`                       // workflow_run_id / cron_job_id 等
    StepID      *string    `json:"step_id"`                         // 关联的workflow step id
    
    // 创建者
    CreatedBy   string     `gorm:"not null;default:'admin'" json:"created_by"`    // admin/lyra/system
    
    // 时间
    DueDate     *time.Time `json:"due_date"`
    StartedAt   *time.Time `json:"started_at"`
    CompletedAt *time.Time `json:"completed_at"`
    CreatedAt   time.Time  `json:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at"`
}
```

### 4.4 Task 状态机

```
                  ┌──────────┐
                  │  pending  │  ← 刚创建，等待分配
                  └─────┬────┘
                        │ assign (指派agent)
                  ┌─────▼────┐
                  │ assigned  │  ← 已指派，等待执行
                  └─────┬────┘
                        │ start (prompt发送给agent)
                  ┌─────▼────┐
                  │  running  │  ← agent正在执行，等待回复
                  └──┬────┬──┘
                     │    │
            success  │    │ failure
           ┌─────────▼┐  ┌▼──────────┐
           │ completed │  │  failed   │
           └──────────┘  └─────┬─────┘
                               │ retry
                         ┌─────▼────┐
                         │  pending  │  （重回队列）
                         └──────────┘

  任何状态 → cancelled（手动取消）
```

**状态转换规则：**
- `pending → assigned`：手动指派或队列自动分配
- `assigned → running`：prompt 已发送给 agent（sessions_send 成功）
- `running → completed`：收到 agent 回复，写入 output
- `running → failed`：超时或 agent 报错
- `failed → pending`：重试（重新入队）

### 4.5 Task API

```
GET    /api/tasks                          → Task[]         # 列表（支持筛选 ?status=&agent_id=&source_type=&priority=）
POST   /api/tasks                          → Task           # 创建（手动）
GET    /api/tasks/:id                      → TaskDetail     # 详情（含input/output）
PUT    /api/tasks/:id                      → Task           # 更新
DELETE /api/tasks/:id                      → { ok: bool }
POST   /api/tasks/:id/assign              → Task           # 指派agent
POST   /api/tasks/:id/cancel              → Task           # 取消
POST   /api/tasks/:id/retry               → Task           # 重试（failed → pending）
```

### 4.6 验收标准

- [ ] Task 有 input（prompt）和 output（agent回复）字段
- [ ] Task 有 source_type 标记来源（manual/workflow/cron/queue）
- [ ] 工作流执行 step 时自动创建 Task（source_type=workflow）
- [ ] Task 状态机严格遵循定义的转换规则
- [ ] 前端任务看板显示所有来源的 Task，可按来源筛选
- [ ] Task 详情页能看到完整的 input 和 output

---

## 5. 工作流执行引擎 v2 — P0

### 5.1 当前问题

```
v1: sessions_send(prompt) → 立即标 completed ❌
v2: sessions_send(prompt) → 等agent回复 → 收集output → 传给下一步 ✅
```

### 5.2 异步等待回复机制

**核心流程：**

```
1. 渲染 prompt 模板（替换上游步骤的 output 变量）
2. 创建 Task 记录（source_type=workflow, status=running）
3. sessions_send(agent, rendered_prompt)
4. 轮询 sessions_history 等待 agent 回复
   - 每 10s 检查一次
   - 识别新的 assistant 消息（比 send 时间更晚的）
   - 超时判定（默认 30min，可配置）
5. 收到回复：
   a. 写入 Task.output
   b. 写入 StepRun.output
   c. 提取 outputs 变量（用于下游步骤）
   d. 标记 step completed
6. 超时/失败：
   a. 标记 step failed
   b. 按 on_failure 策略处理（pause/skip/abort）
```

### 5.3 回复检测算法

```go
// WaitForReply 等待 agent 回复
func (e *WorkflowEngine) WaitForReply(sessionKey string, sentAt time.Time, timeout time.Duration) (string, error) {
    deadline := time.Now().Add(timeout)
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            if time.Now().After(deadline) {
                return "", ErrTimeout
            }
            // 拉取最近消息
            messages := gateway.SessionsHistory(sessionKey, 10, false)
            for _, msg := range messages {
                if msg.Role == "assistant" && msg.Timestamp.After(sentAt) {
                    return msg.Content, nil
                }
            }
        }
    }
}
```

### 5.4 步骤间数据流

上一步的 output 作为下一步的 context/input：

```yaml
steps:
  - id: analyze
    agent: pm
    prompt: "分析需求：{{inputs.task_description}}"
    outputs:
      prd_path: "从回复中提取"
      summary: "从回复中提取"

  - id: design
    agent: ux
    depends_on: [analyze]
    prompt: |
      请根据以下 PRD 进行 UX 设计：
      PRD路径：{{steps.analyze.outputs.prd_path}}
      需求摘要：{{steps.analyze.outputs.summary}}
      
      上一步完整输出供参考：
      {{steps.analyze.output}}
```

**变量解析优先级：**
1. `{{steps.<id>.outputs.<key>}}` — 从 output 中提取的结构化变量
2. `{{steps.<id>.output}}` — 上一步的完整原始 output（agent回复全文）
3. `{{inputs.<key>}}` — 工作流输入参数

**Output 变量提取规则（v2）：**
- Agent 回复中包含 `[output:key=value]` 格式 → 精确提取
- 未找到标记 → 将整个回复作为 `output` 传递给下游

### 5.5 超时和重试

```yaml
steps:
  - id: develop
    agent: main
    timeout: 2h              # 超时时间
    retry:
      max_attempts: 2         # 最多重试次数
      delay: 5m               # 重试间隔
      on: failure             # 触发条件：failure / timeout
    on_failure: pause         # pause = 暂停工作流等人工干预
                              # skip = 跳过继续下游
                              # abort = 终止整个工作流
```

### 5.6 WorkflowStepRun 数据模型更新

```go
type WorkflowStepRun struct {
    ID          string     `gorm:"primaryKey" json:"id"`
    RunID       string     `gorm:"index;not null" json:"run_id"`
    StepID      string     `gorm:"not null" json:"step_id"`
    AgentID     string     `gorm:"not null" json:"agent_id"`
    Status      string     `gorm:"not null;default:'pending'" json:"status"`
    
    Prompt      string     `gorm:"type:text" json:"prompt"`         // 渲染后的prompt（已替换变量）
    Output      string     `gorm:"type:text" json:"output"`         // agent回复全文
    Outputs     string     `gorm:"type:text;default:'{}'" json:"outputs"` // JSON，提取的结构化变量
    
    TaskID      *string    `gorm:"index" json:"task_id"`            // 关联的Task ID（NEW）
    
    Attempt     int        `gorm:"not null;default:1" json:"attempt"`
    Error       *string    `json:"error"`
    StartedAt   *time.Time `json:"started_at"`
    CompletedAt *time.Time `json:"completed_at"`
    CreatedAt   time.Time  `json:"created_at"`
}
```

### 5.7 验收标准

- [ ] 工作流步骤发送 prompt 后等待 agent 回复（轮询 sessions_history）
- [ ] agent 回复写入 StepRun.output 和关联 Task.output
- [ ] 下游步骤能引用上游步骤的 output（`{{steps.<id>.output}}`）
- [ ] 超时后按 on_failure 策略处理（默认 pause）
- [ ] 重试机制生效：失败后自动重试 N 次
- [ ] 每个步骤自动创建 Task 记录，Task 列表页可看到工作流产生的任务
- [ ] 工作流执行详情页显示每个步骤的 input（prompt）和 output（回复）

---

## 6. MCP Server — P0

### 6.1 设计理念

Lyra 通过 MCP（Model Context Protocol）接口操作 ACP。MCP Server 作为 ACP 的编程接口层，暴露全部核心能力。

### 6.2 部署方式

MCP Server 作为独立进程运行，通过 stdio 与 Lyra 的 OpenClaw session 通信。

```
Lyra (OpenClaw Agent)
    ↕ stdio (JSON-RPC)
ACP MCP Server (cmd/acp-mcp/main.go)
    ↕ 内部调用
ACP Service Layer（复用同一套 service 代码）
    ↕
SQLite + OpenClaw Gateway
```

### 6.3 MCP 工具列表

#### 任务管理

| Tool | 参数 | 说明 |
|------|------|------|
| `acp_task_create` | title, description, input, priority, agent_id | 创建任务 |
| `acp_task_list` | status?, agent_id?, source_type?, priority?, limit? | 查询任务列表 |
| `acp_task_get` | task_id | 获取任务详情（含 input/output） |
| `acp_task_assign` | task_id, agent_id | 指派任务给 agent |
| `acp_task_cancel` | task_id | 取消任务 |
| `acp_task_retry` | task_id | 重试失败的任务 |

#### 工作流管理

| Tool | 参数 | 说明 |
|------|------|------|
| `acp_workflow_list` | status? | 查询工作流列表 |
| `acp_workflow_get` | workflow_id | 获取工作流详情 |
| `acp_workflow_create` | name, yaml_content | 创建工作流 |
| `acp_workflow_update` | workflow_id, yaml_content | 更新工作流 YAML |
| `acp_workflow_execute` | workflow_id, inputs? | 触发执行工作流 |
| `acp_workflow_run_status` | run_id | 查看执行进度 |
| `acp_workflow_run_cancel` | run_id | 取消执行 |
| `acp_workflow_step_retry` | run_id, step_id | 重试某个步骤 |

#### Agent 监控

| Tool | 参数 | 说明 |
|------|------|------|
| `acp_agent_list` | — | 列出所有 agent 及状态 |
| `acp_agent_status` | agent_id | 查看 agent 详情 + token 用量 |
| `acp_agent_history` | agent_id, limit? | 查看 agent 对话历史 |
| `acp_agent_send` | agent_id, message | 向 agent 发送消息 |

#### 任务队列

| Tool | 参数 | 说明 |
|------|------|------|
| `acp_queue_status` | — | 查看队列状态（待分配数、各 agent 负载） |
| `acp_queue_push` | task_id, priority? | 将任务推入队列 |
| `acp_queue_pause` | — | 暂停自动调度 |
| `acp_queue_resume` | — | 恢复自动调度 |

### 6.4 MCP 工具示例

```json
{
  "name": "acp_task_create",
  "description": "创建一个新任务并可选指派给 agent",
  "inputSchema": {
    "type": "object",
    "properties": {
      "title": { "type": "string", "description": "任务标题" },
      "description": { "type": "string", "description": "任务描述" },
      "input": { "type": "string", "description": "发给 agent 的 prompt" },
      "priority": { "type": "string", "enum": ["P0", "P1", "P2"], "default": "P1" },
      "agent_id": { "type": "string", "description": "指派的 agent ID（可选，不填则进入队列）" }
    },
    "required": ["title", "input"]
  }
}
```

### 6.5 验收标准

- [ ] MCP Server 独立二进制，通过 stdio 通信
- [ ] Lyra 能通过 MCP 创建任务、触发工作流、查看进度
- [ ] MCP 操作的数据与 Web 前端看到的一致（同一数据库）
- [ ] 所有 MCP 工具有完整的 inputSchema 定义
- [ ] 错误返回标准 MCP error 格式

---

## 7. 任务队列和自动调度 — P1

### 7.1 设计理念

Agent 空闲时自动从队列取下一个任务，不需要人工干预。

### 7.2 队列机制

```
任务入队（手动/工作流/定时）
    ↓
队列按优先级排序（P0 > P1 > P2，同优先级按创建时间 FIFO）
    ↓
调度器每 30s 检查一次
    ↓
检测空闲 agent（无 running 状态的 task）
    ↓
匹配：agent 能力 × 任务要求
    ↓
分配：更新 Task.agent_id + status=assigned → 发送 prompt
```

### 7.3 Agent 能力标签

在 Agent 配置中增加能力标签，用于任务匹配：

```go
type Agent struct {
    // ...existing fields...
    Capabilities string `gorm:"default:'[]'" json:"capabilities"`  // JSON array: ["prd","research","code","design"]
}
```

| Agent | 能力 |
|-------|------|
| main (Lyra) | code, deploy, review, coordination |
| pm | prd, research, task-breakdown, backlog |
| ux | design, prototype, design-review |
| alice | code, test, documentation |

任务可以指定 `required_capability`，队列调度器只会分配给具备该能力的 agent。不指定则任何 agent 都可以接。

### 7.4 空闲检测

Agent 被视为"空闲"的条件：
1. 没有 status=running 的 Task
2. 最近 5 分钟没有新的 assistant 消息（说明不在忙别的事）

### 7.5 负载均衡策略

- 优先分配给完全空闲的 agent
- 同等条件下，优先分配给已完成任务最少的 agent（均衡负载）
- P0 任务可以抢占 P2 任务的 agent（抢占式调度，v2 先不做）

### 7.6 验收标准

- [ ] 未指定 agent 的任务自动进入队列
- [ ] 调度器每 30s 检查并分配任务
- [ ] 分配遵循优先级排序
- [ ] Agent 能力匹配生效
- [ ] 前端能看到队列状态（待分配数、各 agent 当前任务）
- [ ] Lyra 能通过 MCP 暂停/恢复自动调度

---

## 8. 已有功能（保留）

以下为 v1 已实现功能，保持不变：

### 8.1 Agent 监控面板（Dashboard）— ✅ 已完成

路由：`/` — Agent 卡片网格、全局统计栏、10s 自动刷新。

### 8.2 Agent 详情页 — ✅ 已完成

路由：`/agents/:id` — 对话历史流（左60%）+ 状态侧边栏（右40%）。

### 8.3 工作流编排（前端 + YAML DSL）— ✅ 已完成

路由：`/workflows` — YAML 编辑器 + DAG 预览 + 执行 + 进度查看。

### 8.4 任务管理（基础 CRUD）— ✅ 已完成

路由：`/tasks` — 看板视图 + 创建/指派。

**v2 需要升级：** Task 数据模型扩展（加 input/output/source_type 等字段）、看板增加来源筛选。

---

## 9. 工作流 DSL 规范（v2 扩展）

### 9.1 v2 新增字段

在 v1 DSL 基础上新增：

```yaml
steps:
  - id: analyze
    name: 需求分析
    agent: pm
    prompt: |
      分析需求：{{inputs.task_description}}
    timeout: 30m                    # 等待agent回复的超时时间
    
    # NEW: 等待回复配置
    wait_reply: true                # 默认true，是否等待agent回复
    poll_interval: 10s              # 轮询间隔，默认10s
    
    # NEW: 输出提取（从agent回复中）
    outputs:
      prd_path: "regex:PRD.*?(/[\\w/.-]+\\.md)"     # 正则提取
      summary: "section:## 摘要"                      # 按标题段落提取
    
    # 重试（同v1）
    retry:
      max_attempts: 2
      delay: 5m
      on: failure
    
    on_failure: pause               # pause/skip/abort
```

### 9.2 输出变量提取方式

| 方式 | 语法 | 说明 |
|------|------|------|
| 标记提取 | `[output:key=value]` | Agent 回复中显式标记（推荐） |
| 正则提取 | `regex:<pattern>` | 用正则从回复中匹配 |
| 全文传递 | `{{steps.<id>.output}}` | 不提取，直接传完整回复 |

v2 优先用全文传递（`{{steps.<id>.output}}`），简单可靠。结构化提取作为增强能力。

---

## 10. API 设计（v2 新增/变更）

### 10.1 Task API（变更）

```
GET    /api/tasks                          → Task[]         # 新增筛选：?source_type=workflow
POST   /api/tasks                          → Task           # 新增字段：input, source_type
GET    /api/tasks/:id                      → TaskDetail     # 包含 input + output
PUT    /api/tasks/:id                      → Task
DELETE /api/tasks/:id                      → { ok: bool }
POST   /api/tasks/:id/assign              → Task           # 指派
POST   /api/tasks/:id/cancel              → Task           # 取消（NEW）
POST   /api/tasks/:id/retry               → Task           # 重试（NEW）
```

### 10.2 Queue API（新增）

```
GET    /api/queue/status                   → QueueStatus    # 队列状态
POST   /api/queue/pause                    → { ok: bool }   # 暂停调度
POST   /api/queue/resume                   → { ok: bool }   # 恢复调度
```

### 10.3 其他 API 不变

Agent API、Workflow API、System API 与 v1 保持一致。

---

## 11. 前端变更（v2）

### 11.1 Tasks 页面升级

- 看板增加来源标签（手动/工作流/定时/队列）
- Task 详情弹窗/页面显示 input 和 output
- 按来源类型筛选
- 显示关联的工作流步骤（如有）

### 11.2 Dashboard 统计栏升级

- 队列待分配任务数
- 今日完成任务数（真实数据）
- 活跃工作流数（真实数据）

### 11.3 工作流执行详情升级

- 每个步骤显示 input（渲染后的 prompt）和 output（agent 回复）
- 步骤状态增加 "waiting"（等待回复中）
- 进度条或计时器显示等待时长

---

## 12. 页面结构与路由（v2）

```
/                          Dashboard（Agent监控面板）
/agents/:id                Agent详情页
/tasks                     任务管理（看板视图，含来源筛选）
/tasks/:id                 任务详情（input/output）       ← NEW
/workflows                 工作流列表
/workflows/new             创建工作流
/workflows/:id             工作流详情（执行记录）
/workflows/:id/edit        工作流编辑器
/workflow-runs/:id         工作流执行详情（含步骤input/output）
/queue                     任务队列状态                    ← NEW
/settings                  系统设置
/login                     登录页
```

---

## 13. 数据模型变更摘要

| 表 | 变更类型 | 说明 |
|-----|---------|------|
| tasks | ALTER | 新增 input, output, source_type, source_id, step_id, started_at 字段 |
| workflow_step_runs | ALTER | 新增 task_id, outputs(JSON) 字段；output 改为存 agent 回复全文 |
| agents | ALTER | 新增 capabilities(JSON) 字段 |
| queue_config | NEW | 队列配置（调度间隔、是否暂停等） |

---

## 14. 分期实施计划

### Phase 1 — P0（核心闭环，2周）

**目标：** Task 有 input/output + 工作流等回复 + MCP 基础接口

| # | 任务 | 复杂度 | 说明 |
|---|------|--------|------|
| 1 | Task 数据模型升级 | S | 加 input/output/source_type 等字段，migration |
| 2 | 工作流引擎 v2 — 异步等待回复 | L | 核心改造：轮询等回复、写入output、数据流传递 |
| 3 | 工作流 → Task 自动关联 | M | 步骤执行时自动创建Task，output写回Task |
| 4 | MCP Server 骨架 + 核心工具 | M | stdio JSON-RPC server + task/workflow/agent 工具 |
| 5 | 前端 Task 页面升级 | S | 显示 input/output、来源标签、筛选 |
| 6 | 前端工作流执行详情升级 | S | 显示步骤 input/output、waiting 状态 |

### Phase 2 — P1（自动调度，1周）

| # | 任务 | 复杂度 | 说明 |
|---|------|--------|------|
| 7 | 任务队列 + 调度器 | M | 优先级队列、空闲检测、自动分配 |
| 8 | Agent 能力标签 | S | capabilities 字段 + 匹配逻辑 |
| 9 | MCP 队列管理工具 | S | pause/resume/status |
| 10 | 前端队列状态页 | S | /queue 页面 |

### Phase 3 — P2（增强，按需）

| # | 任务 | 说明 |
|---|------|------|
| 11 | Output 结构化提取（正则/段落） | 从 agent 回复中智能提取变量 |
| 12 | 工作流触发器（定时/Webhook） | cron 触发工作流执行 |
| 13 | 抢占式调度 | P0 任务抢占 P2 的 agent |
| 14 | WebSocket 实时推送 | 替代前端轮询 |
| 15 | 任务统计与报表 | 按 agent/时间段统计产出 |

---

## 15. 技术约束

- **技术栈：** Go（Gin + GORM）后端 + React（Ant Design）前端 + SQLite
- **MCP 协议：** JSON-RPC over stdio，遵循 Model Context Protocol 规范
- **工作流引擎：** goroutine 并发执行，轮询间隔可配
- **部署：** 单二进制，43.134.86.237:3001

---

## 16. 依赖

- OpenClaw Gateway WebSocket API（sessions_list / sessions_history / sessions_send / session_status）
- MCP SDK（Go 实现，或手写 JSON-RPC stdio handler）
- 现有 ACP v1 全部代码

---

## 变更记录

| 日期 | 变更内容 | 原因 |
|------|---------|------|
| 2026-02-19 | v1 初版 PRD | 泽斌提出 Agent 管理需求 |
| 2026-02-20 | v2 重大升级 | Lyra 提出：Task 系统重设计、工作流异步等待、MCP Server、任务队列 |
