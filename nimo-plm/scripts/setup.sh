#!/bin/bash

# nimo PLM Setup Script
# This script helps set up the development environment

set -e

echo "🚀 nimo PLM Setup Script"
echo "========================"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check prerequisites
check_prerequisites() {
    echo -e "\n${YELLOW}Checking prerequisites...${NC}"
    
    # Check Go
    if command -v go &> /dev/null; then
        GO_VERSION=$(go version | awk '{print $3}')
        echo -e "${GREEN}✓${NC} Go installed: $GO_VERSION"
    else
        echo -e "${RED}✗${NC} Go not found. Please install Go 1.22+"
        exit 1
    fi
    
    # Check Docker
    if command -v docker &> /dev/null; then
        DOCKER_VERSION=$(docker --version | awk '{print $3}')
        echo -e "${GREEN}✓${NC} Docker installed: $DOCKER_VERSION"
    else
        echo -e "${RED}✗${NC} Docker not found. Please install Docker"
        exit 1
    fi
    
    # Check Docker Compose
    if command -v docker-compose &> /dev/null || docker compose version &> /dev/null; then
        echo -e "${GREEN}✓${NC} Docker Compose installed"
    else
        echo -e "${RED}✗${NC} Docker Compose not found"
        exit 1
    fi
}

# Create .env file
create_env() {
    echo -e "\n${YELLOW}Creating .env file...${NC}"
    if [ ! -f .env ]; then
        cp .env.example .env
        echo -e "${GREEN}✓${NC} .env created from .env.example"
        echo -e "${YELLOW}  Please edit .env and add your Feishu app credentials${NC}"
    else
        echo -e "${GREEN}✓${NC} .env already exists"
    fi
}

# Download dependencies
download_deps() {
    echo -e "\n${YELLOW}Downloading Go dependencies...${NC}"
    go mod download
    go mod tidy
    echo -e "${GREEN}✓${NC} Dependencies downloaded"
}

# Start infrastructure
start_infra() {
    echo -e "\n${YELLOW}Starting infrastructure (PostgreSQL, Redis, RabbitMQ, MinIO)...${NC}"
    docker-compose -f deployments/docker/docker-compose.yaml up -d postgres redis rabbitmq minio
    echo -e "${GREEN}✓${NC} Infrastructure started"
    
    # Wait for PostgreSQL to be ready
    echo -e "${YELLOW}Waiting for PostgreSQL to be ready...${NC}"
    sleep 5
    until docker exec nimo-postgres pg_isready -U nimo -d nimo_plm > /dev/null 2>&1; do
        echo "Waiting for PostgreSQL..."
        sleep 2
    done
    echo -e "${GREEN}✓${NC} PostgreSQL is ready"
}

# Run migrations
run_migrations() {
    echo -e "\n${YELLOW}Running database migrations...${NC}"
    
    # Check if migrate is installed
    if ! command -v migrate &> /dev/null; then
        echo "Installing golang-migrate..."
        go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
    fi
    
    # Run migrations
    export DATABASE_URL="postgres://nimo:nimo123@localhost:5432/nimo_plm?sslmode=disable"
    migrate -path ./database/migrations -database "$DATABASE_URL" up || {
        # If migrate fails, try running SQL directly
        echo "Running migrations directly with psql..."
        docker exec -i nimo-postgres psql -U nimo -d nimo_plm < ./database/migrations/001_init_schema.sql
        docker exec -i nimo-postgres psql -U nimo -d nimo_plm < ./database/migrations/002_seed_data.sql
    }
    
    echo -e "${GREEN}✓${NC} Migrations completed"
}

# Build application
build_app() {
    echo -e "\n${YELLOW}Building application...${NC}"
    go build -o bin/nimo-plm ./cmd/server
    echo -e "${GREEN}✓${NC} Application built: bin/nimo-plm"
}

# Main
main() {
    check_prerequisites
    create_env
    download_deps
    start_infra
    run_migrations
    build_app
    
    echo -e "\n${GREEN}=============================${NC}"
    echo -e "${GREEN}Setup completed successfully!${NC}"
    echo -e "${GREEN}=============================${NC}"
    echo ""
    echo "To start the PLM service:"
    echo "  make run"
    echo ""
    echo "Or with hot reload:"
    echo "  make run-dev"
    echo ""
    echo "API will be available at: http://localhost:8080"
    echo ""
    echo "Other services:"
    echo "  - RabbitMQ Management: http://localhost:15672 (nimo/nimo123)"
    echo "  - MinIO Console: http://localhost:9001 (minioadmin/minioadmin123)"
}

# Run main
main
