package main

import (
	"aATA/internal/config"
	"aATA/internal/domain"
	"aATA/internal/logic/student_data"
	"aATA/internal/model"
	"aATA/internal/svc"
	"aATA/pkg/conf"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

var (
	configFile = flag.String("f", "./etc/local/api.yaml", "config file")
)

func main() {
	flag.Parse()

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var cfg config.Config
	conf.MustLoad(*configFile, &cfg)

	svcCtx, err := svc.NewServiceContext(rootCtx, cfg)
	if err != nil {
		panic(err)
	}

	training := student_data.NewTrainingLogic(
		svcCtx.UsersModel,
		svcCtx.ContestModel,
		svcCtx.DailyModel,
		svcCtx.Crawler,
		nil,
	)

	users, total, err := svcCtx.UsersModel.List(rootCtx, &domain.UserListReq{})
	if err != nil {
		panic(fmt.Errorf("list users failed: %w", err))
	}

	fmt.Printf("开始初始化全量历史训练数据，总用户数: %d\n", total)

	successCnt := 0
	skipCnt := 0
	failCnt := 0
	failedUsers := make([]string, 0)

	for _, u := range users {
		if u == nil {
			continue
		}

		if u.IsSystem == model.IsSystemUser {
			fmt.Printf("[SKIP] %s %s: 系统用户\n", u.Id, u.Name)
			skipCnt++
			continue
		}

		if u.CFHandle == "" {
			fmt.Printf("[SKIP] %s %s: 未设置 CF handle\n", u.Id, u.Name)
			skipCnt++
			continue
		}

		fmt.Printf("[START] %s %s cf=%s\n", u.Id, u.Name, u.CFHandle)

		if err := training.SyncAllHistory(rootCtx, u.Id); err != nil {
			fmt.Printf("[FAIL]  %s %s: %v\n", u.Id, u.Name, err)
			failCnt++
			failedUsers = append(failedUsers, fmt.Sprintf("%s(%s)", u.Id, u.Name))
			continue
		}

		fmt.Printf("[OK]    %s %s\n", u.Id, u.Name)
		successCnt++
	}

	fmt.Println("===================================")
	fmt.Printf("初始化完成: success=%d skip=%d fail=%d\n", successCnt, skipCnt, failCnt)
	if len(failedUsers) > 0 {
		fmt.Println("失败用户列表:")
		for _, s := range failedUsers {
			fmt.Println("-", s)
		}
	}
}
