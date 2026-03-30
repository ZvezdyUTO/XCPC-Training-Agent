package tools

import (
	"aATA/internal/domain"
	applogic "aATA/internal/logic"
	"aATA/internal/logic/agent/tooling"
	"context"
	"encoding/json"
)

// TrainingValueLeaderboardTool 负责把训练价值排行榜暴露给 Agent。
// 具体评分逻辑复用业务层实现，避免 HTTP 入口和 Agent 入口各自维护一套公式。
type TrainingValueLeaderboardTool struct {
	leaderboard applogic.TrainingLeaderboard
}

// NewTrainingValueLeaderboardTool 创建训练价值排行榜工具。
func NewTrainingValueLeaderboardTool(
	leaderboard applogic.TrainingLeaderboard,
) *TrainingValueLeaderboardTool {
	return &TrainingValueLeaderboardTool{leaderboard: leaderboard}
}

// Name 返回工具唯一名称。
func (t *TrainingValueLeaderboardTool) Name() string {
	return "training_value_leaderboard"
}

// Description 返回工具职责说明。
func (t *TrainingValueLeaderboardTool) Description() string {
	return "查询指定时间范围内的训练价值排行榜，综合题量、难度和相对本人能力线的挑战价值进行排序"
}

// Schema 声明该工具接受的输入结构。
func (t *TrainingValueLeaderboardTool) Schema() tooling.ToolSchema {
	return tooling.ToolSchema{
		Parameters: map[string]tooling.Parameter{
			"from": {
				Type:        "string",
				Description: "开始日期，格式 2006-01-02",
			},
			"to": {
				Type:        "string",
				Description: "结束日期，格式 2006-01-02",
			},
			"top_n": {
				Type:        "integer",
				Description: "返回前多少名，默认 20",
			},
		},
		Required: []string{"from", "to"},
	}
}

// Call 执行训练价值排行榜查询。
func (t *TrainingValueLeaderboardTool) Call(ctx context.Context, input json.RawMessage) (any, error) {
	var req domain.TrainingLeaderboardReq
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, err
	}

	return t.leaderboard.Query(ctx, &req)
}
