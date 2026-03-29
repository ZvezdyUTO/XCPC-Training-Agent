package runtime

import (
	"aATA/internal/logic/agent"
	agentcontext "aATA/internal/logic/agent/context"
	agentllm "aATA/internal/logic/agent/llm"
	agentobserve "aATA/internal/logic/agent/observe"
	agenttooling "aATA/internal/logic/agent/tooling"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const (
	defaultMaxSteps  = 10
	wrapUpStepWindow = 2
)

// Session 描述一次运行已经完成装配的执行会话。
// runtime 只消费这些稳定依赖，不负责再创建或翻译其它模块。
type Session struct {
	Input           agent.Input
	ToolNames       []string
	ToolDefinitions []agentllm.ToolDefinition
	Model           agentllm.Client
	Tools           agenttooling.Toolbox
	Context         agentcontext.Manager
	Observer        agentobserve.Observer
}

// Runner 是 Agent 的执行内核。
// 它只关心运行编排，不直接负责依赖装配、协议嫁接或模块私有策略。
type Runner struct {
	MaxSteps int
}

// NewRunner 创建默认配置的运行时。
func NewRunner() *Runner {
	return &Runner{MaxSteps: defaultMaxSteps}
}

// Run 执行完整的模型-工具闭环，直到得到最终 JSON 输出或触发失败条件。
func (r *Runner) Run(ctx context.Context, session Session) (map[string]any, agent.RunTrace, error) {
	if err := validateSession(session); err != nil {
		return nil, agent.RunTrace{}, err
	}

	maxSteps := r.MaxSteps
	if maxSteps <= 0 {
		maxSteps = defaultMaxSteps
	}

	session.Observer.RunStarted(session.Input, session.ToolNames)

	contextState, err := session.Context.Open(ctx, session.Input)
	if err != nil {
		session.Observer.RunFailed(0, "context_open", err, map[string]any{
			"summary": "运行失败：上下文初始化失败",
		})
		return nil, observerTrace(session.Observer), err
	}

	runState := state{context: contextState}
	conversation := make([]agentllm.Message, 0, 16)

	for runState.step = 0; runState.step < maxSteps; runState.step++ {
		messages := session.Context.BuildMessages(contextState, conversation)
		if shouldInjectWrapUpHint(runState.step, maxSteps) {
			messages = append(messages, buildWrapUpHintMessage())
		}

		req := agentllm.ChatRequest{
			Messages:       messages,
			Tools:          session.ToolDefinitions,
			ResponseFormat: finalOutputResponseFormat(),
		}

		completion, err := r.completeStep(ctx, session, runState.step, req)
		if err != nil {
			return nil, observerTrace(session.Observer), err
		}

		if len(completion.ToolCalls) > 0 {
			conversation = append(conversation, completion.Message)
			r.runToolCalls(ctx, session, &runState, completion.ToolCalls, &conversation)
			continue
		}

		final, err := finishOutput(completion.Content)
		if err != nil {
			session.Observer.RunFailed(runState.step, "finish_validate", err, map[string]any{
				"summary": "运行失败：模型最终输出不是合法 JSON",
				"content": completion.Content,
			})
			return nil, observerTrace(session.Observer), err
		}

		session.Observer.RunFinished(runState.step, final)
		return final, observerTrace(session.Observer), nil
	}

	err = errors.New("执行步数超过上限")
	session.Observer.RunFailed(runState.step, "loop_guard", err, map[string]any{
		"summary": "运行失败：执行步数超过上限",
	})
	return nil, observerTrace(session.Observer), err
}

// shouldInjectWrapUpHint 判断当前轮次是否已接近硬步数上限。
// 这个提示只在临近终止时追加一次轻量收尾指令，不改变其它模块边界。
func shouldInjectWrapUpHint(step, maxSteps int) bool {
	return maxSteps > wrapUpStepWindow && step >= maxSteps-wrapUpStepWindow
}

// buildWrapUpHintMessage 为模型提供最小收尾指令。
// 它只提醒模型优先结束，不引入新的运行状态或额外协议。
func buildWrapUpHintMessage() agentllm.Message {
	return agentllm.Message{
		Role:    "system",
		Content: "你已接近本次运行的步骤上限。除非还缺少完成任务必需的信息，否则不要继续调用工具，请优先基于现有信息整理并输出最终结果。最终输出必须是纯 JSON 对象，不要使用代码块包裹 JSON。",
	}
}

// completeStep 负责一次模型调用，并在返回普通答案时提前做最终结果校验。
func (r *Runner) completeStep(ctx context.Context, session Session, step int, req agentllm.ChatRequest) (*agentllm.ChatCompletion, error) {
	session.Observer.ModelStarted(step, req)
	completion, err := session.Model.Chat(ctx, req)
	if err != nil {
		session.Observer.RunFailed(step, "model_call", err, map[string]any{
			"summary":       "运行失败：模型调用失败",
			"message_count": len(req.Messages),
			"tool_count":    len(req.Tools),
		})
		return nil, err
	}

	var parseErr error
	if len(completion.ToolCalls) == 0 {
		_, parseErr = parseFinalOutput(completion.Content)
	}

	session.Observer.ModelFinished(step, completion, parseErr)
	return completion, nil
}

// runToolCalls 顺序执行当前轮次模型返回的所有工具调用，并把 tool message 追加回对话历史。
func (r *Runner) runToolCalls(ctx context.Context, session Session, state *state, calls []agentllm.ToolCall, messages *[]agentllm.Message) {
	for _, call := range calls {
		toolMessage, _ := r.runToolCall(ctx, session, state, call)
		*messages = append(*messages, toolMessage)
	}
}

// runToolCall 负责执行单个工具调用，并把完整结果编码成 role=tool 消息。
func (r *Runner) runToolCall(ctx context.Context, session Session, state *state, call agentllm.ToolCall) (agentllm.Message, error) {
	session.Observer.ToolStarted(state.step, call.Function.Name, call.Function.Arguments, call.ID)

	var (
		result  agenttooling.CallResult
		callErr error
	)
	latencyMs, _ := measureLatency(func() error {
		rawArgs := []byte(strings.TrimSpace(call.Function.Arguments))
		if len(rawArgs) == 0 {
			rawArgs = []byte("{}")
		}
		result, callErr = session.Tools.Call(ctx, call.Function.Name, rawArgs)
		return callErr
	})

	session.Observer.ToolFinished(state.step, result, callErr, latencyMs)
	session.Context.ApplyToolResult(state.context, agentcontext.ToolResultPatch{
		ToolName: call.Function.Name,
		Success:  callErr == nil,
		Args:     decodeToolArguments(call.Function.Arguments),
		Result:   result.Result,
	})

	payload, _ := json.Marshal(result.Result)
	return agentllm.Message{
		Role:       "tool",
		ToolCallID: call.ID,
		Content:    string(payload),
	}, callErr
}

// validateSession 校验一次运行所需的最小依赖是否已经装配完成。
func validateSession(session Session) error {
	switch {
	case session.Model == nil:
		return errors.New("缺少 llm client")
	case session.Tools == nil:
		return errors.New("缺少 toolbox")
	case session.Context == nil:
		return errors.New("缺少 context manager")
	case session.Observer == nil:
		return errors.New("缺少 observer")
	default:
		return nil
	}
}

// decodeToolArguments 把模型返回的 JSON 参数解码成上下文可保存的结构化对象。
func decodeToolArguments(raw string) map[string]any {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	var args map[string]any
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return nil
	}
	return args
}

// observerTrace 在 observer 支持 trace 导出时提取最终结果。
func observerTrace(observer agentobserve.Observer) agent.RunTrace {
	traced, ok := observer.(agentobserve.TraceResult)
	if !ok {
		return agent.RunTrace{}
	}
	return traced.Result()
}

// state 维护 runtime 在单次 Run 中的最小内部状态。
type state struct {
	context *agentcontext.State
	step    int
}

// measureLatency 用于统一采集模型和工具调用的耗时。
func measureLatency(fn func() error) (int64, error) {
	startedAt := time.Now()
	err := fn()
	return time.Since(startedAt).Milliseconds(), err
}
