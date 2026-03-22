package api

import (
	"aATA/internal/logic/student_data"
	"fmt"

	"github.com/gin-gonic/gin"

	"aATA/internal/domain"
	"aATA/internal/svc"
	"aATA/pkg/httpx"
)

type AdminOperator struct {
	svcCtx   *svc.ServiceContext
	training student_data.TrainingLogic
}

func NewAdminOperator(svcCtx *svc.ServiceContext, training student_data.TrainingLogic) *AdminOperator {
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

	type FailedItem struct {
		StudentID string `json:"student_id"`
		Error     string `json:"error"`
	}

	failed := make([]FailedItem, 0)

	for _, stu := range req.Students {

		fmt.Println(stu.StudentID)
		err := h.training.SyncRange(
			ctx.Request.Context(),
			stu.StudentID,
			req.From,
			req.To,
		)

		if err != nil {
			failed = append(failed, FailedItem{
				StudentID: stu.StudentID,
				Error:     err.Error(),
			})
			continue
		}
	}

	if len(failed) == 0 {
		httpx.Ok(ctx)
		return
	}

	httpx.OkWithData(ctx, gin.H{
		"msg":        "partial success",
		"failed_cnt": len(failed),
		"failed":     failed,
	})
}
