# Task Breakdown: ACP v2 — P0 阶段

PRD: `/docs/prd/agent-control-panel.md`（v2）

---

## 任务依赖图

```
Task 1 (Task数据模型升级)
  ├→ Task 2 (工作流引擎v2 — 异步等待回复)
  │    └→ Task 3 (工作流→Task自动关联)
  │         └→ Task 5 (前端Task页面升级)
  ├→ Task 4 (MCP Server)
  └→ Task 6 (前端工作流执行详情升级)
```

Task 1 是基础，完成后 Task 2/4/6 可并行（但CC同时只能跑一个，按 1→2→3→4→5→6 顺序执行）。

---

## Task 1: [后端] Task 数据模型升级 + Migration

**描述：**
扩展 Task 数据模型，增加 input/output/source_type 等字段。更新 entity、service、handler、前端 API 类型定义。这是所有后续任务的基础。

**具体工作：**

1. **修改 `entity/entity.go` 中的 Task struct：**
   ```go
   type Task struct {
       // 保留已有字段：ID, Title, Description, Status, Priority, AgentID, CreatedBy, DueDate, CreatedAt, UpdatedAt
       
       // 新增字段
       Input       string     `gorm:"type:text" json:"input"`           // 发给agent的prompt
       Output      string     `gorm:"type:text" json:"output"`          // agent回复/产出
       SourceType  string     `gorm:"not null;default:'manual'" json:"source_type"`  // manual/workflow/cron/queue
       SourceID    *string    `json:"source_id"`                        // 关联的来源ID
       StepID      *string    `json:"step_id"`                          // 关联的workflow step id
       StartedAt   *time.Time `json:"started_at"`
       CompletedAt *time.Time `json:"completed_at"`
   }
   ```

2. **修改 `entity/entity.go` 中的 WorkflowStepRun struct：**
   - 新增 `TaskID *string` 字段（关联 Task）
   - 新增 `Outputs string` 字段（JSON，结构化变量提取）
   - 新增 `Attempt int` 字段（重试次数）

3. **修改 Task Status 枚举：**
   - v1 状态：pending / in_progress / completed / failed / cancelled
   - v2 状态：pending / assigned / running / completed / failed / cancelled
   - 更新 service 层 status 校验逻辑

4. **修改 Task Priority 枚举：**
   - v1：low / medium / high / urgent
   - v2：P0 / P1 / P2（与 PRD 一致）
   - 更新 service 层 priority 校验

5. **更新 `service/task_service.go`：**
   - `TaskFilter` 新增 `SourceType` 筛选字段
   - `CreateRequest` 新增 `Input`, `SourceType`, `SourceID`, `StepID` 字段
   - 新增 `Cancel(id string)` 方法（任何状态 → cancelled）
   - 新增 `Retry(id string)` 方法（failed → pending，清空 output）
   - 新增 `SetOutput(id string, output string)` 方法（写入 agent 回复）
   - 新增 `SetRunning(id string)` 方法（assigned → running，记录 started_at）

6. **更新 `handler/task_handler.go`：**
   - 新增 `POST /api/tasks/:id/cancel` 路由
   - 新增 `POST /api/tasks/:id/retry` 路由
   - Create 接口支持 input、source_type 等新字段

7. **SQLite Migration：**
   - GORM AutoMigrate 会自动加列，但需确认 SQLite 兼容性
   - 已有数据的 source_type 默认填 'manual'，priority 默认填 'P1'

**涉及文件：**
- `internal/acp/entity/entity.go` — 数据模型
- `internal/acp/service/task_service.go` — 业务逻辑
- `internal/acp/handler/task_handler.go` — HTTP handler
- `internal/acp/handler/handler.go` — 路由注册（新增 cancel/retry 路由）

**验收标准：**
- [ ] Task entity 包含 input, output, source_type, source_id, step_id, started_at, completed_at 字段
- [ ] WorkflowStepRun entity 包含 task_id, outputs, attempt 字段
- [ ] 启动后 AutoMigrate 成功，新字段已加到 SQLite
- [ ] `POST /api/tasks` 支持 input 和 source_type 字段
- [ ] `GET /api/tasks?source_type=workflow` 筛选生效
- [ ] `POST /api/tasks/:id/cancel` 和 `/retry` 正常工作
- [ ] 已有的 Task CRUD 功能不受影响（向后兼容）
- [ ] Task status 使用新枚举（pending/assigned/running/completed/failed/cancelled）
- [ ] Task priority 使用 P0/P1/P2

**依赖：** 无
**复杂度：** S
**优先级：** P0

