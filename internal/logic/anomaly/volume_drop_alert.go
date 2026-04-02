package anomaly

import (
	"aATA/internal/model"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

func (s *service) detectVolumeDrop(
	ctx context.Context,
	studentID string,
	asOf time.Time,
) (*model.TrainingAlert, bool, error) {
	cfg := s.getConfig()
	currentFrom := asOf.AddDate(0, 0, -(cfg.CurrentWindowDays - 1))
	currentTo := asOf

	baselineTo := currentFrom.AddDate(0, 0, -1)
	baselineFrom := baselineTo.AddDate(0, 0, -(cfg.BaselineWindowDays - 1))

	currentStats, err := s.daily.SumRange(ctx, studentID, currentFrom, currentTo)
	if err != nil {
		return nil, false, fmt.Errorf("加载当前窗口训练数据失败: %w", err)
	}
	baselineStats, err := s.daily.SumRange(ctx, studentID, baselineFrom, baselineTo)
	if err != nil {
		return nil, false, fmt.Errorf("加载基线窗口训练数据失败: %w", err)
	}

	currentTotal := 0
	baselineTotal := 0
	if currentStats != nil {
		currentTotal = currentStats.CFNewTotal + currentStats.ACNewTotal
	}
	if baselineStats != nil {
		baselineTotal = baselineStats.CFNewTotal + baselineStats.ACNewTotal
	}

	currentAvg := float64(currentTotal) / float64(cfg.CurrentWindowDays)
	baselineAvg := float64(baselineTotal) / float64(cfg.BaselineWindowDays)
	if baselineAvg < cfg.BaselineMinDaily {
		return nil, false, nil
	}
	if currentAvg >= cfg.CurrentMinDailyForAlert {
		return nil, false, nil
	}

	dropRatio := (baselineAvg - currentAvg) / baselineAvg
	severity := classifyDropSeverity(dropRatio, cfg)
	if severity == "" {
		return nil, false, nil
	}

	evidence, _ := json.Marshal(map[string]any{
		"metric":         "daily_solved",
		"current_window": map[string]string{"from": currentFrom.Format(time.DateOnly), "to": currentTo.Format(time.DateOnly)},
		"baseline_window": map[string]string{
			"from": baselineFrom.Format(time.DateOnly),
			"to":   baselineTo.Format(time.DateOnly),
		},
		"current_total":   currentTotal,
		"baseline_total":  baselineTotal,
		"current_avg":     round2(currentAvg),
		"baseline_avg":    round2(baselineAvg),
		"drop_ratio":      round4(dropRatio),
		"threshold_level": severity,
	})

	actions, _ := json.Marshal([]string{
		"在下一周为该学生安排固定训练时段，先恢复稳定题量（建议每天至少 2 题）。",
		"优先检查是否存在比赛周、课程周或请假等客观因素，避免误报。",
		"下次复盘重点关注日均题量是否回到个人基线的 80% 以上。",
	})

	title := fmt.Sprintf("近7天日均题量较基线下降 %.0f%%", dropRatio*100)

	return &model.TrainingAlert{
		StudentID: studentID,
		AlertDate: asOf,
		AlertType: volumeDropAlertType,
		Severity:  severity,
		Status:    model.AlertStatusNew,
		Title:     title,
		Evidence:  evidence,
		Actions:   actions,
	}, true, nil
}

func classifyDropSeverity(dropRatio float64, cfg RuleConfig) string {
	switch {
	case dropRatio >= cfg.DropHighThreshold:
		return model.AlertSeverityHigh
	case dropRatio >= cfg.DropMediumThreshold:
		return model.AlertSeverityMedium
	case dropRatio >= cfg.DropLowThreshold:
		return model.AlertSeverityLow
	default:
		return ""
	}
}
