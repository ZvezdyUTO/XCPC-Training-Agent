package domain

// AdminAlertListReq 管理员查询预警列表请求。
type AdminAlertListReq struct {
	StudentID string `form:"student_id" json:"student_id"`
	Status    string `form:"status" json:"status"`
	Severity  string `form:"severity" json:"severity"`
	From      string `form:"from" json:"from"`
	To        string `form:"to" json:"to"`
	Page      int    `form:"page" json:"page"`
	Count     int    `form:"count" json:"count"`
}

// AdminAlertItem 预警列表项。
type AdminAlertItem struct {
	ID        int64                  `json:"id"`
	StudentID string                 `json:"student_id"`
	AlertDate string                 `json:"alert_date"`
	AlertType string                 `json:"alert_type"`
	Severity  string                 `json:"severity"`
	Status    string                 `json:"status"`
	Title     string                 `json:"title"`
	Evidence  map[string]interface{} `json:"evidence"`
	Actions   []string               `json:"actions"`
	CreatedAt string                 `json:"created_at"`
	UpdatedAt string                 `json:"updated_at"`
}

// AdminAlertListResp 管理员预警列表响应。
type AdminAlertListResp struct {
	Count int64            `json:"count"`
	List  []AdminAlertItem `json:"list"`
}

