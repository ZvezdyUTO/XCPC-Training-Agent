package anomaly

import (
	"aATA/internal/model"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

func (s *service) detectDifficultyDrop(
	ctx context.Context,
	studentID string,
	asOf time.Time,
) (*model.TrainingAlert, bool, error) {
	cfg := s.getConfig()
	records, err := s.contest.FindByStudent(ctx, studentID)
	if err != nil {
		return nil, false, fmt.Errorf("加载学生比赛记录失败: %w", err)
	}
	level := buildStudentDifficultyLevel(records, cfg)
	if !level.HasCF && !level.HasAC {
		return nil, false, nil
	}

	currentFrom := asOf.AddDate(0, 0, -(cfg.DifficultyDropCurrentWindowDays - 1))
	currentTo := asOf

	baselineTo := currentFrom.AddDate(0, 0, -1)
	baselineFrom := baselineTo.AddDate(0, 0, -(cfg.DifficultyDropBaselineWindowDays - 1))

	currentStats, err := s.daily.SumRange(ctx, studentID, currentFrom, currentTo)
	if err != nil {
		return nil, false, fmt.Errorf("加载难度占比当前窗口训练数据失败: %w", err)
	}
	baselineStats, err := s.daily.SumRange(ctx, studentID, baselineFrom, baselineTo)
	if err != nil {
		return nil, false, fmt.Errorf("加载难度占比基线窗口训练数据失败: %w", err)
	}

	currentTotal, currentHigh, currentEasy := totalHighEasyByLevel(currentStats, level, cfg)
	baselineTotal, baselineHigh, baselineEasy := totalHighEasyByLevel(baselineStats, level, cfg)

	if currentTotal < cfg.DifficultyDropMinCurrentTotal || baselineTotal <= 0 {
		return nil, false, nil
	}

	currentHighRatio := safeDiv(float64(currentHigh), float64(currentTotal))
	baselineHighRatio := safeDiv(float64(baselineHigh), float64(baselineTotal))
	if baselineHighRatio < cfg.DifficultyDropMinBaselineHighRatio {
		return nil, false, nil
	}
	if baselineHighRatio <= 0 {
		return nil, false, nil
	}

	dropRatio := safeDiv(baselineHighRatio-currentHighRatio, baselineHighRatio)
	severity := classifyDifficultyDropSeverity(dropRatio, cfg)
	if severity == "" {
		return nil, false, nil
	}

	evidence, _ := json.Marshal(map[string]any{
		"metric": "high_difficulty_ratio",
		"self_level": map[string]any{
			"cf":              level.CF,
			"ac":              level.AC,
			"has_cf":          level.HasCF,
			"has_ac":          level.HasAC,
			"round_base":      cfg.DifficultyLevelRoundBase,
			"high_delta":      cfg.DifficultyRelativeHighDelta,
			"easy_delta":      cfg.DifficultyRelativeEasyDelta,
		},
		"current_window": map[string]string{
			"from": currentFrom.Format(time.DateOnly),
			"to":   currentTo.Format(time.DateOnly),
		},
		"baseline_window": map[string]string{
			"from": baselineFrom.Format(time.DateOnly),
			"to":   baselineTo.Format(time.DateOnly),
		},
		"current_total":        currentTotal,
		"current_high_total":   currentHigh,
		"current_easy_total":   currentEasy,
		"current_high_ratio":   round4(currentHighRatio),
		"baseline_total":       baselineTotal,
		"baseline_high_total":  baselineHigh,
		"baseline_easy_total":  baselineEasy,
		"baseline_high_ratio":  round4(baselineHighRatio),
		"high_ratio_drop_rate": round4(dropRatio),
	})

	actions, _ := json.Marshal([]string{
		"建议在本周训练计划中明确加入“高于个人水平 200+”的题目配额，逐步恢复高难题占比。",
		"优先检查是否近期训练目标偏向补基础，若是则在阶段切换后恢复挑战题比例。",
		"复盘时对比下一窗口的高难题占比是否回升到历史基线附近。",
	})

	title := fmt.Sprintf("近%d天高难题占比较基线下降 %.0f%%", cfg.DifficultyDropCurrentWindowDays, dropRatio*100)

	return &model.TrainingAlert{
		StudentID: studentID,
		AlertDate: asOf,
		AlertType: difficultyDropAlertType,
		Severity:  severity,
		Status:    model.AlertStatusNew,
		Title:     title,
		Evidence:  evidence,
		Actions:   actions,
	}, true, nil
}

func classifyDifficultyDropSeverity(dropRatio float64, cfg RuleConfig) string {
	switch {
	case dropRatio >= cfg.DifficultyDropHighThreshold:
		return model.AlertSeverityHigh
	case dropRatio >= cfg.DifficultyDropMediumThreshold:
		return model.AlertSeverityMedium
	case dropRatio >= cfg.DifficultyDropLowThreshold:
		return model.AlertSeverityLow
	default:
		return ""
	}
}
