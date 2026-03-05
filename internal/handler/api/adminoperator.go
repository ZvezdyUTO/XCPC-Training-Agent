package api

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"aATA/internal/domain"
	"aATA/internal/logic"
	"aATA/internal/svc"
	"aATA/pkg/httpx"
)

type AdminOperator struct {
	svcCtx   *svc.ServiceContext
	training logic.TrainingLogic
}

func NewAdminOperator(svcCtx *svc.ServiceContext, training logic.TrainingLogic) *AdminOperator {
	return &AdminOperator{
		svcCtx:   svcCtx,
		training: training,
	}
}

func (h *AdminOperator) InitRegister(engine *gin.Engine) {
	// RESTful 架构，用 URL 表示资源，用 HTTP 动词表示动作
	g := engine.Group("v1/admin/op", h.svcCtx.JwtMid.Handler, h.svcCtx.AdminMid.Handler)
	g.POST("/training/sync", h.SyncTraining)
}

func (h *AdminOperator) SyncTraining(ctx *gin.Context) {
	var req domain.AdminSyncTrainingReq
	if err := httpx.BindAndValidate(ctx, &req); err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	for _, stu := range req.Students {
		fmt.Println(stu)
		err := h.training.SyncRange(
			ctx.Request.Context(),
			stu.StudentID,
			stu.CFHandle,
			stu.ACHandle,
			req.From,
			req.To,
		)
		fmt.Println(err)
		if err != nil {
			httpx.FailWithErr(ctx, err)
			return
		}
	}

	httpx.Ok(ctx)
}
