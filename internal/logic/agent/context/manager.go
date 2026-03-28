package context

import (
	"aATA/internal/logic/agent"
	agentmodel "aATA/internal/logic/agent/model"
	stdctx "context"
)

// DefaultManager 是当前默认的上下文实现：
// 从磁盘加载 memory，初始化 snapshot，并按运行进度组装模型输入消息。
type DefaultManager struct {
	loader *Loader
}

// NewManager 创建基于文件系统 memory 的上下文管理器。
func NewManager(memoryDir string) *DefaultManager {
	return &DefaultManager{
		loader: NewLoader(memoryDir),
	}
}

// Open 解析本次任务的 memory，并初始化基础消息与 session snapshot。
func (m *DefaultManager) Open(_ stdctx.Context, input agent.Input) (*State, error) {
	resolvedPaths := input.MemoryPaths()
	bundle, err := m.loader.Load(resolvedPaths)
	if err != nil {
		return nil, err
	}

	return &State{
		BaseMessages:    buildBaseMessages(input, bundle),
		Snapshot:        newSessionSnapshot(input),
		ResolvedPaths:   resolvedPaths,
		AppliedMemories: appliedMemoryNames(bundle),
	}, nil
}

// Messages 生成本轮模型调用所需的完整消息列表。
func (m *DefaultManager) Messages(state *State, conversation []agentmodel.Message) []agentmodel.Message {
	if state == nil {
		return nil
	}
	snapshot, _ := state.Snapshot.(*sessionSnapshot)
	return buildRequestMessages(state.BaseMessages, snapshot, conversation)
}

// RecordTool 将工具调用结果写回 snapshot，供后续轮次上下文使用。
func (m *DefaultManager) RecordTool(state *State, toolName string, ok bool) {
	if state == nil {
		return
	}
	snapshot, _ := state.Snapshot.(*sessionSnapshot)
	if snapshot == nil {
		return
	}
	snapshot.recordToolResult(toolName, ok)
}
