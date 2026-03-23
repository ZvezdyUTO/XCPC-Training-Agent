package logic

import (
	"aATA/internal/domain"
	"aATA/internal/logic/agent"
	"aATA/internal/svc"
	"context"
	"errors"
)

type AgentLogic interface {
	RunTask(ctx context.Context, req *domain.AdminAgentTaskRunReq) (map[string]interface{}, []string, error)
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
) (map[string]interface{}, []string, error) {

	if req == nil || req.Task == "" {
		return nil, nil, errors.New("task required")
	}

	// 1️⃣ 创建 registry（任务级别）
	reg := agent.NewRegistry()

	// 2️⃣ 注册工具（从 svcCtx 拿实例）
	reg.Register(l.svcCtx.TrainingSummaryTool)
	reg.Register(l.svcCtx.ContestRatingSummaryTool)
	reg.Register(l.svcCtx.TrainingDayLeaderboardTool)
	reg.Register(l.svcCtx.TrainingWeekLeaderboardTool)
	reg.Register(l.svcCtx.TrainingMonthLeaderboardTool)

	// 3️⃣ 创建 controller（任务级别）
	ctrl := agent.NewController(
		l.svcCtx.LLMClient,
		reg,
	)

	input := agent.AgentInput{
		Query:  req.Task,
		Params: req.Params,
	}

	return ctrl.Run(ctx, input)
}
