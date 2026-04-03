package model

import (
	"encoding/json"
	"time"
)

// AnomalyRuleConfig 持久化异常检测规则配置。
// 当前按单行配置存储，固定使用 id=1。
type AnomalyRuleConfig struct {
	ID         int             `gorm:"column:id;primaryKey" json:"id"`
	ConfigJSON json.RawMessage `gorm:"column:config_json" json:"config_json"`
	CreatedAt  time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (AnomalyRuleConfig) TableName() string {
	return "anomaly_rule_config"
}

