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
			Messages: messages,
			Tools:    session.ToolDefinitions,
		}

		completion, err := r.completeStep(ctx, session, runState.step, req)
		if err != nil {
			return nil, observerTrace(session.Observer), err
		}
		assistantTurn := session.Context.ApplyAssistantTurn(contextState, completion.Message)
		if assistantTurn.Error != nil {
			session.Observer.RunFailed(runState.step, "plan_parse", assistantTurn.Error, map[string]any{
				"summary": "运行失败：计划协议块解析失败",
				"content": completion.Content,
			})
			return nil, observerTrace(session.Observer), assistantTurn.Error
		}

		if runState.step == 0 && !contextState.Snapshot.PlanState.Initialized {
			err = errors.New("第一轮缺少 PLAN_STATE")
			session.Observer.RunFailed(runState.step, "plan_init_missing", err, map[string]any{
				"summary": "运行失败：第一轮未生成初始计划",
			})
			return nil, observerTrace(session.Observer), err
		}
		if runState.step == 0 && len(completion.ToolCalls) == 0 {
			if runState.firstStepReminderSent {
				err = errors.New("第一轮未执行首个计划步骤")
				session.Observer.RunFailed(runState.step, "plan_first_step_missing", err, map[string]any{
					"summary":    "运行失败：第一轮没有执行首个计划步骤",
					"plan_state": contextState.Snapshot.PlanState,
				})
				return nil, observerTrace(session.Observer), err
			}
			runState.firstStepReminderSent = true
			conversation = append(conversation, buildFirstStepReminderMessage(contextState.Snapshot.PlanState))
			continue
		}

		if len(completion.ToolCalls) > 0 {
			if shouldAppendConversationMessage(assistantTurn.Message) {
				conversation = append(conversation, assistantTurn.Message)
			}
			r.runToolCalls(ctx, session, &runState, completion.ToolCalls, &conversation)
			continue
		}

		if assistantTurn.HasPlanDirective {
			if shouldAppendConversationMessage(assistantTurn.Message) {
				conversation = append(conversation, assistantTurn.Message)
			}
			continue
		}

		// 这里允许分析步骤在中间轮直接给出最终 JSON。
		// 只要没有新的工具调用或计划更新，且正文已经是合法最终结果，就直接接受并收尾。
		if final, parseErr := finishOutput(completion.Content); parseErr == nil {
			session.Context.AcceptDirectOutput(contextState)
			session.Observer.RunFinished(runState.step, final, contextState.Snapshot.PlanState)
			return final, observerTrace(session.Observer), nil
		}

		if !session.Context.PrepareFinalization(contextState) {
			session.Observer.RunFailed(runState.step, "plan_progress_invalid", errors.New("当前计划尚未完成"), map[string]any{
				"summary":    "运行失败：当前计划尚未完成，也没有新的工具调用或计划更新",
				"content":    completion.Content,
				"plan_state": contextState.Snapshot.PlanState,
			})
			return nil, observerTrace(session.Observer), errors.New("当前计划尚未完成")
		}

		final, err := r.finalizeOutput(ctx, session, runState.step, contextState, conversation)
		if err != nil {
			return nil, observerTrace(session.Observer), err
		}
		session.Context.CompleteFinalization(contextState)
		session.Observer.RunFinished(runState.step, final, contextState.Snapshot.PlanState)
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
		Content: "你已接近本次运行的步骤上限。除非还缺少完成任务必需的信息，否则不要继续扩张计划；如果当前计划已完成或仅剩收尾步骤，请准备输出最终 JSON 结果。",
	}
}

// buildFirstStepReminderMessage 在首轮只生成计划但未执行工具时追加一次强提醒。
// 它只要求模型立刻执行当前 running 步骤，不重新定义新的协议或额外状态。
func buildFirstStepReminderMessage(plan agentcontext.PlanState) agentllm.Message {
	currentTitle := "当前 running 步骤"
	for _, step := range plan.Steps {
		if step.Status == "running" {
			currentTitle = step.Title
			break
		}
	}

	return agentllm.Message{
		Role: "system",
		Content: "你上一轮已经给出了 PLAN_STATE，但没有执行首个计划步骤。" +
			"请直接执行当前 running 步骤对应的数据查询，不要重复输出 PLAN_STATE，不要只解释计划。" +
			"当前必须执行的步骤是：" + currentTitle,
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
	if len(completion.ToolCalls) == 0 && req.ResponseFormat != nil {
		_, parseErr = parseFinalOutput(completion.Content)
	}

	session.Observer.ModelFinished(step, completion, parseErr)
	return completion, nil
}

// finalizeOutput 在计划执行完成后，单独发起一轮不带工具的最终收尾请求。
// 这一轮才要求模型进入 JSON mode，避免中间轮计划协议和最终结果协议互相冲突。
func (r *Runner) finalizeOutput(
	ctx context.Context,
	session Session,
	step int,
	contextState *agentcontext.State,
	conversation []agentllm.Message,
) (map[string]any, error) {
	messages := session.Context.BuildMessages(contextState, conversation)
	messages = append(messages, agentllm.Message{
		Role:    "system",
		Content: "当前计划已执行完毕，请基于现有 Session Context、tool_results 和最近工具返回，直接输出最终 JSON 结果。不要再生成 PLAN_STATE，不要再生成 PLAN_UPDATE，不要再调用工具。overall_summary 必须概括整体状态；report 必须写成至少 3 段的完整分析；key_findings 要给出具体结论，不要只写一句很短的话。",
	})

	completion, err := r.completeStep(ctx, session, step, agentllm.ChatRequest{
		Messages:       messages,
		ResponseFormat: finalOutputResponseFormat(),
	})
	if err != nil {
		return nil, err
	}

	final, err := finishOutput(completion.Content)
	if err != nil {
		session.Observer.RunFailed(step, "finish_validate", err, map[string]any{
			"summary": "运行失败：最终收尾阶段输出不是合法 JSON",
			"content": completion.Content,
		})
		return nil, err
	}
	return final, nil
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

func shouldAppendConversationMessage(message agentllm.Message) bool {
	return strings.TrimSpace(message.Content) != "" || len(message.ToolCalls) > 0 || message.ToolCallID != ""
}

// state 维护 runtime 在单次 Run 中的最小内部状态。
type state struct {
	context               *agentcontext.State
	step                  int
	firstStepReminderSent bool
}

// measureLatency 用于统一采集模型和工具调用的耗时。
func measureLatency(fn func() error) (int64, error) {
	startedAt := time.Now()
	err := fn()
	return time.Since(startedAt).Milliseconds(), err
}
