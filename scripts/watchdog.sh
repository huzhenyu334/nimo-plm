#!/bin/bash
# PLM/ERP 服务守护脚本
cd /home/claw/.openclaw/workspace

# Check PLM
if ! curl -s --connect-timeout 2 http://127.0.0.1:8080/health/live > /dev/null 2>&1; then
    echo "[$(date)] PLM down, restarting..."
    nohup ./bin/plm > plm.log 2>&1 &
    echo "[$(date)] PLM started PID: $!"
fi

# Check ERP
if ! curl -s --connect-timeout 2 http://127.0.0.1:8081/health/live > /dev/null 2>&1; then
    echo "[$(date)] ERP down, restarting..."
    nohup ./bin/erp > erp.log 2>&1 &
    echo "[$(date)] ERP started PID: $!"
fi
