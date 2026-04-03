package model

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type (
	AnomalyRuleConfigModel interface {
		Find(ctx context.Context, id int) (*AnomalyRuleConfig, error)
		Upsert(ctx context.Context, data *AnomalyRuleConfig) error
	}

	defaultAnomalyRuleConfig struct {
		db *gorm.DB
	}
)

func NewAnomalyRuleConfigModel(db *gorm.DB) AnomalyRuleConfigModel {
	return &defaultAnomalyRuleConfig{db: db}
}

func (m *defaultAnomalyRuleConfig) model() *gorm.DB {
	return m.db.Model(&AnomalyRuleConfig{})
}

func (m *defaultAnomalyRuleConfig) Find(ctx context.Context, id int) (*AnomalyRuleConfig, error) {
	var res AnomalyRuleConfig
	err := m.model().
		WithContext(ctx).
		Where("id = ?", id).
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

func (m *defaultAnomalyRuleConfig) Upsert(ctx context.Context, data *AnomalyRuleConfig) error {
	return m.model().
		WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"config_json",
				"updated_at",
			}),
		}).
		Create(data).Error
}

