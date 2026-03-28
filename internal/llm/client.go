package llm

import "context"

type Client interface {
	Chat(ctx context.Context, req ChatRequest) (*ChatCompletion, error)
}

type Descriptor interface {
	ModelName() string
}

type ChatRequest struct {
	Messages    []Message
	Tools       []ToolDefinition
	Temperature *float64
}

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ToolDefinition struct {
	Type     string             `json:"type"`
	Function FunctionDefinition `json:"function"`
}

type FunctionDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type ToolCall struct {
	ID       string       `json:"id,omitempty"`
	Type     string       `json:"type,omitempty"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments,omitempty"`
}

type ChatCompletion struct {
	Message      Message         `json:"message"`
	Content      string          `json:"content"`
	ToolCalls    []ToolCall      `json:"tool_calls"`
	FinishReason string          `json:"finish_reason"`
	LatencyMs    int64           `json:"latency_ms"`
	Usage        CompletionUsage `json:"usage"`
	RawResponse  string          `json:"raw_response"`
}

type CompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
