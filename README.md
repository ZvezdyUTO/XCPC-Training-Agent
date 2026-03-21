# XCPC-Training-Agent

一个专为 XCPC 集训队设计的智能化训练数据管理与分析平台。它不仅能自动化抓取 Codeforces / AtCoder 数据，更能通过内置的 AI Agent 像教练一样分析队员的训练走势。

---

## 你能用它做什么

- **智能 Agent 分析系统**
  - 事件循环控制：AI 不只是生成文本，而是通过观察 (Observation) -> 思考 (Thought) -> 行动 (Action) 的循环进行任务调度。
  - 工具自动化调用：Agent 可自主决定何时调用数据库查询接口、统计函数或爬虫，完成复杂的横向/纵向对比分析。

- **全自动数据同步**
  - 多平台支持：统一抓取 Codeforces 与 AtCoder 的训练/比赛记录。
  - T+1 自动任务：按天自动同步数据，支持管理员手动触发区间覆盖同步。

- **完善的权限体系**
  - 基于 JWT 的鉴权机制，区分普通队员（查阅、改密）与管理员（调度爬虫、管理用户、调用 Agent）。

- **容器化一键部署**
  - 基于 Docker Compose 编排，集成 MySQL、Redis 环境，支持 SQL 自动初始化。

---

## 未来希望增加的功能

- 拓展 tool, skill
- 可视化前端
- 部署到校园内网

---

## 技术栈

* **Go + Gin**：提供 RESTful API
* **LLM + Tool-based Agent 框架**：基于事件循环的工具调用式智能分析系统
* **GORM + MySQL**：数据持久化
* **gocron**：定时任务调度
* **Docker Compose**：服务编排与部署
* **Python 爬虫**：独立子进程抓取 CF / AtCoder 数据，JSON 回传

---

## 项目结构

总体架构：**Handler → Logic → Model**

```
internal/
├── handler/
│   ├── api/              # HTTP 接口层
│   └── task/             # 定时任务入口
│
├── logic/                # 业务核心层
│   ├── user.go           # 用户逻辑（登录、注册、权限、批量建号）
│   │
│   ├── student_data/     # 训练数据导入相关逻辑
│   │
│   ├── agent_logic.go    # Agent 调度入口
│   └── agent/            # Agent 框架实现
│       ├── controller.go # 事件循环核心
│       ├── registry.go   # 工具注册与调用
│       ├── prompt.go     # Prompt 组装
│       ├── types.go      # 协议定义
│       └── tools/        # 具体分析工具（训练统计 / rating 统计）
│
├── model/                # 数据访问层
├── crawler/              # Python 爬虫调用封装
sql/
└── init.sql              # 数据库初始化
```

---

## 快速开始（Docker）

### 1. 配置 LLM 环境变量

在启动服务之前，你需要准备好 LLM 的访问凭证。默认支持 **阿里云百炼 (DashScope)** 及其他兼容 OpenAI 接口协议的服务。

请在 `docker-compose.yaml` 中填写你的配置：

* **DASHSCOPE_API_KEY**: 你的 API Key（例如：`sk-xxxx...`）。
* **DASHSCOPE_BASE_URL**: 接口基础地址（百炼通常为 `https://dashscope.aliyuncs.com/compatible-mode/v1`）。

### 2. 启动依赖与服务

确保当前目录下存在 `sql/init.sql` 脚本，随后运行：

```bash
docker compose up -d
```

### 3. 服务编排参考 (`docker-compose.yaml`)

```yaml
services:
  mysql:
    image: mysql:8.0
    container_name: aATA-mysql
    environment:
      MYSQL_ROOT_PASSWORD: 123456
      MYSQL_DATABASE: aATAdb
      MYSQL_ROOT_HOST: "%"
    ports:
      - "3307:3306"
    volumes:
      - mysql_data:/var/lib/mysql
      - ./sql/init.sql:/docker-entrypoint-initdb.d/init.sql
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 5s
      timeout: 5s
      retries: 5

  app:
    build:
      context: .
      dockerfile: Dockerfile.dev
    container_name: aATA-app
    environment:
      # === 需自行填写部分 ===
      - DASHSCOPE_API_KEY=<Token>
      - DASHSCOPE_BASE_URL=<URL>
      # ====================
      - AGENT_TEST=1
    ports:
      - "8888:8888"
    volumes:
      - .:/app
    depends_on:
      mysql:
        condition: service_healthy
    restart: always

volumes:
  mysql_data:

```

