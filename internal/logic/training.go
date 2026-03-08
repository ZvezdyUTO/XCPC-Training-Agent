package logic

import (
	"aATA/internal/crawler"
	"aATA/internal/domain"
	"aATA/internal/model"
	"context"
	"errors"
	"fmt"
	"time"
)

type TrainingLogic interface {
	SyncDay(ctx context.Context, studentID string, day time.Time) error
	SyncAllUsersYesterday(ctx context.Context) error
	SyncAllHistory(ctx context.Context, studentID string) error
	SyncRange(ctx context.Context, studentID string, from, to time.Time) error
}
type trainingLogic struct {
	user    model.UsersModel
	contest model.ContestRecordModel
	daily   model.DailyTrainingStatsModel
	crawler crawler.Crawler
	loc     *time.Location
}

func NewTrainingLogic(
	user model.UsersModel,
	contest model.ContestRecordModel,
	daily model.DailyTrainingStatsModel,
	crawler crawler.Crawler,
	loc *time.Location,
) TrainingLogic {
	if loc == nil {
		loc = time.Local
	}
	return &trainingLogic{
		user:    user,
		contest: contest,
		daily:   daily,
		loc:     loc,
		crawler: crawler,
	}
}

// SyncDay 同步某一天（日常任务）
func (l *trainingLogic) SyncDay(
	ctx context.Context,
	studentID string,
	day time.Time,
) error {
	from, to := dayRange(day, l.loc)
	return l.SyncRange(ctx, studentID, from, to)
}

// SyncAllUsersYesterday 同步所有用户昨日的训练记录（管理员除外）
func (l *trainingLogic) SyncAllUsersYesterday(ctx context.Context) error {
	users, _, err := l.user.List(ctx, &domain.UserListReq{})
	if err != nil {
		return err
	}

	yesterday := time.Now().In(l.loc).AddDate(0, 0, -1)

	for _, u := range users {
		if u.IsSystem == model.IsSystemUser {
			continue
		}
		if u.CFHandle == "" || u.ACHandle == "" {
			continue
		}

		// 执行爬虫逻辑
		if err := l.SyncDay(ctx, u.Id, yesterday); err != nil {
			// 不中断，继续
			continue
		}
	}
	return nil
}

// SyncAllHistory 全量历史（初始化/重建）
func (l *trainingLogic) SyncAllHistory(
	ctx context.Context,
	studentID string,
) error {
	from, to := allHistoryRange(time.Now(), l.loc)
	return l.SyncRange(ctx, studentID, from, to)
}

func (l *trainingLogic) SyncRange(
	ctx context.Context,
	studentID string,
	from, to time.Time,
) error {

	//检查用户是否存在，如果不存在插入
	cfHandle, acHandle, err := l.getStudentHandle(ctx, studentID)
	if err != nil {
		return fmt.Errorf("check or insert user failed: %w", err)
	}

	// 1) 调用爬虫（子进程 python）
	resp, err := l.crawler.FetchRange(ctx, studentID, cfHandle, acHandle, from, to)
	if err != nil {
		fmt.Println("爬虫调用失败")
		fmt.Println(err)
		return err
	}
	fmt.Println("爬虫调用成功")

	// 2) 覆盖式同步：先删后写
	// 比增量同步简单得多，且在“初始化/修复数据”场景可靠
	if err := l.contest.DeleteRange(ctx, []string{studentID}, from, to); err != nil {
		return fmt.Errorf("delete contest range: %w", err)
	}
	if err := l.daily.DeleteRange(ctx, []string{studentID}, from, to); err != nil {
		return fmt.Errorf("delete daily range: %w", err)
	}

	// 3) 写 contest（逐条 insert）
	for i := range resp.ContestRecords {
		cr := resp.ContestRecords[i]
		cr.StudentID = studentID

		// 需要：model 层的结构体 ≈ domain 层结构体（字段一致）
		if err := l.contest.Upsert(ctx, model.ToModelContest(&cr)); err != nil {
			return fmt.Errorf("insert contest: %w", err)
		}
	}

	// 4) 写 daily（逐条 upsert）
	for i := range resp.DailyStats {
		ds := resp.DailyStats[i]
		ds.StudentID = studentID

		if err := l.daily.Upsert(ctx, model.ToModelDaily(&ds)); err != nil {
			return fmt.Errorf("upsert daily: %w", err)
		}
	}

	return nil
}

func (l *trainingLogic) getStudentHandle(ctx context.Context, studentID string) (string, string, error) {

	if l.user == nil {
		return "", "", fmt.Errorf("UsersModel is not initialized")
	}

	u, err := l.user.FindByID(studentID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return "", "", fmt.Errorf("user %s not found", studentID)
		}
		return "", "", fmt.Errorf("failed to find user: %w", err)
	}

	if u == nil {
		return "", "", fmt.Errorf("user %s not found", studentID)
	}

	if u.CFHandle == "" || u.ACHandle == "" {
		return "", "", fmt.Errorf("user %s handle not set", studentID)
	}

	return u.CFHandle, u.ACHandle, nil
}
