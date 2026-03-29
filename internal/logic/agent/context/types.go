package context

import (
	"aATA/internal/logic/agent"
	agentllm "aATA/internal/logic/agent/llm"
	stdctx "context"
)

// Snapshot 是单次运行中持续演进的轻量上下文快照。
// 它只保留对下一轮推理有帮助的状态，不负责完整记录全部历史。
type Snapshot struct {
	Goal           string   `json:"goal"`
	ConfirmedFacts []string `json:"confirmed_facts"`
	DoneItems      []string `json:"done_items"`
	TodoItems      []string `json:"todo_items"`
	Artifacts      []string `json:"artifacts"`
}

// ToolResultSummary 是写入上下文状态的工具结果记录。
// 最近若干次工具调用保留完整结果，更早的结果只保留摘要，避免上下文无限膨胀。
type ToolResultSummary struct {
	ToolName string         `json:"tool_name"`
	Success  bool           `json:"success"`
	Result   any            `json:"result,omitempty"`
	Summary  map[string]any `json:"summary"`
}

// ToolResultPatch 表示一次工具调用完成后写回上下文的最小补丁。
// runtime 负责把运行事实投影成这个结构，再交给 context 更新状态。
type ToolResultPatch struct {
	ToolName string
	Success  bool
	Args     map[string]any
	Result   any
}

// State 保存一次运行已经解析好的上下文状态。
// 它对 runtime 暴露的是稳定结构，而不是 memory loader/prompt builder 的内部细节。
type State struct {
	Snapshot    Snapshot
	ToolResults []ToolResultSummary

	baseMessages []agentllm.Message
}

// Manager 负责为 runtime 提供上下文生命周期操作。
// 它只管理上下文状态，不参与工具调度、模型调用或观测逻辑。
type Manager interface {
	Open(ctx stdctx.Context, input agent.Input) (*State, error)
	BuildMessages(state *State, conversation []agentllm.Message) []agentllm.Message
	ApplyToolResult(state *State, patch ToolResultPatch)
}
