package tools

import (
	"aATA/internal/logic/agent"
	"aATA/internal/model"
	"context"
	"encoding/json"
	"time"
)

type TrainingSummaryTool struct {
	daily model.DailyTrainingStatsModel
}

func NewTrainingSummaryTool(daily model.DailyTrainingStatsModel) *TrainingSummaryTool {
	return &TrainingSummaryTool{daily: daily}
}

func (t *TrainingSummaryTool) Name() string {
	return "training_summary_range"
}

func (t *TrainingSummaryTool) Description() string {
	return "查询某个学生在指定时间范围内的训练累计数据（按难度统计）"
}

func (t *TrainingSummaryTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Parameters: map[string]agent.Parameter{
			"student_id": {
				Type:        "string",
				Description: "学生ID",
			},
			"from": {
				Type:        "string",
				Description: "开始日期，格式 2006-01-02",
			},
			"to": {
				Type:        "string",
				Description: "结束日期，格式 2006-01-02",
			},
		},
		Required: []string{"student_id", "from", "to"},
	}
}

func (t *TrainingSummaryTool) Call(ctx context.Context, input json.RawMessage) (any, error) {
	var args struct {
		StudentID string `json:"student_id"`
		From      string `json:"from"`
		To        string `json:"to"`
	}

	if err := json.Unmarshal(input, &args); err != nil {
		return nil, err
	}

	fromTime, err := time.Parse("2006-01-02", args.From)
	if err != nil {
		return nil, err
	}
	toTime, err := time.Parse("2006-01-02", args.To)
	if err != nil {
		return nil, err
	}

	res, err := t.daily.SumRange(ctx, args.StudentID, fromTime, toTime)
	if err != nil {
		return nil, err
	}

	dist := map[string]int{
		"800_1100":  res.CFNew800 + res.CFNew900 + res.CFNew1000 + res.CFNew1100,
		"1200_1300": res.CFNew1200 + res.CFNew1300,
		"1400_1500": res.CFNew1400 + res.CFNew1500,
		"1600_1800": res.CFNew1600 + res.CFNew1700 + res.CFNew1800,
		"1900_2000": res.CFNew1900 + res.CFNew2000,
		"2100_2200": res.CFNew2100 + res.CFNew2200,
		"2300_plus": res.CFNew2300 + res.CFNew2400 +
			res.CFNew2500 + res.CFNew2600 +
			res.CFNew2700 + res.CFNew2800Plus,
	}

	return map[string]any{
		"student_id":      args.StudentID,
		"from":            args.From,
		"to":              args.To,
		"cf_total":        res.CFNewTotal,
		"cf_distribution": dist,
	}, nil
}
