# ACP 流程审计系统 PRD

## 背景

流程审计是ACP的核心模块。每条工作流跑完后自动生成审计数据，用户可以按任意时间跨度聚合分析，发现趋势、定位瓶颈、驱动agent和流程的持续优化。

## 设计原则

1. **审计数据 = 一等公民**：不是附属功能，是系统核心
2. **单次Run审计自动生成**：流程跑完即产出，零人工介入
3. **时间跨度动态可调**：周期性分析 = 对已有审计数据的聚合查询，不需要额外的定时任务
4. **闭环驱动**：审计发现 → 升级建议 → 应用 → 效果验证 → 再审计

---

## 一、数据模型

### 1.1 RunAudit（单次运行审计）

每条工作流Run完成时（completed/failed/stopped），自动生成一条RunAudit记录。

```go
type RunAudit struct {
    ID              string    `json:"id"`
    RunID           string    `json:"run_id"`           // 关联的workflow run
    WorkflowID      string    `json:"workflow_id"`      // 工作流模板ID
    WorkflowName    string    `json:"workflow_name"`    // 冗余，方便查询
    Status          string    `json:"status"`           // completed / failed / stopped
    
    // 时间指标
    TotalDurationMs int64     `json:"total_duration_ms"` // 总耗时
    StartedAt       time.Time `json:"started_at"`
    CompletedAt     time.Time `json:"completed_at"`
    
    // 步骤统计
    TotalSteps      int       `json:"total_steps"`       // 总步骤数
    CompletedSteps  int       `json:"completed_steps"`   // 完成的步骤数
    FailedSteps     int       `json:"failed_steps"`      // 失败的步骤数
    SkippedSteps    int       `json:"skipped_steps"`     // 跳过的步骤数（condition分支）
    
    // 质量指标
    GatePassRate    float64   `json:"gate_pass_rate"`    // Gate一次通过率 (0-1)
    GateRetries     int       `json:"gate_retries"`      // Gate总重试次数
    ApprovalCount   int       `json:"approval_count"`    // 审批步骤数
    ApprovalRejects int       `json:"approval_rejects"`  // 审批驳回次数
    ModifiedOutputs int       `json:"modified_outputs"`  // 审批时修改output的次数
    EscalationCount int       `json:"escalation_count"`  // 升级次数
    LessonsHit      int       `json:"lessons_hit"`       // 触发的经验教训数
    
    // 步骤级明细
    StepAudits      []StepAudit `json:"step_audits"`     // 每步的审计明细
    
    // 综合评分
    Score           float64   `json:"score"`             // 0-100 综合评分
    
    CreatedAt       time.Time `json:"created_at"`
}
```

### 1.2 StepAudit（步骤级审计明细）

嵌入在RunAudit中，记录每个步骤的执行细节。

```go
type StepAudit struct {
    StepName        string  `json:"step_name"`
    StepType        string  `json:"step_type"`          // agent / gate / approval / condition
    AgentID         string  `json:"agent_id,omitempty"` // 执行的agent
    DurationMs      int64   `json:"duration_ms"`        // 该步骤耗时
    Status          string  `json:"status"`             // completed / failed / skipped
    Retries         int     `json:"retries"`            // 重试次数
    
    // Gate相关
    GatePassed      *bool   `json:"gate_passed,omitempty"`      // Gate是否一次通过
    GateAttempts    int     `json:"gate_attempts,omitempty"`     // Gate尝试次数
    GateAction      string  `json:"gate_action,omitempty"`      // 最终Gate动作 (pass/restart/escalate/abort)
    
    // Approval相关
    ApprovalResult  string  `json:"approval_result,omitempty"`  // approved / rejected
    ApproverID      string  `json:"approver_id,omitempty"`      // 审批人
    ApprovalWaitMs  int64   `json:"approval_wait_ms,omitempty"` // 等待审批的时间
    OutputModified  bool    `json:"output_modified,omitempty"`  // 审批时是否修改了output
    
    // Condition相关
    ConditionResult string  `json:"condition_result,omitempty"` // 走了哪个分支
}
```

### 1.3 综合评分算法

Run完成时自动计算 Score（0-100）：

```
Score = 100
- (failed_steps / total_steps) × 40        // 失败扣分，权重最大
- gate_retries × 5                          // 每次Gate重试扣5分
- approval_rejects × 8                     // 每次审批驳回扣8分
- escalation_count × 10                    // 每次升级扣10分
- time_penalty                              // 超过同类工作流平均耗时的部分，每超50%扣10分
+ lessons_hit × 2                           // 命中经验教训加分（说明系统在学习）

Score = max(0, min(100, Score))
```

评分等级：
- 90-100: 优秀 🟢
- 70-89: 良好 🟡  
- 50-69: 需改进 🟠
- 0-49: 问题严重 🔴

---

## 二、审计数据生成

### 2.1 触发时机

在 `workflow_engine.go` 的 `completeRun()` 中，Run进入终态（completed/failed/stopped）时调用 `generateRunAudit(run)`。

### 2.2 数据来源

| 指标 | 数据来源 |
|------|---------|
| 步骤耗时 | `StepExecution.started_at` / `completed_at` |
| Gate重试 | 现有 `AgentPerformance` 记录（metric_type=gate_failure） |
| 审批结果 | `Approval` 表的action记录 |
| 修改output | `ApprovalAction.modified_output` 是否非空 |
| Escalation | `AgentPerformance`（metric_type=escalation） |
| Lessons | 步骤执行时查询lessons库的命中记录 |

