package model

import (
	"encoding/json"
	"time"
)

const (
	AlertSeverityLow    = "low"
	AlertSeverityMedium = "medium"
	AlertSeverityHigh   = "high"
)

const (
	AlertStatusNew      = "new"
	AlertStatusAck      = "ack"
	AlertStatusResolved = "resolved"
)

// TrainingAlert 表示训练异常检测后落库的一条预警记录。
// 它只描述持久化字段，不负责阈值计算或告警生成策略。
type TrainingAlert struct {
	ID         int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	StudentID  string          `gorm:"column:student_id" json:"student_id"`
	AlertDate  time.Time       `gorm:"column:alert_date;type:date" json:"alert_date"`
	AlertType  string          `gorm:"column:alert_type" json:"alert_type"`
	Severity   string          `gorm:"column:severity" json:"severity"`
	Status     string          `gorm:"column:status" json:"status"`
	Title      string          `gorm:"column:title" json:"title"`
	Evidence   json.RawMessage `gorm:"column:evidence_json" json:"evidence_json"`
	Actions    json.RawMessage `gorm:"column:actions_json" json:"actions_json"`
	CreatedAt  time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (TrainingAlert) TableName() string {
	return "training_alerts"
}

// TrainingAlertListQuery 描述预警列表查询条件。
// 这里仅表达数据层筛选参数，不承载 HTTP 绑定语义。
type TrainingAlertListQuery struct {
	StudentID string
	Status    string
	Severity  string
	From      *time.Time
	To        *time.Time
	Page      int
	Count     int
}
