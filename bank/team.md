# BitFantasy Agent团队

## 成员

### 泽斌（陈泽斌）— CEO
- Feishu open_id (Lyra app): ou_5b159fc157d4042f1e8088b1ffebb2da
- Feishu open_id (ACP app): ou_e229cd56698a8e15e629af2447a8e0ed
- 邮箱：suonety@gmail.com
- 作息：夜猫子型，凌晨2点左右睡觉
- 风格：高效直接，看重实际进展，技术决策果断

### Lyra — COO (main agent)
- Workspace: /home/claw/.openclaw/workspace
- 模型：Opus 4.6
- 职责：全局调度、项目管理、系统建设

### PM — 产品经理
- Workspace: /home/claw/.openclaw/workspace-pm
- 模型：Opus
- 职责：PRD撰写、任务拆解、验收把关

### UX — 设计师
- Workspace: /home/claw/.openclaw/workspace-ux
- 模型：Opus
- 职责：Design System、原型设计、UI验收

### Alice — 开发
- Workspace: /home/claw/.openclaw/workspace-alice
- 模型：Opus
- 职责：执行CC开发任务

### Catherine — 另一台服务器的agent
- 飞书App ID: cli_a90dddef38b8dbc9
- Feishu open_id: ou_3ee5f5cfefb6f57c6e6cd7fffda1bfe1
- Discord ID: 1473763559312724210
- 用GLM5模型

## 协作方式
- Lyra通过ACP工作流调度PM/UX/Alice
- CC Hook通知：Lyra(main) + 泽斌飞书DM
- Discord三方通信已打通（需设allowBots: true）
- Lyra Discord ID: 1473761493807005696

## 关键配置
- agents.list必须包含所有4个agent（main/pm/ux/alice）
- 曾被莫名删过，需注意