---

## Task 2: [后端] 工作流引擎 v2 — 异步等待 Agent 回复

**描述：**
核心改造。把工作流引擎从"发完就忘"改为"发送 → 等待回复 → 收集 output → 传给下一步"。这是最复杂的任务。

**具体工作：**

1. **实现 `WaitForReply` 方法（`service/workflow_engine.go`）：**
   ```go
   // WaitForReply 发送prompt后轮询等待agent回复
   // 返回 agent 的回复内容
   func (e *WorkflowEngine) WaitForReply(sessionKey string, sentAt time.Time, timeout time.Duration) (string, error)
   ```
   - 每 10s 轮询 `gateway.SessionsHistory(sessionKey, 10, false)`
   - 查找 sentAt 之后的 assistant 消息
   - 超时返回 `ErrTimeout`
   - 支持取消（通过 context.Context）

2. **改造 `executeStep` 方法：**
   - v1：`sessions_send` → 立即标 completed
   - v2：
     ```
     a. 渲染 prompt 模板（替换 {{steps.<id>.output}} 变量）
     b. sessions_send(agent, rendered_prompt)
     c. 记录 sentAt 时间
     d. 标记步骤为 "waiting"（新状态）
     e. WaitForReply(sessionKey, sentAt, timeout)
     f. 收到回复 → 写入 StepRun.output → 标记 completed
     g. 超时/失败 → 按 on_failure 策略处理
     ```

3. **实现模板变量渲染（`service/workflow_engine.go`）：**
   ```go
   // RenderPrompt 替换 prompt 中的模板变量
   // 支持：{{inputs.xxx}}, {{steps.<id>.output}}, {{steps.<id>.outputs.<key>}}
   func (e *WorkflowEngine) RenderPrompt(prompt string, inputs map[string]string, stepOutputs map[string]*StepOutput) string
   ```
   - `stepOutputs` 存储每个已完成步骤的 output 和 outputs
   - 用 strings.Replace 或简单的模板引擎

4. **步骤间数据流传递：**
   - 在 `execute()` 中维护 `stepOutputs map[string]*StepOutput`
   - 每个步骤完成后，将其 output 存入 map
   - 下游步骤渲染 prompt 时引用上游 output

5. **步骤状态新增 "waiting"：**
   - 现有状态：pending / running / completed / failed / skipped
   - 新增：waiting（prompt已发送，等待agent回复）
   - 前端和API需要兼容这个新状态

6. **超时和重试逻辑：**
   - 从 YAML step 定义读取 timeout（默认 30m）
   - 从 YAML step 定义读取 retry 配置
   - 失败后根据 retry.max_attempts 决定是否重试
   - 重试时增加 StepRun.attempt 计数

7. **on_failure 策略：**
   - `pause`：标记 run 为 paused，等待人工干预
   - `skip`：跳过当前步骤，继续下游
   - `abort`：终止整个 workflow run

8. **更新 WorkflowDef / StepDef 解析（`service/workflow_service.go`）：**
   - StepDef 新增字段：`Timeout`, `Retry`, `OnFailure`, `WaitReply`, `PollInterval`
   - YAML 解析适配新字段

**涉及文件：**
- `internal/acp/service/workflow_engine.go` — 主要改造文件
- `internal/acp/service/workflow_service.go` — WorkflowDef/StepDef 结构体 + YAML 解析
- `internal/acp/gateway/client.go` — 确认 SessionsHistory 返回消息带时间戳

**验收标准：**
- [ ] 工作流步骤发送 prompt 后进入 "waiting" 状态
- [ ] 引擎轮询 sessions_history 检测 agent 回复（每10s一次）
- [ ] 收到回复后写入 StepRun.output，标记 completed
- [ ] 下游步骤的 prompt 中 `{{steps.<id>.output}}` 被正确替换为上游回复
- [ ] 超时后步骤标记 failed，按 on_failure 策略处理
- [ ] retry 配置生效：失败后自动重试，attempt 计数正确
- [ ] 两步以上的工作流能正确传递数据：step1 output → step2 prompt
- [ ] 取消 run 时，正在 waiting 的步骤能被中断

**依赖：** Task 1（StepRun 新增 outputs/attempt 字段）
**复杂度：** L（核心改造，预计 3-4 小时）
**优先级：** P0

---

## Task 3: [后端] 工作流 → Task 自动关联

**描述：**
工作流引擎执行每个 step 时自动创建 Task 记录，将工作流产出写入 Task.output，实现 Task 与 Workflow 的数据闭环。

