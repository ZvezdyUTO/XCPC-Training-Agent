package svc

import (
	"aATA/internal/config"
	"aATA/internal/crawler"
	"aATA/internal/middleware"
	"aATA/internal/model"
	"aATA/pkg/encrypt"
	"aATA/pkg/jwt"
	"context"
	"errors"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type ServiceContext struct {
	Config config.Config
	ctx    context.Context

	// 基础设施
	JWT          *jwt.JWT
	UsersModel   model.UsersModel
	ContestModel model.ContestRecordModel
	DailyModel   model.DailyTrainingStatsModel
	Crawler      crawler.Crawler

	// Middleware
	JwtMid     *middleware.JWTMid
	AdminMid   *middleware.AdminMid
	LoggingMid *middleware.LoggingMid
}

func NewServiceContext(ctx context.Context, c config.Config) (*ServiceContext, error) {
	db, err := gorm.Open(mysql.Open(c.MySql.DataSource), &gorm.Config{}) // 这就是进行组装了
	if err != nil {
		return nil, err
	}

	jwtTool := jwt.NewJWT(
		c.JWT.Secret,
		c.JWT.Expire,
	)

	craw := &crawler.PythonCrawler{
		ScriptPath: "./internal/crawler/crawler_cli.py",
		PythonBin:  "python3",
	}

	res := &ServiceContext{
		ctx:          ctx,
		Config:       c,
		UsersModel:   model.NewUsersModel(db),
		ContestModel: model.NewContestRecordModel(db),
		DailyModel:   model.NewDailyTrainingStatsModel(db),
		JWT:          jwtTool,
		JwtMid:       middleware.NewJWTMid(jwtTool),
		LoggingMid:   middleware.NewLoggingMid(),
		AdminMid:     middleware.NewAdminMid(),
		Crawler:      craw,
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
