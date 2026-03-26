package student_data

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
	SyncRange(ctx context.Context, studentID string, from, to time.Time) error
	SyncAllHistory(ctx context.Context, studentID string) error

	SyncAllUsers(ctx context.Context) error
	SyncStudent(ctx context.Context, studentID string) error
}
type trainingLogic struct {
	user      model.UsersModel
	contest   model.ContestRecordModel
	daily     model.DailyTrainingStatsModel
	syncState model.StudentSyncStateModel
	crawler   crawler.Crawler
	loc       *time.Location
}

func NewTrainingLogic(
	user model.UsersModel,
	contest model.ContestRecordModel,
	daily model.DailyTrainingStatsModel,
	syncState model.StudentSyncStateModel,
	crawler crawler.Crawler,
	loc *time.Location,
) TrainingLogic {
	if loc == nil {
		loc = time.Local
	}
	return &trainingLogic{
		user:      user,
		contest:   contest,
		daily:     daily,
		syncState: syncState,
		loc:       loc,
		crawler:   crawler,
	}
}

// SyncAllUsers 同步所有普通用户的数据（自动判断全量初始化或范围补齐）
func (l *trainingLogic) SyncAllUsers(ctx context.Context) error {
	users, _, err := l.user.List(ctx, &domain.UserListReq{})
	if err != nil {
		return err
	}

	for _, u := range users {
		if u.IsSystem == model.IsSystemUser {
			continue
		}
		if u.CFHandle == "" && u.ACHandle == "" {
			continue
		}

		// 每个学生按自己的 sync 状态决定如何同步
		if err := l.SyncStudent(ctx, u.Id); err != nil {
			// 单个失败不中断整体
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

// SyncRange 按时间区间进行爬取
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

// getStudentHandle 获取学生 handle
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

	if u.CFHandle == "" && u.ACHandle == "" {
		return "", "", fmt.Errorf("user %s handle not set", studentID)
	}

	return u.CFHandle, u.ACHandle, nil
}

// SyncStudent 根据 sync 状态自动决定是全量同步还是范围同步
func (l *trainingLogic) SyncStudent(ctx context.Context, studentID string) error {
	if l.syncState == nil {
		return fmt.Errorf("StudentSyncStateModel is not initialized")
	}

	yesterday := dateOnly(time.Now().In(l.loc).AddDate(0, 0, -1), l.loc)

	state, err := l.syncState.FindByStudentID(ctx, studentID)
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		return fmt.Errorf("find sync state failed: %w", err)
	}

	// 1. 未初始化：直接全量同步到昨天，成功后写 sync 表，完事退出
	if errors.Is(err, model.ErrNotFound) || state == nil || state.IsFullyInitialized == 0 {
		if err := l.SyncAllHistory(ctx, studentID); err != nil {
			return fmt.Errorf("sync all history failed: %w", err)
		}

		if err := l.syncState.Upsert(ctx, &model.StudentSyncState{
			StudentID:            studentID,
			IsFullyInitialized:   1,
			LatestSuccessfulDate: &yesterday,
		}); err != nil {
			return fmt.Errorf("upsert sync state after full sync failed: %w", err)
		}

		return nil
	}

	// 2. 已初始化：根据 latest_successful_date 计算缺口区间
	if state.LatestSuccessfulDate == nil {
		return fmt.Errorf("student %s sync state invalid: initialized but latest_successful_date is nil", studentID)
	}

	latest := *state.LatestSuccessfulDate
	from := dateOnly(latest.AddDate(0, 0, 1), l.loc)
	to := yesterday

	// 没有缺口，不需要同步
	if from.After(to) {
		return nil
	}

	if err := l.SyncRange(ctx, studentID, from, to); err != nil {
		return fmt.Errorf("sync range failed: %w", err)
	}

	// 3. 范围同步成功后，再推进 sync 表
	if err := l.syncState.Upsert(ctx, &model.StudentSyncState{
		StudentID:            studentID,
		IsFullyInitialized:   1,
		LatestSuccessfulDate: &to,
	}); err != nil {
		return fmt.Errorf("upsert sync state after range sync failed: %w", err)
	}

	return nil
}

func dateOnly(t time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.Local
	}
	tt := t.In(loc)
	return time.Date(tt.Year(), tt.Month(), tt.Day(), 0, 0, 0, 0, loc)
}
