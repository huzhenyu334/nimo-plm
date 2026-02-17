# AGENTS.md - nimo PLM/ERP 项目编码规范

CC（Claude Code）每次启动自动读取此文件，必须遵循以下规范。

## 技术栈
- 后端: Pure Go + Gin + GORM + PostgreSQL
- 前端: React + TypeScript + Ant Design + React Query + Vite
- 飞书集成: cli_a9efa9afff78dcb5

## 编码规范

### 可编辑表格（重要！）
**凡是需要可编辑表格的地方，必须使用 `EditableTable` 组件 + onChange 本地状态模式。**

- 表格UI: `src/components/EditableTable.tsx`（通用 click-to-edit 表格）
- 数据流: onChange 更新本地 useState → UI 即时反映 → useEffect debounce 自动保存到API
- **禁止**: 每次编辑/添加/删除都直接调 useMutation + invalidateQueries 等API返回再刷新
- **原因**: 用户体验差，有1秒以上延迟

已使用此模式的组件（参考实现）：
- BOM管理: `src/pages/BOMManagementDetail.tsx` + `src/components/BOM/EBOMControl.tsx`
- CMF管理: `src/components/CMFEditControl.tsx`

### 组件复用
- BOM控件: EBOMControl/PBOMControl/MBOMControl 共用 DynamicBOMTable → EditableTable
- CMF控件: CMFEditControl 直接用 EditableTable
- 新功能如需表格编辑，直接用 EditableTable，不要重新造轮子

### API模式
- 获取数据: useQuery (react-query)，合理设置 staleTime 缓存
- 保存数据: debounce 自动保存（检测新增/修改/删除，分别调对应API）
- 乐观更新: 先更新本地state，API失败时 invalidateQueries 回滚

### 模板/扩展字段
- 模板数据用 useQuery 缓存（staleTime 5分钟），不要每个组件单独 fetch
- 模板未加载完前显示 loading，不要分两步渲染（先基础字段再扩展字段）

## 编译部署
- 后端: `cd /home/claw/.openclaw/workspace && go build -o bin/plm ./cmd/plm/`
- 前端: `cd nimo-plm-web && npm run build && rm -rf ../web/plm/* && cp -r dist/* ../web/plm/`
- 重启后端: `kill $(pgrep -f 'bin/plm') 2>/dev/null; sleep 1; nohup ./bin/plm > server.log 2>&1 &`

## Git Commit（每次任务完成后必须执行）
```bash
git add -A -- ':!.openclaw/' ':!internal/plm/handler/uploads/' ':!nimo-plm-web/playwright-report/' ':!nimo-plm-web/screenshots/' ':!nimo-plm-web/test-results/' ':!uploads/'
git commit -m "<简洁描述本次改动>"
git push
```
