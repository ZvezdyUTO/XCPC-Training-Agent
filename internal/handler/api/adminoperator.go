package api

import (
	"aATA/internal/domain"
	applogic "aATA/internal/logic"
	anomalylogic "aATA/internal/logic/anomaly"
	"aATA/internal/logic/student_data"
	"aATA/internal/model"
	"errors"
	"io"
	"time"

	"github.com/gin-gonic/gin"

	"aATA/internal/svc"
	"aATA/pkg/httpx"
)

type AdminOperator struct {
	svcCtx      *svc.ServiceContext
	training    student_data.TrainingLogic
	leaderboard applogic.TrainingLeaderboard
	anomaly     anomalylogic.Service
}

func NewAdminOperator(
	svcCtx *svc.ServiceContext,
	training student_data.TrainingLogic,
	leaderboard applogic.TrainingLeaderboard,
	anomaly anomalylogic.Service,
) *AdminOperator {
	return &AdminOperator{
		svcCtx:      svcCtx,
		training:    training,
		leaderboard: leaderboard,
		anomaly:     anomaly,
	}
}

func (h *AdminOperator) InitRegister(engine *gin.Engine) {
	// RESTful 架构，用 URL 表示资源，用 HTTP 动词表示动作
	g := engine.Group("v1/admin/op", h.svcCtx.JwtMid.Handler, h.svcCtx.AdminMid.Handler)
	g.POST("/training/syncall", h.SyncAllTraining)
	g.POST("/training/syncone", h.SyncOneTraining)
	g.POST("/training/detect/run", h.RunTrainingDetect)
	g.GET("/training/syncstate/list", h.ListTrainingSyncState)
	g.GET("/training/summary", h.GetTrainingSummaryRange)
	g.GET("/training/leaderboard", h.GetTrainingLeaderboard)
	g.GET("/contest/ranking", h.GetContestRanking)
}

// SyncAllTraining 检查所有学生的 sync 状态，并自动决定全量或范围更新
func (h *AdminOperator) SyncAllTraining(ctx *gin.Context) {
	var req struct {
		DetectAfterSync bool `json:"detect_after_sync"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		httpx.FailWithErr(ctx, err)
		return
	}

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

	detectCnt := 0
	if req.DetectAfterSync && h.anomaly != nil {
		detectCnt, err = h.anomaly.DetectAllUsers(ctx.Request.Context(), time.Now())
		if err != nil {
			httpx.FailWithErr(ctx, err)
			return
		}
	}

	if len(failed) == 0 {
		httpx.OkWithData(ctx, gin.H{
			"msg":         "success",
			"success_cnt": len(success),
			"success":     success,
			"alert_cnt":   detectCnt,
		})
		return
	}

	httpx.OkWithData(ctx, gin.H{
		"msg":         "partial success",
		"success_cnt": len(success),
		"success":     success,
		"failed_cnt":  len(failed),
		"failed":      failed,
		"alert_cnt":   detectCnt,
	})
}

// SyncOneTraining 触发单个学生的训练同步，并返回本次实际采用的同步模式。
// 该接口复用现有训练同步逻辑，不额外增加批量或重试语义。
func (h *AdminOperator) SyncOneTraining(ctx *gin.Context) {
	var req domain.AdminSyncOneTrainingReq
	if err := httpx.BindAndValidate(ctx, &req); err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	user, err := h.svcCtx.UsersModel.FindByID(req.StudentID)
	if err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}
	if user.CFHandle == "" && user.ACHandle == "" {
		httpx.FailWithErr(ctx, errors.New("student has no cf/ac handle"))
		return
	}

	mode, err := h.training.SyncStudentWithMode(ctx.Request.Context(), req.StudentID)
	if err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	detectCnt := 0
	if req.DetectAfterSync && h.anomaly != nil {
		detectCnt, err = h.anomaly.DetectAllUsers(ctx.Request.Context(), time.Now())
		if err != nil {
			httpx.FailWithErr(ctx, err)
			return
		}
	}

	httpx.OkWithData(ctx, gin.H{
		"msg":         "success",
		"student_id":  req.StudentID,
		"mode":        string(mode),
		"alert_cnt":   detectCnt,
	})
}

// RunTrainingDetect 手动触发一次训练异常检测（无需先执行同步）。
func (h *AdminOperator) RunTrainingDetect(ctx *gin.Context) {
	if h.anomaly == nil {
		httpx.FailWithErr(ctx, errors.New("异常检测服务未初始化"))
		return
	}

	cnt, err := h.anomaly.DetectAllUsers(ctx.Request.Context(), time.Now())
	if err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	httpx.OkWithData(ctx, gin.H{
		"msg":        "success",
		"alert_cnt":  cnt,
		"detected_at": time.Now().Format("2006-01-02 15:04:05"),
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

// GetTrainingSummaryRange 直接查询某个学生在指定时间段内的训练累计数据。
// 该接口复用训练统计表，不调用模型，也不触发补抓取。
func (h *AdminOperator) GetTrainingSummaryRange(ctx *gin.Context) {
	var req struct {
		StudentID string `form:"student_id" binding:"required"`
		From      string `form:"from" binding:"required"`
		To        string `form:"to" binding:"required"`
	}
	if err := httpx.BindAndValidate(ctx, &req); err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	fromTime, err := time.Parse("2006-01-02", req.From)
	if err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}
	toTime, err := time.Parse("2006-01-02", req.To)
	if err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	res, err := h.svcCtx.DailyModel.SumRange(ctx.Request.Context(), req.StudentID, fromTime, toTime)
	if err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}
	records, err := h.svcCtx.ContestModel.FindByStudent(ctx.Request.Context(), req.StudentID)
	if err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	dist, acDist := applogic.BuildTrainingDistributions(res)
	trainingValue := applogic.BuildTrainingValueSummary(res, records, fromTime, toTime)

	httpx.OkWithData(ctx, gin.H{
		"student_id":      req.StudentID,
		"from":            req.From,
		"to":              req.To,
		"cf_total":        res.CFNewTotal,
		"cf_distribution": dist,
		"ac_total":        res.ACNewTotal,
		"ac_distribution": acDist,
		"training_value":  trainingValue,
	})
}

// GetTrainingLeaderboard 返回指定时间区间内的训练价值排行榜。
// 该接口只读取已落库数据，不会触发补抓取或二次推断。
func (h *AdminOperator) GetTrainingLeaderboard(ctx *gin.Context) {
	var req domain.TrainingLeaderboardReq
	if err := httpx.BindAndValidate(ctx, &req); err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	resp, err := h.leaderboard.Query(ctx.Request.Context(), &req)
	if err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	httpx.OkWithData(ctx, resp)
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
