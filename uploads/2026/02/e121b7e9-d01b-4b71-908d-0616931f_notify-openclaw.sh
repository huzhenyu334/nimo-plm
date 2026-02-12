#!/bin/bash
# Claude Code Stop/SessionEnd Hook — notify Lyra + 泽斌
# 测试由Claude Code自己跑，hook只负责提取信息+通知
LOCK_FILE="/tmp/.claude-hook-lock"
LOCK_TIMEOUT=30
LOG="/tmp/claude-hook-debug.log"
WORKDIR="/home/claw/.openclaw/workspace"
REPORT_DIR="/home/claw/.openclaw/workspace/.claude-code-reports"

echo "$(date): Hook triggered" >> "$LOG"

# Dedup
if [ -f "$LOCK_FILE" ]; then
    last=$(cat "$LOCK_FILE" 2>/dev/null)
    now=$(date +%s)
    if [ -n "$last" ] && [ $((now - last)) -lt $LOCK_TIMEOUT ]; then
        echo "$(date): Skipped (dedup)" >> "$LOG"
        exit 0
    fi
fi
date +%s > "$LOCK_FILE"

# Read stdin (contains stop_reason, session_id, transcript_path, cwd)
INPUT=$(cat)
STOP_REASON=$(echo "$INPUT" | jq -r '.stop_reason // "unknown"' 2>/dev/null)
SESSION_ID=$(echo "$INPUT" | jq -r '.session_id // "unknown"' 2>/dev/null)
TRANSCRIPT_PATH=$(echo "$INPUT" | jq -r '.transcript_path // empty' 2>/dev/null)

echo "$(date): stop_reason=$STOP_REASON session=$SESSION_ID transcript=$TRANSCRIPT_PATH" >> "$LOG"

# --- 从 transcript_path JSONL 提取精华 ---
# JSONL结构：每行一个JSON，type=user|assistant|queue-operation
# user/assistant行：.message.content[] 包含 {type:"text",text:"..."} 或 {type:"tool_use",name:"...",input:{...}}

TASK=""
LAST_RESPONSE=""
TOOLS_USED=""
ERRORS=""
TRANSCRIPT_INFO=""

