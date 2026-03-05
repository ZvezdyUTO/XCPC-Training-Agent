package logic

import (
	"aATA/internal/model"
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"aATA/internal/svc"

	"gorm.io/gorm"
)

type TrainingLogic interface {
	SyncDay(ctx context.Context, studentID, cfHandle, acHandle string, day time.Time) error
	SyncAllHistory(ctx context.Context, studentID, cfHandle, acHandle string) error
	SyncRange(ctx context.Context, studentID, cfHandle, acHandle string, from, to time.Time) error
}
type trainingLogic struct {
	svc *svc.ServiceContext
	loc *time.Location
}

func NewTrainingLogic(svcCtx *svc.ServiceContext, loc *time.Location) TrainingLogic {
	if loc == nil {
		loc = time.Local
	}
	return &trainingLogic{svc: svcCtx, loc: loc}
}

// SyncDay：同步某一天（日常任务）
func (l *trainingLogic) SyncDay(
	ctx context.Context,
	studentID, cfHandle, acHandle string,
	day time.Time,
) error {
	from, to := dayRange(day, l.loc)
	return l.SyncRange(ctx, studentID, cfHandle, acHandle, from, to)
}

// SyncAllHistory：全量历史（初始化/重建）
func (l *trainingLogic) SyncAllHistory(
	ctx context.Context,
	studentID, cfHandle, acHandle string,
) error {
	from, to := allHistoryRange(time.Now(), l.loc)
	return l.SyncRange(ctx, studentID, cfHandle, acHandle, from, to)
}

func (l *trainingLogic) SyncRange(
	ctx context.Context,
	studentID, cfHandle, acHandle string,
	from, to time.Time,
) error {

	//检查用户是否存在，如果不存在插入
	if err := l.ensureUserExists(ctx, studentID); err != nil {
		return fmt.Errorf("check or insert user failed: %w", err)
	}
	fmt.Println("检查用户逻辑成功，已执行")

	// 1) 调用爬虫（子进程 python）
	resp, err := l.svc.Crawler.FetchRange(ctx, studentID, cfHandle, acHandle, from, to)
	if err != nil {
		fmt.Println("爬虫调用失败")
		fmt.Println(err)
		return err
	}
	fmt.Println("爬虫调用成功")

	// 2) 覆盖式同步：先删后写
	// 比增量同步简单得多，且在“初始化/修复数据”场景可靠
	if err := l.svc.ContestModel.DeleteRange(ctx, []string{studentID}, from, to); err != nil {
		return fmt.Errorf("delete contest range: %w", err)
	}
	if err := l.svc.DailyModel.DeleteRange(ctx, []string{studentID}, from, to); err != nil {
		return fmt.Errorf("delete daily range: %w", err)
	}

	// 3) 写 contest（逐条 insert）
	for i := range resp.ContestRecords {
		cr := resp.ContestRecords[i]
		// 保险：强制 studentID（避免脚本输出不一致）
		cr.StudentID = studentID

		// 这里的类型是 domain.ContestRecord，但你的 model.Insert 参数是 *model.ContestRecord
		// 需要：model 层的结构体 ≈ domain 层结构体（字段一致）
		if err := l.svc.ContestModel.Upsert(ctx, model.ToModelContest(&cr)); err != nil {
			return fmt.Errorf("insert contest: %w", err)
		}
	}

	// 4) 写 daily（逐条 upsert 更稳）
	for i := range resp.DailyStats {
		ds := resp.DailyStats[i]
		ds.StudentID = studentID

		if err := l.svc.DailyModel.Upsert(ctx, model.ToModelDaily(&ds)); err != nil {
			return fmt.Errorf("upsert daily: %w", err)
		}
	}

	return nil
}

func (l *trainingLogic) ensureUserExists(ctx context.Context, userID string) error {
	// 查询用户是否存在
	if l.svc.UsersModel == nil {
		return fmt.Errorf("UsersModel is not initialized")
	}

	_, err := l.svc.UsersModel.FindByID(userID)
	if err != nil {
		// 如果用户不存在（或者其他错误），返回错误
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 插入用户
			newUser := &model.Users{
				Id:       userID,
				Name:     userID,             // 可以根据需求设置其他字段
				Password: "default_password", // 默认密码，实际应用时需要改成更合适的逻辑
				Status:   model.UserStatusNormal,
				IsSystem: 0,
			}
			// 调用 model 插入用户
			if err := l.svc.UsersModel.Insert(ctx, newUser); err != nil {
				return fmt.Errorf("failed to insert user: %w", err)
			}
			log.Printf("Inserted new user: %s", userID)
		} else {
			// 其他查询错误
			return fmt.Errorf("failed to find user: %w", err)
		}
	}
	return nil
}
