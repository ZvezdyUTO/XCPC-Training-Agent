package tools

import (
	applogic "aATA/internal/logic"
	"aATA/internal/logic/agent/tooling"
	"aATA/internal/model"
	"context"
	"encoding/json"
	"time"
)

// TrainingSummaryTool 负责查询单个学生区间训练累计，并返回统一训练价值评分拆解。
// 它只读取已落库的训练统计和比赛记录，不触发抓取、补数或额外兜底。
type TrainingSummaryTool struct {
	daily   model.DailyTrainingStatsModel
	contest model.ContestRecordModel
}

// NewTrainingSummaryTool 创建单人训练查询工具。
// 该工具复用排行榜同一套评分公式，保证模型查询和手动查询口径一致。
func NewTrainingSummaryTool(
	daily model.DailyTrainingStatsModel,
	contest model.ContestRecordModel,
) *TrainingSummaryTool {
	return &TrainingSummaryTool{
		daily:   daily,
		contest: contest,
	}
}

func (t *TrainingSummaryTool) Name() string {
	return "training_summary_range"
}

func (t *TrainingSummaryTool) Description() string {
	return "查询某个学生在指定时间范围内的训练累计数据（按难度统计）"
}

func (t *TrainingSummaryTool) Schema() tooling.ToolSchema {
	return tooling.ToolSchema{
		Parameters: map[string]tooling.Parameter{
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
	records, err := t.contest.FindByStudent(ctx, args.StudentID)
	if err != nil {
		return nil, err
	}

	cfDist, acDist := applogic.BuildTrainingDistributions(res)
	trainingValue := applogic.BuildTrainingValueSummary(res, records)

	return map[string]any{
		"student_id":      args.StudentID,
		"from":            args.From,
		"to":              args.To,
		"cf_total":        res.CFNewTotal,
		"cf_distribution": cfDist,
		"ac_total":        res.ACNewTotal,
		"ac_distribution": acDist,
		"training_value":  trainingValue,
	}, nil
}
