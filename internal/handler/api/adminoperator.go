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
	g.GET("/training/syncstate/list", h.ListTrainingSyncState)
	g.GET("/contest/ranking", h.GetContestRanking)
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

// ListTrainingSyncState 返回同步状态表中的全部记录。
// 这里只暴露当前已落库的状态快照，便于前端查看初始化情况与最近成功日期。
func (h *AdminOperator) ListTrainingSyncState(ctx *gin.Context) {
	list, err := h.svcCtx.StudentSyncStateModel.List(ctx.Request.Context())
	if err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	httpx.OkWithData(ctx, gin.H{
		"count": len(list),
		"list":  list,
	})
}

// GetContestRanking 直接查询数据库中某场比赛的队内排名。
// 该接口只做数据读取，不调用模型，也不触发任何补抓取逻辑。
func (h *AdminOperator) GetContestRanking(ctx *gin.Context) {
	var req struct {
		Platform  string `form:"platform" binding:"required,oneof=CF AC"`
		ContestID string `form:"contest_id" binding:"required"`
	}
	if err := httpx.BindAndValidate(ctx, &req); err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	list, err := h.svcCtx.ContestModel.FindByContest(ctx.Request.Context(), req.Platform, req.ContestID)
	if err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	res := domain.ContestRankingResult{
		Platform:  req.Platform,
		ContestID: req.ContestID,
		Count:     len(list),
		Items:     make([]domain.ContestRankingItem, 0, len(list)),
	}

	if len(list) > 0 {
		res.ContestName = list[0].ContestName
		if !list[0].ContestDate.IsZero() {
			res.ContestDate = list[0].ContestDate.Format("2006-01-02 15:04:05")
		}
	}

	studentIDs := make([]string, 0, len(list))
	for _, record := range list {
		studentIDs = append(studentIDs, record.StudentID)
	}

	users, _, err := h.svcCtx.UsersModel.List(ctx.Request.Context(), &domain.UserListReq{Ids: studentIDs})
	if err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	nameMap := make(map[string]string, len(users))
	for _, user := range users {
		nameMap[user.Id] = user.Name
	}

	for _, record := range list {
		res.Items = append(res.Items, domain.ContestRankingItem{
			StudentID:    record.StudentID,
			StudentName:  nameMap[record.StudentID],
			Platform:     record.Platform,
			ContestID:    record.ContestID,
			Name:         record.ContestName,
			Date:         record.ContestDate,
			Rank:         record.ContestRank,
			OldRating:    record.OldRating,
			NewRating:    record.NewRating,
			RatingChange: record.RatingChange,
		})
	}

	httpx.OkWithData(ctx, res)
}
