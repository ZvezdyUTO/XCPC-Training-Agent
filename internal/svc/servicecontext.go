package svc

import (
	"aATA/internal/config"
	"aATA/internal/crawler"
	agentllm "aATA/internal/logic/agent/llm"
	"aATA/internal/logic/agent/tooling"
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

// Models 收拢所有数据库模型依赖。
// 采用嵌入方式是为了在 ServiceContext 上继续保持短字段访问。
type Models struct {
	UsersModel            model.UsersModel
	ContestModel          model.ContestRecordModel
	DailyModel            model.DailyTrainingStatsModel
	StudentSyncStateModel model.StudentSyncStateModel
}

// Infra 收拢与业务无关的基础设施依赖。
type Infra struct {
	JWT     *jwt.JWT
	Crawler crawler.Crawler
}

// MiddlewareSet 收拢 HTTP 中间件依赖。
type MiddlewareSet struct {
	JwtMid     *middleware.JWTMid
	AdminMid   *middleware.AdminMid
	LoggingMid *middleware.LoggingMid
}

// AgentDeps 收拢 Agent 模块运行所需的依赖。
type AgentDeps struct {
	LLMClient agentllm.Client

	TrainingSummaryTool          tooling.Tool
	ContestRatingSummaryTool     tooling.Tool
	TrainingDayLeaderboardTool   tooling.Tool
	TrainingWeekLeaderboardTool  tooling.Tool
	TrainingMonthLeaderboardTool tooling.Tool
	ContestRankingTool           tooling.Tool
}

// ServiceContext 是应用层依赖的轻量装配入口。
// 这里按领域分组依赖，但通过嵌入保持原有访问方式不变。
type ServiceContext struct {
	Config config.Config
	ctx    context.Context

	Models
	Infra
	MiddlewareSet
	AgentDeps
}

func NewServiceContext(ctx context.Context, c config.Config) (*ServiceContext, error) {
	db, err := gorm.Open(mysql.Open(c.MySql.DataSource), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	models := newModels(db)
	infra := newInfra(c)
	middlewareSet := newMiddlewareSet(infra.JWT)
	agentDeps := newAgentDeps(models)

	res := &ServiceContext{
		ctx:           ctx,
		Config:        c,
		Models:        models,
		Infra:         infra,
		MiddlewareSet: middlewareSet,
		AgentDeps:     agentDeps,
	}

	return res, initServer(res)
}

func newModels(db *gorm.DB) Models {
	return Models{
		UsersModel:            model.NewUsersModel(db),
		ContestModel:          model.NewContestRecordModel(db),
		DailyModel:            model.NewDailyTrainingStatsModel(db),
		StudentSyncStateModel: model.NewStudentSyncStateModel(db),
	}
}

func newInfra(c config.Config) Infra {
	return Infra{
		JWT: jwt.NewJWT(c.JWT.Secret, c.JWT.Expire),
		Crawler: &crawler.PythonCrawler{
			ScriptPath: "./internal/crawler/crawler_cli.py",
			PythonBin:  "python3",
		},
	}
}

func newMiddlewareSet(jwtTool *jwt.JWT) MiddlewareSet {
	return MiddlewareSet{
		JwtMid:     middleware.NewJWTMid(jwtTool),
		AdminMid:   middleware.NewAdminMid(),
		LoggingMid: middleware.NewLoggingMid(),
	}
}

func newAgentDeps(models Models) AgentDeps {
	modelName := os.Getenv("LLM_MODEL")
	if modelName == "" {
		modelName = "deepseek-chat"
	}

	return AgentDeps{
		LLMClient:                    agentllm.NewOpenAICompatibleClient(modelName),
		TrainingSummaryTool:          tools.NewTrainingSummaryTool(models.DailyModel),
		ContestRatingSummaryTool:     tools.NewContestRatingSummaryTool(models.ContestModel),
		TrainingDayLeaderboardTool:   tools.NewTrainingDayLeaderboardTool(models.DailyModel, models.UsersModel),
		TrainingWeekLeaderboardTool:  tools.NewTrainingWeekLeaderboardTool(models.DailyModel, models.UsersModel),
		TrainingMonthLeaderboardTool: tools.NewTrainingMonthLeaderboardTool(models.DailyModel, models.UsersModel),
		ContestRankingTool:           tools.NewContestRankingTool(models.ContestModel, models.UsersModel),
	}
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
