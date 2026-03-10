package tools

import (
	"aATA/internal/logic/agent"
	"aATA/internal/model"
	"context"
	"encoding/json"
)

type ContestRatingSummaryTool struct {
	contest model.ContestRecordModel
}

func NewContestRatingSummaryTool(contest model.ContestRecordModel) *ContestRatingSummaryTool {
	return &ContestRatingSummaryTool{contest: contest}
}

func (t *ContestRatingSummaryTool) Name() string {
	return "contest_rating_summary"
}

func (t *ContestRatingSummaryTool) Description() string {
	return "查询某个学生的比赛 rating 统计信息（最新、历史最高、平均等）"
}

func (t *ContestRatingSummaryTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Parameters: map[string]agent.Parameter{
			"student_id": {
				Type:        "string",
				Description: "学生ID",
			},
			"include_trend": {
				Type:        "boolean",
				Description: "是否返回 rating 变化趋势数据",
			},
		},
		Required: []string{"student_id"},
	}
}

func (t *ContestRatingSummaryTool) Call(ctx context.Context, input json.RawMessage) (any, error) {
	var args struct {
		StudentID    string `json:"student_id"`
		IncludeTrend bool   `json:"include_trend"`
	}

	if err := json.Unmarshal(input, &args); err != nil {
		return nil, err
	}

	list, err := t.contest.FindByStudent(ctx, args.StudentID)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return map[string]any{
			"student_id": args.StudentID,
			"message":    "no contest records found",
		}, nil
	}

	var (
		latestRating   int
		maxRating      int
		sumRating      int
		sumPerformance int
		trend          []map[string]any
	)

	for i, record := range list {
		if i == 0 {
			maxRating = record.NewRating
		}

		if record.NewRating > maxRating {
			maxRating = record.NewRating
		}

		sumRating += record.NewRating
		sumPerformance += record.Performance

		if args.IncludeTrend {
			trend = append(trend, map[string]any{
				"contest_id":    record.ContestID,
				"contest_date":  record.ContestDate,
				"new_rating":    record.NewRating,
				"rating_change": record.RatingChange,
				"performance":   record.Performance,
			})
		}
	}

	latestRating = list[len(list)-1].NewRating
	avgRating := sumRating / len(list)
	avgPerformance := sumPerformance / len(list)

	result := map[string]any{
		"student_id":      args.StudentID,
		"contest_count":   len(list),
		"latest_rating":   latestRating,
		"max_rating":      maxRating,
		"avg_rating":      avgRating,
		"avg_performance": avgPerformance,
	}

	if args.IncludeTrend {
		result["rating_trend"] = trend
	}

	return result, nil
}
