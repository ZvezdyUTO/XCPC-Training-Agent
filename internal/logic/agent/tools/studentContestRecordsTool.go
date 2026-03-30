package tools

import (
	"aATA/internal/logic/agent/tooling"
	"aATA/internal/model"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	defaultStudentContestLimit = 20
	maxStudentContestLimit     = 50
)

// StudentContestRecordsTool 负责查询单个学生的比赛记录。
// 该工具只做记录读取和轻量过滤，不负责生成统计摘要或排行榜。
type StudentContestRecordsTool struct {
	contest model.ContestRecordModel
}

// NewStudentContestRecordsTool 创建单人比赛记录查询工具。
func NewStudentContestRecordsTool(contest model.ContestRecordModel) *StudentContestRecordsTool {
	return &StudentContestRecordsTool{contest: contest}
}

// Name 返回工具唯一名称。
func (t *StudentContestRecordsTool) Name() string {
	return "student_contest_records"
}

// Description 返回工具职责说明。
func (t *StudentContestRecordsTool) Description() string {
	return "查询某个学生的比赛记录，支持按平台过滤并限制返回条数，默认按时间倒序返回最近记录"
}

// Schema 声明该工具接受的输入结构。
func (t *StudentContestRecordsTool) Schema() tooling.ToolSchema {
	return tooling.ToolSchema{
		Parameters: map[string]tooling.Parameter{
			"student_id": {
				Type:        "string",
				Description: "学生ID",
			},
			"platform": {
				Type:        "string",
				Description: "比赛平台，可选 CF 或 AC；为空表示不过滤平台",
				Enum:        []string{"CF", "AC"},
			},
			"limit": {
				Type:        "integer",
				Description: "最多返回多少条记录，默认 20，最大 50",
			},
		},
		Required: []string{"student_id"},
	}
}

// Call 执行单人比赛记录查询。
func (t *StudentContestRecordsTool) Call(ctx context.Context, input json.RawMessage) (any, error) {
	var args struct {
		StudentID string `json:"student_id"`
		Platform  string `json:"platform"`
		Limit     int    `json:"limit"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return nil, err
	}

	platform := strings.ToUpper(strings.TrimSpace(args.Platform))
	if platform != "" && platform != "CF" && platform != "AC" {
		return nil, fmt.Errorf("invalid platform: %s", args.Platform)
	}

	limit := args.Limit
	if limit <= 0 {
		limit = defaultStudentContestLimit
	}
	if limit > maxStudentContestLimit {
		limit = maxStudentContestLimit
	}

	list, err := t.contest.FindByStudent(ctx, args.StudentID)
	if err != nil {
		return nil, err
	}

	items := make([]map[string]any, 0, limit)
	for i := len(list) - 1; i >= 0 && len(items) < limit; i-- {
		record := list[i]
		if platform != "" && record.Platform != platform {
			continue
		}
		items = append(items, compactStudentContestRecord(record))
	}

	return map[string]any{
		"student_id": args.StudentID,
		"platform":   platform,
		"count":      len(items),
		"items":      items,
	}, nil
}

// compactStudentContestRecord 只保留比赛记录查询真正需要的字段。
// 这里按“查记录”语义返回，不再额外拼装高层统计结论。
func compactStudentContestRecord(record *model.ContestRecord) map[string]any {
	return map[string]any{
		"platform":      record.Platform,
		"contest_id":    record.ContestID,
		"contest_name":  record.ContestName,
		"contest_date":  record.ContestDate.Format(time.DateOnly),
		"rank":          record.ContestRank,
		"old_rating":    record.OldRating,
		"new_rating":    record.NewRating,
		"rating_change": record.RatingChange,
		"performance":   record.Performance,
	}
}
