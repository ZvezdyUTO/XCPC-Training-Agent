# 接口文档

## 基本信息

- Base URL: `http://localhost:8888`
- API Prefix: `/v1`

统一响应结构：

```json
{
  "code": 200,
  "data": {},
  "msg": "success",
  "err_code": ""
}
```

字段说明：

- `code`：业务状态码，成功固定为 `200`
- `data`：接口返回数据
- `msg`：成功或失败消息
- `err_code`：稳定错误码，成功时为空

## 认证

- 公开接口：无需认证
- 用户接口：需要 `Authorization: Bearer <TOKEN>`
- 管理员接口：需要 JWT 且具备管理员权限

## 接口总览

### 公开接口

- `POST /v1/user/login`

### 用户接口

- `GET /v1/user/me`
- `POST /v1/user/password`
- `DELETE /v1/user/me`

### 管理员接口

- `GET /v1/admin/users/list`
- `POST /v1/admin/users/create`
- `DELETE /v1/admin/users/:id`
- `POST /v1/admin/op/training/syncall`
- `POST /v1/admin/op/training/syncone`
- `POST /v1/admin/op/training/detect/run`
- `GET /v1/admin/op/training/syncstate/list`
- `GET /v1/admin/op/training/summary`
- `GET /v1/admin/op/training/leaderboard`
- `GET /v1/admin/op/contest/ranking`
- `POST /v1/admin/anomaly/detect/run`
- `GET /v1/admin/anomaly/config`
- `POST /v1/admin/anomaly/config`
- `GET /v1/admin/alerts/list`
- `POST /v1/admin/alerts/:id/ack`
- `POST /v1/admin/alerts/:id/resolve`
- `POST /v1/admin/agent/task/run`

## 公开接口

### `POST /v1/user/login`

用户登录。

请求体：

```json
{
  "username": "20001",
  "password": "000000"
}
```

返回重点：

- `data.user`
- `data.token`

## 用户接口

### `GET /v1/user/me`

获取当前登录用户信息。

### `POST /v1/user/password`

修改当前用户密码。

请求体：

```json
{
  "oldPwd": "old",
  "newPwd": "new"
}
```

### `DELETE /v1/user/me`

注销当前用户。

## 管理员接口

### 用户管理

### `GET /v1/admin/users/list`

查询用户列表。

可用查询参数：

- `ids`
- `name`
- `page`
- `count`

### `POST /v1/admin/users/create`

批量创建用户。

请求体：

```json
{
  "users": [
    {
      "id": "230000001",
      "name": "张三",
      "password": "123456",
      "cf_handle": "demo_cf",
      "ac_handle": "demo_ac"
    }
  ]
}
```

### `DELETE /v1/admin/users/:id`

删除指定用户。

约束：

- 不允许删除当前登录用户自身
- 不允许删除系统用户

说明：

- 按学号删除用户
- 会硬删除该用户本身
- 依赖数据库外键级联删除其比赛记录、训练记录和同步状态数据

### 数据同步

### `POST /v1/admin/op/training/syncall`

触发训练数据同步。

请求体（可选）：

```json
{
  "detect_after_sync": true
}
```

说明：

- 遍历所有有效学生
- 自动判断全量同步、区间同步或跳过
- `detect_after_sync=true` 时，同步完成后会立即执行一次异常检测

成功响应重点：

- `data.msg`
- `data.success_cnt`
- `data.success[].student_id`
- `data.success[].mode`
- `data.failed[].student_id`
- `data.failed[].error`
- `data.alert_cnt`：本次同步后检测写入/更新的预警数量（仅当 `detect_after_sync=true`）

### `POST /v1/admin/op/training/syncone`

触发单个学生的训练数据同步。

请求体：

```json
{
  "student_id": "230000001",
  "detect_after_sync": true
}
```

返回重点：

- `data.student_id`
- `data.mode`：`full` / `range` / `skip`
- `data.alert_cnt`：本次同步后检测写入/更新的预警数量（仅当 `detect_after_sync=true`）

### `POST /v1/admin/op/training/detect/run`

手动触发一次训练异常检测（无需先执行同步）。

成功响应重点：

- `data.msg`
- `data.alert_cnt`
- `data.detected_at`

### `GET /v1/admin/op/training/syncstate/list`

查询同步状态表。

返回重点：

- `data.count`
- `data.list[].student_id`
- `data.list[].is_fully_initialized`
- `data.list[].latest_successful_date`

### `GET /v1/admin/op/training/summary`

查询单个学生在指定时间范围内的训练累计与训练价值评分。

查询参数：

- `student_id`
- `from`
- `to`

成功响应重点：

- `data.cf_total`
- `data.cf_distribution`
- `data.ac_total`
- `data.ac_distribution`
- `data.training_value`

