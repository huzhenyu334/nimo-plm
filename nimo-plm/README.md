# nimo PLM System

nimo智能眼镜产品生命周期管理系统

## 技术栈

- **后端**: Go 1.22 + Gin
- **数据库**: PostgreSQL 16
- **缓存**: Redis 7
- **消息队列**: RabbitMQ 3.13
- **文件存储**: MinIO
- **认证**: 飞书OAuth + JWT

## 快速开始

### 前置要求

- Go 1.22+
- Docker & Docker Compose
- Make

### 开发环境启动

1. **克隆项目**
```bash
git clone https://github.com/bitfantasy/nimo-plm.git
cd nimo-plm
```

2. **配置环境变量**
```bash
cp .env.example .env
# 编辑 .env 配置飞书应用信息
```

3. **启动基础设施**
```bash
make docker-up
```

4. **执行数据库迁移**
```bash
make migrate-up
```

5. **运行服务**
```bash
make run
# 或使用热重载
make run-dev
```

6. **访问服务**
- API: http://localhost:8080
- RabbitMQ管理界面: http://localhost:15672 (nimo/nimo123)
- MinIO控制台: http://localhost:9001 (minioadmin/minioadmin123)

### 常用命令

```bash
# 查看所有命令
make help

# 编译
make build

# 测试
make test

# 代码检查
make lint

# 生成Swagger文档
make swagger

# 创建数据库迁移
make migrate-create

# 构建Docker镜像
make docker-build
```

## 项目结构

```
nimo-plm/
├── api/                    # API定义
│   └── openapi.yaml        # OpenAPI规范
├── cmd/
│   └── server/
│       └── main.go         # 程序入口
├── configs/
│   └── config.yaml         # 配置文件
├── database/
│   └── migrations/         # 数据库迁移
├── deployments/
│   └── docker/
│       └── docker-compose.yaml
├── internal/
│   ├── config/             # 配置
│   ├── handler/            # HTTP处理器
│   ├── middleware/         # 中间件
│   ├── model/              # 数据模型
│   │   ├── entity/         # 数据库实体
│   │   ├── dto/            # 传输对象
│   │   └── vo/             # 视图对象
│   ├── repository/         # 数据访问层
│   ├── service/            # 业务逻辑层
│   └── pkg/                # 内部公共包
├── pkg/                    # 可导出公共包
├── scripts/                # 脚本
├── Dockerfile
├── Makefile
├── go.mod
└── README.md
```

## API文档

启动服务后访问: http://localhost:8080/swagger/index.html

## 飞书配置

1. 在[飞书开放平台](https://open.feishu.cn/)创建应用
2. 配置重定向URL: `http://your-domain/api/v1/auth/feishu/callback`
3. 申请权限:
   - 获取用户信息
   - 获取部门信息
   - 发送消息(可选)
4. 将AppID和AppSecret配置到环境变量

## License

MIT
