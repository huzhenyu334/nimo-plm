# ACP (Agent Control Panel)

> 内部AI Agent团队管理平台，泽斌发起，Lyra主导建设

## 基本信息
- 技术栈：Go + React + SQLite，零依赖单二进制
- 端口：3001 | 地址：http://43.134.86.237:3001
- 密码：openclaw2026
- 代码目录：/home/claw/.openclaw/workspace/agent-control-panel/
- systemd管理：`systemctl --user restart acp`
- 飞书App ID：cli_a9122d58b5f8dcca

## 定位
- **ACP是给Lyra用的**，泽斌只看前端页面
- 两套界面：泽斌=前端Web，Lyra=MCP接口（plugin tools）
- 目标：7×24持续工作，系统驱动agent团队不间断
- 核心理念："平台只是平台，核心是流程"（泽斌语）

## 引擎能力
- 结构化输入输出 + schema校验（422打回）
- Gate条件循环（所有step通用）
- Expand子任务集（PM输出JSON数组→引擎展开→逐个分配）
- Approval审批节点（飞书card + on_reject策略）
- Condition条件分支
- Escalate升级 + Blocked状态
- Lessons自动采集（4字段必填）
- Credential vault（AES-GCM加密）
- 崩溃恢复（已修4个bug）

## 工作流模板
- 新模块开发：6f0b1040（PRD→拆任务→expand开发→交付报告）
- 功能优化/修复：b0995080（发现→检查→审批→开发→验收→报告）v11

## Plugin Tools
14个原生agent tools，通过OpenClaw plugins.load.paths加载

## 版本历史
- v2.2: 审批系统+飞书OAuth
- v2.3: 审批升级+条件分支
- v2.4: 工作流停止+Agent进化模块
- v2.5: Flow审计系统+分析Dashboard

## 关键技术决策
- API响应：不返回agent_context，truncate output>2000字符
- 前端部署：`rm -rf web && cp -r acp-web/dist web`，生成.gz文件
- 模板变量限制：Max 32000/64000/100000 chars
- 泽斌ACP-app open_id：ou_e229cd56698a8e15e629af2447a8e0ed（和Lyra app不同！）
