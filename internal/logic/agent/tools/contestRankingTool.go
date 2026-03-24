package tools

import (
	"aATA/internal/domain"
	"aATA/internal/logic/agent"
	"aATA/internal/model"
	"context"
	"encoding/json"
)

type ContestRankingTool struct {
	contest model.ContestRecordModel
	users   model.UsersModel
}

func NewContestRankingTool(
	contest model.ContestRecordModel,
	users model.UsersModel,
) *ContestRankingTool {
	return &ContestRankingTool{
		contest: contest,
		users:   users,
	}
}

func (t *ContestRankingTool) Name() string {
	return "contest_ranking"
}

func (t *ContestRankingTool) Description() string {
	return "查询某一场比赛中数据库内所有成员的排名情况"
}

func (t *ContestRankingTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Parameters: map[string]agent.Parameter{
			"platform": {
				Type:        "string",
				Description: "比赛平台，可选 CF 或 AC",
				Enum:        []string{"CF", "AC"},
			},
			"contest_id": {
				Type:        "string",
				Description: "比赛ID",
			},
		},
		Required: []string{"platform", "contest_id"},
	}
}

func (t *ContestRankingTool) Call(ctx context.Context, input json.RawMessage) (any, error) {
	var args struct {
		Platform  string `json:"platform"`
		ContestID string `json:"contest_id"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return nil, err
	}

	list, err := t.contest.FindByContest(ctx, args.Platform, args.ContestID)
	if err != nil {
		return nil, err
	}

	res := domain.ContestRankingResult{
		Platform:  args.Platform,
		ContestID: args.ContestID,
		Count:     len(list),
		Items:     make([]domain.ContestRankingItem, 0, len(list)),
	}

	if len(list) > 0 {
		res.ContestName = list[0].ContestName
		if !list[0].ContestDate.IsZero() {
			res.ContestDate = list[0].ContestDate.Format("2006-01-02 15:04:05")
		}
	}

	for _, record := range list {
		name := ""
		u, err := t.users.FindByID(record.StudentID)
		if err == nil && u != nil {
			name = u.Name
		}

		res.Items = append(res.Items, domain.ContestRankingItem{
			StudentID:    record.StudentID,
			Name:         name,
			ContestRank:  record.ContestRank,
			OldRating:    record.OldRating,
			NewRating:    record.NewRating,
			RatingChange: record.RatingChange,
			Performance:  record.Performance,
		})
	}

	return res, nil
}
