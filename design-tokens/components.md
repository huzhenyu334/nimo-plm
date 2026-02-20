# 组件使用规范 - nimo PLM/ERP

> 由 UX Agent 维护。CC开发时必须遵守。

## 按钮（Button）

| 场景 | 类型 | 示例 |
|------|------|------|
| 主操作（每页最多1个） | `type="primary"` | 新建项目、提交审批 |
| 次操作 | `type="default"` | 编辑、导出 |
| 危险操作 | `danger` | 删除、撤销 |
| 文字按钮（表格行内） | `type="link"` | 查看、编辑 |

**文案约定：**
- "新建" 不是 "添加" / "创建"
- "保存" 不是 "提交"（表单保存用"保存"，审批提交用"提交"）
- "删除" 必须有二次确认

## 表格（Table/ProTable）

- 使用 `ProTable` 而非原生 `Table`
- 顶部：搜索筛选栏（`search` prop）
- 右上角：操作按钮（toolBarRender）
- 行操作：用 `type="link"` 按钮，不用下拉菜单（≤3个操作时）
- 列宽：固定关键列宽度，剩余列自适应
- 对齐：文字左对齐，数字右对齐，状态居中
- 分页：默认 `pageSize: 20`

## 表单（Form/ProForm）

- 使用 `ProForm` 系列组件
- 标签位置：`layout="vertical"`（移动端友好）
- 分组：用 `Card` 分区，每个Card一个逻辑分组
- 必填标记：Ant Design自带红色星号
- 校验：实时校验 + 提交时二次校验
- 保存方式：onChange debounce自动保存（可编辑表格）或 手动保存按钮

## 详情页（Descriptions）

- 基本信息用 `Descriptions` 组件
- 多维信息用 `Tabs` 分区
- 状态展示用 `Tag`（颜色语义化）
- 时间线用 `Timeline` 组件

## 状态展示（Tag）

| 状态类型 | 颜色 |
|---------|------|
| 进行中 | `blue` / `processing` |
| 已完成 | `green` / `success` |
| 待处理 | `orange` / `warning` |
| 已关闭/已拒绝 | `red` / `error` |
| 草稿 | `default`（灰色）|

## 空状态

- 列表无数据：`<Empty description="暂无数据" />`
- 搜索无结果：`<Empty description="未找到匹配的结果" />`
- 新功能引导：`<Empty description="还没有xxx，点击新建" />`+ 新建按钮

## 加载状态

- 首次加载页面：`Skeleton`（骨架屏）
- 局部数据刷新：`Spin`
- 按钮提交中：`loading` prop
- 表格加载：ProTable内置loading

## 通知/反馈

- 操作成功：`message.success`（顶部轻提示，2s自动消失）
- 操作失败：`message.error`
- 需要确认：`Modal.confirm`
- 重要通知：`notification`（右上角，需手动关闭）

## 布局

- 侧边栏：`ProLayout` + `Menu`，支持折叠
- 面包屑：所有二级及以下页面必须有
- 页面结构：`PageContainer`（ProComponents）包裹内容
