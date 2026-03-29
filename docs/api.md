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

### 数据同步

### `POST /v1/admin/op/training/syncall`

触发训练数据同步。

说明：

- 遍历所有有效学生
- 自动判断全量同步、区间同步或跳过

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
- `invalid_api_key`：模型 API Key 无效
- `llm_request_failed`：模型请求失败
- `llm_response_invalid_json`：模型响应不是合法 JSON
- `internal_error`：服务内部错误

返回重点：

- `data.task`
- `data.result`
- `data.token_usage`
- `data.trace`：仅在显式请求 `summary/debug` 时返回