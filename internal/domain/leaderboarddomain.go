package domain

// TrainingLeaderboardReq 描述训练价值排行榜的手动查询参数。
// 这里只接受明确时间区间和前 N 名限制，不负责猜测默认时间范围。
type TrainingLeaderboardReq struct {
	From string `form:"from" json:"from" binding:"required"`
	To   string `form:"to" json:"to" binding:"required"`
	TopN int    `form:"top_n" json:"top_n"`
}

// TrainingLeaderboardResp 表示一次训练价值排行榜查询结果。
// 返回值同时保留总分和拆解项，便于前端解释排序原因。
type TrainingLeaderboardResp struct {
	ScoringVersion string                    `json:"scoring_version"`
	From           string                    `json:"from"`
	To             string                    `json:"to"`
	TopN           int                       `json:"top_n"`
	Count          int                       `json:"count"`
	Items          []TrainingLeaderboardItem `json:"items"`
}

// TrainingLeaderboardItem 表示排行榜中的单个学生条目。
type TrainingLeaderboardItem struct {
	Rank            int                              `json:"rank"`
	StudentID       string                           `json:"student_id"`
	StudentName     string                           `json:"student_name"`
	SolvedTotal     int                              `json:"solved_total"`
	Score           float64                          `json:"score"`
	VolumeScore     float64                          `json:"volume_score"`
	DifficultyScore float64                          `json:"difficulty_score"`
	ChallengeScore  float64                          `json:"challenge_score"`
	UndefinedTotal  int                              `json:"undefined_total"`
	UndefinedRatio  float64                          `json:"undefined_ratio"`
	CFRating        TrainingLeaderboardRatingProfile `json:"cf_rating"`
	ACRating        TrainingLeaderboardRatingProfile `json:"ac_rating"`
	CF              TrainingLeaderboardPlatformScore `json:"cf"`
	AC              TrainingLeaderboardPlatformScore `json:"ac"`
}

// TrainingLeaderboardRatingProfile 表示某个平台的当前分、峰值分和能力参考线。
// 能力参考线用于估计“高于本人水平的训练价值”，不直接作为奖励项加分。
type TrainingLeaderboardRatingProfile struct {
	Current       *int     `json:"current"`
	Peak          *int     `json:"peak"`
	AbilityAnchor *float64 `json:"ability_anchor"`
}

// TrainingLeaderboardPlatformScore 表示单个平台上的训练贡献拆解。
type TrainingLeaderboardPlatformScore struct {
	SolvedTotal     int     `json:"solved_total"`
	KnownTotal      int     `json:"known_total"`
	UndefinedTotal  int     `json:"undefined_total"`
	Score           float64 `json:"score"`
	VolumeScore     float64 `json:"volume_score"`
	DifficultyScore float64 `json:"difficulty_score"`
	ChallengeScore  float64 `json:"challenge_score"`
}

// TrainingValueSummary 表示单个学生在指定时间区间内的训练价值评分拆解。
// 该结构复用排行榜同一套公式，保证手动查询、前端展示和 Agent 工具口径一致。
type TrainingValueSummary struct {
	ScoringVersion  string                           `json:"scoring_version"`
	SolvedTotal     int                              `json:"solved_total"`
	Score           float64                          `json:"score"`
	VolumeScore     float64                          `json:"volume_score"`
	DifficultyScore float64                          `json:"difficulty_score"`
	ChallengeScore  float64                          `json:"challenge_score"`
	UndefinedTotal  int                              `json:"undefined_total"`
	UndefinedRatio  float64                          `json:"undefined_ratio"`
	CFRating        TrainingLeaderboardRatingProfile `json:"cf_rating"`
	ACRating        TrainingLeaderboardRatingProfile `json:"ac_rating"`
	CF              TrainingLeaderboardPlatformScore `json:"cf"`
	AC              TrainingLeaderboardPlatformScore `json:"ac"`
}
