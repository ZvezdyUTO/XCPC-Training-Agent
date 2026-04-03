package api

import (
	"aATA/internal/app/apperr"
	"aATA/internal/domain"
	anomalylogic "aATA/internal/logic/anomaly"
	"aATA/internal/svc"
	"aATA/pkg/httpx"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// AdminAlert 提供训练异常预警相关接口。
type AdminAlert struct {
	svcCtx  *svc.ServiceContext
	anomaly anomalylogic.Service
}

func NewAdminAlert(svcCtx *svc.ServiceContext, anomaly anomalylogic.Service) *AdminAlert {
	return &AdminAlert{
		svcCtx:  svcCtx,
		anomaly: anomaly,
	}
}

func (h *AdminAlert) InitRegister(engine *gin.Engine) {
	gDetect := engine.Group("v1/admin/anomaly", h.svcCtx.JwtMid.Handler, h.svcCtx.AdminMid.Handler)
	gDetect.POST("/detect/run", h.RunDetect)
	gDetect.GET("/config", h.GetRuleConfig)
	gDetect.POST("/config", h.UpdateRuleConfig)

	gAlerts := engine.Group("v1/admin/alerts", h.svcCtx.JwtMid.Handler, h.svcCtx.AdminMid.Handler)
	gAlerts.GET("/list", h.ListAlerts)
	gAlerts.POST("/resolve/all", h.ResolveAllAlerts)
	gAlerts.POST("/:id/ack", h.AckAlert)
	gAlerts.POST("/:id/resolve", h.ResolveAlert)
}

// RunDetect 手动触发一次全量异常检测。
func (h *AdminAlert) RunDetect(ctx *gin.Context) {
	cnt, err := h.anomaly.DetectAllUsers(ctx.Request.Context(), time.Now())
	if err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	httpx.OkWithData(ctx, gin.H{
		"msg":         "success",
		"alert_cnt":   cnt,
		"detected_at": time.Now().Format("2006-01-02 15:04:05"),
	})
}

// GetRuleConfig 查询当前异常检测规则参数。
func (h *AdminAlert) GetRuleConfig(ctx *gin.Context) {
	cfg := h.anomaly.GetRuleConfig(ctx.Request.Context())
	httpx.OkWithData(ctx, cfg)
}

// UpdateRuleConfig 更新异常检测规则参数。
func (h *AdminAlert) UpdateRuleConfig(ctx *gin.Context) {
	var req anomalylogic.RuleConfigPatch
	if err := ctx.ShouldBindJSON(&req); err != nil {
		httpx.FailWithErr(ctx, apperr.New(apperr.KindUser, "invalid_request_body", "请求体格式错误，请使用 JSON", 400))
		return
	}

	next, err := h.anomaly.PatchRuleConfig(ctx.Request.Context(), req)
	if err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	httpx.OkWithData(ctx, gin.H{
		"msg":    "success",
		"config": next,
	})
}

// ListAlerts 查询预警列表。
func (h *AdminAlert) ListAlerts(ctx *gin.Context) {
	var req domain.AdminAlertListReq
	if err := httpx.BindAndValidate(ctx, &req); err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	res, err := h.anomaly.ListAlerts(ctx.Request.Context(), &req)
	if err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	httpx.OkWithData(ctx, res)
}

// AckAlert 把某条预警标记为已确认。
func (h *AdminAlert) AckAlert(ctx *gin.Context) {
	idRaw := ctx.Param("id")
	id, err := strconv.ParseInt(idRaw, 10, 64)
	if err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	if err := h.anomaly.AckAlert(ctx.Request.Context(), id); err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	httpx.Ok(ctx)
}

// ResolveAlert 把某条预警标记为已处理完成。
func (h *AdminAlert) ResolveAlert(ctx *gin.Context) {
	idRaw := ctx.Param("id")
	id, err := strconv.ParseInt(idRaw, 10, 64)
	if err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	if err := h.anomaly.ResolveAlert(ctx.Request.Context(), id); err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	httpx.Ok(ctx)
}

// ResolveAllAlerts 一键将所有未处理预警标记为已处理完成。
func (h *AdminAlert) ResolveAllAlerts(ctx *gin.Context) {
	cnt, err := h.anomaly.ResolveAllAlerts(ctx.Request.Context())
	if err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	httpx.OkWithData(ctx, gin.H{
		"msg":          "success",
		"resolved_cnt": cnt,
	})
}