`data.training_value` 字段说明：

- `scoring_version`：当前训练价值评分版本
- `solved_total`：总题量
- `score`：综合分
- `volume_score`：题量分
- `difficulty_score`：难度分
- `challenge_score`：挑战分
- `undefined_total` / `undefined_ratio`：未标难度题数量与占比
- `cf_rating` / `ac_rating`：当前分、峰值分与能力参考线
- `cf` / `ac`：分平台的题量、已知题、undefined 和分数拆解

### `GET /v1/admin/op/training/leaderboard`

查询指定时间范围内的训练价值排行榜。

查询参数：

- `from`
- `to`
- `top_n`

成功响应重点：

- `data.scoring_version`
- `data.from`
- `data.to`
- `data.top_n`
- `data.count`
- `data.items[]`

说明：

- 排行榜与单人训练查询复用同一套评分公式
- 评分同时考虑题量、难度和相对本人能力线的挑战价值
- `undefined` 题会谨慎参与折扣估计，不直接按高难题处理

### `GET /v1/admin/op/contest/ranking`

查询某一场比赛在数据库中的队内排名。

查询参数：

- `platform`：`CF` 或 `AC`
- `contest_id`

成功响应重点：

- `data.contest_name`
- `data.contest_date`
- `data.count`
- `data.items[].student_id`
- `data.items[].student_name`
- `data.items[].rank`
- `data.items[].old_rating`
- `data.items[].new_rating`
- `data.items[].rating_change`

### 异常预警

### `POST /v1/admin/anomaly/detect/run`

手动触发一次异常检测。

说明：

- 与 `/v1/admin/op/training/detect/run` 语义一致，均可用于独立检测

成功响应重点：

- `data.msg`
- `data.alert_cnt`
- `data.detected_at`

### `GET /v1/admin/anomaly/config`

查询当前异常检测规则参数。

成功响应重点：

- `data.current_window_days`
- `data.baseline_window_days`
- `data.baseline_min_daily`
- `data.current_min_daily_for_alert`
- `data.drop_low_threshold`
- `data.drop_medium_threshold`
- `data.drop_high_threshold`
- `data.inactive_days_threshold`
- `data.inactive_baseline_min_daily`
- `data.difficulty_drop_current_window_days`
- `data.difficulty_drop_baseline_window_days`
- `data.difficulty_drop_min_current_total`
- `data.difficulty_drop_min_baseline_high_ratio`
- `data.difficulty_level_round_base`
- `data.difficulty_relative_high_delta`
- `data.difficulty_relative_easy_delta`
- `data.difficulty_drop_low_threshold`
- `data.difficulty_drop_medium_threshold`
- `data.difficulty_drop_high_threshold`

### `POST /v1/admin/anomaly/config`

更新异常检测规则参数（支持部分字段更新）。

请求体示例（完整）：

```json
{
  "current_window_days": 7,
  "baseline_window_days": 30,
  "baseline_min_daily": 1.0,
  "current_min_daily_for_alert": 2.0,
  "drop_low_threshold": 0.35,
  "drop_medium_threshold": 0.5,
  "drop_high_threshold": 0.7,
  "inactive_days_threshold": 3,
  "inactive_baseline_min_daily": 1.0,
  "difficulty_drop_current_window_days": 7,
  "difficulty_drop_baseline_window_days": 30,
  "difficulty_drop_min_current_total": 5,
  "difficulty_drop_min_baseline_high_ratio": 0.15,
  "difficulty_level_round_base": 100,
  "difficulty_relative_high_delta": 200,
  "difficulty_relative_easy_delta": 200,
  "difficulty_drop_low_threshold": 0.35,
  "difficulty_drop_medium_threshold": 0.5,
  "difficulty_drop_high_threshold": 0.7
}
```

请求体示例（只修改一项）：

```json
{
  "inactive_days_threshold": 2
}
```

说明：

- 参数非法时会返回中文错误提示，例如阈值范围或大小关系不合法。
- 配置会持久化到数据库，服务重启后仍会保留最近一次保存的配置。
- 未传入的字段保持原值不变。

### `GET /v1/admin/alerts/list`

查询异常预警列表。

可用查询参数：

- `student_id`
- `status`：`new` / `ack` / `resolved`
- `severity`：`low` / `medium` / `high`
- `from`：开始日期，格式 `YYYY-MM-DD`
- `to`：结束日期，格式 `YYYY-MM-DD`
- `page`
- `count`

成功响应重点：

- `data.count`
- `data.list[].id`
- `data.list[].student_id`
- `data.list[].alert_date`
- `data.list[].alert_type`
- `data.list[].severity`
- `data.list[].status`
- `data.list[].title`
- `data.list[].evidence`
- `data.list[].actions`
- `data.list[].created_at`
- `data.list[].updated_at`

