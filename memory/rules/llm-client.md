---
paths:
  - internal/logic/agent/model/**
---
修改 LLM 客户端时遵守：

- 保持 OpenAI-compatible chat completions 兼容层最小化。
- 模型能力优先通过结构化字段暴露，不回退到文本协议。
- 对 content 和 tool_calls 做容错解析，但不要在客户端层拼业务决策。
