# API 文档

## 概览

- 默认服务地址：`http://localhost:8888`
- 接口规范源：`docs/openapi.yaml`
- 认证方式：`Authorization: Bearer <JWT>`

当前项目的统一响应格式如下：

```json
{
  "code": 200,
  "data": {},
  "msg": "success"
}
```

说明：

- `code = 200` 表示业务成功
- 失败时当前实现通常返回 `code = 50000`
- 失败语义主要依赖 HTTP 状态码和 `msg`

## 文档入口

- OpenAPI 规范：[openapi.yaml](/docs/openapi.yaml)

## 鉴权说明

1. 调用 `POST /v1/user/login` 获取 JWT
2. 后续在请求头中添加：

```http
Authorization: Bearer <TOKEN>
```

管理员接口需要：

- JWT 有效
- `is_admin = true`

注意：当前实现中，鉴权失败（`401/403`）由中间件直接返回 JSON，例如 `{"msg":"未登录"}` / `{"msg":"无权限"}`，不走统一 `{code,data,msg}` 响应结构。

## 接口总览

### Auth

- `POST /v1/user/login`：用户登录

### User Self

- `GET /v1/user/me`：获取当前用户信息
- `POST /v1/user/password`：修改当前用户密码
- `DELETE /v1/user/me`：注销当前用户

### Admin Users

- `GET /v1/admin/users/list`：管理员查询用户列表
- `POST /v1/admin/users/create`：管理员批量创建用户
- `DELETE /v1/admin/users/{id}`：管理员删除用户

### Admin Training

- `POST /v1/admin/op/training/sync`：管理员按时间区间同步训练数据

### Admin Agent

- `POST /v1/admin/agent/task/run`：管理员触发 Agent 分析任务

## 典型调用流程

### 1. 手动同步训练数据

```bash
# 1) 登录获取 token
curl -s http://localhost:8888/v1/user/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"20001","password":"000000"}'

# 2) 手动同步区间
# 注意：当前实现字段是 students[].id（不是 student_id）
curl -s http://localhost:8888/v1/admin/op/training/sync \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <TOKEN>' \
  -d '{"students":[{"id":"示例学号"}],"from":"2026-03-01T00:00:00+08:00","to":"2026-03-07T23:59:59+08:00"}'
```

## 已知实现细节

- 用户列表接口当前返回字段名为 `List`，不是 `list`
- `POST /v1/admin/users/create` 在部分用户创建失败时仍返回 HTTP `200`，需要自行检查 `data.failed`
- `POST /v1/admin/op/training/sync` 在部分学生同步失败时也会返回 HTTP `200`，需要检查 `data.failed`
- 端口以配置文件和实际代码为准，当前默认端口是 `8888`
