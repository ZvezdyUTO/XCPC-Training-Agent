package observe

import (
	"aATA/internal/logic/agent"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Collector 是默认的内存内 trace sink。
// 它负责记录 event/span，并在结束时汇总为对外可返回的 RunTrace。
type Collector struct {
	mode      agent.Mode
	runID     string
	startedAt time.Time

	mu          sync.Mutex
	events      []agent.Event
	spans       []agent.Span
	activeSpans map[string]*agent.Span
}

var idSeq uint64

// NewCollector 创建一个按指定模式记录的 trace collector。
func NewCollector(mode agent.Mode) *Collector {
	if mode != agent.ModeDebug {
		mode = agent.ModeSummary
	}

	now := time.Now()
	return &Collector{
		mode:        mode,
		runID:       newID("run"),
		startedAt:   now,
		events:      make([]agent.Event, 0, 16),
		spans:       make([]agent.Span, 0, 8),
		activeSpans: make(map[string]*agent.Span),
	}
}

// Record 追加一条瞬时事件记录。
func (c *Collector) Record(step int, eventType agent.EventType, parentID string, payload any) string {
	eventID := newID("evt")
	event := agent.Event{
		EventID:   eventID,
		RunID:     c.runID,
		ParentID:  parentID,
		Step:      step,
		EventType: eventType,
		Timestamp: time.Now(),
		Payload:   summarizePayload(c.mode, eventType, payload),
	}

	c.mu.Lock()
	c.events = append(c.events, event)
	c.mu.Unlock()

	return eventID
}

// StartSpan 打开一个新的耗时区间。
func (c *Collector) StartSpan(step int, spanType agent.SpanType, parentSpanID string, payload any) string {
	spanID := newID("span")
	now := time.Now()

	span := &agent.Span{
		SpanID:       spanID,
		RunID:        c.runID,
		ParentSpanID: parentSpanID,
		Step:         step,
		SpanType:     spanType,
		StartedAt:    now,
		Status:       "running",
		Payload:      summarizeSpanPayload(c.mode, payload),
	}

	c.mu.Lock()
	c.activeSpans[spanID] = span
	c.mu.Unlock()

	return spanID
}

// FinishSpan 关闭一个已经打开的 span，并补齐最终状态和耗时。
func (c *Collector) FinishSpan(spanID, status string, payload any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	span, ok := c.activeSpans[spanID]
	if !ok {
		return
	}

	span.FinishedAt = time.Now()
	span.LatencyMs = span.FinishedAt.Sub(span.StartedAt).Milliseconds()
	if status == "" {
		status = "unknown"
	}
	span.Status = status
	span.Payload = mergeSummary(span.Payload, summarizeSpanPayload(c.mode, payload))
	span.Payload["status"] = status
	span.Payload["latency_ms"] = span.LatencyMs

	c.spans = append(c.spans, *span)
	delete(c.activeSpans, spanID)
}

// Result 导出当前 collector 中累计的 trace 结果。
func (c *Collector) Result() agent.RunTrace {
	c.mu.Lock()
	defer c.mu.Unlock()

	events := make([]agent.Event, len(c.events))
	copy(events, c.events)

	spans := make([]agent.Span, len(c.spans))
	copy(spans, c.spans)

	return agent.RunTrace{
		RunID:      c.runID,
		Mode:       c.mode,
		StartedAt:  c.startedAt,
		FinishedAt: time.Now(),
		TokenUsage: summarizeTokenUsage(events),
		Spans:      spans,
		Events:     events,
	}
}

// newID 为 run/event/span 生成局部唯一标识。
func newID(prefix string) string {
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().UnixNano(), atomic.AddUint64(&idSeq, 1))
}

// summarizePayload 把 event payload 压缩成适合长期保存的摘要形式。
func summarizePayload(mode agent.Mode, eventType agent.EventType, payload any) map[string]any {
	summary := summarizeMap(normalizeEnvelope(payload))

	if mode == agent.ModeDebug {
		summary["debug"] = rawPayload(payload)
	}
	return summary
}

// summarizeSpanPayload 把 span payload 压缩成适合长期保存的摘要形式。
func summarizeSpanPayload(mode agent.Mode, payload any) map[string]any {
	summary := summarizeMap(normalizeEnvelope(payload))

	if mode == agent.ModeDebug {
		summary["debug"] = rawPayload(payload)
	}
	return summary
}

