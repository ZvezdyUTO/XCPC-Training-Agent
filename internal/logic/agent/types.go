/*
定义整个 agent 系统的统一数据结构（非常关键，后面都用它）：

AgentInput：mode/date/student_id
AgentState：输入 + 已有工具结果 + reasoning trace + step count
LLMAction：call_tool / finish
LLMResponse：严格 JSON 协议
FinalOutput：两种模式的输出 schema（coach_attention / student_diagnosis）
ToolCall：tool_name + arguments

这份文件就是“协议中心”。
*/

package agent

type AgentInput struct {
	Query  string                 `json:"query"`
	Params map[string]interface{} `json:"params"`
}

type ToolResult struct {
	ToolName string
	Result   interface{}
}

type AgentState struct {
	Input        AgentInput
	ToolResults  []ToolResult
	ReasoningLog []string
	Step         int
}

type FinalOutput struct {
	DecisionType  string                 `json:"decision_type"`
	FocusStudents []string               `json:"focus_students"`
	Confidence    float64                `json:"confidence"`
	Report        string                 `json:"report"`
	Metrics       map[string]interface{} `json:"metrics"`
}

type LLMResponse struct {
	Action      string                 `json:"action"`
	ToolName    string                 `json:"tool_name"`
	Arguments   map[string]interface{} `json:"arguments"`
	Reasoning   string                 `json:"reasoning"`
	FinalOutput *FinalOutput           `json:"final_output"`
}
