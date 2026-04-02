package anomaly

import (
	"aATA/internal/app/apperr"
	"aATA/internal/model"
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

const anomalyRuleConfigID = 1

func (s *service) GetRuleConfig(ctx context.Context) RuleConfig {
	_ = s.loadRuleConfigIfExists(ctx)
	return s.getConfig()
}

func (s *service) UpdateRuleConfig(ctx context.Context, cfg RuleConfig) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	if err := s.persistRuleConfig(ctx, cfg); err != nil {
		return err
	}

	s.mu.Lock()
	s.cfg = cfg
	s.mu.Unlock()
	return nil
}

// PatchRuleConfig 以“部分更新”的方式修改异常检测规则配置。
// 只会覆盖 patch 中显式传入的字段，其余字段保持不变。
func (s *service) PatchRuleConfig(ctx context.Context, patch RuleConfigPatch) (RuleConfig, error) {
	if err := s.loadRuleConfigIfExists(ctx); err != nil {
		return RuleConfig{}, err
	}

	next := s.getConfig()
	mergeRuleConfigPatch(&next, patch)

	if err := next.Validate(); err != nil {
		return RuleConfig{}, apperr.New(apperr.KindUser, "invalid_rule_config", err.Error(), 400)
	}
	if err := s.persistRuleConfig(ctx, next); err != nil {
		return RuleConfig{}, err
	}

	s.mu.Lock()
	s.cfg = next
	s.mu.Unlock()
	return next, nil
}

func (s *service) getConfig() RuleConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

func (s *service) loadRuleConfigIfExists(ctx context.Context) error {
	if s.configs == nil {
		return nil
	}

	stored, err := s.configs.Find(ctx, anomalyRuleConfigID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("加载异常规则配置失败: %w", err)
	}

	var cfg RuleConfig
	if err := json.Unmarshal(stored.ConfigJSON, &cfg); err != nil {
		return fmt.Errorf("解析异常规则配置失败: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("异常规则配置不合法: %w", err)
	}

	s.mu.Lock()
	s.cfg = cfg
	s.mu.Unlock()
	return nil
}

func (s *service) persistRuleConfig(ctx context.Context, cfg RuleConfig) error {
	if s.configs == nil {
		return nil
	}

	raw, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("序列化异常规则配置失败: %w", err)
	}

	return s.configs.Upsert(ctx, &model.AnomalyRuleConfig{
		ID:         anomalyRuleConfigID,
		ConfigJSON: raw,
	})
}

func mergeRuleConfigPatch(dst *RuleConfig, patch RuleConfigPatch) {
	if dst == nil {
		return
	}

	if patch.CurrentWindowDays != nil {
		dst.CurrentWindowDays = *patch.CurrentWindowDays
	}
	if patch.BaselineWindowDays != nil {
		dst.BaselineWindowDays = *patch.BaselineWindowDays
	}
	if patch.BaselineMinDaily != nil {
		dst.BaselineMinDaily = *patch.BaselineMinDaily
	}
	if patch.CurrentMinDailyForAlert != nil {
		dst.CurrentMinDailyForAlert = *patch.CurrentMinDailyForAlert
	}
	if patch.DropLowThreshold != nil {
		dst.DropLowThreshold = *patch.DropLowThreshold
	}
	if patch.DropMediumThreshold != nil {
		dst.DropMediumThreshold = *patch.DropMediumThreshold
	}
	if patch.DropHighThreshold != nil {
		dst.DropHighThreshold = *patch.DropHighThreshold
	}
	if patch.InactiveDaysThreshold != nil {
		dst.InactiveDaysThreshold = *patch.InactiveDaysThreshold
	}
	if patch.InactiveBaselineMinDaily != nil {
		dst.InactiveBaselineMinDaily = *patch.InactiveBaselineMinDaily
	}
	if patch.DifficultyDropCurrentWindowDays != nil {
		dst.DifficultyDropCurrentWindowDays = *patch.DifficultyDropCurrentWindowDays
	}
	if patch.DifficultyDropBaselineWindowDays != nil {
		dst.DifficultyDropBaselineWindowDays = *patch.DifficultyDropBaselineWindowDays
	}
	if patch.DifficultyDropMinCurrentTotal != nil {
		dst.DifficultyDropMinCurrentTotal = *patch.DifficultyDropMinCurrentTotal
	}
	if patch.DifficultyDropMinBaselineHighRatio != nil {
		dst.DifficultyDropMinBaselineHighRatio = *patch.DifficultyDropMinBaselineHighRatio
	}
	if patch.DifficultyLevelRoundBase != nil {
		dst.DifficultyLevelRoundBase = *patch.DifficultyLevelRoundBase
	}
	if patch.DifficultyRelativeHighDelta != nil {
		dst.DifficultyRelativeHighDelta = *patch.DifficultyRelativeHighDelta
	}
	if patch.DifficultyRelativeEasyDelta != nil {
		dst.DifficultyRelativeEasyDelta = *patch.DifficultyRelativeEasyDelta
	}
	if patch.DifficultyDropLowThreshold != nil {
		dst.DifficultyDropLowThreshold = *patch.DifficultyDropLowThreshold
	}
	if patch.DifficultyDropMediumThreshold != nil {
		dst.DifficultyDropMediumThreshold = *patch.DifficultyDropMediumThreshold
	}
	if patch.DifficultyDropHighThreshold != nil {
		dst.DifficultyDropHighThreshold = *patch.DifficultyDropHighThreshold
	}
}