**具体工作：**

1. **在 `executeStep` 中创建 Task：**
   - 步骤开始执行时，调用 `TaskService.Create()` 创建 Task：
     ```go
     task := TaskService.Create({
         Title:      fmt.Sprintf("[%s] %s", workflow.Name, step.Name),
         Description: fmt.Sprintf("工作流 %s 的步骤 %s", workflow.Name, step.Name),
         Input:      renderedPrompt,
         AgentID:    step.Agent,
         Status:     "running",
         Priority:   "P1",        // 继承工作流优先级或默认P1
         SourceType: "workflow",
         SourceID:   runID,
         StepID:     step.ID,
         CreatedBy:  "system",
     })
     ```
   - 将 Task.ID 写入 StepRun.TaskID

2. **回复收到后更新 Task：**
   - `TaskService.SetOutput(taskID, agentReply)`
   - 更新 Task.status = "completed"
   - 更新 Task.completed_at

3. **步骤失败时更新 Task：**
   - Task.status = "failed"
   - 如果重试，创建新 Task（或复用原 Task 更新 status 回 running）

4. **WorkflowEngine 依赖注入 TaskService：**
   - `WorkflowEngine` struct 新增 `TaskService *TaskService` 字段
   - 修改 `NewWorkflowEngine()` 签名，接收 TaskService
   - 更新 `cmd/acp/main.go` 中的初始化代码

**涉及文件：**
- `internal/acp/service/workflow_engine.go` — 在 executeStep 中创建/更新 Task
- `internal/acp/service/task_service.go` — 确认 Create/SetOutput/SetRunning 方法可用
- `cmd/acp/main.go` — 更新 WorkflowEngine 初始化

**验收标准：**
- [ ] 执行工作流后，Tasks 页面能看到自动创建的 Task（source_type=workflow）
- [ ] Task.input = 渲染后的 prompt
- [ ] Task.output = agent 的回复内容
- [ ] Task.source_id = workflow run ID
- [ ] Task.step_id = workflow step ID
- [ ] StepRun.task_id 关联正确的 Task
- [ ] 步骤失败时 Task 状态也是 failed
- [ ] 手动创建的 Task（source_type=manual）不受影响

**依赖：** Task 2（异步等待回复机制，才有 output 可写）
**复杂度：** M
**优先级：** P0

---

## Task 4: [后端] MCP Server 骨架 + 核心工具

**描述：**
创建 ACP MCP Server，通过 stdio JSON-RPC 暴露 Task/Workflow/Agent 操作工具，供 Lyra 编程调度。

**具体工作：**

1. **创建 `cmd/acp-mcp/main.go`：**
   - 独立 main 入口
   - 初始化 SQLite 连接（复用同一数据库文件）
   - 初始化 Gateway Client
   - 初始化 Service 层（TaskService, WorkflowService, AgentService, WorkflowEngine）
   - 启动 MCP stdio server

2. **实现 MCP JSON-RPC stdio handler（`internal/acp/mcp/server.go`）：**
   - 从 stdin 读取 JSON-RPC 请求
   - 路由到对应 tool handler
   - 结果写入 stdout
   - 支持 MCP 协议：`initialize`, `tools/list`, `tools/call`
   - 错误返回标准 MCP error 格式

3. **实现 MCP 工具（`internal/acp/mcp/tools.go`）：**

   **任务管理（6个工具）：**
   - `acp_task_create` — 创建任务（title, input, priority, agent_id, description）
   - `acp_task_list` — 查询列表（status, agent_id, source_type, priority, limit）
   - `acp_task_get` — 获取详情（task_id）
   - `acp_task_assign` — 指派（task_id, agent_id）
   - `acp_task_cancel` — 取消（task_id）
   - `acp_task_retry` — 重试（task_id）

   **工作流管理（6个工具）：**
   - `acp_workflow_list` — 列表（status）
   - `acp_workflow_get` — 详情（workflow_id）
   - `acp_workflow_create` — 创建（name, yaml_content）
   - `acp_workflow_update` — 更新（workflow_id, yaml_content）
   - `acp_workflow_execute` — 触发执行（workflow_id, inputs）
   - `acp_workflow_run_status` — 查看执行进度（run_id）

   **Agent 监控（4个工具）：**
   - `acp_agent_list` — 列出所有 agent + 状态
   - `acp_agent_status` — 查看详情 + token 用量（agent_id）
   - `acp_agent_history` — 对话历史（agent_id, limit）
   - `acp_agent_send` — 发送消息（agent_id, message）

