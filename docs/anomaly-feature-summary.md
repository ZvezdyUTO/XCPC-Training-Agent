# 训练异常预警前端对接说明（给前端同学）

## 1. 你要做什么

目标：在现有教练端里补一套“异常预警”功能，包含：

1. 预警列表页（筛选、查看证据、查看建议）
2. 预警状态流转（`new -> ack -> resolved`）
3. 手动触发检测
4. 规则参数配置页（部分更新）
5. 与同步功能联动（可选：同步后立即检测）

你不需要改后端逻辑，直接调用接口即可。

---

## 2. 接口清单（前端只用这些）

## 2.1 预警检测

1. `POST /v1/admin/op/training/detect/run`
- 用途：手动触发一次检测（推荐在“预警页”放一个按钮）
- 响应重点：`data.alert_cnt`, `data.detected_at`

2. `POST /v1/admin/anomaly/detect/run`
- 用途：同上（与上面语义一致，二选一即可）

## 2.2 预警列表与状态

1. `GET /v1/admin/alerts/list`
- 查询参数：
  - `student_id`
  - `status`: `new | ack | resolved`
  - `severity`: `low | medium | high`
  - `from`, `to`（`YYYY-MM-DD`）
  - `page`, `count`
- 响应重点：
  - `data.count`
  - `data.list[].id`
  - `data.list[].title`
  - `data.list[].evidence`（对象）
  - `data.list[].actions`（数组）

2. `POST /v1/admin/alerts/:id/ack`
- 用途：将预警标记为“已确认”

3. `POST /v1/admin/alerts/:id/resolve`
- 用途：将预警标记为“已处理完成”

## 2.3 规则配置

1. `GET /v1/admin/anomaly/config`
- 用途：读取当前检测阈值配置

2. `POST /v1/admin/anomaly/config`
- 用途：部分更新配置（只传要改的字段）
- 例子：只改一项
```json
{
  "inactive_days_threshold": 2
}
```

## 2.4 同步联动（可选）

1. `POST /v1/admin/op/training/syncall`
2. `POST /v1/admin/op/training/syncone`

这两个接口请求体可加：
```json
{
  "detect_after_sync": true
}
```
响应会返回 `alert_cnt`。

---

## 3. 前端页面拆分建议

## 3.1 新增 `AlertsPage`

建议放在导航一级菜单，路径如 `/alerts`。

页面分区建议：

1. 顶部操作区
- “立即检测”按钮（调用 `POST /v1/admin/op/training/detect/run`）
- 检测结果提示（本次新增预警数量）

2. 筛选区
- 学号输入
- 状态下拉（new/ack/resolved）
- 等级下拉（low/medium/high）
- 日期范围（from/to）
- 分页（page/count）

3. 列表区
- 关键信息列：学号、日期、类型、等级、状态、标题
- 展开内容：`evidence`、`actions`

4. 行操作
- 当状态是 `new`：显示“确认（ack）”
- 当状态是 `new` 或 `ack`：显示“完成（resolve）”
- 操作后刷新当前列表

## 3.2 配置页（可与 AlertsPage 同页）

最小可用做法：

1. 进入页面时 `GET /v1/admin/anomaly/config`
2. 表单显示配置项
3. 点保存时只提交修改过的字段（patch）

---

## 4. 字段展示建议

## 4.1 状态显示

- `new` -> `新预警`
- `ack` -> `已确认`
- `resolved` -> `已处理`

## 4.2 等级显示

- `low` -> `低`
- `medium` -> `中`
- `high` -> `高`

## 4.3 `evidence` 显示

推荐用 `<details>` + JSON pretty 展示，先保真再美化。

## 4.4 `actions` 显示

按数组渲染成 1..N 条建议。

---

## 5. 类型定义建议（TS）

建议在 `frontend/src/shared/types.ts` 增加：

1. `AnomalyConfig`
2. `AlertListItem`
3. `AlertListPayload`
4. `RunDetectPayload`

关键注意：

- `evidence` 用 `Record<string, unknown>`
- `actions` 用 `string[]`

---

## 6. API 封装建议

建议在 `frontend/src/shared/api.ts` 增加：

1. `runTrainingDetect(token)`
2. `listAlerts(token, query)`
3. `ackAlert(token, id)`
4. `resolveAlert(token, id)`
5. `getAnomalyConfig(token)`
6. `updateAnomalyConfig(token, patch)`

其中 `updateAnomalyConfig` 的第二个参数必须是“部分字段对象”。

---

## 7. 联调顺序（推荐）

1. 先做“立即检测”按钮，确认能拿到 `alert_cnt`
2. 做列表查询和筛选
3. 做 `ack/resolve`，保证状态刷新正确
4. 做配置读取和部分更新
5. 最后做同步页上的 `detect_after_sync`

---

## 8. 验收标准（前端侧）

1. 可在页面触发检测并看到结果数量
2. 列表能筛选并展示 `evidence/actions`
3. `ack/resolve` 能正确改变状态
4. 配置修改支持“只改一个字段”
5. 刷新页面后数据一致，接口错误提示可读（中文）

---

## 9. 项目内落点路径（给 AI/新同学直接开工）

建议按以下文件落地：

1. 路由接入
- `frontend/src/app/App.tsx`
- 新增路由：`/alerts`

2. 侧边栏导航
- `frontend/src/components/AppShell.tsx`
- 新增菜单项：`预警`（指向 `/alerts`）

3. 新页面
- `frontend/src/pages/AlertsPage.tsx`
- 页面内容：检测按钮、筛选区、列表区、`ack/resolve` 操作、配置区

4. API 封装
- `frontend/src/shared/api.ts`
- 新增：
  - `runTrainingDetect`
  - `listAlerts`
  - `ackAlert`
  - `resolveAlert`
  - `getAnomalyConfig`
  - `updateAnomalyConfig`

5. 类型定义
- `frontend/src/shared/types.ts`
- 新增：
  - `AlertListItem`
  - `AlertListPayload`
  - `RunDetectPayload`
  - `AnomalyConfig`
  - `AnomalyConfigPatch`

6. 样式（按现有风格复用）
- 优先复用 `frontend/src/shared/styles.css` 里的现有卡片、表格、按钮样式类
- 尽量保持和 `QueryPage`/`AgentPage` 的布局风格一致

7. 同步页联动开关（可选增强）
- `frontend/src/pages/StudentsPage.tsx`
- 在“触发训练同步”动作附近增加 `detect_after_sync` 开关并透传给 API

---

## 10. 给 AI 的最小提示词模板（可直接复制）

```text
请在这个项目中实现异常预警前端功能，要求：
1) 新增 AlertsPage，路径 /alerts，并在侧边栏加入入口。
2) 对接后端接口：
   - POST /v1/admin/op/training/detect/run
   - GET /v1/admin/alerts/list
   - POST /v1/admin/alerts/:id/ack
   - POST /v1/admin/alerts/:id/resolve
   - GET /v1/admin/anomaly/config
   - POST /v1/admin/anomaly/config（部分更新）
3) 在 frontend/src/shared/types.ts 增加对应类型；
   在 frontend/src/shared/api.ts 增加对应 API 方法。
4) AlertsPage 需要：筛选、列表、evidence/actions 展示、ack/resolve 按钮、配置编辑与保存。
5) 保持现有项目 UI 风格，不要大改全局样式。
6) 完成后说明改了哪些文件，并给出手动测试步骤。
```