---

## 调用示例                  

1. 登录 root，获取管理员 token
```
curl -s http://localhost:8080/v1/user/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"20001","password":"000000"}'
```

2. 批量创建用户（把 token 填到 Authorization）
```
curl -s http://localhost:8080/v1/admin/users/create \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <TOKEN>' \
  -d '{"users":[{"id":"示例学号","name":"示例姓名","password":"默认密码","cf_handle":"示例codeforcesID","ac_handle":"示例atCoderID"}]}'
```

3. 手动同步区间（只传 student_id）
```
curl -s http://localhost:8080/v1/admin/op/training/sync \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <TOKEN>' \
  -d '{"students":[{"student_id":"示例学号"}],"from":"2026-03-01T00:00:00+08:00","to":"2026-03-07T23:59:59+08:00"}'
```

4. 调用 Agent 进行分析，此处示例分析某位学生的表现
```
curl -s http://localhost:8080/v1/admin/agent/task/run \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <TOKEN>' \
  -d '{ "task": "观察学号为<示例学号>的学生在 2025 年的训练情况，可以从比赛数据和做题数据中进行分析" }'
```


模型返回示例：
```JSON
{
	"code": 200,
	"data": {
		"result": {
			"confidence": 0.9,
			"decision_type": "student_analysis",
			"focus_students": [
				"<示例学号>"
			],
			"metrics": {
				"avg_rating": 1257,
				"contest_count": 89,
				"latest_rating": 1674,
				"max_rating": 1856,
				"primary_difficulty_range": "1600-1800",
				"total_problems_solved": 376
			},
			"report": "学号 <示例学号> 的学生在 2025 年展现了活跃的训练参与度和显著的比赛成绩提升。\n\n【做题训练情况】\n全年累计完成 Codeforces 题目 376 道，训练量十分充足。难度分布呈现以下特点：\n- 中高难度（1600-1800）题目为主力，完成 125 道，占比约 33%\n- 基础题目（800-1100）保持了 92 道的训练量，占比约 24%\n- 进阶难度（1900-2300+）题目合计 90 道，显示出挑战高难度的意愿\n\n这种分布表明学生在巩固基础的同时，积极向高难度题目进阶，训练策略较为合理。\n\n【比赛表现分析】\n全年参与 89 场比赛，参赛频率很高。\n- 最新 Rating：1674（紫名边缘）\n- 历史最高：1856（2025 年 10 月 24 日达到）\n- 平均 Rating：1257\n\n比赛表现可以分为两个阶段：\n1. 上半年（1-4 月）：Rating 在 1370-1525 区间内大幅震荡，最高达到 1525，最低跌至 1370，稳定性较差\n2. 下半年（8-11 月）：进入稳定上升期，从 1667 稳步攀升至历史最高的 1856，虽然 11 月略有回落至 1674，但整体保持在较高水平\n\n特别值得注意的是，学生在 AtCoder ABC 系列比赛中表现强劲，Performance 多次突破 1000 分，最高达到 1619 分，说明在标准算法竞赛中的实际能力已经很强。\n\n【综合评价】\n该学生是一名勤奋且有潜力的竞赛选手：\n1. 训练态度端正，年训练量 376 题属于高水平\n2. 比赛经验丰富，89 场比赛的参与度体现了极强的竞技热情\n3. Rating 整体呈上升趋势，从年初的 1400+ 突破到 1800+，进步明显\n4. 需要改进的是比赛的稳定性，上半年波动较大\n\n【建议】\n1. 继续保持当前的训练强度，建议适当增加 2100+ 难度题目的比例，以突破当前瓶颈\n2. 在比赛中注意心态调整，减少大幅波动，争取稳定在 1700-1800 区间\n3. 加强赛后复盘，总结失分原因，提升抗干扰能力"
		},
		"task": "观察学号为 <示例学号> 的学生的 2025 年的训练情况，可以从比赛数据和做题数据中进行分析",
		"trace": [
			"Step 0: 查询该学生2025年全年的训练累计数据（按难度统计）",
			"Step 1: 需要获取该学生的比赛 rating 统计信息，结合已有的训练数据进行全面分析",
			"Step 2: 已获取该学生 2025 年完整的训练数据和比赛 rating 数据，信息充足，可以进行综合分析并生成报告"
		]
	},
	"msg": "success"
}
```
