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

## 效率规则（重要！）
1. **先定位后修复**：读最少的文件定位问题，不要把整个调用链从前端到数据库全读一遍
2. **最小读取原则**：如果bug在前端，先只读前端代码；确认需要后端信息时才读后端
3. **不要用假token测API**：curl用`Bearer test`必然401，浪费时间。要测就用正确的方式
4. **测试与改动成正比**：改1-2行的小bug，更新已有测试即可，不要写全新测试文件
5. **目标是又快又好**：5分钟能搞定的bug不要花20分钟

## 部署步骤（修完代码后执行）
```bash
cd /home/claw/.openclaw/workspace && go build -o bin/plm ./cmd/plm/
cd /home/claw/.openclaw/workspace/nimo-plm-web && npm run build
rm -rf /home/claw/.openclaw/workspace/web/plm/* && cp -r /home/claw/.openclaw/workspace/nimo-plm-web/dist/* /home/claw/.openclaw/workspace/web/plm/
kill $(pgrep -f "bin/plm" | head -1) 2>/dev/null; sleep 1 && cd /home/claw/.openclaw/workspace && nohup ./bin/plm > server.log 2>&1 &
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/
```

## 目录结构
- 后端入口: cmd/plm/main.go（路由注册、DB迁移、服务初始化）
- 前端路由: nimo-plm-web/src/routes/index.tsx
- 后端测试工具: internal/plm/testutil/testutil.go
- 前端Playwright配置: nimo-plm-web/playwright.config.ts

## 功能→文件速查表（改bug前先查这里！）

### PLM模块 (internal/plm/)
| 功能 | 前端页面 | 前端API | 后端handler | 后端service |
|------|---------|---------|------------|------------|
| 角色管理 | pages/RoleManagement.tsx | api/roles.ts | plm/handler/role_handler.go | plm/service/role_service.go |
| 项目管理 | pages/Projects.tsx | api/projects.ts | plm/handler/project_handler.go | plm/service/project_service.go |
| 任务管理 | pages/Projects.tsx(内嵌) | api/projects.ts | plm/handler/project_handler.go | plm/service/project_service.go |
| BOM管理 | pages/BomManagement.tsx | api/bom.ts | plm/handler/bom_handler.go | plm/service/bom_service.go |
| 产品管理 | pages/Products.tsx | api/products.ts | plm/handler/product_handler.go | plm/service/product_service.go |
| 物料管理 | pages/Materials.tsx | api/materials.ts | plm/handler/material_handler.go | plm/service/material_service.go |
| 模板管理 | pages/Templates.tsx | api/templates.ts | plm/handler/template_handler.go | plm/service/template_service.go |
| 登录/认证 | pages/Login.tsx | api/auth.ts | plm/handler/auth_handler.go | - |

### SRM模块 (internal/srm/)
| 功能 | 前端页面 | 前端API | 后端handler | 后端service |
|------|---------|---------|------------|------------|
| 采购看板 | pages/srm/KanbanBoard.tsx | api/srm.ts | srm/handler/pr_item_handler.go | srm/service/pr_item_service.go |
| 采购总览 | pages/srm/SRMDashboard.tsx | api/srm.ts | srm/handler/dashboard_handler.go | srm/service/dashboard_service.go |
| 供应商管理 | pages/srm/Suppliers.tsx | api/srm.ts | srm/handler/supplier_handler.go | srm/service/supplier_service.go |
| 采购申请 | pages/srm/PurchaseRequests.tsx | api/srm.ts | srm/handler/pr_handler.go | srm/service/procurement_service.go |
| 采购订单 | pages/srm/PurchaseOrders.tsx | api/srm.ts | srm/handler/po_handler.go | srm/service/procurement_service.go |
| 来料检验 | pages/srm/Inspections.tsx | api/srm.ts | srm/handler/inspection_handler.go | srm/service/inspection_service.go |
| 询价(RFQ) | - | api/srm.ts | srm/handler/rfq_handler.go | srm/service/rfq_service.go |
| SRM项目 | pages/srm/Projects.tsx | api/srm.ts | srm/handler/project_handler.go | srm/service/project_service.go |

### 前端通用
| 文件 | 用途 |
|------|------|
| src/layouts/MainLayout.tsx | 侧边栏菜单 |
| src/contexts/AuthContext.tsx | 登录状态管理 |
| src/api/client.ts | Axios实例+拦截器 |
| src/types/index.ts | TypeScript类型定义 |

### 重要提示
- SRM的API全在 api/srm.ts 一个文件里（不像PLM分开的）
- 前端用 Ant Design v5 + App.useApp()（不要用静态Modal.confirm/message）
- 后端用 Gin + GORM + PostgreSQL

## 测试规则（铁律 — 必须严格遵守）
1. 每次后端代码变更，必须编写或更新对应的 Go test
2. 每次前端代码变更，必须编写或更新对应的 Playwright e2e test
3. 新功能必须有测试覆盖，bug修复必须有回归测试
4. **任务完成前必须自己运行全部测试，测试全部通过才算任务完成**
5. 如果测试失败，必须修复代码直到测试通过，不允许带着失败的测试结束任务

### 任务完成检查清单（每次任务结束前必须执行）
```bash
# Step 1: 编译通过
cd /home/claw/.openclaw/workspace && go build -o bin/plm ./cmd/plm/

# Step 2: 后端测试全部通过
go test ./internal/plm/... -v

# Step 3: 前端编译通过
cd /home/claw/.openclaw/workspace/nimo-plm-web && npm run build

# Step 4: 前端E2E测试全部通过
npx playwright test

# Step 5: 部署并验证服务启动
# （按下方部署步骤执行）
```
**以上5步全部通过后，任务才算完成。任何一步失败都必须修复后重试。**

## 测试命令
```bash
# 后端测试
go test ./internal/plm/... -v

# 前端E2E测试
cd /home/claw/.openclaw/workspace/nimo-plm-web && npx playwright test

# 后端单模块测试
go test ./internal/plm/handler/ -v -run TestRole
go test ./internal/plm/handler/ -v -run TestProject
go test ./internal/plm/handler/ -v -run TestFeishu
```
