# PRD: ACP v2.2 — 审批系统 + 飞书集成

## 背景

ACP工作流中需要人工审批节点（如：agent完成开发后，CEO审批是否部署）。审批通知通过飞书应用发送给飞书用户，用户可在飞书中直接审批。

## 飞书应用信息

- 应用名称：ACP
- App ID：`cli_a9122d58b5f8dcca`
- App Secret：`xSdWhANrlU7bhultnsiuTdfPSWnAUf3g`

## 核心功能

### 1. 飞书登录（OAuth2）

ACP前端支持飞书扫码/点击登录，获取用户open_id，绑定到ACP用户系统。

**流程：**
```
用户点击"飞书登录" → 跳转飞书OAuth授权页 → 用户授权 
→ 回调ACP后端 → 获取access_token + open_id + 用户信息
→ 创建/关联ACP用户 → 返回JWT → 登录成功
```

**API：**
```
GET  /api/auth/feishu/login     → 返回飞书OAuth授权URL
GET  /api/auth/feishu/callback  → 飞书回调，换token，返回ACP JWT
GET  /api/auth/me               → 当前用户信息（含feishu_open_id）
```

**数据模型扩展：**
```go
// User 用户表（新增）
type User struct {
    ID           string    `gorm:"primaryKey" json:"id"`
    Username     string    `gorm:"uniqueIndex;not null" json:"username"`
    Name         string    `gorm:"not null" json:"name"`           // 显示名
    Avatar       string    `gorm:"default:''" json:"avatar"`       // 头像URL
    FeishuOpenID string    `gorm:"uniqueIndex" json:"feishu_open_id"` // 飞书open_id
    FeishuUnionID string   `json:"feishu_union_id"`
    Role         string    `gorm:"default:'user'" json:"role"`     // admin/user
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}
```

### 2. 审批系统

#### 2.1 工作流DSL中的审批步骤

```yaml
steps:
  - id: develop
    agent: alice
    prompt: "开发xxx功能"
    
  - id: review_deploy
    type: approval                    # 新step类型：审批
    depends_on: [develop]
    approval:
      title: "部署审批"
      description: "Alice已完成开发，是否批准部署到生产环境？"
      approvers:                      # 审批人列表
        - type: feishu_user           # 飞书用户
          open_id: "ou_5b159fc157d4042f1e8088b1ffebb2da"  # 泽斌
        - type: agent                 # Agent审批
          agent_id: "main"            # Lyra
      strategy: any                   # any=任一通过即可, all=全部通过
      timeout: 24h                    # 审批超时
      on_timeout: abort               # abort/skip/auto_approve
      notify:
        feishu: true                  # 通过飞书发送审批通知
      context:                        # 审批时展示的上下文
        - "{{steps.develop.output}}"  # 上一步的产出
        
  - id: deploy
    depends_on: [review_deploy]       # 审批通过后才执行
    agent: alice
    prompt: "部署到生产环境"
```

#### 2.2 数据模型

```go
// Approval 审批记录
type Approval struct {
    ID           string     `gorm:"primaryKey" json:"id"`
    RunID        string     `gorm:"index;not null" json:"run_id"`        // 关联工作流run
    StepID       string     `gorm:"not null" json:"step_id"`             // 关联step
    Title        string     `gorm:"not null" json:"title"`
    Description  string     `gorm:"type:text" json:"description"`
    Context      string     `gorm:"type:text" json:"context"`            // 审批上下文（上游output等）
    Status       string     `gorm:"not null;default:'pending'" json:"status"` // pending/approved/rejected/timeout
    Strategy     string     `gorm:"default:'any'" json:"strategy"`       // any/all
    Timeout      string     `gorm:"default:'24h'" json:"timeout"`
    CreatedAt    time.Time  `json:"created_at"`
    UpdatedAt    time.Time  `json:"updated_at"`
    CompletedAt  *time.Time `json:"completed_at"`
}

// ApprovalAction 审批操作记录（每个审批人的操作）
type ApprovalAction struct {
    ID          string     `gorm:"primaryKey" json:"id"`
    ApprovalID  string     `gorm:"index;not null" json:"approval_id"`
    ApproverType string    `gorm:"not null" json:"approver_type"`   // feishu_user/agent
    ApproverID   string    `gorm:"not null" json:"approver_id"`     // open_id 或 agent_id
    ApproverName string    `gorm:"default:''" json:"approver_name"`
    Action       string    `gorm:"default:'pending'" json:"action"` // pending/approved/rejected
    Comment      string    `gorm:"type:text" json:"comment"`        // 审批意见
    ActedAt      *time.Time `json:"acted_at"`
    CreatedAt    time.Time  `json:"created_at"`
}
```

