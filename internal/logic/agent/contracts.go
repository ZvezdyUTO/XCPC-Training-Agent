package agent

import (
	"strings"
	"time"
)

// Input 是一次 Agent 运行的外部任务输入。
// Query 承载用户任务描述，Params 承载可选的附加参数。
type Input struct {
	Query  string                 `json:"query"`
	Params map[string]interface{} `json:"params"`
}

// MemoryPaths 从通用参数里提取 memory 相关路径。
// 这里兼容多个别名，避免 API 层和调用方被单一字段名绑死。
func (in Input) MemoryPaths() []string {
	if in.Params == nil {
		return nil
	}

	keys := []string{"memory_paths", "context_paths", "paths"}
	paths := make([]string, 0, 4)
	for _, key := range keys {
		raw, ok := in.Params[key]
		if !ok {
			continue
		}
		switch v := raw.(type) {
		case []string:
			for _, item := range v {
				item = strings.TrimSpace(item)
				if item != "" {
					paths = append(paths, item)
				}
			}
		case []any:
			for _, item := range v {
				text, ok := item.(string)
				if !ok {
					continue
				}
				text = strings.TrimSpace(text)
				if text != "" {
					paths = append(paths, text)
				}
			}
		case string:
			for _, item := range strings.Split(v, ",") {
				item = strings.TrimSpace(item)
				if item != "" {
					paths = append(paths, item)
				}
			}
		}
	}

	if len(paths) == 0 {
		return nil
	}
	return paths
}

// Mode 控制 trace 的记录粒度。
type Mode string

const (
	// ModeNone 表示不向调用方返回 trace，但内部仍可保留最小运行摘要。
	ModeNone Mode = "none"
	// ModeSummary 只保留适合接口返回和日常排障的摘要信息。
	ModeSummary Mode = "summary"
	// ModeDebug 记录更多原始 payload，用于深度调试模型与工具链路。
	ModeDebug Mode = "debug"
)

// EventType 表示一次运行中的离散事件类型。
type EventType string

// SpanType 表示一次运行中的耗时区间类型。
type SpanType string

const (
	EventRunStarted      EventType = "run_started"
	EventToolsRegistered EventType = "tools_registered"
	EventModelCalled     EventType = "model_called"
	EventModelReturned   EventType = "model_returned"
	EventToolCalled      EventType = "tool_called"
	EventToolReturned    EventType = "tool_returned"
	EventRunFinished     EventType = "run_finished"
	EventRunFailed       EventType = "run_failed"
)

const (
	SpanModelCall SpanType = "model_call"
	SpanToolCall  SpanType = "tool_call"
)

// RunTrace 是一次运行结束后对外暴露的完整 trace 结果。
type RunTrace struct {
	RunID      string            `json:"-"`
	Mode       Mode              `json:"mode"`
	StartedAt  time.Time         `json:"started_at"`
	FinishedAt time.Time         `json:"finished_at"`
	TokenUsage TokenUsageSummary `json:"token_usage"`
	Spans      []Span            `json:"spans"`
	Events     []Event           `json:"events"`
}

// TokenUsageSummary 汇总本次运行中所有模型调用的 token 消耗。
type TokenUsageSummary struct {
	ModelCallCount int `json:"model_call_count"`
	InputTokens    int `json:"input_tokens"`
	OutputTokens   int `json:"output_tokens"`
	TotalTokens    int `json:"total_tokens"`
}

// Event 是 trace 中的一条瞬时事件记录。
type Event struct {
	EventID   string         `json:"-"`
	RunID     string         `json:"-"`
	ParentID  string         `json:"-"`
	Step      int            `json:"step"`
	EventType EventType      `json:"event_type"`
	Timestamp time.Time      `json:"timestamp"`
	Payload   map[string]any `json:"payload"`
}

// Span 是 trace 中的一段耗时区间记录。
type Span struct {
	SpanID       string         `json:"-"`
	RunID        string         `json:"-"`
	ParentSpanID string         `json:"-"`
	Step         int            `json:"step"`
	SpanType     SpanType       `json:"span_type"`
	StartedAt    time.Time      `json:"started_at"`
	FinishedAt   time.Time      `json:"finished_at"`
	Status       string         `json:"status"`
	LatencyMs    int64          `json:"latency_ms"`
	Payload      map[string]any `json:"payload"`
}
