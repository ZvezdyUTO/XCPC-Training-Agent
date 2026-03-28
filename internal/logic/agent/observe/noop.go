package observe

import "aATA/internal/logic/agent"

// NoopSink 在调用方不关心 trace 时提供一个零成本占位实现。
type NoopSink struct{}

func (NoopSink) Record(step int, eventType agent.EventType, parentID string, payload any) string {
	return ""
}

func (NoopSink) StartSpan(step int, spanType agent.SpanType, parentSpanID string, payload any) string {
	return ""
}

func (NoopSink) FinishSpan(spanID, status string, payload any) {}

// Result 返回一个空 trace，保持接口语义稳定。
func (NoopSink) Result() agent.RunTrace {
	return agent.RunTrace{
		Mode:   agent.ModeSummary,
		Spans:  []agent.Span{},
		Events: []agent.Event{},
	}
}
