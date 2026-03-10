package model

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type (
	DailyTrainingStatsModel interface {
		// Insert 插入某个用户某一天的训练增量数据。
		Insert(ctx context.Context, data *DailyTrainingStats) error
		// Upsert 如果该用户在该日期已有记录则更新，否则插入。
		Upsert(ctx context.Context, data *DailyTrainingStats) error
		// DeleteByDate 删除某用户某天的训练记录。
		DeleteByDate(ctx context.Context, studentID string, date time.Time) error
		//DeleteRange 批量删除训练记录
		DeleteRange(ctx context.Context, studentIDs []string, from, to time.Time) error

		// FindByDate 查询某个用户在指定日期的训练数据。
		FindByDate(ctx context.Context, studentID string, date time.Time) (*DailyTrainingStats, error)
		// FindRange 查询某用户在指定时间区间内的训练数据。
		FindRange(ctx context.Context, studentID string, from, to time.Time) ([]*DailyTrainingStats, error)
		// SumRange 统计某个学生在时间区间内的累计训练数据
		SumRange(ctx context.Context, studentID string, from, to time.Time) (*DailyTrainingStats, error)
		// RankByTotal 区间总题量排行
		RankByTotal(ctx context.Context, from, to time.Time, limit int, asc bool) ([]*RankItem, error)
	}

	defaultDailyTrainingStats struct {
		db *gorm.DB
	}
)

func NewDailyTrainingStatsModel(db *gorm.DB) DailyTrainingStatsModel {
	return &defaultDailyTrainingStats{db: db}
}

func (m *defaultDailyTrainingStats) model() *gorm.DB {
	return m.db.Model(&DailyTrainingStats{})
}

func (m *defaultDailyTrainingStats) Insert(ctx context.Context, data *DailyTrainingStats) error {
	return m.model().
		WithContext(ctx).
		Create(data).Error
}

func (m *defaultDailyTrainingStats) Upsert(ctx context.Context, data *DailyTrainingStats) error {
	return m.model().
		WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "student_id"},
				{Name: "stat_date"},
			},
			UpdateAll: true,
		}).
		Create(data).Error
}

func (m *defaultDailyTrainingStats) FindByDate(
	ctx context.Context,
	studentID string,
	date time.Time,
) (*DailyTrainingStats, error) {

	var res DailyTrainingStats
	err := m.model().
		WithContext(ctx).
		Where("student_id = ? AND stat_date = ?", studentID, date).
		First(&res).Error

	if err == gorm.ErrRecordNotFound {
		return nil, ErrNotFound
	}
	return &res, err
}

func (m *defaultDailyTrainingStats) FindRange(
	ctx context.Context,
	studentID string,
	from, to time.Time,
) ([]*DailyTrainingStats, error) {

	var list []*DailyTrainingStats
	err := m.model().
		WithContext(ctx).
		Where("student_id = ?", studentID).
		Where("stat_date BETWEEN ? AND ?", from, to).
		Order("stat_date ASC").
		Find(&list).Error

	return list, err
}

func (m *defaultDailyTrainingStats) DeleteByDate(
	ctx context.Context,
	studentID string,
	date time.Time,
) error {
	return m.model().
		WithContext(ctx).
		Where("student_id = ? AND stat_date = ?", studentID, date).
		Delete(&DailyTrainingStats{}).Error
}

func (m *defaultDailyTrainingStats) DeleteRange(
	ctx context.Context,
	studentIDs []string,
	from, to time.Time,
) error {

	tx := m.model().
		WithContext(ctx).
		Where("stat_date BETWEEN ? AND ?", from, to)

	if len(studentIDs) > 0 {
		tx = tx.Where("student_id IN ?", studentIDs)
	}

	return tx.Delete(&DailyTrainingStats{}).Error
}

func (m *defaultDailyTrainingStats) SumRange(
	ctx context.Context,
	studentID string,
	from, to time.Time,
) (*DailyTrainingStats, error) {

	var res DailyTrainingStats

	err := m.model().
		WithContext(ctx).
		Select(`
			student_id,
			SUM(cf_new_total) as cf_new_total,
			SUM(cf_new_800) as cf_new_800,
			SUM(cf_new_900) as cf_new_900,
			SUM(cf_new_1000) as cf_new_1000,
			SUM(cf_new_1100) as cf_new_1100,
			SUM(cf_new_1200) as cf_new_1200,
			SUM(cf_new_1300) as cf_new_1300,
			SUM(cf_new_1400) as cf_new_1400,
			SUM(cf_new_1500) as cf_new_1500,
			SUM(cf_new_1600) as cf_new_1600,
			SUM(cf_new_1700) as cf_new_1700,
			SUM(cf_new_1800) as cf_new_1800,
			SUM(cf_new_1900) as cf_new_1900,
			SUM(cf_new_2000) as cf_new_2000,
			SUM(cf_new_2100) as cf_new_2100,
			SUM(cf_new_2200) as cf_new_2200,
			SUM(cf_new_2300) as cf_new_2300,
			SUM(cf_new_2400) as cf_new_2400,
			SUM(cf_new_2500) as cf_new_2500,
			SUM(cf_new_2600) as cf_new_2600,
			SUM(cf_new_2700) as cf_new_2700,
			SUM(cf_new_2800_plus) as cf_new_2800_plus
		`).
		Where("student_id = ?", studentID).
		Where("stat_date BETWEEN ? AND ?", from, to).
		Group("student_id").
		Scan(&res).Error

	if err != nil {
		return nil, err
	}

	return &res, nil
}

func (m *defaultDailyTrainingStats) RankByTotal(
	ctx context.Context,
	from, to time.Time,
	limit int,
	asc bool,
) ([]*RankItem, error) {

	var list []*RankItem

	order := "DESC"
	if asc {
		order = "ASC"
	}

	err := m.model().
		WithContext(ctx).
		Select("student_id, SUM(cf_new_total) as total").
		Where("stat_date BETWEEN ? AND ?", from, to).
		Group("student_id").
		Order("total " + order).
		Limit(limit).
		Scan(&list).Error

	return list, err
}
