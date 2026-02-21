# ACP v2.3：审批系统升级 + 条件分支

## 背景

当前审批节点只能"通过/拒绝"，缺少行业标准能力。需要对齐 n8n / 企业BPM 的审批标准。

## 项目位置

`/home/claw/.openclaw/workspace/agent-control-panel/`

## 现有代码结构（必读）

- 审批实体：`internal/acp/entity/entity.go` → Approval, ApprovalAction
- 审批服务：`internal/acp/service/approval_service.go`
- 审批处理器：`internal/acp/handler/approval_handler.go`
- 工作流定义：`internal/acp/service/workflow_service.go` → ApprovalDef, StepDef
- 工作流引擎：`internal/acp/service/workflow_engine.go` → executeApprovalStep
- 前端审批页：`acp-web/src/pages/Approvals.tsx`, `ApprovalDetail.tsx`
- 前端API：`acp-web/src/api/approvals.ts`
- **工作流配置参考文档**：`docs/workflow-reference.md`（所有功能改完后同步更新）

## 任务清单（按顺序执行）

### 任务1：审批通过时支持修改上游输出（P0）

**需求：** 审批人在通过审批时，可以编辑上一步的结构化输出内容（增删改），修改后的版本替换原输出传给下游步骤。

**后端改动：**

1. `approval_handler.go` — Approve API 增加 `modified_output` 字段（JSON object，可选）
```go
type approvalActionRequest struct {
    Comment        string                 `json:"comment"`
    ModifiedOutput map[string]interface{} `json:"modified_output,omitempty"` // 新增
}
```

2. `approval_service.go` — ProcessAction 增加 modified_output 参数。当审批通过且有 modified_output 时，更新对应步骤的 structured_output 和 output 字段。

3. `workflow_engine.go` — executeApprovalStep 中，审批通过后检查是否有 modified_output：
   - 有：用 modified_output 覆盖 approval.depends_on 指向的上一步的 structured_output
   - 无：保持原输出不变
   - 将审批人的 comment 存入 `stepOutputs[stepID]`，后续步骤可通过 `{{steps.approve_step.output}}` 引用

4. `entity.go` — ApprovalAction 增加 `ModifiedOutput` 字段：
```go
ModifiedOutput string `gorm:"type:text" json:"modified_output,omitempty"`
```

**前端改动：**

5. `ApprovalDetail.tsx` — 审批详情页增加"编辑输出"功能：
   - 展示上一步的 structured_output（JSON 格式）
   - 提供可编辑的 JSON 编辑器（可以用简单的 textarea + JSON 校验，不需要复杂组件）
   - "通过"按钮提交时携带修改后的 JSON
   - "通过（不修改）"按钮 = 原有行为

### 任务2：审批意见结构化传递（P0）

**需求：** 审批步骤完成后，后续步骤可通过模板变量引用审批结果。

**改动：**

1. `workflow_engine.go` — 审批步骤 approved 时，将 comment 和 modified_output 信息写入 stepOutputs：
```go
stepOutputs[stepID] = &StepOutput{
    Output: comment,  // 审批意见文本
    Outputs: map[string]string{
        "result": "approved",
        "comment": comment,
        "modified": "true/false",  // 是否修改了输出
    },
}
```

2. 这样后续步骤可以用 `{{steps.approve_step.output}}` 拿到审批意见。

### 任务3：通用条件分支节点（P1）

**需求：** 工作流支持 `type: condition` 节点，根据表达式走不同分支。

**YAML 设计：**
```yaml
- id: route_decision
  type: condition
  depends_on: [approve_step]
  conditions:
    - when: "{{steps.approve_step.outputs.result}} == approved"
      then: [next_step_a]
    - when: "{{steps.some_step.structured_output.priority}} == P0"
      then: [urgent_path]
    - default: [normal_path]
```

**后端改动：**

1. `workflow_service.go` — StepDef 增加：
```go
Conditions []ConditionDef `yaml:"conditions,omitempty" json:"conditions,omitempty"`
```

