package tools

import (
	"aATA/internal/logic/agent/tooling"
	"aATA/internal/model"
	"context"
	"encoding/json"
	"time"
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
	return "查询某个学生的比赛 rating 摘要信息，默认返回压缩后的统计、关键比赛和有限趋势采样，避免返回完整逐场记录"
}

func (t *ContestRatingSummaryTool) Schema() tooling.ToolSchema {
	return tooling.ToolSchema{
		Parameters: map[string]tooling.Parameter{
			"student_id": {
				Type:        "string",
				Description: "学生ID",
			},
			"include_trend": {
				Type:        "boolean",
				Description: "是否返回压缩后的 rating 趋势摘要",
			},
			"max_points": {
				Type:        "integer",
				Description: "趋势采样点上限，仅在 include_trend=true 时生效，默认 8，最大 12",
			},
		},
		Required: []string{"student_id"},
	}
}

func (t *ContestRatingSummaryTool) Call(ctx context.Context, input json.RawMessage) (any, error) {
	var args struct {
		StudentID    string `json:"student_id"`
		IncludeTrend bool   `json:"include_trend"`
		MaxPoints    int    `json:"max_points"`
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

	if args.MaxPoints <= 0 {
		args.MaxPoints = 8
	}
	if args.MaxPoints > 12 {
		args.MaxPoints = 12
	}

	var (
		latestRating    int
		maxRating       int
		sumRating       int
		sumPerformance  int
		totalChange     int
		positiveCount   int
		negativeCount   int
		stableCount     int
		maxGain         = list[0].RatingChange
		maxDrop         = list[0].RatingChange
		bestPerformance = list[0]
		bestGainRecord  = list[0]
		worstDropRecord = list[0]
		platformCount   = map[string]int{}
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
		totalChange += record.RatingChange
		platformCount[record.Platform]++

		switch {
		case record.RatingChange > 0:
			positiveCount++
		case record.RatingChange < 0:
			negativeCount++
		default:
			stableCount++
		}

		if record.RatingChange > maxGain {
			maxGain = record.RatingChange
			bestGainRecord = record
		}
		if record.RatingChange < maxDrop {
			maxDrop = record.RatingChange
			worstDropRecord = record
		}
		if record.Performance > bestPerformance.Performance {
			bestPerformance = record
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
		"date_range": map[string]any{
			"first_contest_date": list[0].ContestDate.Format(time.DateOnly),
			"last_contest_date":  list[len(list)-1].ContestDate.Format(time.DateOnly),
		},
		"platform_distribution": platformCount,
		"rating_change_summary": map[string]any{
			"net_change":     totalChange,
			"avg_change":     totalChange / len(list),
			"positive_count": positiveCount,
			"negative_count": negativeCount,
			"stable_count":   stableCount,
			"max_gain":       maxGain,
			"max_drop":       maxDrop,
		},
		"key_contests": map[string]any{
			"latest":           compactContestRecord(list[len(list)-1]),
			"best_performance": compactContestRecord(bestPerformance),
			"best_gain":        compactContestRecord(bestGainRecord),
			"worst_drop":       compactContestRecord(worstDropRecord),
		},
	}

	if args.IncludeTrend {
		result["rating_trend"] = buildTrendDigest(list, args.MaxPoints)
		result["trend_compressed"] = true
	}

	return result, nil
}

func compactContestRecord(record *model.ContestRecord) map[string]any {
	return map[string]any{
		"platform":      record.Platform,
		"contest_id":    record.ContestID,
		"contest_name":  record.ContestName,
		"contest_date":  record.ContestDate.Format(time.DateOnly),
		"new_rating":    record.NewRating,
		"rating_change": record.RatingChange,
		"performance":   record.Performance,
		"rank":          record.ContestRank,
	}
}

func buildTrendDigest(list []*model.ContestRecord, maxPoints int) map[string]any {
	if len(list) == 0 {
		return map[string]any{
			"sample_count": 0,
			"points":       []map[string]any{},
		}
	}

	points := sampleContestRecords(list, maxPoints)
	return map[string]any{
		"sample_count": len(points),
		"sampled_from": len(list),
		"points":       points,
	}
}

func sampleContestRecords(list []*model.ContestRecord, maxPoints int) []map[string]any {
	n := len(list)
	if maxPoints <= 0 || n <= maxPoints {
		out := make([]map[string]any, 0, n)
		for _, record := range list {
			out = append(out, compactTrendPoint(record))
		}
		return out
	}

	indexes := make([]int, 0, maxPoints)
	indexes = append(indexes, 0)
	if maxPoints > 1 {
		step := float64(n-1) / float64(maxPoints-1)
		last := 0
		for i := 1; i < maxPoints-1; i++ {
			idx := int(step * float64(i))
			if idx <= last {
				idx = last + 1
			}
			if idx >= n-1 {
				idx = n - 2
			}
			indexes = append(indexes, idx)
			last = idx
		}
		indexes = append(indexes, n-1)
	}

	out := make([]map[string]any, 0, len(indexes))
	for _, idx := range indexes {
		out = append(out, compactTrendPoint(list[idx]))
	}
	return out
}

func compactTrendPoint(record *model.ContestRecord) map[string]any {
	return map[string]any{
		"contest_date":  record.ContestDate.Format(time.DateOnly),
		"platform":      record.Platform,
		"new_rating":    record.NewRating,
		"rating_change": record.RatingChange,
		"performance":   record.Performance,
	}
}
