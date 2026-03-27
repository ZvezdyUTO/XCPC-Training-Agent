package logic

import (
	"aATA/internal/domain"
	"aATA/internal/logic/agent"
	"aATA/internal/logic/agenttrace"
	"aATA/internal/svc"
	"context"
	"errors"
	"strings"
)

type AgentLogic interface {
	RunTask(ctx context.Context, req *domain.AdminAgentTaskRunReq) (map[string]interface{}, agenttrace.RunTrace, error)
}

type defaultAgentLogic struct {
	svcCtx *svc.ServiceContext
}

func NewAgentLogic(svcCtx *svc.ServiceContext) AgentLogic {
	return &defaultAgentLogic{
		svcCtx: svcCtx,
	}
}

func (l *defaultAgentLogic) RunTask(
	ctx context.Context,
	req *domain.AdminAgentTaskRunReq,
) (map[string]interface{}, agenttrace.RunTrace, error) {

	if req == nil || req.Task == "" {
		return nil, agenttrace.RunTrace{}, errors.New("task required")
	}

	// 1️⃣ 创建 registry（任务级别）
	reg := agent.NewRegistry()

	// 2️⃣ 注册工具（从 svcCtx 拿实例）
	reg.Register(l.svcCtx.TrainingSummaryTool)
	reg.Register(l.svcCtx.ContestRatingSummaryTool)
	reg.Register(l.svcCtx.TrainingDayLeaderboardTool)
	reg.Register(l.svcCtx.TrainingWeekLeaderboardTool)
	reg.Register(l.svcCtx.TrainingMonthLeaderboardTool)
	reg.Register(l.svcCtx.ContestRankingTool)

	traceCollector := agenttrace.NewCollector(parseTraceMode(req.TraceMode))

	// 3️⃣ 创建 controller（任务级别）
	ctrl := agent.NewController(
		l.svcCtx.LLMClient,
		reg,
		traceCollector,
	)

	input := agent.AgentInput{
		Query:  req.Task,
		Params: req.Params,
	}

	return ctrl.Run(ctx, input)
}

func parseTraceMode(raw string) agenttrace.Mode {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", string(agenttrace.ModeSummary):
		return agenttrace.ModeSummary
	case string(agenttrace.ModeDebug):
		return agenttrace.ModeDebug
	default:
		return agenttrace.ModeSummary
	}
}
