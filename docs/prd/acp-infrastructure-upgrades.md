# ACP 基础设施升级 PRD

## 背景

ACP问题分三层：①系统基础设施 ②流程设计 ③Agent能力。第②③层已有审计闭环驱动持续优化。第①层是地基，需要一次性解决。

本PRD覆盖两个基础设施改造：
1. MCP工具响应限流 + 引擎自动注入上下文
2. 凭证金库（Credential Vault）

---

## 一、MCP工具响应限流 + 引擎上下文注入

### 1.1 问题

- `acp_workflow_run_detail` 返回所有步骤的完整output，数据量可达数万字符
- Agent拿到后传给LLM，导致上下文溢出
- 任何MCP工具都可能返回超大响应，没有通用保护机制

### 1.2 方案

#### A. 工具侧限流（防御层）

**所有ACP MCP工具的响应体增加全局大小上限：**

```go
const MaxToolResponseChars = 8000 // 默认8000字符上限

func truncateToolResponse(response string) string {
    if len([]rune(response)) <= MaxToolResponseChars {
        return response
    }
    return string([]rune(response)[:MaxToolResponseChars]) + 
        "\n\n[⚠️ 响应已截断，原始长度" + strconv.Itoa(len([]rune(response))) + "字符。" +
        "请使用 acp_get_step_output 查看单个步骤的完整输出]"
}
```

**`acp_workflow_run_detail` 专项优化：**
- 新增参数 `summary_only`（bool，默认false）
  - `true`: 每步只返回 status + duration + structured_output keys（不含values），总响应<2KB
  - `false`: 返回完整数据，但受全局truncate保护
- 新增参数 `steps`（string，逗号分隔）
  - 只返回指定步骤的output，如 `steps=discover_modules,inspect_module_1`

**新增工具 `acp_get_step_output`：**
- 参数：`run_id`, `step_name`, `output_type`（full/structured/summary）
- 只返回单个步骤的output，精确控制数据量

#### B. 引擎侧上下文注入（根本解法）

**工作流引擎在派发步骤给Agent时，自动注入前序步骤的output：**

根据步骤的 `depends_on` 声明，引擎收集所有依赖步骤的 `structured_output`，注入到当前步骤的 `agent_context` 中。

```go
// workflow_engine.go - executeAgentStep()
func (e *Engine) buildAgentContext(step StepDef, stepOutputs map[string]string) string {
    var ctx strings.Builder
    
    // 自动注入依赖步骤的structured_output
    for _, dep := range step.DependsOn {
        if output, ok := stepOutputs[dep]; ok {
            // 只注入structured_output（体积小）
            var parsed map[string]interface{}
            if json.Unmarshal([]byte(output), &parsed) == nil {
                if so, exists := parsed["structured_output"]; exists {
                    soJSON, _ := json.Marshal(so)
                    ctx.WriteString(fmt.Sprintf("\n## %s 的输出\n```json\n%s\n```\n", dep, string(soJSON)))
                }
            }
        }
    }
    
    // 追加用户定义的context
    for _, c := range step.Context {
        rendered := RenderPrompt(c, inputs, stepOutputs)
        ctx.WriteString(rendered)
    }
    
    // 最终限流保护
    return truncateContext(ctx.String(), MaxAgentContextChars)
}
```

**`MaxAgentContextChars = 12000`** — Agent接收的上下文硬上限，超过时智能截断（保留structured_output，截断raw text）。

#### C. 步骤YAML增加 `inject_outputs` 选项

```yaml
- id: consolidate
  type: agent
  agent: pm
  depends_on: [step_a, step_b, step_c]
  inject_outputs: structured  # auto | structured | full | none
  prompt: "汇总以上步骤的结果..."
```

- `auto`（默认）: 引擎自动注入 depends_on 步骤的 structured_output
- `structured`: 只注入 structured_output
- `full`: 注入完整 output（受限流保护）
- `none`: 不注入，agent自己通过工具获取

### 1.3 实现清单

1. Plugin: 所有工具响应加 `truncateToolResponse()` 保护
2. Plugin: `acp_workflow_run_detail` 加 `summary_only` 和 `steps` 参数
3. Plugin: 新增 `acp_get_step_output` 工具
4. Engine: `buildAgentContext()` 自动注入依赖步骤output
5. Engine: 支持 `inject_outputs` YAML配置
6. Engine: `MaxAgentContextChars` 硬上限保护

---

## 二、凭证金库（Credential Vault）

### 2.1 问题

