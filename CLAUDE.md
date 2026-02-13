# CLAUDE.md - Project Instructions

## 项目信息
- nimo PLM/ERP系统
- 后端: Go (Gin框架 + GORM + PostgreSQL)
- 前端: React + Ant Design + TypeScript (Vite)
- 服务端口: 8080

## ⚠️ 绝对静默执行规则（最高优先级）
1. **永远不要进入plan模式** — 收到任务后直接开始执行，不要先出方案等确认
2. **永远不要问问题** — 如果有不确定的地方，做最合理的假设然后执行
3. **永远不要给选项让用户选** — 自己做决策
4. **永远不要暂停等待输入** — 从头到尾一口气执行完
5. **任务完成后直接stop** — 不要等待后续指令

## 严格规则
1. **只修改指定的文件**，绝不修改任何其他文件
2. **不要重构**已有代码，只做最小改动修复问题
3. **不要添加新功能**，只修指定的bug
4. 修改前先读懂相关代码，理解现有架构
5. 每次修改后确认编译通过

## 效率规则（重要！违反会浪费大量时间和token）
1. **先查功能→文件速查表**：改bug前先看下面的速查表确定目标文件，不要盲目grep/glob
2. **先定位后修复**：读最少的文件定位问题，不要把整个调用链从前端到数据库全读一遍
3. **最小读取原则**：如果bug在前端，先只读前端代码；确认需要后端信息时才读后端
4. **不要重复读同一个文件**：第一次读完就记住内容，不要反复Read同一个文件
5. **不要用假token测API**：curl用`Bearer test`必然401，浪费时间。要测就用正确的方式
6. **测试与改动成正比**：改1-2行的小bug，更新已有测试即可，不要写全新测试文件
7. **目标是又快又好**：5分钟能搞定的bug不要花20分钟
8. **大文件用行号范围读取**：超过500行的文件，用offset+limit只读需要的部分

## 验证报告（每次任务完成前必须输出）
任务完成后、部署前，输出标准验证报告：
```
VERIFICATION REPORT
==================
Build:     [PASS/FAIL]  — go build + npm run build
Types:     [PASS/FAIL]  — tsc --noEmit (前端) / go vet (后端)
Tests:     [PASS/FAIL]  — go test + npx playwright test (X/Y passed)
Deploy:    [PASS/FAIL]  — 服务重启 + HTTP 200
Changed:   X files (+Y -Z lines)

Overall:   [READY/NOT READY]
Issues:    (如有未解决的问题列在这里)
```
如果任何项FAIL，必须修复后重新验证，不能带着FAIL部署。

## 前端改动必须写UI测试（强制规则）
每次修改或新增前端功能，必须同时在 e2e/ 目录下新增或更新对应的Playwright UI测试：
1. **新功能** → 写新的 .spec.ts 文件，用浏览器打开页面验证UI元素存在、可交互
2. **改bug** → 在现有测试中追加回归断言，确保bug不复现
3. **改布局/样式** → 添加布局断言（元素可见、位置正确、不超出视口）
4. 测试必须通过真正的浏览器渲染（不是纯API测试）
5. 使用已有的测试登录helper（storageState）
6. 改完代码后先跑 `npx playwright test` 全量测试，全部通过才能部署

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

### 项目详情页组件（pages/ProjectDetail/ 目录）
| 组件 | 文件 | 行数范围 | 用途 |
|------|------|---------|------|
| ProjectDetail | ProjectDetail.tsx | 主组件 | Tab切换路由 |
| OverviewTab | OverviewTab.tsx | - | 项目概览 |
| GanttChart | GanttChart.tsx | - | 甘特图 |
| BOMTab | BOMTab.tsx | - | BOM管理(EBOM/SBOM) |
| SKUTab | SKUTab.tsx | - | SKU配色管理 |
| DocumentsTab | DocumentsTab.tsx | - | 文档管理 |
| DeliverablesTab | DeliverablesTab.tsx | - | 交付物管理 |
| ECNTab | ECNTab.tsx | - | ECN变更管理 |
| RoleAssignmentTab | RoleAssignmentTab.tsx | - | 角色分配 |
| TaskActions | TaskActions.tsx | - | 任务操作(确认/驳回) |
| FormSubmissionDisplay | FormSubmissionDisplay.tsx | - | 表单提交显示 |
| MaterialSearchModal | MaterialSearchModal.tsx | - | 物料搜索弹窗 |
| PhaseProgressBar | PhaseProgressBar.tsx | - | 阶段进度条 |

### SKU相关
| 文件 | 用途 |
|------|------|
| api/sku.ts | SKU API调用 |
| handler/sku_handler.go | SKU HTTP处理 |
| service/sku_service.go | SKU业务逻辑 |
| repository/sku_repository.go | SKU数据访问 |
| entity/sku.go | SKU实体(ProductSKU/SKUCMFConfig/SKUBOMItem) |

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
