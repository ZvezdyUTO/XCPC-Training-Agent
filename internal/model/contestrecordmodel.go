package model

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type (
	ContestRecordModel interface {
		// Insert 插入一场比赛记录。
		Insert(ctx context.Context, data *ContestRecord) error
		// Upsert 如果有记录则更新否则插入比赛记录
		Upsert(ctx context.Context, data *ContestRecord) error
		// FindByStudent 查询某用户的全部比赛历史。
		FindByStudent(ctx context.Context, studentID string) ([]*ContestRecord, error)
		// FindByStudent 查询某比赛的用户排名
		FindByContest(ctx context.Context, platform, contestID string) ([]*ContestRecord, error)
		// FindRecent 查询某用户最近 N 天的比赛记录。
		FindRecent(ctx context.Context, studentID string, days int) ([]*ContestRecord, error)
		// Delete 删除某场比赛记录。
		Delete(ctx context.Context, studentID, platform, contestID string) error
		// DeleteRange 批量删除比赛记录
		DeleteRange(ctx context.Context, studentIDs []string, from, to time.Time) error
	}

	defaultContestRecord struct {
		db *gorm.DB
	}
)

func NewContestRecordModel(db *gorm.DB) ContestRecordModel {
	return &defaultContestRecord{db: db}
}

func (m *defaultContestRecord) model() *gorm.DB {
	return m.db.Model(&ContestRecord{})
}

func (m *defaultContestRecord) Insert(ctx context.Context, data *ContestRecord) error {
	return m.model().Create(data).Error
}

func (m *defaultContestRecord) FindByStudent(
	ctx context.Context,
	studentID string,
) ([]*ContestRecord, error) {

	var list []*ContestRecord
	err := m.model().
		WithContext(ctx).
		Where("student_id = ?", studentID).
		Order("contest_date ASC").
		Find(&list).Error

	return list, err
}

func (m *defaultContestRecord) Upsert(
	ctx context.Context,
	data *ContestRecord,
) error {

	return m.model().
		WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "student_id"},
				{Name: "platform"},
				{Name: "contest_id"},
			},
			DoUpdates: clause.Assignments(map[string]any{
				"contest_name":  data.ContestName,
				"contest_date":  data.ContestDate,
				"contest_rank":  data.ContestRank,
				"old_rating":    data.OldRating,
				"new_rating":    data.NewRating,
				"rating_change": data.RatingChange,
				"performance":   data.Performance,
				"created_at":    data.CreatedAt,
				"deleted_at":    nil,
			}),
		}).
		Create(data).Error
}

func (m *defaultContestRecord) FindRecent(
	ctx context.Context,
	studentID string,
	days int,
) ([]*ContestRecord, error) {

	var list []*ContestRecord
	cutoff := time.Now().AddDate(0, 0, -days)

	err := m.model().
		WithContext(ctx).
		Where("student_id = ?", studentID).
		Where("contest_date >= ?", cutoff).
		Order("contest_date ASC").
		Find(&list).Error

	return list, err
}

func (m *defaultContestRecord) Delete(
	ctx context.Context,
	studentID, platform, contestID string,
) error {

	return m.model().
		Unscoped().
		Where("student_id = ? AND platform = ? AND contest_id = ?",
			studentID, platform, contestID).
		Delete(&ContestRecord{}).Error
}

func (m *defaultContestRecord) DeleteRange(
	ctx context.Context,
	studentIDs []string,
	from, to time.Time,
) error {

	tx := m.model().
		WithContext(ctx).
		Unscoped().
		Where("contest_date BETWEEN ? AND ?", from, to)

	if len(studentIDs) > 0 {
		tx = tx.Where("student_id IN ?", studentIDs)
	}

	return tx.Delete(&ContestRecord{}).Error
}

func (m *defaultContestRecord) FindByContest(
	ctx context.Context,
	platform, contestID string,
) ([]*ContestRecord, error) {
	var list []*ContestRecord
	err := m.model().
		WithContext(ctx).
		Where("platform = ? AND contest_id = ?", platform, contestID).
		Order("contest_rank ASC").
		Find(&list).Error
	return list, err
}
