package model

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type (
	StudentSyncStateModel interface {
		// Upsert 按学生维度写入最新同步状态。
		Upsert(ctx context.Context, data *StudentSyncState) error
		// FindByStudentID 查询单个学生的同步状态。
		FindByStudentID(ctx context.Context, studentID string) (*StudentSyncState, error)
		// List 查询当前表内全部同步状态记录。
		// 这里只返回状态表原始内容，不负责补齐缺失学生或拼装额外业务字段。
		List(ctx context.Context) ([]*StudentSyncState, error)
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

func (m *defaultStudentSyncState) List(ctx context.Context) ([]*StudentSyncState, error) {
	var list []*StudentSyncState
	err := m.model().
		WithContext(ctx).
		Order("updated_at DESC").
		Find(&list).Error
	return list, err
}
