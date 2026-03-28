# 架构说明

## 概述

XCPC-Training-Agent 是一个面向集训队训练管理的后端服务，包含两项核心能力：

- 训练数据同步：采集 Codeforces / AtCoder 的训练与比赛数据
- 训练分析：基于 LLM 与本地工具输出结构化分析结果

## 分层

系统主链路如下：

`Handler -> Logic -> Model -> MySQL`

其中，Agent 请求在 Logic 层内进入专用执行链路：

`agent/service -> runtime -> context / tooling / model / observe`

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
  service/    任务级依赖装配
  runtime/    执行循环与最终输出校验
  tooling/    工具定义、注册、调用、摘要
  context/    memory、snapshot、消息组装
  model/      LLM provider 适配
  observe/    trace 记录与导出
  tools/      业务工具实现
```

## Agent 模块职责

各子模块职责如下：

- `service`：创建本轮运行所需的 `Toolbox`、`ContextManager`、`Observer` 与 `model.Client`
- `runtime`：驱动模型调用、工具调用与最终输出收敛
- `tooling`：管理工具协议与工具执行
- `context`：加载 memory，维护 session snapshot，构造模型输入
- `model`：适配 OpenAI-compatible chat completions 与 `tool_calls`
- `observe`：记录运行过程，不参与决策

## 执行流程

一次 Agent 请求的执行流程如下：

1. API 接收任务请求。
2. `agent/service` 完成本轮依赖装配。
3. `runtime` 打开 context，生成基础消息。
4. `runtime` 调用模型。
5. 若模型返回 `tool_calls`，则由 `tooling` 执行工具，并将摘要结果回传模型。
6. 若模型不再请求工具，`runtime` 校验最终 JSON 输出。
7. 返回分析结果与 trace。

## Memory 与 Trace

当前 memory 采用文件驱动方式：

- `memory/project.md`
- `memory/rules/*.md`

规则按路径匹配加载，不做全量注入。

trace 提供两种模式：

- `summary`
- `debug`

## 工程约束

以下约束用于保持模块边界稳定：

- `runtime` 只负责编排，不承载业务规则
- `tooling` 只负责工具，不拼接 prompt
- `context` 只负责上下文，不执行工具
- `observe` 只负责观测，不影响主流程
- provider 协议细节仅出现在 `agent/model`
