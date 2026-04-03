package tools

import (
	"aATA/internal/model"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"aATA/internal/logic/agent/tooling"
)

const (
	defaultAlertPage  = 1
	defaultAlertCount = 20
	maxAlertCount     = 100
)

// TrainingAlertsTool 负责查询异常预警列表。
// 该工具只读取预警落库结果，不触发检测流程。
type TrainingAlertsTool struct {
	alerts model.TrainingAlertModel
}

// NewTrainingAlertsTool 创建预警列表查询工具。
func NewTrainingAlertsTool(alerts model.TrainingAlertModel) *TrainingAlertsTool {
	return &TrainingAlertsTool{alerts: alerts}
}

func (t *TrainingAlertsTool) Name() string {
	return "training_alerts_list"
}

func (t *TrainingAlertsTool) Description() string {
	return "查询训练异常预警列表，支持按学生、状态、严重等级和日期范围过滤"
}

func (t *TrainingAlertsTool) Schema() tooling.ToolSchema {
	return tooling.ToolSchema{
		Parameters: map[string]tooling.Parameter{
			"student_id": {
				Type:        "string",
				Description: "学生学号，可选",
			},
			"status": {
				Type:        "string",
				Description: "预警状态，可选：new/ack/resolved",
				Enum:        []string{"new", "ack", "resolved"},
			},
			"severity": {
				Type:        "string",
				Description: "预警等级，可选：low/medium/high",
				Enum:        []string{"low", "medium", "high"},
			},
			"from": {
				Type:        "string",
				Description: "开始日期，格式 2006-01-02，可选",
			},
			"to": {
				Type:        "string",
				Description: "结束日期，格式 2006-01-02，可选",
			},
			"page": {
				Type:        "integer",
				Description: "页码，默认 1",
			},
			"count": {
				Type:        "integer",
				Description: "每页数量，默认 20，最大 100",
			},
		},
		Required: []string{},
	}
}

func (t *TrainingAlertsTool) Call(ctx context.Context, input json.RawMessage) (any, error) {
	var args struct {
		StudentID string `json:"student_id"`
		Status    string `json:"status"`
		Severity  string `json:"severity"`
		From      string `json:"from"`
		To        string `json:"to"`
		Page      int    `json:"page"`
		Count     int    `json:"count"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return nil, err
	}

	query, err := buildAlertListQueryArgs(args)
	if err != nil {
		return nil, err
	}

	list, total, err := t.alerts.List(ctx, query)
	if err != nil {
		return nil, err
	}

	items := make([]map[string]any, 0, len(list))
	for _, item := range list {
		if item == nil {
			continue
		}
		evidence := map[string]any{}
		_ = json.Unmarshal(item.Evidence, &evidence)

		actions := []string{}
		_ = json.Unmarshal(item.Actions, &actions)

		items = append(items, map[string]any{
			"id":         item.ID,
			"student_id": item.StudentID,
			"alert_date": item.AlertDate.Format(time.DateOnly),
			"alert_type": item.AlertType,
			"severity":   item.Severity,
			"status":     item.Status,
			"title":      item.Title,
			"evidence":   evidence,
			"actions":    actions,
		})
	}

	return map[string]any{
		"count": total,
		"items": items,
	}, nil
}

func buildAlertListQueryArgs(args struct {
	StudentID string `json:"student_id"`
	Status    string `json:"status"`
	Severity  string `json:"severity"`
	From      string `json:"from"`
	To        string `json:"to"`
	Page      int    `json:"page"`
	Count     int    `json:"count"`
}) (*model.TrainingAlertListQuery, error) {
	query := &model.TrainingAlertListQuery{
		StudentID: args.StudentID,
		Status:    args.Status,
		Severity:  args.Severity,
		Page:      args.Page,
		Count:     args.Count,
	}

	if query.Page <= 0 {
		query.Page = defaultAlertPage
	}
	if query.Count <= 0 {
		query.Count = defaultAlertCount
	}
	if query.Count > maxAlertCount {
		query.Count = maxAlertCount
	}

	if args.From != "" {
		from, err := time.Parse(time.DateOnly, args.From)
		if err != nil {
			return nil, fmt.Errorf("开始日期格式非法，应为 YYYY-MM-DD")
		}
		query.From = &from
	}
	if args.To != "" {
		to, err := time.Parse(time.DateOnly, args.To)
		if err != nil {
			return nil, fmt.Errorf("结束日期格式非法，应为 YYYY-MM-DD")
		}
		query.To = &to
	}
	if query.From != nil && query.To != nil && query.To.Before(*query.From) {
		return nil, fmt.Errorf("结束日期不能早于开始日期")
	}

	return query, nil
}

