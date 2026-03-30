package context

import (
	"aATA/internal/logic/agent"
	agentllm "aATA/internal/logic/agent/llm"
	stdctx "context"
)

// DefaultManager 是当前默认的上下文实现：
// 从磁盘加载 memory，初始化运行状态，并在每轮调用前生成模型消息。
type DefaultManager struct {
	loader *Loader
}

// NewManager 创建基于文件系统 memory 的上下文管理器。
func NewManager(memoryDir string) *DefaultManager {
	return &DefaultManager{
		loader: NewLoader(memoryDir),
	}
}

// Open 解析本次任务的 memory，并初始化本次运行的上下文状态。
func (m *DefaultManager) Open(_ stdctx.Context, input agent.Input) (*State, error) {
	bundle, err := m.loader.Load(input.MemoryPaths())
	if err != nil {
		return nil, err
	}

	return &State{
		Snapshot:     newSessionSnapshot(input),
		ToolResults:  []ToolResultSummary{},
		baseMessages: buildBaseMessages(input, bundle), // 构造基础消息
	}, nil
}

// BuildMessages 根据当前状态和最近会话生成下一轮模型调用消息。
func (m *DefaultManager) BuildMessages(state *State, conversation []agentllm.Message) []agentllm.Message {
	if state == nil {
		return nil
	}

	out := make([]agentllm.Message, 0, len(state.baseMessages)+1+recentConversationLimit)
	out = append(out, state.baseMessages[:len(state.baseMessages)-1]...) // 稳定 system 消息
	out = append(out, agentllm.Message{                                  // 当前运行状态消息
		Role:    "system",
		Content: buildContextStateMessage(state),
	})
	out = append(out, state.baseMessages[len(state.baseMessages)-1]) // 当前任务输入
	out = append(out, recentConversation(conversation)...)           // 最近会话历史
	return out
}

// ApplyAssistantTurn 解析 assistant 回合中的计划协议块，并返回适合继续写入对话历史的消息。
// 计划原文在解析后不再作为长期真相来源，而是转成结构化状态回写到 Snapshot。
func (m *DefaultManager) ApplyAssistantTurn(state *State, message agentllm.Message) AssistantTurnOutcome {
	if state == nil {
		return AssistantTurnOutcome{Message: message}
	}
	return applyAssistantTurnToSnapshot(&state.Snapshot, message)
}

// ApplyToolResult 将一次工具调用结果写回上下文状态，供后续轮次推理使用。
func (m *DefaultManager) ApplyToolResult(state *State, patch ToolResultPatch) {
	if state == nil {
		return
	}
	state.ToolResults = appendToolResultSummary(state.ToolResults, patch)
	applyToolResultToSnapshot(&state.Snapshot, patch)
}

// PrepareFinalization 在进入最终 JSON 收尾前整理计划状态。
// 它只在计划已经执行完毕，或仅剩最后一个收尾步骤时返回 true。
func (m *DefaultManager) PrepareFinalization(state *State) bool {
	if state == nil {
		return false
	}
	return preparePlanForFinalization(&state.Snapshot.PlanState)
}

// CompleteFinalization 在最终 JSON 输出成功后提交计划完成状态。
// 这里只做最后一步的状态收尾，不负责判断当前是否允许进入收尾阶段。
func (m *DefaultManager) CompleteFinalization(state *State) {
	if state == nil {
		return
	}
	completePlanAfterFinalization(&state.Snapshot.PlanState)
}

// AcceptDirectOutput 在中间轮直接收到合法最终 JSON 时强制收尾计划状态。
// 这里用于“分析步骤直接给出最终答案”的场景，避免 trace 中残留 waiting 步骤。
func (m *DefaultManager) AcceptDirectOutput(state *State) {
	if state == nil {
		return
	}
	completePlanAfterDirectOutput(&state.Snapshot.PlanState)
}
