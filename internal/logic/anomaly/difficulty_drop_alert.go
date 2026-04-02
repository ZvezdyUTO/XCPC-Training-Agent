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

	maxDays := cfg.DifficultyDropHighDaysThreshold
	currentFrom := asOf.AddDate(0, 0, -(maxDays - 1))
	currentTo := asOf

	items, err := s.daily.FindRange(ctx, studentID, currentFrom, currentTo)
	if err != nil {
		return nil, false, fmt.Errorf("加载高难题连续未达标窗口训练数据失败: %w", err)
	}
	byDate := make(map[string]*model.DailyTrainingStats, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		byDate[item.StatDate.Format(time.DateOnly)] = item
	}

	highMinPerDay := cfg.DifficultyDropMinCurrentTotal
	windowTotal := 0
	windowHigh := 0
	dailyHigh := make([]map[string]any, 0, maxDays)
	for i := 0; i < maxDays; i++ {
		day := currentFrom.AddDate(0, 0, i)
		dayKey := day.Format(time.DateOnly)
		stat := byDate[dayKey]
		total, high, _ := totalHighEasyByLevel(stat, level, cfg)
		dailyHigh = append(dailyHigh, map[string]any{
			"date":        dayKey,
			"high_count":  high,
			"total_count": total,
			"is_low":      high < highMinPerDay,
		})
	}
	// 从 asOf 向前统计连续“高难题未达标”的天数。
	inactiveDays := 0
	for i := 0; i < maxDays; i++ {
		day := asOf.AddDate(0, 0, -i).Format(time.DateOnly)
		stat := byDate[day]
		total, high, _ := totalHighEasyByLevel(stat, level, cfg)
		windowTotal += total
		windowHigh += high
		if high >= highMinPerDay {
			break
		}
		inactiveDays++
	}
	if inactiveDays < cfg.DifficultyDropCurrentWindowDays {
		return nil, false, nil
	}
	effectiveFrom := asOf.AddDate(0, 0, -(inactiveDays - 1))

	severity := model.AlertSeverityLow
	if inactiveDays >= cfg.DifficultyDropHighDaysThreshold {
		severity = model.AlertSeverityHigh
	} else if inactiveDays >= cfg.DifficultyDropMediumDaysThreshold {
		severity = model.AlertSeverityMedium
	}

	evidence, _ := json.Marshal(map[string]any{
		"metric": "high_difficulty_inactive_days",
		"self_level": map[string]any{
			"cf":         level.CF,
			"ac":         level.AC,
			"has_cf":     level.HasCF,
			"has_ac":     level.HasAC,
			"round_base": cfg.DifficultyLevelRoundBase,
			"high_delta": cfg.DifficultyRelativeHighDelta,
		},
		"current_window": map[string]string{
			"from": effectiveFrom.Format(time.DateOnly),
			"to":   currentTo.Format(time.DateOnly),
		},
		"inactive_days":     inactiveDays,
		"high_min_per_day":  highMinPerDay,
		"window_high_total": windowHigh,
		"window_total":      windowTotal,
		"window_high_ratio": round4(safeDiv(float64(windowHigh), float64(windowTotal))),
		"thresholds": map[string]int{
			"low_days":    cfg.DifficultyDropCurrentWindowDays,
			"medium_days": cfg.DifficultyDropMediumDaysThreshold,
			"high_days":   cfg.DifficultyDropHighDaysThreshold,
		},
		"daily_high_detail": dailyHigh,
	})

	actions, _ := json.Marshal([]string{
		"建议在本周训练计划中增加高难题硬性配额（例如每天至少 1 题高于个人水平阈值）。",
		"先排查是否存在阶段性降难安排，若有计划内原因可先标记观察。",
		"下一轮复盘重点关注连续天数内每日高难题数量是否恢复达标。",
	})

	title := fmt.Sprintf("已连续 %d 天高难题未达标（每日少于 %d 题）", inactiveDays, highMinPerDay)

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
