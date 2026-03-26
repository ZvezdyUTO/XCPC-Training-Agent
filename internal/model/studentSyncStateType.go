package model

import (
	"time"
)

type StudentSyncState struct {
	StudentID            string     `gorm:"column:student_id"`
	IsFullyInitialized   int64      `gorm:"column:is_fully_initialized"`
	LatestSuccessfulDate *time.Time `gorm:"column:latest_successful_date"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (m *StudentSyncState) TableName() string {
	return "student_sync_state"
}
