package llm

import "context"

// Client 抽象出 Agent 运行时对 LLM 层的最小依赖。
// 它只负责消息协议收发，不拥有工具领域模型或业务语义。
type Client interface {
	Chat(ctx context.Context, req ChatRequest) (*ChatCompletion, error)
}

// Descriptor 允许上层以可选方式读取模型标识，用于 trace 和调试。
type Descriptor interface {
	ModelName() string
}

// ChatRequest 是运行时发给 LLM 层的统一请求结构。
// Tools 字段承载的是已经完成协议投影的工具描述，而不是 tooling 域的原始定义。
type ChatRequest struct {
	Messages       []Message
	Tools          []ToolDefinition
	ResponseFormat *ResponseFormat
	Temperature    *float64
}

// Message 是 LLM 协议中的单条消息表示。
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ToolDefinition 描述一个可暴露给 LLM 的工具协议结构。
// 它只表达 provider 需要的 JSON 形态，不定义工具的领域含义。
type ToolDefinition struct {
	Type     string                 `json:"type"`
	Function ToolFunctionDefinition `json:"function"`
}

// ToolFunctionDefinition 是工具定义中 provider 期望的函数元数据。
type ToolFunctionDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// ResponseFormat 描述当前模型调用期望的原生结构化输出协议。
// 它只表达 provider 协议字段，不承载 runtime 的业务语义。
type ResponseFormat struct {
	Type       string              `json:"type"`
	JSONSchema *ResponseJSONSchema `json:"json_schema,omitempty"`
}

// ResponseJSONSchema 定义 OpenAI-compatible structured outputs 所需的 schema 包装。
type ResponseJSONSchema struct {
	Name   string         `json:"name"`
	Schema map[string]any `json:"schema"`
	Strict bool           `json:"strict"`
}

// ToolCall 表示 LLM 返回的一次工具调用请求。
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

// ChatCompletion 是 LLM 层向 runtime 返回的统一结果。
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
