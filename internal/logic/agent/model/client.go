package model

import "context"

// Client 抽象出 Agent 运行时对模型层的最小依赖。
type Client interface {
	Chat(ctx context.Context, req ChatRequest) (*ChatCompletion, error)
}

// Descriptor 允许上层以可选方式读取模型标识，用于 trace 和调试。
type Descriptor interface {
	ModelName() string
}

// ChatRequest 是运行时发给模型层的统一请求结构。
type ChatRequest struct {
	Messages    []Message
	Tools       []ToolDefinition
	Temperature *float64
}

// Message 是模型协议中的单条消息表示。
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ToolDefinition 描述一个可暴露给模型的工具定义。
type ToolDefinition struct {
	Type     string             `json:"type"`
	Function FunctionDefinition `json:"function"`
}

// FunctionDefinition 是工具定义中 provider 期望的函数元数据。
type FunctionDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// ToolCall 表示模型返回的一次工具调用请求。
type ToolCall struct {
	ID       string       `json:"id,omitempty"`
	Type     string       `json:"type,omitempty"`
	Function FunctionCall `json:"function"`
}

// FunctionCall 描述单次工具调用的名称和参数。
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments,omitempty"`
}

// ChatCompletion 是模型层向 runtime 返回的统一结果。
type ChatCompletion struct {
	Message      Message         `json:"message"`
	Content      string          `json:"content"`
	ToolCalls    []ToolCall      `json:"tool_calls"`
	FinishReason string          `json:"finish_reason"`
	LatencyMs    int64           `json:"latency_ms"`
	Usage        CompletionUsage `json:"usage"`
	RawResponse  string          `json:"raw_response"`
}

// CompletionUsage 汇总单次模型调用的 token 使用情况。
type CompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
