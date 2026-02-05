#!/bin/bash

# nimo PLM API 测试脚本

BASE_URL="http://localhost:8080"

echo "=========================================="
echo "       nimo PLM API 测试"
echo "=========================================="

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

test_endpoint() {
    local method=$1
    local endpoint=$2
    local expected=$3
    local data=$4
    
    echo -n "测试 $method $endpoint ... "
    
    if [ -n "$data" ]; then
        response=$(curl -s -X $method "$BASE_URL$endpoint" \
            -H "Content-Type: application/json" \
            -d "$data" -w "\n%{http_code}")
    else
        response=$(curl -s -X $method "$BASE_URL$endpoint" -w "\n%{http_code}")
    fi
    
    http_code=$(echo "$response" | tail -1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" = "$expected" ]; then
        echo -e "${GREEN}✓ HTTP $http_code${NC}"
        return 0
    else
        echo -e "${RED}✗ HTTP $http_code (expected $expected)${NC}"
        echo "  Response: $body"
        return 1
    fi
}

# 基础检查
echo -e "\n${YELLOW}[1] 基础健康检查${NC}"
test_endpoint GET "/health/live" "200"
test_endpoint GET "/health/ready" "200"
test_endpoint GET "/version" "200"

# 公开接口
echo -e "\n${YELLOW}[2] 认证接口${NC}"
test_endpoint GET "/api/v1/auth/feishu/login" "302"

# 需要认证的接口 (应该返回401)
echo -e "\n${YELLOW}[3] 需认证接口 (预期401)${NC}"
test_endpoint GET "/api/v1/products" "401"
test_endpoint GET "/api/v1/projects" "401"
test_endpoint GET "/api/v1/ecns" "401"
test_endpoint GET "/api/v1/documents" "401"

echo -e "\n=========================================="
echo "测试完成"
echo "=========================================="

# 如果有JWT token，可以测试更多接口
if [ -n "$JWT_TOKEN" ]; then
    echo -e "\n${YELLOW}[4] 带认证的接口测试${NC}"
    
    AUTH_HEADER="Authorization: Bearer $JWT_TOKEN"
    
    echo -n "测试 GET /api/v1/auth/me ... "
    response=$(curl -s "$BASE_URL/api/v1/auth/me" -H "$AUTH_HEADER" -w "\n%{http_code}")
    http_code=$(echo "$response" | tail -1)
    if [ "$http_code" = "200" ]; then
        echo -e "${GREEN}✓ HTTP $http_code${NC}"
    else
        echo -e "${RED}✗ HTTP $http_code${NC}"
    fi
fi