```go
type ConditionDef struct {
    When    string   `yaml:"when,omitempty" json:"when,omitempty"`       // 条件表达式
    Default bool     `yaml:"default,omitempty" json:"default,omitempty"` // 是否为默认分支
    Then    []string `yaml:"then" json:"then"`                          // 满足条件时触发的步骤 ID 列表
}
```

2. `workflow_engine.go` — 引擎执行 condition 步骤时：
   - 渲染所有 `when` 表达式中的模板变量
   - 按顺序评估条件（支持 `==`, `!=`, `contains`, `>`, `<` 等基本运算符）
   - 第一个匹配的条件，触发其 `then` 列表中的步骤
   - 无匹配则走 `default`
   - condition 步骤本身不需要 agent/prompt

3. 验证逻辑 — `validateWorkflow` 需要豁免 condition 类型的 agent/prompt 要求（和 approval 一样），并校验 then 引用的步骤存在。

**注意：** condition 节点的 `then` 步骤不通过 `depends_on` 关联，而是动态触发。引擎需要处理好 DAG 中这种条件边的关系——condition 的 then 步骤应该只在条件满足时被触发，其他时候 skip。

### 任务4：审批人动态指定（P2）

**需求：** 审批人支持模板变量，运行时解析。

**YAML 设计：**
```yaml
approval:
  approvers:
    - type: feishu_user
      open_id: "{{steps.assign.structured_output.reviewer_id}}"
    - type: role
      role: project_owner   # 按角色查找
```

**后端改动：**

1. `workflow_engine.go` — executeApprovalStep 中，对 approvers 的 open_id 做模板变量渲染：
```go
for i, a := range approvalDef.Approvers {
    a.OpenID = RenderPrompt(a.OpenID, inputs, stepOutputs)
    approvers[i] = a
}
```

2. 支持 `type: role` 类型的审批人 — 需要查询用户表按 role 匹配：
   - `entity.go` — User 已有 Role 字段
   - `approval_service.go` — 解析 role 类型时，查 Users 表找到匹配 role 的用户，取其 feishu_open_id

### 任务5：多级审批链（P2）

**需求：** 支持串联多个审批节点，YAML 里直接写多个 approval 步骤就行（已天然支持），但需要优化体验。

**改动：**

1. 前端 `Approvals.tsx` — 审批列表页增加显示"审批链"标记：
   - 如果同一个 run 有多个 approval 步骤，显示"第 X/N 级审批"
   - 显示当前等待的是哪一级

2. 前端 `ApprovalDetail.tsx` — 审批详情展示审批链上下文：
   - 显示前序审批的结果和意见
   - 方便后续审批人了解之前的决策

3. `approval_service.go` — `GetDetail` 返回同 run 下所有审批步骤的状态列表，供前端展示。

### 任务6：更新文档

完成以上所有任务后，更新 `docs/workflow-reference.md`：
- 新增 condition 节点文档
- 更新 approval 节点文档（modified_output、动态审批人）
- 新增审批链使用示例

## 编译部署

```bash
# 后端
cd /home/claw/.openclaw/workspace/agent-control-panel
go build -o bin/acp ./cmd/acp/

# 前端
cd acp-web && npm run build
cp -r dist/* ../web/

# 部署
kill $(pgrep -f 'bin/acp') 2>/dev/null; sleep 1; nohup ./bin/acp > acp.log 2>&1 &
```

## 验证要求

1. 审批通过+修改输出：修改后的 structured_output 要能被后续步骤的 `{{steps.xxx.structured_output}}` 正确引用
2. 条件分支：创建一个包含 condition 节点的测试工作流，验证条件匹配和默认分支
3. 动态审批人：用模板变量指定 open_id，验证运行时能正确解析
4. 多级审批：创建两个串联 approval 步骤的工作流，验证链式审批体验
5. 每个任务完成后 commit + push