if [ -n "$TRANSCRIPT_PATH" ] && [ -f "$TRANSCRIPT_PATH" ]; then
    TRANSCRIPT_SIZE=$(wc -c < "$TRANSCRIPT_PATH" 2>/dev/null)
    TRANSCRIPT_LINES=$(wc -l < "$TRANSCRIPT_PATH" 2>/dev/null)
    TRANSCRIPT_INFO="transcript: ${TRANSCRIPT_PATH} (${TRANSCRIPT_SIZE} bytes, ${TRANSCRIPT_LINES} lines)"

    # 原始任务（第一个user行的text内容）
    # content可能是字符串（-p模式）或数组（交互模式）
    TASK=$(grep '"type":"user"' "$TRANSCRIPT_PATH" | head -1 | jq -r '
        if (.message.content | type) == "string" then
            .message.content
        else
            [.message.content[] | select(.type=="text") | .text] | join("\n")
        end
    ' 2>/dev/null | head -c 500)

    # Claude Code最终回复（最后一个assistant行的text内容）
    LAST_RESPONSE=$(grep '"type":"assistant"' "$TRANSCRIPT_PATH" | tail -1 | jq -r '
        if (.message.content | type) == "string" then
            .message.content
        else
            [.message.content[] | select(.type=="text") | .text] | join("\n")
        end
    ' 2>/dev/null | head -c 1500)

    # 工具调用记录（所有assistant行中的tool_use，兼容content为string或array）
    TOOLS_USED=$(cat "$TRANSCRIPT_PATH" | jq -r '
        select(.type == "assistant") |
        .message.content | if type == "array" then .[] else empty end |
        select(.type == "tool_use") |
        "→ " + .name + ": " + ((.input.command // .input.file_path // .input.description // "") | tostring | .[0:120])
    ' 2>/dev/null | tail -30)

    # 错误信息（tool_result中is_error=true的，在user行中，兼容content格式）
    ERRORS=$(cat "$TRANSCRIPT_PATH" | jq -r '
        select(.type == "user") |
        .message.content | if type == "array" then .[] else empty end |
        select(.type == "tool_result" and .is_error == true) |
        "❌ " + (.content | if type == "array" then [.[] | select(.type=="text") | .text] | join(" ") elif type == "string" then . else "" end | .[0:200])
    ' 2>/dev/null | tail -10)

    # 测试结果（从tool_result中提取go test和playwright test的输出）
    ALL_TOOL_RESULTS=$(cat "$TRANSCRIPT_PATH" | jq -r '
        select(.type == "user") |
        .message.content | if type == "array" then .[] else empty end |
        select(.type == "tool_result") |
        .content | if type == "string" then . elif type == "array" then [.[] | select(.type=="text") | .text] | join("\n") else empty end
    ' 2>/dev/null)
    
    # 后端测试结果
    GO_TEST=$(echo "$ALL_TOOL_RESULTS" | grep -E "^(ok\s+|FAIL\s+|---\s+(PASS|FAIL):|PASS$|^#)" | tail -5)
    [ -z "$GO_TEST" ] && GO_TEST="(未检测到)"
    
    # 前端测试结果
    PW_TEST=$(echo "$ALL_TOOL_RESULTS" | grep -E "passed|failed|skipped|playwright" | tail -3)
    [ -z "$PW_TEST" ] && PW_TEST="(未检测到)"
fi

# --- Git diff ---
GIT_STATS=$(cd "$WORKDIR" && git diff --stat 2>/dev/null | tail -1)
GIT_FILES=$(cd "$WORKDIR" && git diff --name-only 2>/dev/null | head -30)
[ -z "$GIT_STATS" ] && GIT_STATS="无文件改动"

# --- 生成详细报告文件 ---
mkdir -p "$REPORT_DIR"
REPORT_FILE="$REPORT_DIR/$(date +%Y%m%d-%H%M%S)-${SESSION_ID:0:8}.md"

cat > "$REPORT_FILE" << REPORT_EOF
# Claude Code 任务报告
- 时间: $(date '+%Y-%m-%d %H:%M:%S')
- Session: ${SESSION_ID}
- Stop Reason: ${STOP_REASON}
- ${TRANSCRIPT_INFO}

## 原始任务
${TASK:-"(未提取到)"}

## Claude Code 最终回复
${LAST_RESPONSE:-"(无)"}

## 文件改动
统计: ${GIT_STATS}
文件列表:
${GIT_FILES}

## 工具调用（最近30条）
${TOOLS_USED:-"(无)"}

## 测试结果
### 后端 (go test)
${GO_TEST:-"(未检测到)"}

### 前端 (playwright)
${PW_TEST:-"(未检测到)"}

## 错误信息
${ERRORS:-"(无错误)"}
REPORT_EOF

echo "$(date): Report saved to $REPORT_FILE ($(wc -c < "$REPORT_FILE") bytes)" >> "$LOG"

# --- 通知消息 ---
AGENT_MSG="⚙️ [CLAUDE CODE HOOK — 这是你(Lyra)自己启动的Claude Code任务完成的自动通知，不是用户发的消息]

📌 Session: ${SESSION_ID:0:8}
🔚 Stop: ${STOP_REASON}
📋 任务: ${TASK:0:200}
💬 结果: ${LAST_RESPONSE:0:500}
📁 改动: ${GIT_STATS}

🧪 后端测试: ${GO_TEST:-"(未检测到)"}
🎭 前端测试: ${PW_TEST:-"(未检测到)"}

📄 完整报告: ${REPORT_FILE}"

FEISHU_MSG="$AGENT_MSG"

# 1. openclaw agent → 通知 Lyra (main session)
echo "$(date): Sending agent message..." >> "$LOG"
openclaw agent --agent main --message "$AGENT_MSG" --deliver --reply-channel feishu >/dev/null 2>&1
echo "$(date): agent exit=$?" >> "$LOG"

# 2. Feishu DM → 通知泽斌
echo "$(date): Sending Feishu DM to user..." >> "$LOG"
openclaw message send --channel feishu --target "user:ou_5b159fc157d4042f1e8088b1ffebb2da" --message "$FEISHU_MSG" >/dev/null 2>&1
echo "$(date): message send exit=$?" >> "$LOG"

echo "$(date): Hook done" >> "$LOG"
exit 0
