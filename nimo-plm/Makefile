.PHONY: all build test lint run docker clean migrate help

# 变量
APP_NAME := nimo-plm
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Go参数
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOLINT := golangci-lint

# 目录
CMD_DIR := ./cmd/server
BIN_DIR := ./bin
MIGRATION_DIR := ./database/migrations

# Docker
DOCKER_IMAGE := $(APP_NAME)
DOCKER_TAG := $(VERSION)

# 数据库
DATABASE_URL ?= postgres://nimo:nimo123@localhost:5432/nimo_plm?sslmode=disable

## help: 显示帮助信息
help:
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

## all: 执行lint、test、build
all: lint test build

## build: 编译二进制文件
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(APP_NAME) $(CMD_DIR)
	@echo "Build complete: $(BIN_DIR)/$(APP_NAME)"

## build-linux: 交叉编译Linux版本
build-linux:
	@echo "Building $(APP_NAME) for Linux..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(APP_NAME)-linux-amd64 $(CMD_DIR)

## test: 运行测试
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	@echo "Test complete"

## test-coverage: 生成测试覆盖率报告
test-coverage: test
	@$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## lint: 运行代码检查
lint:
	@echo "Running linter..."
	$(GOLINT) run ./...

## run: 本地运行服务
run:
	@echo "Starting $(APP_NAME)..."
	$(GOCMD) run $(CMD_DIR)/main.go

## run-dev: 使用air热重载运行(需要安装air)
run-dev:
	@air -c .air.toml

## deps: 下载依赖
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

## deps-update: 更新依赖
deps-update:
	@echo "Updating dependencies..."
	$(GOGET) -u ./...
	$(GOMOD) tidy

## docker-build: 构建Docker镜像
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):latest

## docker-push: 推送Docker镜像
docker-push:
	@echo "Pushing Docker image..."
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	docker push $(DOCKER_IMAGE):latest

## docker-up: 启动开发环境(docker-compose)
docker-up:
	@echo "Starting development environment..."
	docker-compose -f deployments/docker/docker-compose.yaml up -d

## docker-down: 停止开发环境
docker-down:
	@echo "Stopping development environment..."
	docker-compose -f deployments/docker/docker-compose.yaml down

## docker-logs: 查看容器日志
docker-logs:
	docker-compose -f deployments/docker/docker-compose.yaml logs -f

## migrate-up: 执行数据库迁移
migrate-up:
	@echo "Running migrations..."
	migrate -path $(MIGRATION_DIR) -database "$(DATABASE_URL)" up

## migrate-down: 回滚最近一次迁移
migrate-down:
	@echo "Rolling back migration..."
	migrate -path $(MIGRATION_DIR) -database "$(DATABASE_URL)" down 1

## migrate-status: 查看迁移状态
migrate-status:
	@echo "Migration status:"
	migrate -path $(MIGRATION_DIR) -database "$(DATABASE_URL)" version

## migrate-create: 创建新的迁移文件
migrate-create:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir $(MIGRATION_DIR) -seq $$name

## swagger: 生成Swagger文档
swagger:
	@echo "Generating Swagger docs..."
	swag init -g cmd/server/main.go -o api/docs

## mock: 生成Mock文件
mock:
	@echo "Generating mocks..."
	mockery --all --output internal/mocks

## wire: 生成依赖注入代码
wire:
	@echo "Generating wire..."
	wire ./internal/wire/...

## clean: 清理构建文件
clean:
	@echo "Cleaning..."
	@rm -rf $(BIN_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

## install-tools: 安装开发工具
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	go install github.com/vektra/mockery/v2@latest
	go install github.com/google/wire/cmd/wire@latest
	go install github.com/cosmtrek/air@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@echo "Tools installed"
