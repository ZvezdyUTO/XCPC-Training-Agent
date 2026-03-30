package service

import (
	"aATA/internal/app/logx"
	"aATA/internal/domain"
	"aATA/internal/logic/agent"
	agentcontext "aATA/internal/logic/agent/context"
	agentllm "aATA/internal/logic/agent/llm"
	agentobserve "aATA/internal/logic/agent/observe"
	agentruntime "aATA/internal/logic/agent/runtime"
	agenttooling "aATA/internal/logic/agent/tooling"
	"aATA/internal/svc"
	"context"
	"errors"
	"os"
	"strings"
)

// Service 是 Agent 模块暴露给 HTTP/Logic 层的最小服务接口。
type Service interface {
	RunTask(ctx context.Context, req *domain.AdminAgentTaskRunReq) (map[string]interface{}, agent.RunTrace, error)
}

// defaultService 负责把 ServiceContext 中的基础设施装配成一次可执行的 Agent 运行。
type defaultService struct {
	svcCtx *svc.ServiceContext
}

// New 创建 Agent 服务入口。
func New(svcCtx *svc.ServiceContext) Service {
	return &defaultService{svcCtx: svcCtx}
}

// RunTask 把外部任务请求翻译成 runtime 可执行的输入，并完成本次运行的依赖装配。
func (l *defaultService) RunTask(
	ctx context.Context,
	req *domain.AdminAgentTaskRunReq,
) (map[string]interface{}, agent.RunTrace, error) {
	if req == nil || req.Task == "" {
		return nil, agent.RunTrace{}, errors.New("任务不能为空")
	}

	tools := agenttooling.NewToolbox()
	tools.Register(l.svcCtx.TrainingSummaryTool)
	tools.Register(l.svcCtx.StudentContestRecordsTool)
	tools.Register(l.svcCtx.TrainingValueLeaderboardTool)
	tools.Register(l.svcCtx.ContestRankingTool)

	traceCollector := agentobserve.NewCollector(parseCollectorMode(req.TraceMode))
	contextManager := agentcontext.NewManager(os.Getenv("AGENT_MEMORY_DIR"))
	toolSpecs := tools.Definitions()

	input := agent.Input{
		Query:  req.Task,
		Params: req.Params,
	}

	toolNames := toolNamesFromSpecs(toolSpecs)
	toolDefs := renderLLMToolDefinitions(toolSpecs)
	observer := agentobserve.NewTraceObserverFactory(traceCollector).New(
		l.svcCtx.LLMClient,
		input,
		toolNames,
	)

	runner := agentruntime.NewRunner()
	result, trace, err := runner.Run(ctx, agentruntime.Session{
		Input:           input,
		ToolNames:       toolNames,
		ToolDefinitions: toolDefs,
		Model:           l.svcCtx.LLMClient,
		Tools:           tools,
		Context:         contextManager,
		Observer:        observer,
	})
	logTraceSummary(trace, err)
	return result, trace, err
}

// parseTraceMode 将外部字符串参数收敛到受支持的 trace 返回模式。
// 默认 none，表示不向 HTTP 调用方返回完整 trace。
func parseTraceMode(raw string) agent.Mode {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", string(agent.ModeNone):
		return agent.ModeNone
	case string(agent.ModeSummary):
		return agent.ModeSummary
	case string(agent.ModeDebug):
		return agent.ModeDebug
	default:
		return agent.ModeNone
	}
}

// parseCollectorMode 决定内部 trace 的采集粒度。
// 默认虽然不回传 trace，但仍保留 summary 用于内部日志。
func parseCollectorMode(raw string) agent.Mode {
	mode := parseTraceMode(raw)
	if mode == agent.ModeNone {
		return agent.ModeSummary
	}
	return mode
}

// toolNamesFromSpecs 导出本次运行对外注册的工具名称列表。
func toolNamesFromSpecs(specs []agenttooling.ToolSpec) []string {
	names := make([]string, 0, len(specs))
	for _, spec := range specs {
		if spec.Name != "" {
			names = append(names, spec.Name)
		}
	}
	return names
}

// renderLLMToolDefinitions 在装配层把工具规格投影成当前 LLM 协议所需结构。
func renderLLMToolDefinitions(specs []agenttooling.ToolSpec) []agentllm.ToolDefinition {
	defs := make([]agentllm.ToolDefinition, 0, len(specs))
	for _, spec := range specs {
		defs = append(defs, agentllm.ToolDefinition{
			Type: "function",
			Function: agentllm.ToolFunctionDefinition{
				Name:        spec.Name,
				Description: spec.Description,
				Parameters:  renderToolSchema(spec.Schema),
			},
		})
	}
	return defs
}

// renderToolSchema 将 tooling 域参数结构转为当前 LLM 协议需要的 JSON Schema 片段。
func renderToolSchema(schema agenttooling.ToolSchema) map[string]any {
	properties := make(map[string]any, len(schema.Parameters))
	for name, param := range schema.Parameters {
		property := map[string]any{
			"type":        param.Type,
			"description": param.Description,
		}
		if len(param.Enum) > 0 {
			property["enum"] = param.Enum
		}
		properties[name] = property
	}

	return map[string]any{
		"type":                 "object",
		"properties":           properties,
		"required":             schema.Required,
		"additionalProperties": false,
	}
}

// logTraceSummary 把一次运行的最小 trace 摘要打印到内部日志。
// 这是默认模式下排查 agent 运行问题的主出口。
func logTraceSummary(trace agent.RunTrace, err error) {
	fields := map[string]any{
		"run_id":        trace.RunID,
		"mode":          trace.Mode,
		"model_calls":   trace.TokenUsage.ModelCallCount,
		"input_tokens":  trace.TokenUsage.InputTokens,
		"output_tokens": trace.TokenUsage.OutputTokens,
		"total_tokens":  trace.TokenUsage.TotalTokens,
		"event_count":   len(trace.Events),
		"span_count":    len(trace.Spans),
	}
	if err != nil {
		logx.Error("agent.run", err, fields)
		return
	}
	logx.Info("agent.run", fields)
}
