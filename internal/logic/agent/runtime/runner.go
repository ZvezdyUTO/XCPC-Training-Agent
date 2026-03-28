package runtime

import (
	"aATA/internal/logic/agent"
	agentcontext "aATA/internal/logic/agent/context"
	agentmodel "aATA/internal/logic/agent/model"
	agentobserve "aATA/internal/logic/agent/observe"
	agenttooling "aATA/internal/logic/agent/tooling"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const maxSteps = 10

// Runner 是 Agent 的执行内核。
// 它只关心运行编排，不直接负责 memory 规则、工具实现或 trace 落盘细节。
type Runner struct {
	Model   agentmodel.Client
	Tools   agenttooling.Toolbox
	Context agentcontext.Manager
	Observe agentobserve.Factory
}

// NewRunner 创建一个最小依赖集的运行时。
func NewRunner(
	model agentmodel.Client,
	tools agenttooling.Toolbox,
	contextManager agentcontext.Manager,
	observerFactory agentobserve.Factory,
) *Runner {
	return &Runner{
		Model:   model,
		Tools:   tools,
		Context: contextManager,
		Observe: observerFactory,
	}
}

// Run 执行完整的模型-工具闭环，直到得到最终 JSON 输出或触发失败条件。
func (r *Runner) Run(ctx context.Context, input agent.Input) (map[string]any, agent.RunTrace, error) {
	if r.Model == nil {
		return nil, agent.RunTrace{}, errors.New("缺少 llm client")
	}
	if r.Tools == nil {
		return nil, agent.RunTrace{}, errors.New("缺少 toolbox")
	}
	if r.Context == nil {
		return nil, agent.RunTrace{}, errors.New("缺少 context manager")
	}
	if r.Observe == nil {
		return nil, agent.RunTrace{}, errors.New("缺少 observer factory")
	}

	toolDefs := r.Tools.Definitions()
	toolNames := toolNamesFromDefinitions(toolDefs)
	observer := r.Observe.New(r.Model, input, toolNames)
	observer.RunStarted(input, toolNames)

	contextState, err := r.Context.Open(ctx, input)
	if err != nil {
		observer.RunFailed(0, "context_open", err, map[string]any{
			"summary": "运行失败：上下文初始化失败",
		})
		return nil, observerTrace(observer), err
	}

	state := state{
		input:   input,
		context: contextState,
	}
	conversation := make([]agentmodel.Message, 0, 16)

	for state.step = 0; state.step < maxSteps; state.step++ {
		req := agentmodel.ChatRequest{
			Messages: r.Context.Messages(contextState, conversation),
			Tools:    toolDefs,
		}

		completion, err := r.completeStep(ctx, state.step, observer, req)
		if err != nil {
			return nil, observerTrace(observer), err
		}

		if len(completion.ToolCalls) > 0 {
			conversation = append(conversation, completion.Message)
			r.runToolCalls(ctx, &state, observer, completion.ToolCalls, &conversation)
			continue
		}

		final, err := finishOutput(completion.Content)
		if err != nil {
			observer.RunFailed(state.step, "finish_validate", err, map[string]any{
				"summary": "运行失败：模型最终输出不是合法 JSON",
				"content": completion.Content,
			})
			return nil, observerTrace(observer), err
		}

		observer.RunFinished(state.step, final)
		return final, observerTrace(observer), nil
	}

	err = errors.New("执行步数超过上限")
	observer.RunFailed(state.step, "loop_guard", err, map[string]any{
		"summary": "运行失败：执行步数超过上限",
	})
	return nil, observerTrace(observer), err
}

// completeStep 负责一次模型调用，并在返回普通答案时提前做最终结果校验。
func (r *Runner) completeStep(ctx context.Context, step int, observer agentobserve.Observer, req agentmodel.ChatRequest) (*agentmodel.ChatCompletion, error) {
	observer.ModelStarted(step, req)
	completion, err := r.Model.Chat(ctx, req)
	if err != nil {
		observer.RunFailed(step, "model_call", err, map[string]any{
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

	observer.ModelFinished(step, completion, parseErr)
	return completion, nil
}

// runToolCalls 顺序执行当前轮次模型返回的所有工具调用，并把 tool message 追加回对话历史。
func (r *Runner) runToolCalls(ctx context.Context, state *state, observer agentobserve.Observer, calls []agentmodel.ToolCall, messages *[]agentmodel.Message) {
	for _, call := range calls {
		toolMessage, _ := r.runToolCall(ctx, state, observer, call)
		*messages = append(*messages, toolMessage)
	}
}

// runToolCall 负责执行单个工具调用，并把摘要结果编码成 role=tool 消息。
func (r *Runner) runToolCall(ctx context.Context, state *state, observer agentobserve.Observer, call agentmodel.ToolCall) (agentmodel.Message, error) {
	observer.ToolStarted(state.step, call.Function.Name, call.Function.Arguments, call.ID)

	var (
		result  agenttooling.CallResult
		callErr error
	)
	latencyMs, _ := measureLatency(func() error {
		rawArgs := []byte(strings.TrimSpace(call.Function.Arguments))
		if len(rawArgs) == 0 {
			rawArgs = []byte("{}")
		}
		result, callErr = r.Tools.Call(ctx, call.Function.Name, rawArgs)
		return callErr
	})

	observer.ToolFinished(state.step, result, callErr, latencyMs)
	r.Context.RecordTool(state.context, call.Function.Name, callErr == nil)

	payload, _ := json.Marshal(result.Summary)
	return agentmodel.Message{
		Role:       "tool",
		ToolCallID: call.ID,
		Content:    string(payload),
	}, callErr
}

// toolNamesFromDefinitions 用于给 observer 提供稳定的工具名称列表。
func toolNamesFromDefinitions(defs []agentmodel.ToolDefinition) []string {
	names := make([]string, 0, len(defs))
	for _, def := range defs {
		if def.Function.Name != "" {
			names = append(names, def.Function.Name)
		}
	}
	return names
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
	input   agent.Input
	context *agentcontext.State
	step    int
}

// measureLatency 用于统一采集模型和工具调用的耗时。
func measureLatency(fn func() error) (int64, error) {
	startedAt := time.Now()
	err := fn()
	return time.Since(startedAt).Milliseconds(), err
}
