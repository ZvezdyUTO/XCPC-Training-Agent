package svc

import (
	"aATA/internal/config"
	"aATA/internal/crawler"
	"aATA/internal/llm"
	"aATA/internal/logic/agent"
	"aATA/internal/logic/agent/tools"
	"aATA/internal/middleware"
	"aATA/internal/model"
	"aATA/pkg/encrypt"
	"aATA/pkg/jwt"
	"context"
	"errors"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type ServiceContext struct {
	Config config.Config
	ctx    context.Context

	// 基础设施
	JWT                      *jwt.JWT
	UsersModel               model.UsersModel
	ContestModel             model.ContestRecordModel
	DailyModel               model.DailyTrainingStatsModel
	StudentSyncStateModel    model.StudentSyncStateModel
	Crawler                  crawler.Crawler
	LLMClient                llm.Client
	GetUserTrainingRangeTool agent.Tool

	// Middleware
	JwtMid     *middleware.JWTMid
	AdminMid   *middleware.AdminMid
	LoggingMid *middleware.LoggingMid

	// AgentTools
	TrainingSummaryTool          agent.Tool
	ContestRatingSummaryTool     agent.Tool
	TrainingDayLeaderboardTool   agent.Tool
	TrainingWeekLeaderboardTool  agent.Tool
	TrainingMonthLeaderboardTool agent.Tool
	ContestRankingTool           agent.Tool
}

func NewServiceContext(ctx context.Context, c config.Config) (*ServiceContext, error) {
	db, err := gorm.Open(mysql.Open(c.MySql.DataSource), &gorm.Config{}) // 这就是进行组装了
	if err != nil {
		return nil, err
	}

	// 统一拼装 model
	dailyModel := model.NewDailyTrainingStatsModel(db)
	userModel := model.NewUsersModel(db)
	contestModel := model.NewContestRecordModel(db)
	studentSyncStateModel := model.NewStudentSyncStateModel(db)

	jwtTool := jwt.NewJWT(
		c.JWT.Secret,
		c.JWT.Expire,
	)

	craw := &crawler.PythonCrawler{
		ScriptPath: "./internal/crawler/crawler_cli.py",
		PythonBin:  "python3",
	}

	// 拼装 agent 工具
	modelName := os.Getenv("LLM_MODEL")
	if modelName == "" {
		modelName = "deepseek-chat" // 默认值
	}

	llmClient := llm.NewAliyunQwenClient(modelName)
	TrainingSummaryTool := tools.NewTrainingSummaryTool(dailyModel)
	ContestRatingSummaryTool := tools.NewContestRatingSummaryTool(contestModel)
	TrainingDayLeaderboardTool := tools.NewTrainingDayLeaderboardTool(dailyModel, userModel)
	TrainingWeekLeaderboardTool := tools.NewTrainingWeekLeaderboardTool(dailyModel, userModel)
	TrainingMonthLeaderboardTool := tools.NewTrainingMonthLeaderboardTool(dailyModel, userModel)
	ContestRankingTool := tools.NewContestRankingTool(contestModel, userModel)

	res := &ServiceContext{
		ctx:                   ctx,
		Config:                c,
		UsersModel:            userModel,
		ContestModel:          contestModel,
		DailyModel:            dailyModel,
		StudentSyncStateModel: studentSyncStateModel,

		JWT:        jwtTool,
		JwtMid:     middleware.NewJWTMid(jwtTool),
		LoggingMid: middleware.NewLoggingMid(),
		AdminMid:   middleware.NewAdminMid(),

		Crawler:                      craw,
		LLMClient:                    llmClient,
		TrainingSummaryTool:          TrainingSummaryTool,
		ContestRatingSummaryTool:     ContestRatingSummaryTool,
		TrainingDayLeaderboardTool:   TrainingDayLeaderboardTool,
		TrainingWeekLeaderboardTool:  TrainingWeekLeaderboardTool,
		TrainingMonthLeaderboardTool: TrainingMonthLeaderboardTool,
		ContestRankingTool:           ContestRankingTool,
	}

	return res, initServer(res)
}

func initServer(svc *ServiceContext) error {
	if err := initSystemUser(svc.ctx, svc); err != nil {
		return err
	}
	return nil
}

func initSystemUser(ctx context.Context, svc *ServiceContext) error {
	systemUser, err := svc.UsersModel.SystemUser()

	if err != nil && !errors.Is(err, model.ErrNotFound) {
		return err
	}

	if systemUser != nil {
		return nil
	}

	// 防止重复 root
	u, err := svc.UsersModel.FindByID("20001")
	if err == nil && u != nil {
		return nil
	}

	pwd, err := encrypt.GenPasswordHash([]byte("000000"))
	if err != nil {
		return err
	}

	return svc.UsersModel.Insert(ctx, &model.Users{
		Id:       "20001",
		Name:     "root",
		Password: string(pwd),
		Status:   model.UserStatusNormal,
		IsSystem: model.IsSystemUser,
	})
}
