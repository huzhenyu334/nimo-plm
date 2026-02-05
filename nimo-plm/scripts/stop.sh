#!/bin/bash

echo "停止 nimo PLM 服务..."

# 停止PLM进程
if [ -f .pid ]; then
    kill $(cat .pid) 2>/dev/null
    rm .pid
    echo "✓ PLM服务已停止"
fi

# 停止Docker服务
cd deployments/docker
if docker compose version &> /dev/null; then
    docker compose down
else
    docker-compose down
fi
cd ../..

echo "✓ 所有服务已停止"