4. **每个工具完整的 inputSchema 定义**

5. **Makefile 更新：**
   - 新增 `make build-mcp` target
   - 编译 `cmd/acp-mcp/main.go` → `bin/acp-mcp`

6. **OpenClaw MCP 配置说明：**
   - 在 README 或注释中说明如何在 Lyra 的 OpenClaw config 中配置 MCP tool

**涉及文件：**
- `cmd/acp-mcp/main.go` — MCP Server 入口（新建）
- `internal/acp/mcp/server.go` — MCP JSON-RPC stdio handler（新建）
- `internal/acp/mcp/tools.go` — 工具定义 + handler（新建）
- `Makefile` — 新增 build-mcp target

**验收标准：**
- [ ] `make build-mcp` 编译出 `bin/acp-mcp` 二进制
- [ ] `echo '{"jsonrpc":"2.0","id":1,"method":"initialize",...}' | ./bin/acp-mcp` 返回正确的 initialize 响应
- [ ] `tools/list` 返回 16 个工具及其完整 inputSchema
- [ ] `tools/call` 调用 `acp_task_create` 能成功创建任务
- [ ] `tools/call` 调用 `acp_task_list` 返回任务列表
- [ ] `tools/call` 调用 `acp_workflow_execute` 能触发工作流执行
- [ ] `tools/call` 调用 `acp_agent_list` 返回 agent 列表和在线状态
- [ ] 所有工具的错误情况返回标准 MCP error 格式
- [ ] MCP Server 和 Web Server 操作同一个 SQLite 数据库

**依赖：** Task 1（Task 新字段）
**复杂度：** M（工作量中等，逻辑简单但工具多）
**优先级：** P0

---

## Task 5: [前端] Task 页面升级 — input/output + 来源筛选

**描述：**
升级 Tasks 页面，展示 Task 的 input/output，增加来源类型筛选，适配新的状态和优先级枚举。

**具体工作：**

1. **更新 `acp-web/src/api/tasks.ts` 类型定义：**
   ```typescript
   export type TaskStatus = 'pending' | 'assigned' | 'running' | 'completed' | 'failed' | 'cancelled';
   export type TaskPriority = 'P0' | 'P1' | 'P2';
   export type TaskSourceType = 'manual' | 'workflow' | 'cron' | 'queue';
   
   export interface Task {
     // ...existing fields...
     input: string;
     output: string;
     source_type: TaskSourceType;
     source_id: string | null;
     step_id: string | null;
     started_at: string | null;
     completed_at: string | null;
   }
   ```

2. **更新 `TaskFilter` 增加 `source_type` 参数**

3. **新增 API 函数：**
   - `cancelTask(id: string)` → `POST /api/tasks/:id/cancel`
   - `retryTask(id: string)` → `POST /api/tasks/:id/retry`

4. **更新 `acp-web/src/pages/Tasks.tsx`：**

   a. **状态选项更新：**
   ```
   pending → 待处理
   assigned → 已指派
   running → 执行中
   completed → 已完成
   failed → 失败
   cancelled → 已取消
   ```

   b. **优先级选项更新：**
   ```
   P0 → 紧急 (红色)
   P1 → 正常 (蓝色)
   P2 → 低 (灰色)
   ```

   c. **增加来源类型筛选器：**
   - 筛选栏新增 Select：全部 / 手动 / 工作流 / 定时 / 队列

   d. **来源标签展示：**
   - 列表/看板卡片上显示来源 Tag（手动=蓝、工作流=紫、定时=绿、队列=橙）

   e. **Task 详情弹窗增强：**
   - 显示 Input 区域（代码块或可折叠文本，展示发给 agent 的 prompt）
   - 显示 Output 区域（展示 agent 回复，支持 Markdown 渲染）
   - 显示来源信息（如果是 workflow，显示关联的工作流名称和步骤）

   f. **操作按钮：**
   - Failed 状态的任务显示"重试"按钮
   - Running/Assigned 状态的任务显示"取消"按钮

5. **创建任务弹窗增加 Input 字段：**
   - TextArea，用于输入发给 agent 的 prompt
   - label: "任务指令（Prompt）"

**涉及文件：**
- `acp-web/src/api/tasks.ts` — 类型定义 + 新 API
- `acp-web/src/pages/Tasks.tsx` — 页面组件
- `acp-web/src/components/KanbanBoard.tsx` — 看板卡片（如需加来源标签）
- `acp-web/src/tokens.ts` — 可能需要新增来源类型颜色

**UX 设计依赖：** 无（使用 Ant Design 标准组件即可）

