package api

import (
	"aATA/internal/app/apperr"
	"aATA/internal/logic/agent"
	agentservice "aATA/internal/logic/agent/service"
	"fmt"

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
		h.failRunTaskWithTrace(ctx, req, trace, err)
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

// failRunTaskWithTrace 在 Agent 运行失败时保留最小运行上下文。
// 这里不改变全局错误协议，只在本接口额外带回 trace 和 token 使用概况，便于前端排障。
func (h *AdminAgent) failRunTaskWithTrace(
	ctx *gin.Context,
	req domain.AdminAgentTaskRunReq,
	trace agent.RunTrace,
	err error,
) {
	body := gin.H{
		"task":        req.Task,
		"token_usage": trace.TokenUsage,
	}
	if shouldReturnTrace(req.TraceMode) {
		body["trace"] = trace
	}

	httpCode := 500
	bizCode := 50000
	msg := "internal server error"
	errCode := ""

	if appErr, ok := apperr.As(err); ok {
		if appErr.Kind == apperr.KindInternal {
			fmt.Println("[internal error]", err)
		}
		if appErr.HTTPStatus > 0 {
			httpCode = appErr.HTTPStatus
		}
		if appErr.Message != "" {
			msg = appErr.Message
		}
		errCode = appErr.Code
	} else if err != nil {
		fmt.Println("[internal error]", err)
		msg = err.Error()
	}

	httpx.Result(ctx, httpCode, bizCode, body, msg, errCode)
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