### 2.3 存储

RunAudit作为独立表存储（不是JSON塞在Run里），方便按时间范围高效查询聚合。StepAudits以JSON字段存在RunAudit行内（SQLite JSON支持足够）。

---

## 三、聚合分析（动态时间跨度）

### 3.1 API设计

```
GET /api/audit/summary?from=2026-02-01&to=2026-02-21&workflow_id=xxx&agent_id=xxx
```

参数：
- `from` / `to`：时间范围（必填）
- `workflow_id`：按工作流筛选（可选）
- `agent_id`：按agent筛选（可选）
- `group_by`：聚合维度 — `day` / `week` / `workflow` / `agent`（可选，默认不分组）

### 3.2 聚合响应

```json
{
  "period": { "from": "2026-02-01", "to": "2026-02-21" },
  "total_runs": 47,
  "completed_runs": 42,
  "failed_runs": 5,
  "avg_score": 78.5,
  "score_trend": [72, 75, 78, 81, 78],  // 按group_by分段的趋势
  
  "metrics": {
    "avg_duration_ms": 185000,
    "avg_gate_pass_rate": 0.82,
    "total_gate_retries": 34,
    "total_approval_rejects": 8,
    "total_escalations": 3,
    "avg_modified_output_rate": 0.15
  },
  
  "bottlenecks": [
    {
      "step_name": "develop_code",
      "avg_duration_ms": 95000,
      "duration_pct": 51.3,
      "avg_retries": 1.2,
      "issue": "占总耗时51%，重试率高"
    }
  ],
  
  "agent_rankings": [
    {
      "agent_id": "pm",
      "runs": 15,
      "avg_score": 85,
      "gate_pass_rate": 0.9,
      "trend": "improving"
    }
  ],
  
  "degradation_alerts": [
    {
      "metric": "gate_pass_rate",
      "agent_id": "alice",
      "current": 0.65,
      "previous": 0.85,
      "change_pct": -23.5,
      "severity": "warning"
    }
  ],
  
  "upgrade_effectiveness": [
    {
      "proposal_id": "xxx",
      "applied_at": "2026-02-15",
      "metric": "gate_pass_rate",
      "before": 0.7,
      "after": 0.88,
      "verdict": "effective"
    }
  ]
}
```

### 3.3 退化预警逻辑

对比当前时间段 vs 上一个同等长度时间段：
- Gate通过率下降 > 15% → warning
- Gate通过率下降 > 30% → critical
- 平均评分下降 > 10分 → warning
- 失败率上升 > 20% → critical
- 某步骤平均耗时增长 > 100% → warning

---

## 四、前端页面

### 4.1 审计总览页（/audit）

**顶部：时间选择器**
- 预设：今天 / 本周 / 本月 / 自定义范围
- 筛选：工作流 / Agent

**核心指标卡片（4个）：**
- 总运行数 & 成功率
- 平均评分 & 趋势箭头（↑↓）
- Gate一次通过率
- 平均耗时

**趋势图：**
- 折线图：评分趋势（按天/周）
- 柱状图：运行数量分布

**瓶颈分析：**
- 表格：步骤名 | 平均耗时 | 占比 | 重试率 | 问题描述
- 按耗时占比降序

**退化预警：**
- 卡片列表：哪个指标在退化，严重程度，对比数据

### 4.2 单次Run审计详情

在现有RunDetail页面增加"审计"Tab：
- 评分（大数字 + 颜色等级）
- 步骤时间线（甘特图式，每步一条横条，颜色表示状态）
- 步骤明细表格：名称 | 类型 | Agent | 耗时 | 重试 | Gate结果 | 审批结果
- 与同类Run的对比（本次 vs 平均）

### 4.3 Agent健康度页面

在现有AgentPerformance页面增强：
- 每个Agent一个健康度评分（综合gate通过率、完成率、耗时）
- 时间范围内的能力雷达图
- 趋势对比：本周 vs 上周

---

## 五、MCP工具扩展

新增Plugin工具供Lyra使用：

```
acp_audit_summary    — 获取时间范围内的审计聚合数据
acp_audit_run        — 获取单次Run的审计详情
acp_audit_alerts     — 获取当前的退化预警
```

---

## 六、与现有模块的关系

```
Performance采集（第1层，已有）
    ↓ 原始数据
RunAudit生成（第2层，本PRD）
    ↓ 单次审计
聚合分析（第3层，本PRD）
    ↓ 趋势洞察
UpgradeProposal（已有）
    ↓ 自动建议
审批应用 → 效果验证（闭环）
```

现有的 `AgentPerformance` 表保留，作为原始数据源。`RunAudit` 是在其上的聚合计算结果，避免每次查询都重新遍历原始记录。

---

## 七、实现优先级

**P0（核心）：**
1. RunAudit数据模型 + 自动生成
2. 评分算法
3. 聚合查询API
4. 审计总览前端页面

**P1（增强）：**
5. 退化预警逻辑
6. 单次Run审计详情Tab
7. MCP工具（acp_audit_*）

**P2（优化）：**
8. Agent健康度雷达图
9. 步骤时间线甘特图
10. 升级效果自动验证
