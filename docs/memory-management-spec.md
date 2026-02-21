# Agent记忆管理规范 v1.0（草稿）

> 这份规范定义了BitFantasy所有AI Agent的记忆管理标准。
> 待泽斌review确认后，写入各agent的AGENTS.md。

---

## 一、问题陈述

当前记忆系统的问题：
1. Agent不知道该往哪里写——MEMORY.md vs daily log vs TOOLS.md边界模糊
2. 子agent（PM/UX/Alice）完全没有记忆管理规则，memory为空
3. MEMORY.md只增不删，不控制大小，迟早被截断
4. 没有人知道agent记得什么、不记得什么
5. compaction发生时大量上下文丢失，没有pre-compaction记忆保存习惯
6. ACP工作流调用后，agent不会主动记录经验

---

## 二、记忆层次定义

### L1: 身份层（每次session自动加载，永远在context里）

| 文件 | 内容 | 大小上限 | 更新频率 |
|------|------|---------|---------|
| AGENTS.md | 行为规则、工作流程、协作方式 | 8KB | 低频，规则变动时 |
| SOUL.md | 人格、语气、边界 | 2KB | 极低频 |
| USER.md | 用户信息 | 2KB | 低频 |
| IDENTITY.md | agent名字和角色 | 1KB | 极低频 |
| TOOLS.md | 工具配置、项目结构、技术笔记 | 8KB | 中频，新工具/新配置时 |

**原则：这些文件每次session都全量加载，消耗context tokens。必须精简。**

### L2: 知识层（每次主session加载，需要控制大小）

| 文件 | 内容 | 大小上限 | 更新频率 |
|------|------|---------|---------|
| MEMORY.md | 精华记忆：重要决策、关键人物、核心教训、项目里程碑 | 5KB（~150行） | 每周整理1次 |

**MEMORY.md的定位：相当于人脑的"工作记忆 + 重要回忆"。**

不该放的：
- ❌ 每日流水账（放daily log）
- ❌ 代码细节/commit hash（放TOOLS.md或daily log）
- ❌ 完整的PRD/设计文档（放docs/目录）
- ❌ 临时状态（放daily log）

该放的：
- ✅ 泽斌说过的重要决策和原则
- ✅ 项目里程碑和状态总结（不是细节）
- ✅ 反复犯过的错误和教训
- ✅ 关键人物关系和协作模式
- ✅ 长期目标和优先级

### L3: 日志层（不自动加载，按需检索）

| 文件 | 内容 | 大小上限 | 更新频率 |
|------|------|---------|---------|
| memory/YYYY-MM-DD.md | 当天工作日志 | 无硬限（建议<500行） | 实时追加 |

**写入时机：**
- 完成一个重要任务后
- 做了一个关键决策后
- 收到新的规则/要求后
- 遇到问题并解决后
- ACP工作流步骤完成后

**格式要求：**
```markdown
## [时间段] 事件标题

### 做了什么
- 具体动作

### 决策/结论
- 为什么这么做

### 待办/后续
- 下一步是什么
```

### L4: ACP经验层（系统强制采集，结构化存储）

| 存储位置 | 内容 | 管理方式 |
|---------|------|---------|
| ACP SQLite (step_lessons) | 每次工作流步骤的执行经验 | acp_complete_step自动采集 |
| ACP condensed（待建设） | 浓缩后的经验 | 系统自动合并 |

**这一层不依赖agent自觉——系统强制要求，422打回。**

---

## 三、写入规则——什么放哪里

| 信息类型 | 举例 | 应该放 | 不应该放 |
|---------|------|--------|---------|
| CEO的战略决策 | "ACP定位是COO工具" | MEMORY.md | daily log（太容易被淹没） |
| 技术配置 | "PLM API Token是xxx" | TOOLS.md | MEMORY.md（不是记忆，是配置） |
| 今天修了什么bug | "修了BOM保存400错误" | memory/日期.md | MEMORY.md（太细节） |
| 反复犯的错误 | "不要直接调CC，走PM流程" | MEMORY.md + AGENTS.md | 只放daily log（会忘） |
| 项目状态 | "ACP v2.5已发布" | MEMORY.md（简要） | AGENTS.md（不是规则） |
| commit hash | "5d6e75c" | memory/日期.md | MEMORY.md（太细节） |
| 新学到的工具用法 | "tmux load-buffer解决CJK问题" | TOOLS.md | MEMORY.md |
| 协作方式变更 | "CC hook现在通知Lyra+泽斌" | MEMORY.md + TOOLS.md | 只放daily log |
| ACP任务执行经验 | "verify必须截图" | ACP lessons（自动）| 不需要手动记 |

