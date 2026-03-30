# XCPC-Training-Agent

XCPC-Training-Agent 是一个面向集训队训练管理的后端服务，提供两项核心能力：

- 同步 Codeforces / AtCoder 的训练与比赛数据
- 基于 LLM 与本地工具输出结构化训练分析结果

## 功能概览

XCPC-Training-Agent 面向集训队的日常训练管理场景，支持手动触发训练数据爬取，也支持基于同步状态自动计算补抓范围。当前采集的数据覆盖 Codeforces 与 AtCoder 的训练做题记录、按天训练统计、比赛记录、当前分与历史峰值分，既能满足日常补数，也能支撑后续分析和排行榜计算。

在教练端前端中，可以批量导入学生、单独同步某个学生或触发全体训练同步，查看指定时间范围内的训练价值排行榜，查询某场比赛的队内所有人记录，以及直接查看单个学生在某段时间内的训练累计和价值评分拆解。系统同时提供基于 Agent 的自然语言分析能力，模型会结合当前保留的 4 个本地工具完成单人训练查询、单人比赛记录查询、训练价值排行榜查询和某场比赛队内排名查询，并返回结构化结论与 Trace，方便教练做日常判断和复盘。

## 技术栈

- Go + Gin
- GORM + MySQL
- OpenAI-compatible LLM API
- Python crawler
- React + Vite coach frontend
- Docker Compose

## 快速开始

### 1. 配置环境

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

### 2. 启动服务

确保仓库内存在 `sql/init.sql`，随后执行：

```bash
docker compose up -d
```

默认服务地址：

- API: `http://localhost:8888`
- Frontend: `http://localhost:5173`
- MySQL: `127.0.0.1:3307`

### 3. 默认管理员账号

系统初始化后会创建默认管理员：

- Username: `20001`
- Password: `000000`

## 使用方式

推荐直接使用教练端前端，不再需要手动调用 `curl`。

使用流程：

1. 打开浏览器访问 `http://localhost:5173`
2. 使用默认管理员登录
   - Username: `20001`
   - Password: `000000`
3. 在学生页批量导入学生、触发训练同步、单独同步或删除学生
4. 在查询页查看训练价值排行榜、单人训练统计和某场比赛的队内记录
5. 在 Agent 页发起自然语言分析并查看 Trace

如果需要查看接口明细或调试 API，请参考：

- `docs/api.md`

## 前端说明

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

## 文档

- `docs/architecture.md`
- `docs/api.md`
- `docs/agent-tools.md`
