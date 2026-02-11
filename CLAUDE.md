# CLAUDE.md - Project Instructions

## 项目信息
- nimo PLM/ERP系统
- 后端: Go (Gin框架 + GORM + PostgreSQL)
- 前端: React + Ant Design + TypeScript (Vite)
- 服务端口: 8080

## 严格规则
1. **只修改指定的文件**，绝不修改任何其他文件
2. **不要重构**已有代码，只做最小改动修复问题
3. **不要添加新功能**，只修指定的bug
4. 修改前先读懂相关代码，理解现有架构
5. 每次修改后确认编译通过

## 部署步骤（修完代码后执行）
```bash
cd /home/claw/.openclaw/workspace && go build -o bin/plm ./cmd/plm/
cd /home/claw/.openclaw/workspace/nimo-plm-web && npm run build
rm -rf /home/claw/.openclaw/workspace/web/plm/* && cp -r /home/claw/.openclaw/workspace/nimo-plm-web/dist/* /home/claw/.openclaw/workspace/web/plm/
kill $(pgrep -f "bin/plm" | head -1) 2>/dev/null; sleep 1 && cd /home/claw/.openclaw/workspace && nohup ./bin/plm > server.log 2>&1 &
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/
```

## 目录结构
- 后端入口: cmd/plm/main.go
- 后端handler: internal/plm/handler/
- 后端service: internal/plm/service/
- 后端entity: internal/plm/entity/
- 前端页面: nimo-plm-web/src/pages/
- 前端API: nimo-plm-web/src/api/
- 前端路由: nimo-plm-web/src/routes/index.tsx
