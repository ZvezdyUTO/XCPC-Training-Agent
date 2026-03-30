package model

import (
	"time"
)

// StudentSyncState 表示单个学生当前持久化的训练同步状态。
// 该结构只负责描述状态表字段本身，不负责推导同步模式或补齐缺失用户信息。
type StudentSyncState struct {
	StudentID            string     `gorm:"column:student_id" json:"student_id"`
	IsFullyInitialized   int64      `gorm:"column:is_fully_initialized" json:"is_fully_initialized"`
	LatestSuccessfulDate *time.Time `gorm:"column:latest_successful_date" json:"latest_successful_date"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (m *StudentSyncState) TableName() string {
	return "student_sync_state"
}
