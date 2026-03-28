package context

import (
	"aATA/internal/logic/agent"
	agentmodel "aATA/internal/logic/agent/model"
	stdctx "context"
)

// State 保存一次运行已经解析好的上下文状态。
// 它对 runtime 暴露的是稳定结构，而不是 memory loader/snapshot builder 的细节。
type State struct {
	BaseMessages    []agentmodel.Message
	Snapshot        any
	ResolvedPaths   []string
	AppliedMemories []string
}

// Manager 负责为 runtime 提供上下文生命周期操作。
type Manager interface {
	Open(ctx stdctx.Context, input agent.Input) (*State, error)
	Messages(state *State, conversation []agentmodel.Message) []agentmodel.Message
	RecordTool(state *State, toolName string, ok bool)
}