#### 2.3 审批API

```
GET    /api/approvals                          → 审批列表（可筛选status）
GET    /api/approvals/:id                      → 审批详情
POST   /api/approvals/:id/approve              → 通过（body: {comment?}）
POST   /api/approvals/:id/reject               → 驳回（body: {comment?}）

# 飞书回调（审批卡片点击后回调）
POST   /api/webhooks/feishu/approval           → 飞书交互卡片回调
```

#### 2.4 飞书通知

审批创建时，通过飞书ACP应用向审批人发送**交互式卡片消息**：

```json
{
  "msg_type": "interactive",
  "card": {
    "header": {
      "title": {"tag": "plain_text", "content": "🔔 ACP审批：部署审批"},
      "template": "orange"
    },
    "elements": [
      {
        "tag": "markdown",
        "content": "**工作流：** 新功能开发\n**步骤：** Alice已完成开发\n\n**详情：**\n开发产出摘要..."
      },
      {
        "tag": "action",
        "actions": [
          {
            "tag": "button",
            "text": {"tag": "plain_text", "content": "✅ 通过"},
            "type": "primary",
            "value": {"action": "approve", "approval_id": "xxx"}
          },
          {
            "tag": "button", 
            "text": {"tag": "plain_text", "content": "❌ 驳回"},
            "type": "danger",
            "value": {"action": "reject", "approval_id": "xxx"}
          },
          {
            "tag": "button",
            "text": {"tag": "plain_text", "content": "📋 查看详情"},
            "type": "default",
            "url": "http://43.134.86.237:3001/approvals/xxx"
          }
        ]
      }
    ]
  }
}
```

用户在飞书点击"通过"或"驳回"→ 飞书回调ACP → 更新审批状态 → 工作流继续/终止。

#### 2.5 Agent审批

当审批人是Agent时，通过sessions_send发送审批请求，agent回复"approve"或"reject"：

```
发送给agent: "[ACP审批请求] 标题：部署审批\n描述：...\n请回复 approve 或 reject"
agent回复包含"approve" → 标记通过
agent回复包含"reject" → 标记驳回
```

### 3. 前端页面

#### 3.1 审批列表页 `/approvals`
- 列表展示所有审批，状态标签（待审批/已通过/已驳回/已超时）
- 筛选：按状态、按工作流
- 待审批的排在最前面

#### 3.2 审批详情页 `/approvals/:id`
- 审批标题、描述、上下文
- 审批人列表及各自状态
- 通过/驳回按钮（当前用户是审批人时显示）
- 评论输入框
- 关联的工作流step链接

#### 3.3 登录页面更新
- 现有密码登录保留
- 新增"飞书登录"按钮

## 开发任务

| # | 任务 | 涉及文件 |
|---|------|---------|
| 1 | User数据模型 + 飞书OAuth2登录 | entity/entity.go, handler/auth_handler.go, service/feishu_service.go |
| 2 | Approval + ApprovalAction数据模型 | entity/entity.go |
| 3 | 审批Repository + Service | repository/approval_repository.go, service/approval_service.go |
| 4 | 审批Handler（API） | handler/approval_handler.go, handler/handler.go |
| 5 | 工作流引擎支持approval类型step | service/workflow_engine.go |
| 6 | 飞书消息发送（交互式卡片） | service/feishu_service.go |
| 7 | 飞书卡片回调处理 | handler/feishu_webhook_handler.go |
| 8 | Agent审批（sessions_send + 回复检测） | service/approval_service.go |
| 9 | 前端：飞书登录按钮 + OAuth回调页 | acp-web/ |
| 10 | 前端：审批列表页 + 详情页 | acp-web/ |
| 11 | Plugin新增审批相关tool | plugin/index.ts |

## 技术要点

- 飞书OAuth2文档：https://open.feishu.cn/document/common-capabilities/sso/web-application-sso/web-app-overview
- 飞书发消息API：https://open.feishu.cn/document/server-docs/im-v1/message/create
- 飞书交互卡片：https://open.feishu.cn/document/common-capabilities/message-card/message-cards-content
- 卡片回调：需要在飞书开放平台配置事件订阅URL

## 配置

ACP的.env或配置文件新增：
```
ACP_FEISHU_APP_ID=cli_a9122d58b5f8dcca
ACP_FEISHU_APP_SECRET=xSdWhANrlU7bhultnsiuTdfPSWnAUf3g
ACP_FEISHU_REDIRECT_URI=http://43.134.86.237:3001/api/auth/feishu/callback
ACP_BASE_URL=http://43.134.86.237:3001
```

## 变更记录

| 日期 | 变更 | 原因 |
|------|------|------|
| 2026-02-21 | 初版 | 泽斌提出审批系统需求 |
