package observe

import (
	"aATA/internal/logic/agent"
	agentmodel "aATA/internal/logic/agent/model"
	"aATA/internal/logic/agent/tooling"
)

// Sink 是 observe 层底层 trace backend 的最小抽象。
type Sink interface {
	Record(step int, eventType agent.EventType, parentID string, payload any) string
	StartSpan(step int, spanType agent.SpanType, parentSpanID string, payload any) string
	FinishSpan(spanID, status string, payload any)
	Result() agent.RunTrace
}

// Observer 是 runtime 向 observe 层发出的运行事件接口。
type Observer interface {
	RunStarted(input agent.Input, toolNames []string)
	ModelStarted(step int, req agentmodel.ChatRequest)
	ModelFinished(step int, completion *agentmodel.ChatCompletion, parseErr error)
	ToolStarted(step int, name string, args string, toolCallID string)
	ToolFinished(step int, result tooling.CallResult, err error, latencyMs int64)
	RunFinished(step int, output map[string]any)
	RunFailed(step int, stage string, err error, extra map[string]any)
}

// Factory 负责为一次运行创建对应的 observer 实例。
type Factory interface {
	New(llmClient agentmodel.Client, input agent.Input, toolNames []string) Observer
}

// TraceResult 允许 observer 在运行结束后导出最终 trace。
type TraceResult interface {
	Result() agent.RunTrace
}
