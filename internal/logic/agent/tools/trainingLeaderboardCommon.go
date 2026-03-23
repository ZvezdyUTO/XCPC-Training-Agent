package tools

import (
	"aATA/internal/model"
	"context"
	"fmt"
	"time"
)

type leaderboardPeriod string

const (
	leaderboardDay   leaderboardPeriod = "day"
	leaderboardWeek  leaderboardPeriod = "week"
	leaderboardMonth leaderboardPeriod = "month"
)

type trainingLeaderboardRunner struct {
	daily model.DailyTrainingStatsModel
	users model.UsersModel
}

func newTrainingLeaderboardRunner(
	daily model.DailyTrainingStatsModel,
	users model.UsersModel,
) *trainingLeaderboardRunner {
	return &trainingLeaderboardRunner{
		daily: daily,
		users: users,
	}
}

func (r *trainingLeaderboardRunner) run(
	ctx context.Context,
	period leaderboardPeriod,
	dateStr string,
	limit int,
	asc bool,
) (map[string]any, error) {
	baseDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 20
	}

	from, to, err := calcLeaderboardRange(period, baseDate)
	if err != nil {
		return nil, err
	}

	rankList, err := r.daily.RankByTotal(ctx, from, to, limit, asc)
	if err != nil {
		return nil, err
	}

	items := make([]map[string]any, 0, len(rankList))
	for i, item := range rankList {
		name := ""
		u, err := r.users.FindByID(item.StudentID)
		if err == nil && u != nil {
			name = u.Name
		}

		items = append(items, map[string]any{
			"rank":       i + 1,
			"student_id": item.StudentID,
			"name":       name,
			"total":      item.Total,
		})
	}

	return map[string]any{
		"period": string(period),
		"from":   from.Format("2006-01-02"),
		"to":     to.Format("2006-01-02"),
		"count":  len(items),
		"items":  items,
	}, nil
}

func calcLeaderboardRange(period leaderboardPeriod, base time.Time) (time.Time, time.Time, error) {
	loc := base.Location()
	dayStart := time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, loc)

	switch period {
	case leaderboardDay:
		return dayStart, dayStart, nil
	case leaderboardWeek:
		weekday := int(dayStart.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		from := dayStart.AddDate(0, 0, -(weekday - 1))
		to := from.AddDate(0, 0, 6)
		return from, to, nil
	case leaderboardMonth:
		from := time.Date(dayStart.Year(), dayStart.Month(), 1, 0, 0, 0, 0, loc)
		to := from.AddDate(0, 1, -1)
		return from, to, nil
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("invalid leaderboard period: %s", period)
	}
}
