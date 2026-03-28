package tools

import (
	"aATA/internal/logic/agent/tooling"
	"aATA/internal/model"
	"context"
	"encoding/json"
)

type TrainingDayLeaderboardTool struct {
	runner *trainingLeaderboardRunner
}

func NewTrainingDayLeaderboardTool(
	daily model.DailyTrainingStatsModel,
	users model.UsersModel,
) *TrainingDayLeaderboardTool {
	return &TrainingDayLeaderboardTool{
		runner: newTrainingLeaderboardRunner(daily, users),
	}
}

func (t *TrainingDayLeaderboardTool) Name() string {
	return "training_day_leaderboard"
}

func (t *TrainingDayLeaderboardTool) Description() string {
	return "查询数据库内所有人在某一天的过题量排行榜"
}

func (t *TrainingDayLeaderboardTool) Schema() tooling.ToolSchema {
	return tooling.ToolSchema{
		Parameters: map[string]tooling.Parameter{
			"date": {
				Type:        "string",
				Description: "查询日期，格式 2006-01-02",
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

func (t *TrainingDayLeaderboardTool) Call(ctx context.Context, input json.RawMessage) (any, error) {
	var args struct {
		Date  string `json:"date"`
		Limit int    `json:"limit"`
		Asc   bool   `json:"asc"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return nil, err
	}

	return t.runner.run(ctx, leaderboardDay, args.Date, args.Limit, args.Asc)
}
