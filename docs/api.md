# 接口文档

## 基本信息

- Base URL: `http://localhost:8888`
- API Prefix: `/v1`

统一响应结构：

```json
{
  "code": 200,
  "data": {},
  "msg": "success"
}
```

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
  "task": "分析某个学生最近训练情况",
  "params": {
    "memory_paths": [
      "internal/logic/agent/runtime/runner.go"
    ]
  },
  "trace_mode": "summary"
}
```

字段说明：

- `task`：必填，任务描述
- `params`：可选，附加参数
- `trace_mode`：可选，`summary` 或 `debug`

返回重点：

- `data.task`
- `data.result`
- `data.token_usage`
- `data.trace`

## 最小调用路径

本项目最常用的两条接口是：

1. `POST /v1/user/login`
2. `POST /v1/admin/agent/task/run`
