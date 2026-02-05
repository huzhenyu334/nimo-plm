#!/bin/bash
set -e

# nimo PLM 部署脚本
# 用法: ./scripts/deploy.sh

echo "=========================================="
echo "       nimo PLM 部署脚本"
echo "=========================================="

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# 检查Docker
check_docker() {
    if ! command -v docker &> /dev/null; then
        echo -e "${RED}❌ Docker 未安装${NC}"
        echo "请先安装 Docker: https://docs.docker.com/engine/install/"
        exit 1
    fi
    echo -e "${GREEN}✓ Docker 已安装${NC}"
}

# 检查Docker Compose
check_docker_compose() {
    if docker compose version &> /dev/null; then
        COMPOSE_CMD="docker compose"
    elif command -v docker-compose &> /dev/null; then
        COMPOSE_CMD="docker-compose"
    else
        echo -e "${RED}❌ Docker Compose 未安装${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ Docker Compose 已安装${NC}"
}

# 创建必要目录
setup_dirs() {
    echo -e "\n${YELLOW}[1/6] 创建数据目录...${NC}"
    mkdir -p data/{postgres,redis,minio,rabbitmq}
    echo -e "${GREEN}✓ 目录创建完成${NC}"
}

# 配置环境变量
setup_env() {
    echo -e "\n${YELLOW}[2/6] 配置环境变量...${NC}"
    if [ ! -f .env ]; then
        cp .env.example .env
        echo -e "${YELLOW}⚠ 请编辑 .env 文件配置飞书应用信息${NC}"
    fi
    
    # 生成JWT密钥（如果未设置）
    if grep -q "your-jwt-secret" .env 2>/dev/null; then
        JWT_SECRET=$(openssl rand -base64 32)
        sed -i "s/your-jwt-secret-key-change-in-production/$JWT_SECRET/" .env
        echo -e "${GREEN}✓ JWT密钥已自动生成${NC}"
    fi
    echo -e "${GREEN}✓ 环境变量配置完成${NC}"
}

# 启动基础服务
start_infra() {
    echo -e "\n${YELLOW}[3/6] 启动基础服务 (PostgreSQL, Redis, MinIO)...${NC}"
    cd deployments/docker
    $COMPOSE_CMD up -d postgres redis minio
    cd ../..
    echo -e "${GREEN}✓ 基础服务启动中${NC}"
}

# 等待服务就绪
wait_services() {
    echo -e "\n${YELLOW}[4/6] 等待服务就绪...${NC}"
    
    echo -n "等待 PostgreSQL..."
    for i in {1..30}; do
        if docker exec nimo-postgres pg_isready -U nimo -d nimo_plm &>/dev/null; then
            echo -e " ${GREEN}✓${NC}"
            break
        fi
        echo -n "."
        sleep 2
    done
    
    echo -n "等待 Redis..."
    for i in {1..30}; do
        if docker exec nimo-redis redis-cli ping &>/dev/null; then
            echo -e " ${GREEN}✓${NC}"
            break
        fi
        echo -n "."
        sleep 2
    done
    
    echo -n "等待 MinIO..."
    for i in {1..30}; do
        if curl -s http://localhost:9000/minio/health/live &>/dev/null; then
            echo -e " ${GREEN}✓${NC}"
            break
        fi
        echo -n "."
        sleep 2
    done
}

# 创建MinIO Bucket
setup_minio() {
    echo -e "\n${YELLOW}[5/6] 配置 MinIO...${NC}"
    
    # 使用docker运行mc命令
    docker run --rm --network nimo-network \
        -e MC_HOST_myminio=http://minioadmin:minioadmin123@minio:9000 \
        minio/mc mb myminio/nimo-plm --ignore-existing 2>/dev/null || true
    
    echo -e "${GREEN}✓ MinIO Bucket 创建完成${NC}"
}

# 编译并启动PLM服务
start_plm() {
    echo -e "\n${YELLOW}[6/6] 编译并启动 PLM 服务...${NC}"
    
    # 检查Go环境
    if ! command -v go &> /dev/null; then
        echo -e "${YELLOW}Go未安装，使用Docker编译...${NC}"
        cd deployments/docker
        $COMPOSE_CMD up -d --build plm-service
        cd ../..
    else
        echo "编译中..."
        go build -o server ./cmd/server/...
        
        # 加载环境变量并启动
        source .env 2>/dev/null || true
        export SERVER_PORT=8080
        export DB_HOST=localhost
        export DB_PORT=5432
        export DB_USER=nimo
        export DB_PASSWORD=nimo123
        export DB_NAME=nimo_plm
        export REDIS_HOST=localhost
        export REDIS_PORT=6379
        export MINIO_ENDPOINT=localhost:9000
        export MINIO_ACCESS_KEY=minioadmin
        export MINIO_SECRET_KEY=minioadmin123
        export MINIO_BUCKET=nimo-plm
        
        echo -e "${GREEN}启动服务...${NC}"
        nohup ./server > logs/plm.log 2>&1 &
        echo $! > .pid
    fi
    
    echo -e "${GREEN}✓ PLM 服务启动完成${NC}"
}

# 健康检查
health_check() {
    echo -e "\n${YELLOW}健康检查...${NC}"
    sleep 3
    
    if curl -s http://localhost:8080/health/live | grep -q "ok"; then
        echo -e "${GREEN}✓ PLM 服务运行正常${NC}"
    else
        echo -e "${YELLOW}⚠ PLM 服务可能还在启动中，请稍后检查${NC}"
    fi
}

# 打印信息
print_info() {
    echo -e "\n=========================================="
    echo -e "${GREEN}✅ 部署完成！${NC}"
    echo "=========================================="
    echo ""
    echo "服务地址:"
    echo "  - PLM API:    http://localhost:8080"
    echo "  - MinIO控制台: http://localhost:9001 (minioadmin/minioadmin123)"
    echo ""
    echo "API文档:"
    echo "  - 健康检查:   GET  /health/live"
    echo "  - 版本信息:   GET  /version"
    echo "  - 飞书登录:   GET  /api/v1/auth/feishu/login"
    echo ""
    echo "测试命令:"
    echo "  curl http://localhost:8080/health/live"
    echo "  curl http://localhost:8080/version"
    echo ""
    echo "查看日志:"
    echo "  tail -f logs/plm.log"
    echo ""
    echo "停止服务:"
    echo "  ./scripts/stop.sh"
    echo ""
}

# 主流程
main() {
    check_docker
    check_docker_compose
    setup_dirs
    setup_env
    start_infra
    wait_services
    setup_minio
    mkdir -p logs
    start_plm
    health_check
    print_info
}

main "$@"
