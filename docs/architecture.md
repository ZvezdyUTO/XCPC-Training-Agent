# 架构说明

## 概述

XCPC-Training-Agent 是一个面向集训队训练管理的后端服务，包含两项核心能力：

- 训练数据同步：采集 Codeforces / AtCoder 的训练与比赛数据
- 训练分析：基于 LLM 与本地工具输出结构化分析结果

## 分层

系统主链路如下：

`Handler -> Logic -> Model -> MySQL`

其中，Agent 请求在 Logic 层内进入专用执行链路：

`agent/service -> runtime -> context / tooling / llm / observe`

## 目录结构

```text
internal/
  handler/    HTTP 接口与定时任务入口
  logic/      业务编排
  model/      数据访问
  crawler/    Python 爬虫调用
```

Agent 模块目录：

```text
internal/logic/agent/
  service/    单次请求的依赖装配与协议嫁接
  runtime/    单次运行的执行循环与终止控制
  tooling/    工具接入契约、注册管理、调用入口
  context/    上下文状态维护与模型消息组装
  llm/        OpenAI-compatible 协议收发
  observe/    运行事件与 trace 采集
  tools/      业务工具实现
```

## Agent 模块职责

各子模块职责如下：

- `service`：接收一次外部任务请求，创建本轮运行所需实例，并把 `tooling` 规格翻译成当前 `llm` 协议结构。
- `runtime`：只负责执行闭环，依次调度 `context -> llm -> tooling -> context / observe`，不再创建依赖。
- `tooling`：只负责工具接入标准、工具列表导出、工具调用入口，不关心 LLM 协议。
- `context`：负责单次运行状态，加载 memory，维护 `Snapshot` 与工具结果记录，并按轮次产出 `messages`。
- `llm`：负责 OpenAI-compatible chat completions 请求与响应，不关心工具领域模型。
- `observe`：只接收运行事件并导出 trace，不参与运行决策。

这几个模块是并列能力模块，彼此不直接耦合。
真正负责把它们接起来的是 `service` 和 `runtime`：

- `service` 负责准备和装配。
- `runtime` 负责运行和终止。

## 执行流程

一次 Agent 请求的执行流程如下：

1. API 接收任务请求。
2. `agent/service` 把请求翻译成 `agent.Input`，创建 `Toolbox`、`ContextManager`、`Observer`，并准备 `llm` 所需工具协议结构。
3. `runtime` 打开 `context`，得到本次运行的 `State`。
4. `runtime` 向 `context` 请求本轮 `messages`，再连同工具定义一起调用 `llm`。
5. 若模型返回 `tool_calls`，则由 `runtime` 顺序调用 `tooling`。
6. 工具完整结果会先通过 `role=tool` 消息回传模型，同时写入 `context`。
7. `context` 会把最近工具结果保存在 `ToolResults` 中，并将更轻量的进度信息压入 `Snapshot`。
8. 若模型不再请求工具，`runtime` 校验最终 JSON 输出并结束运行。
9. `observe` 在整个过程中记录事件和 span，最终导出 trace。

## Memory 与 Trace

当前 memory 采用文件驱动方式：

- `memory/project.md`
- `memory/rules/*.md`

规则按路径匹配加载，只作为静态背景消息注入，不承担运行中状态。

trace 提供两种模式：

- `summary`
- `debug`

## 工程约束

以下约束用于保持模块边界稳定：

- `service` 只负责装配，不负责执行循环
- `runtime` 只负责编排和终止控制，不承载业务规则
- `tooling` 只负责工具接入和调用，不负责协议翻译
- `context` 只负责状态和消息，不执行工具、不发模型请求
- `llm` 只负责模型协议，不知道 `tooling` 领域模型
- `observe` 只负责观测，不影响主流程
- provider 协议细节仅出现在 `agent/llm`

## 当前终止机制

`runtime` 当前采用两层终止策略：

- 主终止条件：模型给出最终 JSON 输出，不再请求工具。
- 保护性终止条件：达到 `max_steps` 时强制结束。

此外，在接近步数上限时，`runtime` 会追加一条轻量收尾提示，提醒模型优先基于现有信息结束，而不是继续展开新的工具调用。
