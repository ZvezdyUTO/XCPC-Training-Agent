package anomaly

import (
	"aATA/internal/model"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

func (s *service) detectInactiveDays(
	ctx context.Context,
	studentID string,
	asOf time.Time,
) (*model.TrainingAlert, bool, error) {
	cfg := s.getConfig()
	maxDays := cfg.InactiveDaysHighThreshold
	currentFrom := asOf.AddDate(0, 0, -(maxDays - 1))
	currentTo := asOf

	records, err := s.daily.FindRange(ctx, studentID, currentFrom, currentTo)
	if err != nil {
		return nil, false, fmt.Errorf("加载连续停训窗口训练数据失败: %w", err)
	}

	byDate := make(map[string]*model.DailyTrainingStats, len(records))
	for _, item := range records {
		if item == nil {
			continue
		}
		byDate[item.StatDate.Format(time.DateOnly)] = item
	}

	inactiveDays := 0
	for i := 0; i < maxDays; i++ {
		day := asOf.AddDate(0, 0, -i).Format(time.DateOnly)
		stat := byDate[day]
		total := 0
		if stat != nil {
			total = stat.CFNewTotal + stat.ACNewTotal
		}
		if total > 0 {
			break
		}
		inactiveDays++
	}
	if inactiveDays < cfg.InactiveDaysThreshold {
		return nil, false, nil
	}

	effectiveFrom := asOf.AddDate(0, 0, -(inactiveDays - 1))
	baselineTo := effectiveFrom.AddDate(0, 0, -1)
	baselineFrom := baselineTo.AddDate(0, 0, -(cfg.BaselineWindowDays - 1))
	baselineStats, err := s.daily.SumRange(ctx, studentID, baselineFrom, baselineTo)
	if err != nil {
		return nil, false, fmt.Errorf("加载停训规则基线窗口训练数据失败: %w", err)
	}

	baselineTotal := 0
	if baselineStats != nil {
		baselineTotal = baselineStats.CFNewTotal + baselineStats.ACNewTotal
	}
	baselineAvg := float64(baselineTotal) / float64(cfg.BaselineWindowDays)
	if baselineAvg < cfg.InactiveBaselineMinDaily {
		return nil, false, nil
	}

	severity := model.AlertSeverityLow
	if inactiveDays >= cfg.InactiveDaysHighThreshold {
		severity = model.AlertSeverityHigh
	} else if inactiveDays >= cfg.InactiveDaysMediumThreshold {
		severity = model.AlertSeverityMedium
	}

	evidence, _ := json.Marshal(map[string]any{
		"metric": "inactive_days",
		"current_window": map[string]string{
			"from": effectiveFrom.Format(time.DateOnly),
			"to":   currentTo.Format(time.DateOnly),
		},
		"inactive_days": inactiveDays,
		"current_total": 0,
		"thresholds": map[string]int{
			"low_days":    cfg.InactiveDaysThreshold,
			"medium_days": cfg.InactiveDaysMediumThreshold,
			"high_days":   cfg.InactiveDaysHighThreshold,
		},
		"baseline_avg_daily": round2(baselineAvg),
		"baseline_window": map[string]string{
			"from": baselineFrom.Format(time.DateOnly),
			"to":   baselineTo.Format(time.DateOnly),
		},
	})

	actions, _ := json.Marshal([]string{
		"先确认是否存在请假、课程周、考试周等客观原因，避免误判为训练懈怠。",
		"若无客观原因，建议立即安排短周期恢复计划（例如连续 3 天每天至少 2 题）。",
		"下一次复盘重点观察是否恢复到稳定日均训练量。",
	})

	title := fmt.Sprintf("已连续 %d 天无训练记录", inactiveDays)

	return &model.TrainingAlert{
		StudentID: studentID,
		AlertDate: asOf,
		AlertType: inactiveDaysAlertType,
		Severity:  severity,
		Status:    model.AlertStatusNew,
		Title:     title,
		Evidence:  evidence,
		Actions:   actions,
	}, true, nil
}
