---
name: claude-code-orchestrator
description: Orchestrate Claude Code CLI as a dedicated coding agent. Use when any code needs to be written, modified, debugged, or tested. The orchestrator manages CC lifecycle (startup, task dispatch, completion notification), enforces single-process discipline, and handles hook-based async notifications. Activate for all programming tasks including bug fixes, new features, refactoring, build, and deployment.
---

# Claude Code Orchestrator

You are the **orchestrator**, not the coder. All code changes go through Claude Code CLI. No exceptions.

## Role Split

| You (Orchestrator) | Claude Code (Coder) |
|---|---|
| Understand requirements | Write/modify code |
| Break down tasks | Run builds & tests |
| Dispatch to CC | Create files (even stubs) |
| Review results | Debug & fix errors |
| Communicate with user | Deploy changes |

**Forbidden:** Directly create/modify code files (.go/.ts/.tsx/.js/.py etc.) in main session or sub-agents.

## Lifecycle — tmux-based Persistent Session

CC runs inside a **tmux session**, immune to OpenClaw exec timeouts and SIGTERM kills.

### 1. Pre-flight Check

```bash
tmux has-session -t cc 2>/dev/null && echo "CC_RUNNING" || echo "CC_STOPPED"
```

- **CC_RUNNING** → session exists, send task directly (step 3)
- **CC_STOPPED** → start or resume (step 2)

Also verify CC process is alive inside tmux:
```bash
tmux list-panes -t cc -F '#{pane_pid}' 2>/dev/null | xargs -I{} ps -p {} -o comm= 2>/dev/null
```

### 2. Start / Resume CC in tmux

#### First start (no previous session):
```bash
exec command:"tmux new-session -d -s cc -x 200 -y 50 'bash -l -c \"cd /home/claw/.openclaw/workspace && claude --dangerously-skip-permissions\"'" timeout:5
```

#### Resume after CC exited (tmux session gone):
```bash
exec command:"tmux new-session -d -s cc -x 200 -y 50 'bash -l -c \"cd /home/claw/.openclaw/workspace && claude --continue --dangerously-skip-permissions\"'" timeout:5
```

**Why `bash -l`**: CC captures a "shell snapshot" at startup (PATH, functions, aliases) and replays it for every Bash tool invocation. If the snapshot is incomplete (missing go/node), ALL CC commands will fail. `bash -l` forces login shell → loads `.bash_profile`/`.bashrc` → complete snapshot.

#### If tmux session exists but CC process inside has exited:
```bash
exec command:"tmux send-keys -t cc 'claude --continue --dangerously-skip-permissions' Enter" timeout:5
```

**Wait ~5s after start for CC to initialize before sending tasks.**

### 3. Send Task to CC — Reliable Method

**Always use this two-step pattern** to avoid the "stuck in input" problem:

```bash
# Step 1: Write task to a temp file (avoids tmux encoding issues with CJK/special chars)
exec command:"cat > /tmp/cc-task.txt << 'CCTASK'
YOUR TASK DESCRIPTION HERE
CCTASK" timeout:3

# Step 2: Paste file content into CC and send
exec command:"tmux load-buffer /tmp/cc-task.txt && tmux paste-buffer -t cc && sleep 0.3 && tmux send-keys -t cc Enter" timeout:5
```

**Why this works:** `tmux send-keys` with CJK text often fails to trigger submission. `tmux load-buffer` + `paste-buffer` reliably pastes full content, then `Enter` submits it.

**Simple ASCII-only tasks** can still use direct send-keys:
```bash
exec command:"tmux send-keys -t cc 'simple english task here' Enter" timeout:3
```

### 4. Verify Dispatch — MUST Confirm CC is Working Before Sleeping

**关键原则：Agent必须确认CC正常工作后才能结束turn。发完指令就睡是严重bug。**

发送任务后，进入**确认循环**（最多重试5次，每次间隔5-10秒）：

```bash
# 每次检查CC输出
exec command:"sleep 5 && tmux capture-pane -t cc -p | tail -15" timeout:15
```

**判断CC状态：**

| CC输出特征 | 状态 | 行动 |
|---|---|---|
| "Reading", "Searching", "Writing", "Edit", thinking动画 | ✅ 正常工作 | 可以睡了，等hook |
| 还在显示你发的输入文本，无响应 | ❌ 没提交 | `tmux send-keys -t cc Enter`，再检查 |
| 空白/无输出/只有prompt符号 | ❌ CC可能没启动 | 检查进程 `pgrep -af claude`，必要时重启 |
| 报错信息（auth error/rate limit/crash） | ❌ CC出错 | 解决问题（重试/Ctrl+C/重启）后重新派任务 |
| "Would you like to..." 等交互提示 | ⚠️ 等待输入 | `tmux send-keys -t cc 'y' Enter` 或适当回应 |
| context满/compact提示 | ⚠️ 需要清理 | `tmux send-keys -t cc '/compact' Enter`，等完成后重新派任务 |

