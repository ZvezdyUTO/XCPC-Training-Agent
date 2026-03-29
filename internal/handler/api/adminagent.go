package api

import (
	"aATA/internal/logic/agent"
	agentservice "aATA/internal/logic/agent/service"

	"github.com/gin-gonic/gin"

	"aATA/internal/domain"
	"aATA/internal/svc"
	"aATA/pkg/httpx"
	"strings"
)

type AdminAgent struct {
	svcCtx *svc.ServiceContext
	agent  agentservice.Service
}

func NewAdminAgent(svcCtx *svc.ServiceContext, agent agentservice.Service) *AdminAgent {
	return &AdminAgent{svcCtx: svcCtx, agent: agent}
}

func (h *AdminAgent) InitRegister(engine *gin.Engine) {
	g := engine.Group("v1/admin/agent", h.svcCtx.JwtMid.Handler, h.svcCtx.AdminMid.Handler)
	g.POST("/task/run", h.RunTask)
}

func (h *AdminAgent) RunTask(ctx *gin.Context) {
	var req domain.AdminAgentTaskRunReq
	if err := httpx.BindAndValidate(ctx, &req); err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	res, trace, err := h.agent.RunTask(ctx.Request.Context(), &req)
	if err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	body := gin.H{
		"task":        req.Task,
		"result":      res,
		"token_usage": trace.TokenUsage,
	}
	if shouldReturnTrace(req.TraceMode) {
		body["trace"] = trace
	}

	httpx.OkWithData(ctx, body)
}

// shouldReturnTrace 控制是否向 HTTP 调用方返回完整 trace。
// 默认不返回，只有显式请求 summary/debug 时才返回。
func shouldReturnTrace(raw string) bool {
	switch agent.Mode(strings.ToLower(strings.TrimSpace(raw))) {
	case agent.ModeSummary, agent.ModeDebug:
		return true
	default:
		return false
	}
}
