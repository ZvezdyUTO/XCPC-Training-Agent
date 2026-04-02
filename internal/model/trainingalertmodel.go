package model

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type (
	TrainingAlertModel interface {
		// UpsertMany 批量写入预警；同 student_id+alert_date+alert_type 时执行更新。
		UpsertMany(ctx context.Context, data []*TrainingAlert) error
		// List 按条件查询预警列表，并返回总数。
		List(ctx context.Context, query *TrainingAlertListQuery) ([]*TrainingAlert, int64, error)
		// UpdateStatus 更新单条预警状态。
		UpdateStatus(ctx context.Context, id int64, status string) error
	}

	defaultTrainingAlert struct {
		db *gorm.DB
	}
)

func NewTrainingAlertModel(db *gorm.DB) TrainingAlertModel {
	return &defaultTrainingAlert{db: db}
}

func (m *defaultTrainingAlert) model() *gorm.DB {
	return m.db.Model(&TrainingAlert{})
}

func (m *defaultTrainingAlert) UpsertMany(ctx context.Context, data []*TrainingAlert) error {
	if len(data) == 0 {
		return nil
	}

	return m.model().
		WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "student_id"},
				{Name: "alert_date"},
				{Name: "alert_type"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"severity",
				"status",
				"title",
				"evidence_json",
				"actions_json",
				"updated_at",
			}),
		}).
		Create(&data).Error
}

func (m *defaultTrainingAlert) List(
	ctx context.Context,
	query *TrainingAlertListQuery,
) ([]*TrainingAlert, int64, error) {
	if query == nil {
		query = &TrainingAlertListQuery{}
	}

	db := m.model().WithContext(ctx)

	if query.StudentID != "" {
		db = db.Where("student_id = ?", query.StudentID)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.Severity != "" {
		db = db.Where("severity = ?", query.Severity)
	}
	if query.From != nil {
		db = db.Where("alert_date >= ?", *query.From)
	}
	if query.To != nil {
		db = db.Where("alert_date <= ?", *query.To)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if query.Page > 0 && query.Count > 0 {
		offset := (query.Page - 1) * query.Count
		db = db.Offset(offset).Limit(query.Count)
	}

	var list []*TrainingAlert
	err := db.Order("alert_date DESC, id DESC").Find(&list).Error
	if err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

func (m *defaultTrainingAlert) UpdateStatus(ctx context.Context, id int64, status string) error {
	return m.model().
		WithContext(ctx).
		Where("id = ?", id).
		Update("status", status).Error
}
