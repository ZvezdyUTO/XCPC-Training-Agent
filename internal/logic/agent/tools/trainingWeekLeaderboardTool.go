package tools

import (
	"aATA/internal/logic/agent/tooling"
	"aATA/internal/model"
	"context"
	"encoding/json"
)

type TrainingWeekLeaderboardTool struct {
	runner *trainingLeaderboardRunner
}

func NewTrainingWeekLeaderboardTool(
	daily model.DailyTrainingStatsModel,
	users model.UsersModel,
) *TrainingWeekLeaderboardTool {
	return &TrainingWeekLeaderboardTool{
		runner: newTrainingLeaderboardRunner(daily, users),
	}
}

func (t *TrainingWeekLeaderboardTool) Name() string {
	return "training_week_leaderboard"
}

func (t *TrainingWeekLeaderboardTool) Description() string {
	return "查询数据库内所有人在某一周的过题量排行榜"
}

func (t *TrainingWeekLeaderboardTool) Schema() tooling.ToolSchema {
	return tooling.ToolSchema{
		Parameters: map[string]tooling.Parameter{
			"date": {
				Type:        "string",
				Description: "基准日期，格式 2006-01-02，自动查询该日期所在周",
			},
			"limit": {
				Type:        "integer",
				Description: "返回前多少名，默认 20",
			},
			"asc": {
				Type:        "boolean",
				Description: "是否升序，默认 false（从高到低）",
			},
		},
		Required: []string{"date"},
	}
}

func (t *TrainingWeekLeaderboardTool) Call(ctx context.Context, input json.RawMessage) (any, error) {
	var args struct {
		Date  string `json:"date"`
		Limit int    `json:"limit"`
		Asc   bool   `json:"asc"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return nil, err
	}

	return t.runner.run(ctx, leaderboardWeek, args.Date, args.Limit, args.Asc)
}
