package service

import (
	"aATA/internal/domain"
	"aATA/internal/logic/agent"
	agentcontext "aATA/internal/logic/agent/context"
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
	tools.Register(l.svcCtx.ContestRatingSummaryTool)
	tools.Register(l.svcCtx.TrainingDayLeaderboardTool)
	tools.Register(l.svcCtx.TrainingWeekLeaderboardTool)
	tools.Register(l.svcCtx.TrainingMonthLeaderboardTool)
	tools.Register(l.svcCtx.ContestRankingTool)

	traceCollector := agentobserve.NewCollector(parseTraceMode(req.TraceMode))
	contextManager := agentcontext.NewManager(os.Getenv("AGENT_MEMORY_DIR"))
	observerFactory := agentobserve.NewTraceObserverFactory(traceCollector)

	runner := agentruntime.NewRunner(
		l.svcCtx.LLMClient,
		tools,
		contextManager,
		observerFactory,
	)

	input := agent.Input{
		Query:  req.Task,
		Params: req.Params,
	}

	return runner.Run(ctx, input)
}

// parseTraceMode 将外部字符串参数收敛到受支持的 trace 模式。
func parseTraceMode(raw string) agent.Mode {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", string(agent.ModeSummary):
		return agent.ModeSummary
	case string(agent.ModeDebug):
		return agent.ModeDebug
	default:
		return agent.ModeSummary
	}
}
