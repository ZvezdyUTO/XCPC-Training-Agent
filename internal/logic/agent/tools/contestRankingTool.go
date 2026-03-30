package tools

import (
	"aATA/internal/domain"
	"aATA/internal/logic/agent/tooling"
	"aATA/internal/model"
	"context"
	"encoding/json"
)

func loadStudentNameMap(ctx context.Context, users model.UsersModel, studentIDs []string) (map[string]string, error) {
	if len(studentIDs) == 0 {
		return map[string]string{}, nil
	}

	list, _, err := users.List(ctx, &domain.UserListReq{Ids: studentIDs})
	if err != nil {
		return nil, err
	}

	nameMap := make(map[string]string, len(list))
	for _, user := range list {
		nameMap[user.Id] = user.Name
	}
	return nameMap, nil
}

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

func (t *ContestRankingTool) Schema() tooling.ToolSchema {
	return tooling.ToolSchema{
		Parameters: map[string]tooling.Parameter{
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

	studentIDs := make([]string, 0, len(list))
	for _, record := range list {
		studentIDs = append(studentIDs, record.StudentID)
	}

	nameMap, err := loadStudentNameMap(ctx, t.users, studentIDs)
	if err != nil {
		return nil, err
	}

	for _, record := range list {
		res.Items = append(res.Items, domain.ContestRankingItem{
			StudentID:    record.StudentID,
			StudentName:  nameMap[record.StudentID],
			Platform:     record.Platform,
			ContestID:    record.ContestID,
			Name:         record.ContestName,
			Date:         record.ContestDate,
			Rank:         record.ContestRank,
			OldRating:    record.OldRating,
			NewRating:    record.NewRating,
			RatingChange: record.RatingChange,
		})
	}

	return res, nil
}