### `POST /v1/admin/alerts/:id/ack`

将某条预警标记为“已确认”。

说明：

- `:id` 为预警 ID
- 成功返回统一 `success` 响应

### `POST /v1/admin/alerts/:id/resolve`

将某条预警标记为“已处理完成”。

说明：

- `:id` 为预警 ID
- 成功返回统一 `success` 响应

### Agent 分析

### `POST /v1/admin/agent/task/run`

执行一次 Agent 分析任务。

请求体：

```json
{
  "task": "分析<示例学号>学生最近训练情况",
  "params": {
    "memory_paths": [
      "xcpc/training"
    ]
  },
  "trace_mode": "summary"
}
```

字段说明：

- `task`：必填，任务描述
- `params`：可选，附加参数对象，会原样传给 Agent
- `params.memory_paths`：可选，memory 路径提示；也兼容 `context_paths` 或 `paths`
- `trace_mode`：可选，支持 `none`、`summary`、`debug`

`trace_mode` 行为：

- 省略或传 `none`：不在 HTTP 响应中返回 `trace`
- 传 `summary`：返回摘要级 trace
- 传 `debug`：返回更完整的调试级 trace

注意：

- 即使不返回 `trace`，服务内部仍会记录一份摘要级运行日志
- Agent 最终结果由模型原生结构化输出生成，接口期望返回合法 JSON 结果对象

成功响应示例（默认 `trace_mode=none`）：

```json
{
  "code": 200,
  "data": {
    "task": "分析某个学生最近训练情况",
    "result": {
      "decision_type": "student_focus",
      "focus_students": ["<示例学号>"],
      "confidence": 0.92,
      "report": "该同学最近一周训练活跃，但比赛表现波动较大，建议重点复盘最近两场比赛。",
      "metrics": {
        "training_days": 6,
        "contest_count": 2
      }
    },
    "token_usage": {
      "model_call_count": 3,
      "input_tokens": 2410,
      "output_tokens": 516,
      "total_tokens": 2926
    }
  },
  "msg": "success"
}
```

成功响应示例（`trace_mode=summary`）：

```json
{
  "code": 200,
  "data": {
    "task": "分析某个学生最近训练情况",
    "result": {
      "decision_type": "student_focus",
      "focus_students": ["<示例学号>"],
      "confidence": 0.92,
      "report": "该同学最近一周训练活跃，但比赛表现波动较大，建议重点复盘最近两场比赛。",
      "metrics": {
        "training_days": 6,
        "contest_count": 2
      }
    },
    "token_usage": {
      "model_call_count": 3,
      "input_tokens": 2410,
      "output_tokens": 516,
      "total_tokens": 2926
    },
    "trace": {
      "run_id": "run_xxx",
      "mode": "summary",
      "started_at": "2026-03-30T12:00:00+08:00",
      "finished_at": "2026-03-30T12:00:02+08:00",
      "token_usage": {
        "model_call_count": 3,
        "input_tokens": 2410,
        "output_tokens": 516,
        "total_tokens": 2926
      },
      "spans": [],
      "events": []
    }
  },
  "msg": "success"
}
```

`data.result` 字段说明：

- `decision_type`：结果类型，例如重点关注、正常观察、批量分析等
- `focus_students`：需要重点关注的学号列表
- `confidence`：模型对当前结果的置信度
- `report`：可直接展示给管理员的分析结论
- `metrics`：结构化指标补充，字段名不固定

`data.token_usage` 字段说明：

- `model_call_count`：本次运行的模型调用次数
- `input_tokens`：输入 token 总量
- `output_tokens`：输出 token 总量
- `total_tokens`：总 token 数

`data.trace` 字段说明：

- `run_id`：本次运行唯一标识
- `mode`：trace 粒度，`summary` 或 `debug`
- `started_at` / `finished_at`：运行时间范围
- `token_usage`：本次运行累计 token 统计
- `spans`：耗时区间列表
- `events`：关键事件列表

失败响应示例：

```json
{
  "code": 50000,
  "data": {},
  "msg": "缺少 OPENAI_API_KEY 或 OPENAI_BASE_URL 配置",
  "err_code": "openai_config_missing"
}
```

常见错误码：

- `openai_config_missing`：缺少模型配置
- `openai_base_url_invalid`：模型地址配置非法
- `invalid_api_key`：模型 API Key 无效
- `llm_request_failed`：模型请求失败
- `llm_response_invalid_json`：模型响应不是合法 JSON
- `internal_error`：服务内部错误

返回重点：

- `data.task`
- `data.result`
- `data.token_usage`
- `data.trace`：仅在显式请求 `summary/debug` 时返回
