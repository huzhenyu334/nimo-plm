#!/bin/bash
# watchdog.sh — 服务健康监控 + 自动恢复 + 飞书告警
# 由系统crontab每5分钟执行一次

LOG="/tmp/watchdog.log"
ALERT_FILE="/tmp/watchdog-last-alert"
ALERT_COOLDOWN=300  # 同一问题5分钟内不重复告警

log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" >> "$LOG"; }

# 飞书webhook告警（备用，当OpenClaw本身挂了时用）
# 如果配置了FEISHU_WEBHOOK_URL环境变量，会通过webhook发告警
alert() {
    local msg="$1"
    log "🚨 ALERT: $msg"
    
    # 去重：同一消息ALERT_COOLDOWN秒内不重复
    local hash=$(echo "$msg" | md5sum | cut -d' ' -f1)
    if [ -f "$ALERT_FILE-$hash" ]; then
        local last=$(cat "$ALERT_FILE-$hash")
        local now=$(date +%s)
        if [ $((now - last)) -lt $ALERT_COOLDOWN ]; then
            return
        fi
    fi
    date +%s > "$ALERT_FILE-$hash"
    
    # 通过飞书webhook告警（如果配置了的话）
    if [ -n "$FEISHU_WEBHOOK_URL" ]; then
        curl -s -X POST "$FEISHU_WEBHOOK_URL" \
            -H 'Content-Type: application/json' \
            -d "{\"msg_type\":\"text\",\"content\":{\"text\":\"🚨 服务监控告警\\n$msg\\n时间: $(date '+%Y-%m-%d %H:%M:%S')\"}}" \
            > /dev/null 2>&1
    fi
}

recover() {
    local service="$1"
    local method="$2"
    log "🔧 Recovering $service via $method"
    
    case "$method" in
        systemd-user)
            systemctl --user restart "$service" 2>> "$LOG"
            ;;
        systemd-system)
            sudo systemctl restart "$service" 2>> "$LOG"
            ;;
        docker)
            cd /home/claw/.openclaw/workspace/openclaw-mission-control
            REDIS_PORT=6380 docker compose restart "$service" 2>> "$LOG"
            ;;
    esac
}

ISSUES=0

# --- 1. OpenClaw Gateway ---
if systemctl --user is-active openclaw-gateway.service > /dev/null 2>&1; then
    # 进程在，检查RPC是否响应
    if ! timeout 5 curl -sf http://127.0.0.1:18789/ > /dev/null 2>&1; then
        alert "OpenClaw Gateway 进程在但HTTP无响应，尝试重启"
        recover "openclaw-gateway.service" "systemd-user"
        ISSUES=$((ISSUES+1))
    fi
else
    alert "OpenClaw Gateway 进程不存在，自动重启"
    recover "openclaw-gateway.service" "systemd-user"
    ISSUES=$((ISSUES+1))
fi

# --- 2. Nimo PLM ---
if systemctl --user is-active nimo-plm.service > /dev/null 2>&1; then
    if ! timeout 5 curl -sf http://127.0.0.1:8080/ > /dev/null 2>&1; then
        alert "PLM 进程在但HTTP无响应，尝试重启"
        recover "nimo-plm.service" "systemd-user"
        ISSUES=$((ISSUES+1))
    fi
else
    alert "PLM 服务不存在，自动重启"
    recover "nimo-plm.service" "systemd-user"
    ISSUES=$((ISSUES+1))
fi

# --- 3. Command Center ---
if systemctl is-active openclaw-command-center.service > /dev/null 2>&1; then
    if ! timeout 5 curl -sf http://127.0.0.1:3002/ > /dev/null 2>&1; then
        alert "Command Center 进程在但HTTP无响应，尝试重启"
        recover "openclaw-command-center.service" "systemd-system"
        ISSUES=$((ISSUES+1))
    fi
else
    alert "Command Center 服务不存在，自动重启"
    recover "openclaw-command-center.service" "systemd-system"
    ISSUES=$((ISSUES+1))
fi

# --- 4. Mission Control (Docker) ---
MC_DIR="/home/claw/.openclaw/workspace/openclaw-mission-control"
if [ -d "$MC_DIR" ]; then
    cd "$MC_DIR"
    # 检查关键容器
    for svc in backend frontend; do
        status=$(REDIS_PORT=6380 docker compose ps "$svc" --format '{{.Status}}' 2>/dev/null)
        if [[ ! "$status" =~ "Up" ]]; then
            alert "Mission Control $svc 容器异常: $status"
            recover "$svc" "docker"
            ISSUES=$((ISSUES+1))
        fi
    done
fi

# --- 5. Catherine-Build Node连接 ---
# 通过Gateway API检查node状态（轻量级）
NODE_STATUS=$(timeout 5 curl -sf "http://127.0.0.1:18789/" 2>/dev/null | grep -o "Catherine" || echo "")
# Node检查比较复杂，先记录状态不自动恢复（需要远程操作）

# --- 6. 磁盘空间 ---
DISK_USAGE=$(df / | tail -1 | awk '{print $5}' | tr -d '%')
if [ "$DISK_USAGE" -gt 90 ]; then
    alert "磁盘使用率 ${DISK_USAGE}%，超过90%警戒线"
    # 自动清理日志
    find /tmp/openclaw -name "*.log" -mtime +7 -delete 2>/dev/null
    docker system prune -f > /dev/null 2>&1
    ISSUES=$((ISSUES+1))
fi

# --- 7. 内存 ---
MEM_AVAIL=$(free -m | awk '/Mem:/ {print $7}')
if [ "$MEM_AVAIL" -lt 200 ]; then
    alert "可用内存仅 ${MEM_AVAIL}MB，低于200MB警戒线"
    ISSUES=$((ISSUES+1))
fi

# --- 总结 ---
if [ $ISSUES -eq 0 ]; then
    log "✅ All services healthy"
else
    log "⚠️ Found $ISSUES issues, recovery attempted"
fi

# 保留最近1000行日志
tail -1000 "$LOG" > "$LOG.tmp" && mv "$LOG.tmp" "$LOG"