- Agent需要访问PLM、ERP等内部系统，但这些系统需要认证
- 每个系统认证方式不同（Token、OAuth、Cookie...）
- 在工作流prompt里硬编码token不安全、不可维护
- 新增系统时需要修改所有相关工作流

### 2.2 数据模型

```go
type Credential struct {
    ID          string    `json:"id" gorm:"primaryKey"`
    Name        string    `json:"name"`         // 显示名称，如 "PLM系统"
    Slug        string    `json:"slug"`         // 引用标识，如 "plm"
    Type        string    `json:"type"`         // api_token / basic_auth / bearer / custom_header / oauth_client
    BaseURL     string    `json:"base_url"`     // 目标系统地址
    Config      string    `json:"config"`       // JSON加密存储，内容按type不同
    Description string    `json:"description"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

**Config按type的结构：**

```json
// api_token
{"token": "xxx", "header": "Authorization", "prefix": "Bearer"}

// basic_auth
{"username": "agent", "password": "xxx"}

// custom_header
{"headers": {"X-API-Key": "xxx", "X-Agent-ID": "lyra"}}

// oauth_client (client_credentials grant)
{"client_id": "xxx", "client_secret": "xxx", "token_url": "https://...", "scopes": ["read"]}
```

### 2.3 工作流YAML集成

```yaml
steps:
  - id: inspect_module
    type: agent
    agent: ux
    credentials: [plm]  # 引用凭证slug
    prompt: "巡检{{module_name}}模块的UI体验..."
```

引擎在派发任务时自动注入认证信息到agent_context：

```
## 系统访问凭证

### PLM系统
- 地址: http://43.134.86.237:8080
- 认证方式: Bearer Token
- 请求时在Header添加: Authorization: Bearer xxx
- 请使用API方式访问，不要通过浏览器登录
```

### 2.4 安全设计

- Config字段**AES加密存储**（密钥从环境变量 `ACP_ENCRYPTION_KEY` 读取）
- API返回凭证列表时**不返回Config内容**，只返回name/slug/type/base_url
- 只有引擎内部派发任务时才解密Config
- 前端编辑凭证时，已保存的secret显示为 `***` ，不回显

### 2.5 前端页面

**Settings → 凭证管理**

- 凭证列表：名称 | 类型 | 目标地址 | 描述 | 操作（编辑/删除）
- 新增/编辑弹窗：表单根据type动态切换字段
  - api_token: Token输入框 + Header名 + Prefix
  - basic_auth: 用户名 + 密码
  - custom_header: Key-Value对列表
- 测试连接按钮（可选）：用凭证请求BaseURL，验证认证是否有效

### 2.6 API

```
GET    /api/credentials          — 列表（不含secret）
POST   /api/credentials          — 创建
PUT    /api/credentials/:id      — 更新
DELETE /api/credentials/:id      — 删除
POST   /api/credentials/:id/test — 测试连接（可选）
```

### 2.7 实现清单

1. Entity: Credential模型 + 加密/解密函数
2. Repository: CRUD
3. Handler: REST API（列表不返回secret）
4. Service: 凭证解密 + 注入逻辑
5. Engine: 步骤派发时读取credentials声明，解密注入agent_context
6. 前端: Settings页面新增"凭证管理"Tab
7. YAML验证: credentials引用的slug必须存在

### 2.8 PLM Token认证（配套改造）

ACP凭证金库就绪后，PLM后端需要支持API Token认证：

```go
// PLM middleware - 在飞书SSO之前增加Token认证
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 优先检查API Token
        token := c.GetHeader("Authorization")
        if strings.HasPrefix(token, "Bearer ") {
            apiToken := strings.TrimPrefix(token, "Bearer ")
            if validateAPIToken(apiToken) {
                c.Set("user_id", "agent_service_account")
                c.Set("user_name", "Agent")
                c.Next()
                return
            }
        }
        // 回退到飞书SSO JWT验证
        // ...existing logic...
    }
}
```

在PLM的.env中配置: `PLM_API_TOKEN=<随机生成的安全token>`

---

## 三、实现优先级

**P0（解决当前blocking问题）：**
1. MCP工具响应限流（truncateToolResponse）
2. `acp_workflow_run_detail` 加 summary_only 参数
3. 新增 `acp_get_step_output` 工具
4. 引擎自动注入依赖步骤output
5. PLM API Token认证
6. 凭证金库基础功能（Entity + API + 引擎注入）

**P1（完善体验）：**
7. 凭证前端管理页面
8. 凭证加密存储
9. `inject_outputs` YAML配置支持
10. 凭证测试连接功能
