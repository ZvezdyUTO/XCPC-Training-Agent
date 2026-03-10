package agent

import (
	"encoding/json"
	"fmt"
	"strings"
)

// buildPrompt 组装发送给 LLM 的话
func buildPrompt(state AgentState, registry *Registry) string {

	inputJSON, _ := json.MarshalIndent(state.Input, "", "  ")

	var toolResultText string
	for _, tr := range state.ToolResults {
		resultJSON, _ := json.MarshalIndent(tr.Result, "", "  ")
		toolResultText += "\n工具返回(" + tr.ToolName + "):\n" + string(resultJSON) + "\n"
	}

	// 自动生成工具说明
	var toolDescBuilder strings.Builder

	toolDescBuilder.WriteString("你只能使用以下工具：\n\n")

	for _, t := range registry.List() {

		schema := t.Schema()

		paramsJSON, _ := json.MarshalIndent(schema.Parameters, "", "  ")
		requiredJSON, _ := json.MarshalIndent(schema.Required, "", "  ")

		toolDescBuilder.WriteString(fmt.Sprintf(
			"Name: %s\nDescription: %s\nParameters:\n%s\nRequired:\n%s\n\n",
			t.Name(),
			t.Description(),
			string(paramsJSON),
			string(requiredJSON),
		))
	}

	return fmt.Sprintf(`
你是一个严格的 JSON 协议执行引擎，同时也是一个智能体训练分析师。
你不是对话助手。

%s

你必须始终输出一个合法 JSON 对象。
禁止输出 Markdown。
禁止输出 JSON 之外的任何文本。

输出格式如下（必须完全匹配）：

{
  "action": "call_tool" 或 "finish",
  "tool_name": string,
  "arguments": object,
  "reasoning": string,
  "final_output": object 或 null
}

规则：

1. 如果 action="call_tool"：
   - tool_name 必须是已提供工具之一
   - arguments 必须符合工具参数结构
   - final_output 必须为 null

2. 如果 action="finish"：
   - tool_name 必须为空字符串 ""
   - arguments 必须是 {}
   - final_output 必须是：

     {
       "decision_type": string,
       "focus_students": string[],
       "confidence": number,   // 0~1
       "report": string,
       "metrics": object
     }

   - 即使为空，也必须输出：
       focus_students: []
       metrics: {}

3. 所有对人类的完整分析必须写入 final_output.report。
4. reasoning 只写当前决策逻辑，不要写完整分析。
5. 不允许输出任何额外字段。

下面给你一个简短示例：
示例 1：调用工具
{
  "action": "call_tool",
  "tool_name": "training_summary_range",
  "arguments": {
    "student_id": "240511307",
    "from": "2026-02-01",
    "to": "2026-03-01"
  },
  "reasoning": "需要获取该学生近一个月训练数据",
  "final_output": null
}
示例 2：结束任务
{
  "action": "finish",
  "tool_name": "",
  "arguments": {},
  "reasoning": "数据已足够，生成最终报告",
  "final_output": {
    "decision_type": "student_diagnosis",
    "focus_students": [],
    "confidence": 0.82,
    "report": "完整自然语言分析...",
    "metrics": {}
  }
}

以上是你的背景信息，下面给出你当前任务：

当前输入：
%s

历史工具结果：
%s
`,
		toolDescBuilder.String(),
		string(inputJSON),
		toolResultText,
	)
}