**验收标准：**
- [ ] 来源类型筛选生效：选"工作流"只显示 source_type=workflow 的任务
- [ ] 列表/看板卡片显示来源 Tag
- [ ] 任务详情弹窗显示 Input（prompt）和 Output（agent 回复）
- [ ] 状态使用新枚举（pending/assigned/running/completed/failed/cancelled）
- [ ] 优先级使用 P0/P1/P2
- [ ] "重试"按钮对 failed 任务生效
- [ ] "取消"按钮对 running/assigned 任务生效
- [ ] 创建任务弹窗有 Input（Prompt）字段

**依赖：** Task 3（需要有 workflow 产生的 task 数据来测试）
**复杂度：** M
**优先级：** P0

---

## Task 6: [前端] 工作流执行详情页升级 — 步骤 input/output + waiting 状态

**描述：**
升级工作流执行详情页（WorkflowRunDetail），展示每个步骤的 input（渲染后 prompt）和 output（agent 回复），增加 waiting 状态显示。

**具体工作：**

1. **更新 `acp-web/src/pages/WorkflowRunDetail.tsx`：**

   a. **步骤时间线增加 "waiting" 状态：**
   - pending: 灰色圆点
   - running: 蓝色旋转
   - **waiting: 蓝色脉动动画 + "等待回复中..."文字 + 已等待时长计时器**
   - completed: 绿色勾
   - failed: 红色叉
   - skipped: 灰色跳过

   b. **步骤展开详情增强：**
   - **Input 区域：** 显示渲染后的 prompt（StepRun.prompt），代码块样式，可折叠
   - **Output 区域：** 显示 agent 回复（StepRun.output），Markdown 渲染，可折叠
   - 如果 output 为空且状态为 waiting，显示"等待 agent 回复中..."
   - 显示关联的 Task ID（可点击跳转到 Task 详情）

   c. **等待时长显示：**
   - waiting 状态的步骤显示 "已等待 X 分钟"（实时计时）
   - 显示超时时间（如果 YAML 配置了 timeout）

   d. **步骤操作按钮：**
   - failed 步骤：显示"重试"按钮（调用 `POST /api/workflow-runs/:id/steps/:sid/retry`）
   - waiting 步骤：显示"跳过"按钮（调用 `POST /api/workflow-runs/:id/steps/:sid/skip`）

2. **更新步骤 API 类型：**
   - StepRun 类型新增 `output`, `outputs`, `task_id`, `attempt` 字段
   - 确认 `GET /api/workflow-runs/:id` 返回的 step_runs 包含这些字段

3. **Dashboard 统计栏升级：**
   - "今日任务数"：从 `GET /api/tasks?source_type=&completed_at=today` 获取真实数据
   - "活跃工作流数"：从运行中的 workflow runs 计算
   - "队列待分配"：预留字段（P1 实现）

**涉及文件：**
- `acp-web/src/pages/WorkflowRunDetail.tsx` — 主要改造
- `acp-web/src/api/workflows.ts` — StepRun 类型更新
- `acp-web/src/pages/Dashboard.tsx` — 统计栏真实数据

**UX 设计依赖：** 无

**验收标准：**
- [ ] 步骤时间线支持 waiting 状态（脉动动画 + 等待时长）
- [ ] 点击步骤能看到 Input（prompt）和 Output（回复）
- [ ] Output 支持 Markdown 渲染
- [ ] waiting 状态显示实时计时器"已等待 X 分钟"
- [ ] failed 步骤有"重试"按钮
- [ ] 步骤详情显示关联的 Task ID（可点击）
- [ ] Dashboard 统计栏显示真实的今日任务数和活跃工作流数

**依赖：** Task 1（StepRun 新字段）
**复杂度：** M
**优先级：** P0

---

## 执行顺序

```
Task 1 (S, ~1h)   → 数据模型基础，最先做
Task 2 (L, ~3-4h) → 核心引擎改造，最复杂
Task 3 (M, ~2h)   → 依赖 Task 2 的异步等待机制
Task 4 (M, ~2-3h) → 依赖 Task 1，与 2/3 无代码冲突但串行执行
Task 5 (M, ~2h)   → 需要 Task 3 产生测试数据
Task 6 (M, ~2h)   → 依赖 Task 1 的新字段
```

**预计总工时：** 12-14 小时 CC 开发时间（6 个 session）

---

## 变更记录

| 日期 | 变更 | 原因 |
|------|------|------|
| 2026-02-20 | P0 阶段任务拆解 | PRD v2 审核通过，开始开发 |
