# 基础设施

## 服务器
- 腾讯云 OpenCloudOS，IP 43.134.86.237
- 2核CPU，7.5GB RAM
- 限制：不能并行编译+E2E测试，CPU会拉满

## 服务
| 服务 | 端口 | 管理方式 |
|------|------|---------|
| PLM | 8080 | 手动 nohup |
| ACP | 3001 | systemd --user |
| Mission Control前端 | 3001(已被ACP占) | Docker |
| Mission Control后端 | 8001 | Docker |

## Mission Control
- 项目：abhi1693/openclaw-mission-control
- 部署目录：/home/claw/.openclaw/workspace/openclaw-mission-control
- Auth token：a90714a6... (LOCAL_AUTH_TOKEN)
- 关键修复：Gateway device identity认证（Ed25519签名）
- Redis端口注意：环境变量REDIS_PORT=6379会覆盖.env的6380

## 技术生态评估
- Antfarm：已卸载，泽斌认为没用（CC原生能做同样的事）
- Moltworker：Cloudflare部署OpenClaw，$10-35/月
- AionUi（⭐16.4K）：多Agent桌面客户端
- Claude Flow（⭐14.1K）：Claude多Agent编排