// normalizeEnvelope 把任意 payload 统一规整成 map 结构。
func normalizeEnvelope(payload any) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	if m, ok := payload.(map[string]any); ok {
		out := make(map[string]any, len(m))
		for k, v := range m {
			out[k] = v
		}
		return out
	}
	return map[string]any{
		"summary": payload,
	}
}

// summarizeMap 递归压缩 payload 中的值，保留适合直接阅读的摘要结构。
// 这里会主动跳过只适合 debug 模式的大字段，避免 summary 返回过重。
func summarizeMap(input map[string]any) map[string]any {
	output := make(map[string]any, len(input))
	for k, v := range input {
		if shouldOmitSummaryKey(k) {
			continue
		}
		output[k] = summarizeValue(v)
	}
	return output
}

// mergeSummary 把新增摘要字段合并回已有摘要。
func mergeSummary(base, extra map[string]any) map[string]any {
	if len(base) == 0 {
		return extra
	}
	merged := make(map[string]any, len(base)+len(extra))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range extra {
		merged[k] = v
	}
	return merged
}

// summarizeValue 根据值类型生成轻量摘要。
func summarizeValue(v any) any {
	switch x := v.(type) {
	case nil:
		return nil
	case string:
		return preview(x, 240)
	case []string:
		return x
	case []any:
		if len(x) > 8 {
			x = x[:8]
		}
		out := make([]any, 0, len(x))
		for _, item := range x {
			out = append(out, summarizeValue(item))
		}
		return out
	case map[string]any:
		return summarizeMap(x)
	case error:
		return x.Error()
	case bool, int, int64, float64:
		return x
	case json.RawMessage:
		return preview(string(x), 240)
	default:
		var normalized any
		if tryNormalizeValue(v, &normalized) {
			return summarizeValue(normalized)
		}
		return preview(fmt.Sprintf("%v", v), 240)
	}
}

// shouldOmitSummaryKey 定义 summary 模式下应主动丢弃的大字段。
// 这些字段仍可通过 debug 模式下的 raw payload 查看，不应污染日常排障结果。
func shouldOmitSummaryKey(key string) bool {
	switch key {
	case "messages", "content", "tool_calls", "arguments", "arguments_raw", "result", "raw_response":
		return true
	default:
		return false
	}
}

// tryNormalizeValue 尝试把结构体等复杂值转成通用 JSON 结构，便于递归摘要。
func tryNormalizeValue(value any, target *any) bool {
	data, err := json.Marshal(value)
	if err != nil {
		return false
	}
	if err := json.Unmarshal(data, target); err != nil {
		return false
	}
	return true
}

// rawPayload 生成 debug 模式下可附带的原始 payload 表示。
func rawPayload(v any) any {
	if v == nil {
		return nil
	}
	if s, ok := v.(string); ok {
		return s
	}

	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return json.RawMessage(b)
}

// jsonLen 粗略计算一个值编码后的 JSON 长度。
func jsonLen(v any) int {
	b, err := json.Marshal(v)
	if err != nil {
		return len(fmt.Sprintf("%v", v))
	}
	return len(b)
}

// previewJSON 生成 JSON 的截断预览字符串。
func previewJSON(v any, limit int) string {
	b, err := json.Marshal(v)
	if err != nil {
		return preview(fmt.Sprintf("%v", v), limit)
	}
	return preview(string(b), limit)
}

// summarizeTokenUsage 从模型返回事件中提取累计 token 用量。
func summarizeTokenUsage(events []agent.Event) agent.TokenUsageSummary {
	summary := agent.TokenUsageSummary{}
	for _, event := range events {
		if event.EventType != agent.EventModelReturned {
			continue
		}
		summary.ModelCallCount++
		summary.InputTokens += asInt(event.Payload["input_tokens"])
		summary.OutputTokens += asInt(event.Payload["output_tokens"])
		summary.TotalTokens += asInt(event.Payload["total_tokens"])
	}
	return summary
}

// asInt 将常见数值类型统一转换成 int。
func asInt(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int8:
		return int(x)
	case int16:
		return int(x)
	case int32:
		return int(x)
	case int64:
		return int(x)
	case float32:
		return int(x)
	case float64:
		return int(x)
	default:
		return 0
	}
}

// preview 生成一段去换行、限长的字符串预览。
func preview(s string, limit int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "..."
}
