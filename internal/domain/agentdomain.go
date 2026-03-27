package domain

// AdminAgentTaskRunReq /v1/admin/agent/task/run
type AdminAgentTaskRunReq struct {
	Task      string                 `json:"task" binding:"required"`
	Params    map[string]interface{} `json:"params"`
	TraceMode string                 `json:"trace_mode"`
}
