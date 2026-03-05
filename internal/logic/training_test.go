package logic

import (
	"aATA/internal/crawler"
	"aATA/internal/model"
	"aATA/internal/svc"
	"context"
	"testing"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func initTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(mysql.Open("root:123456@tcp(127.0.0.1:3308)/aata_test?charset=utf8mb4&parseTime=True&loc=Local"), &gorm.Config{})
	if err != nil {
		t.Fatalf("db open failed: %v", err)
	}
	return db
}

var testExecuted = false // 用来标记测试是否已经执行

func TestTrainingLogic_RealSync(t *testing.T) {
	// 如果测试已经执行过，跳过
	if testExecuted {
		t.Skip("Test has already been executed once.")
	}

	// 标记测试已经执行
	testExecuted = true

	// 如果是短时间模式，跳过这个测试
	if testing.Short() {
		t.Skip("Skipping real integration test in short mode.")
	}

	// 初始化数据库
	db := initTestDB(t)

	contestModel := model.NewContestRecordModel(db)
	dailyModel := model.NewDailyTrainingStatsModel(db)

	// 初始化 crawler
	py := &crawler.PythonCrawler{
		PythonBin:  "/home/zvezdyuto/GolandProjects/agentAcmTrainingAnalysis/venv/bin/python",
		ScriptPath: "/home/zvezdyuto/GolandProjects/agentAcmTrainingAnalysis/internal/crawler/crawler_cli.py",
	}

	usersModel := model.NewUsersModel(db)
	svcCtx := &svc.ServiceContext{
		Crawler:      py,
		ContestModel: contestModel,
		DailyModel:   dailyModel,
		UsersModel:   usersModel,
	}

	logic := NewTrainingLogic(svcCtx, time.Local)

	// 进行实际的同步操作
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	from := time.Date(2024, 4, 15, 0, 0, 0, 0, time.Local)
	to := time.Date(2024, 5, 1, 23, 59, 59, 0, time.Local)

	err := logic.SyncRange(
		ctx,
		"230511213",
		"Utonut-Zvezdy",
		"",
		from,
		to,
	)

	if err != nil {
		t.Fatalf("SyncDay failed: %v", err)
	}

	t.Log("SyncDay success")
}
