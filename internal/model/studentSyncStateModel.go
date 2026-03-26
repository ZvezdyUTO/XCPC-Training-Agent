package model

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type (
	StudentSyncStateModel interface {
		Upsert(ctx context.Context, data *StudentSyncState) error
		FindByStudentID(ctx context.Context, studentID string) (*StudentSyncState, error)
	}

	defaultStudentSyncState struct {
		db *gorm.DB
	}
)

func NewStudentSyncStateModel(db *gorm.DB) StudentSyncStateModel {
	return &defaultStudentSyncState{
		db: db,
	}
}

func (m *defaultStudentSyncState) model() *gorm.DB {
	return m.db.Model(&StudentSyncState{})
}

func (m *defaultStudentSyncState) Upsert(ctx context.Context, data *StudentSyncState) error {
	return m.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "student_id"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"is_fully_initialized",
				"latest_successful_date",
				"updated_at",
			}),
		}).
		Create(data).Error
}

func (m *defaultStudentSyncState) FindByStudentID(ctx context.Context, studentID string) (*StudentSyncState, error) {
	var res StudentSyncState
	err := m.db.WithContext(ctx).
		Where("student_id = ?", studentID).
		First(&res).Error

	switch err {
	case nil:
		return &res, nil
	case gorm.ErrRecordNotFound:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}
