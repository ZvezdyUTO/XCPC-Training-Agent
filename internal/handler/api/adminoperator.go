package api

import (
	"aATA/internal/domain"
	"aATA/internal/logic/student_data"
	"aATA/internal/model"

	"github.com/gin-gonic/gin"

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
	g.POST("/training/syncall", h.SyncAllTraining)
}

// SyncAllTraining 检查所有学生的 sync 状态，并自动决定全量或范围更新
func (h *AdminOperator) SyncAllTraining(ctx *gin.Context) {
	users, _, err := h.svcCtx.UsersModel.List(ctx.Request.Context(), &domain.UserListReq{})
	if err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	type SuccessItem struct {
		StudentID string `json:"student_id"`
		Mode      string `json:"mode"` // full / range / skip
	}

	type FailedItem struct {
		StudentID string `json:"student_id"`
		Error     string `json:"error"`
	}

	success := make([]SuccessItem, 0)
	failed := make([]FailedItem, 0)

	for _, u := range users {
		if u.IsSystem == model.IsSystemUser {
			continue
		}
		if u.CFHandle == "" && u.ACHandle == "" {
			continue
		}

		mode, err := h.training.SyncStudentWithMode(ctx.Request.Context(), u.Id)
		if err != nil {
			failed = append(failed, FailedItem{
				StudentID: u.Id,
				Error:     err.Error(),
			})
			continue
		}

		success = append(success, SuccessItem{
			StudentID: u.Id,
			Mode:      string(mode),
		})
	}

	if len(failed) == 0 {
		httpx.OkWithData(ctx, gin.H{
			"msg":         "success",
			"success_cnt": len(success),
			"success":     success,
		})
		return
	}

	httpx.OkWithData(ctx, gin.H{
		"msg":         "partial success",
		"success_cnt": len(success),
		"success":     success,
		"failed_cnt":  len(failed),
		"failed":      failed,
	})
}
