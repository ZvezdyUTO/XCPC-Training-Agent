# XCPC-Training-Agent

XCPC-Training-Agent 是一个面向集训队训练管理的后端服务，提供两项核心能力：

- 同步 Codeforces / AtCoder 的训练与比赛数据
- 基于 LLM 与本地工具输出结构化训练分析结果

## Features

- 用户与管理员权限体系
- 训练数据自动同步
- 基于原生 `tools / tool_calls` 的 Agent 分析
- Docker Compose 本地部署

## Tech Stack

- Go + Gin
- GORM + MySQL
- OpenAI-compatible LLM API
- Python crawler
- React + Vite coach frontend
- Docker Compose

## Quick Start

### 1. Configure environment

启动前请配置模型访问参数。当前 Agent 要求模型服务支持 OpenAI-compatible chat completions 与原生 `tool_calls`。

推荐环境变量：

- `LLM_API_KEY`
- `LLM_BASE_URL`
- `LLM_MODEL`

兼容旧配置：

- `OPENAI_API_KEY`
- `OPENAI_BASE_URL`
- `DASHSCOPE_API_KEY`
- `DASHSCOPE_BASE_URL`

### 2. Start services

确保仓库内存在 `sql/init.sql`，随后执行：

```bash
docker compose up -d
```

默认服务地址：

- API: `http://localhost:8888`
- Frontend: `http://localhost:5173`
- MySQL: `127.0.0.1:3307`

### 3. Default admin account

系统初始化后会创建默认管理员：

- Username: `20001`
- Password: `000000`

## Common Usage

### Login

```bash
curl -s http://localhost:8888/v1/user/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"20001","password":"000000"}'
```

### Create users

```bash
curl -s http://localhost:8888/v1/admin/users/create \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <TOKEN>' \
  -d '{"users":[{"id":"<示例学号>","name":"<示例姓名>","password":"<默认密码>","cf_handle":"<CF_HANDLE>","ac_handle":"<AC_HANDLE>"}]}'
```

### Sync training data

```bash
curl -s http://localhost:8888/v1/admin/op/training/syncall \
  -H 'Authorization: Bearer <TOKEN>'
```

### Run agent

```bash
curl -s http://localhost:8888/v1/admin/agent/task/run \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <TOKEN>' \
  -d '{
    "task": "分析学号为 <示例学号> 的学生近期训练情况",
    "trace_mode": "summary"
  }'
```

## Frontend

教练端前端位于 [frontend/package.json](/home/zvezdyuto/GolandProjects/XCPC-Training-Agent/frontend/package.json)，与后端代码独立维护。

本地开发：

```bash
cd frontend
npm install
npm run dev
```

默认开发地址：

- Frontend dev: `http://localhost:5173`
- API 代理到: `http://localhost:8888`

容器部署：

- `docker compose up -d --build`
- 浏览器访问 `http://localhost:5173`

## Repository Layout

```text
internal/
  handler/    HTTP 接口与定时任务入口
  logic/      业务编排
  model/      数据访问
  crawler/    Python 爬虫调用
```

## Documentation

- `docs/architecture.md`
- `docs/api.md`