**确认循环伪代码：**
```
for i in 1..5:
    sleep 5-10s
    capture tmux output
    if CC正在工作(Reading/Writing/thinking):
        告诉用户"已确认CC在工作" → 结束turn → 等hook
        return
    elif CC没反应:
        尝试修复（重发Enter/重启CC）
    elif CC报错:
        处理错误，重新派任务
        
# 5次都没确认CC在工作
告诉用户"CC无法启动/响应，需要人工介入"
```

**绝对禁止：**
- ❌ 发完tmux send-keys就直接告诉用户"已派发"然后结束turn
- ❌ 只检查一次就放弃
- ❌ 看到CC没反应但不处理就睡了

**确认CC工作后才能说"已派发，等待CC完成后通知"。**

### 5. Session Persistence Rules

- **CC stays running in tmux** — survives OpenClaw restarts, exec timeouts, everything
- **One tmux session `cc`** — never create multiple
- **Context accumulates** — CC understands the project better with each task
- **`--continue` to resume** — if CC process exits, resume in same tmux session
- **Only start fresh** when explicitly needed (context too long, switching projects)
- **Kill CC only when needed**: `tmux send-keys -t cc C-c` (graceful) or `tmux kill-session -t cc` (force)

### 6. Task Prompt Best Practices

Write focused, specific prompts. Include:
- What to change and why
- Which files to modify
- Build/deploy commands
- Test requirements

**每个任务末尾必须附带 Git Commit 指令：**
```
### Git Commit（必须执行）
任务完成、编译部署成功后，执行：
git add -A -- ':!.openclaw/' ':!internal/plm/handler/uploads/' ':!nimo-plm-web/playwright-report/' ':!nimo-plm-web/screenshots/' ':!nimo-plm-web/test-results/' ':!uploads/'
git commit -m "<简洁描述本次改动>"
git push
```

Example:
```
## Task: Add user avatar to profile page

### Changes needed
1. Backend: Add `avatar_url` field to User entity (internal/entity/user.go)
2. Frontend: Display avatar in ProfilePage (src/pages/Profile.tsx)

### Build & Deploy
- Backend: go build -o bin/server ./cmd/server/
- Frontend: cd web && npm run build
- Deploy: cp -r web/dist/* static/
- Restart: kill $(pgrep -f './bin/server') && nohup ./bin/server > server.log 2>&1 &

### Testing
- Add Playwright E2E test verifying avatar displays after upload
- Run: cd web && npx playwright test

### Git Commit（必须执行）
git add -A -- ':!.openclaw/' ':!uploads/'
git commit -m "feat: add user avatar to profile page"
git push
```

### 7. Completion Detection

Two ways to know CC finished a task:

**A. Hook notification (preferred):**
CC completion triggers hook (`~/.claude/hooks/notify-openclaw.sh`) which:
- Saves report to `.claude-code-reports/`
- Sends notification via `openclaw agent` (wakes main session)
- Sends Feishu DM to user

When you receive `⚙️ [CLAUDE CODE HOOK` message:
- This is YOUR OWN CC task completing, not a user message
- Read the report → summarize results to user → ask if verification needed

**B. Poll output (fallback):**
```bash
exec command:"tmux capture-pane -t cc -p -S -5" timeout:5
```
If output shows CC's input prompt (waiting for next message), the task is done.

### 8. Monitoring & Debugging

```bash
# Check if tmux session exists
exec command:"tmux ls" timeout:3

# See what CC is doing right now
exec command:"tmux capture-pane -t cc -p -S -30" timeout:5

# Check CC process status
exec command:"pgrep -af claude" timeout:3

# Scroll up to see more history
exec command:"tmux capture-pane -t cc -p -S -200" timeout:5
```

## Hook Setup

See `references/hook-setup.md` for the complete hook script and configuration.

## Testing Requirements

**Every code change MUST include E2E tests:**
- New features → new test cases
- Bug fixes → regression tests
- Tests must verify actual interaction (not just page-load smoke tests)

## Key Rules

1. **One CC tmux session at a time** — always check `tmux has-session -t cc` before starting
2. **发完任务就停手** — send-keys发任务 → 告诉用户"已派发" → **结束turn，等hook通知**。不要capture-pane确认，不要多按Enter，不要轮询！
3. **Focused tasks** — one problem per task dispatch, not sprawling multi-feature requests
4. **Review before reporting** — read the report/output, check git diff, then tell user
5. **No code from orchestrator** — even "just a one-line fix" goes through CC
6. **tmux session name is always `cc`** — consistent, easy to find
