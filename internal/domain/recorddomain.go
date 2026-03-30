package domain

import "time"

type SyncStudentItem struct {
	StudentID string `json:"student_id" binding:"required"`
}

// AdminSyncOneTrainingReq 描述管理员触发单个学生训练同步的请求。
// 这里只接收明确的 student_id，不额外支持批量或模糊匹配。
type AdminSyncOneTrainingReq struct {
	StudentID string `json:"student_id" binding:"required"`
}

type AdminSyncTrainingReq struct {
	Students []SyncStudentItem `json:"students" binding:"required,dive"`
	From     time.Time         `json:"from" binding:"required"`
	To       time.Time         `json:"to" binding:"required"`
}

type ContestRecord struct {
	StudentID string `json:"student_id"`

	Platform  string `json:"platform"` // CF / AC
	ContestID string `json:"contest_id"`

	Name string    `json:"name"`
	Date time.Time `json:"date"`

	Rank         int `json:"rank"`
	OldRating    int `json:"old_rating"`
	NewRating    int `json:"new_rating"`
	RatingChange int `json:"rating_change"`
	Performance  int `json:"performance"`
}

// ContestRankingItem 表示某场比赛中一个成员的排名结果。
// 该结构专门用于排名查询返回，保留展示所需字段，并显式补充学生姓名。
type ContestRankingItem struct {
	StudentID    string    `json:"student_id"`
	StudentName  string    `json:"student_name"`
	Platform     string    `json:"platform"`
	ContestID    string    `json:"contest_id"`
	Name         string    `json:"name"`
	Date         time.Time `json:"date"`
	Rank         int       `json:"rank"`
	OldRating    int       `json:"old_rating"`
	NewRating    int       `json:"new_rating"`
	RatingChange int       `json:"rating_change"`
}

type DailyTrainingStats struct {
	StudentID string    `json:"student_id"`
	Date      time.Time `json:"date"`

	// CF
	CFNewTotal     int         `json:"cf_new_total"`
	CFNewUndefined int         `json:"cf_new_undefined"`
	CFNew          map[int]int `json:"cf_new"` // JSON object keys 会是字符串 "800"，Go 这边能转成 int

	// AC
	ACNewTotal     int            `json:"ac_new_total"`
	ACNewUndefined int            `json:"ac_new_undefined"`
	ACNewRange     map[string]int `json:"ac_new_range"`
}

// ImportContestRecordsReq 批量导入比赛记录请求
type ImportContestRecordsReq struct {
	StudentID string
	Records   []*ContestRecord
}

// ImportTrainingStatsReq 批量导入每日训练记录请求
type ImportTrainingStatsReq struct {
	StudentID string
	Stats     []*DailyTrainingStats
}

// DeleteContestRangeReq 批量删除比赛记录请求
type DeleteContestRangeReq struct {
	StudentIDs []string // 为空表示全部
	From       time.Time
	To         time.Time
}

// DeleteTrainingRangeReq 批量删除每日训练记录请求
type DeleteTrainingRangeReq struct {
	StudentIDs []string
	From       time.Time
	To         time.Time
}

// DeleteContestByIDReq 精准按比赛 ID 和 学生 ID 删除比赛记录
type DeleteContestByIDReq struct {
	StudentID string
	Platform  string
	ContestID string
}

// DeleteTrainingByDateReq 精准按日期和学生 ID 删除训练记录
type DeleteTrainingByDateReq struct {
	StudentID string
	Date      time.Time
}

type ContestRankingResult struct {
	Platform    string               `json:"platform"`
	ContestID   string               `json:"contest_id"`
	ContestName string               `json:"contest_name"`
	ContestDate string               `json:"contest_date"`
	Count       int                  `json:"count"`
	Items       []ContestRankingItem `json:"items"`
}