**黄金规则：如果一条信息你希望3天后还能想起来，它必须在MEMORY.md或AGENTS.md里。daily log 3天后基本不会被主动检索到。**

---

## 四、MEMORY.md结构模板

```markdown
# MEMORY.md - [Agent名字] 长期记忆

> 上次整理：YYYY-MM-DD | 大小：约XXX行
> 整理原则：只保留3天后还有用的信息

## 核心规则（从泽斌/Lyra获得的指令）
- [规则1]
- [规则2]

## 项目状态（每周更新）
### [项目名]
- 当前阶段：
- 关键里程碑：
- 未解决问题：

## 重要教训
- [日期] [教训内容] — [来源/背景]

## 关键人物和协作
- [人物]: [角色], [协作要点]

## 近期优先级
1. [最重要的事]
2. [第二重要]
3. [第三]
```

---

## 五、记忆维护机制

### 5.1 日常维护（每次session）

**Session开始时：**
1. 读MEMORY.md（自动注入）
2. 如果是主session，检查memory/今天.md和memory/昨天.md

**Session中：**
3. 发生重要事件时，立即写入memory/今天.md
4. 不要攒到最后再写——compaction可能随时发生

**Session结束前（或compaction触发时）：**
5. 检查本次session是否有值得长期记住的信息
6. 如果有，更新MEMORY.md

### 5.2 定期整理（每周1次）

通过heartbeat或cron触发：
1. 回顾过去7天的daily log
2. 提取值得长期保留的信息→更新MEMORY.md
3. 检查MEMORY.md大小，如果>5KB，精简过时内容
4. 检查AGENTS.md，如果有反复出现的教训，考虑写入规则

### 5.3 ACP工作流后的记忆

Agent完成ACP工作流步骤后：
1. lessons字段由系统强制采集（不需要手动）
2. 如果这次任务学到了**通用经验**（不只是这个step），写入memory/日期.md
3. 如果是**必须改变行为的教训**，写入AGENTS.md

---

## 六、各Agent的差异化配置

### Lyra（main）
- 完整的MEMORY.md + daily log
- 负责整个公司视角的知识
- 额外维护：PROJECT_STATUS.md（项目全局状态）

### PM
- MEMORY.md重点记录：PRD经验、需求拆解教训、和CC/UX的协作模式
- daily log记录每次评审的发现
- 不需要记技术细节

### UX
- MEMORY.md重点记录：设计规范变更、评审标准演化、常见UI问题模式
- daily log记录每次验收的截图和问题
- 维护design-tokens相关知识

### Alice（Dev）
- MEMORY.md重点记录：代码架构决策、编译部署经验、常踩的坑
- daily log记录每次开发任务
- TOOLS.md重点维护：项目结构、编译命令、部署流程

---

## 七、与ACP Lessons的关系

```
ACP Lessons（L4）          Agent Memory（L2+L3）
系统强制采集               Agent主动记录
结构化、可查询              自由文本、语义搜索
跨agent共享                每个agent私有
绑定(workflow, step)        自由组织
用途：流程优化              用途：agent个人成长
```

**不冲突，互补：**
- ACP lessons = "组织的知识"（跨agent、跨workflow、系统驱动）
- Agent memory = "个人的记忆"（agent自己的经验、偏好、习惯）

**流动方向：**
- ACP lessons中的高频经验 → 浓缩后注入step prompt（待建设）
- ACP lessons中的通用教训 → 人工/自动写入agent的AGENTS.md
- Agent memory中的通用发现 → 可以被ACP lessons系统引用

---

## 八、待讨论问题

1. **MEMORY.md 5KB上限是否合理？** 当前我的MEMORY.md已经7.8KB
2. **定期整理用heartbeat还是cron？** heartbeat更自然但可能跟任务冲突
3. **谁来触发"lessons→AGENTS.md"的内化？** 人工审阅？自动？Lyra代理？
4. **共享知识目录要不要建？** 用OpenClaw的memorySearch.extraPaths让所有agent共享一个知识库
5. **旧的daily log要不要归档/删除？** 防止memory/目录无限增长
