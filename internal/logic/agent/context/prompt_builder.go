package context

import (
	"aATA/internal/logic/agent"
	agentllm "aATA/internal/logic/agent/llm"
	"encoding/json"
	"fmt"
	"time"
)

const recentConversationLimit = 2

// buildBaseMessages 构造与本次任务稳定相关的基础消息：
// system prompt、project memory、路径规则和任务输入。
func buildBaseMessages(input agent.Input, bundle Bundle) []agentllm.Message {
	messages := []agentllm.Message{
		{
			Role:    "system",
			Content: systemPrompt(),
		},
	}

	if bundle.Project != "" {
		messages = append(messages, agentllm.Message{
			Role:    "system",
			Content: "Project Memory:\n" + bundle.Project,
		})
	}

	for _, rule := range bundle.Rules {
		if rule.Content == "" {
			continue
		}
		messages = append(messages, agentllm.Message{
			Role:    "system",
			Content: fmt.Sprintf("Path Rule (%s):\n%s", rule.Name, rule.Content),
		})
	}

	inputJSON, _ := json.MarshalIndent(input, "", "  ")
	messages = append(messages, agentllm.Message{
		Role:    "user",
		Content: fmt.Sprintf("当前任务输入如下：\n%s", string(inputJSON)),
	})

	return messages
}

// systemPrompt 定义运行时要求模型遵守的最小系统约束。
func systemPrompt() string {
	return fmt.Sprintf(`
你是 XCPC 集训队训练分析智能体。
今天日期：%s

规则：
1. 所有分析必须建立在已获取的数据之上；当信息不足时，优先调用已提供的工具获取数据。
2. 如果当前 plan_state 尚未初始化，你必须先输出一个 PLAN_STATE，然后在同一轮为第一个 running 步骤发起 tool_calls；第一个 running 步骤必须是数据查询步骤。
3. 如果当前 plan_state 已初始化，后续轮次默认沿当前计划继续，不要重复输出完整计划。
4. 如需调整计划，只能输出一个 PLAN_UPDATE；它只允许局部修改，不允许重写整份计划。
5. 如果当前 plan_state 没有 running 步骤且任务尚未完成，你必须先提交 PLAN_UPDATE 修复计划，再继续执行。
6. tool_results 和 tool message 是事实来源；如果它们与 plan_state 冲突，以事实为准。
7. 不要虚构工具、数据或结论。
8. 最终输出必须是一个合法 JSON 对象，不要输出 Markdown，不要输出额外说明，不要使用代码块包裹 JSON。
9. 最终结果不能过度简写。overall_summary 需要概括整体状态，report 需要写成至少 3 段的完整分析，覆盖整体趋势、重点学生和训练方向判断。
10. key_findings 应尽量给出 2 到 5 条具体结论，不要只写空泛短语。

PLAN_STATE 格式：
PLAN_STATE:
{
  "current_step": 1,
  "steps": [
    {"index": 1, "title": "查询训练记录", "status": "running"},
    {"index": 2, "title": "查询比赛记录", "status": "waiting"}
  ]
}

PLAN_UPDATE 格式：
PLAN_UPDATE:
{
  "action": "append | insert_after_current | drop",
  "target_step": 2,
  "title": "补查某项数据"
}

最终 JSON 结构：
{
  "decision_type": string,
  "focus_students": string[],
  "confidence": number,
  "overall_summary": string,
  "report": string,
  "key_findings": string[],
  "metrics": object
}
`, time.Now().Format("2006-01-02"))
}

// buildContextStateMessage 将当前上下文状态序列化成 system message。
func buildContextStateMessage(state *State) string {
	body, _ := json.MarshalIndent(map[string]any{
		"snapshot": map[string]any{
			"goal":           state.Snapshot.Goal,
			"tool_summaries": state.Snapshot.ToolSummaries,
		},
		"plan_state":   state.Snapshot.PlanState,
		"tool_results": state.ToolResults,
	}, "", "  ")
	return "Session Context:\n" + string(body)
}

// recentConversation 截取最近若干条对话，避免上下文无限增长。
func recentConversation(messages []agentllm.Message) []agentllm.Message {
	if len(messages) <= recentConversationLimit {
		return append([]agentllm.Message(nil), messages...)
	}
	return append([]agentllm.Message(nil), messages[len(messages)-recentConversationLimit:]...)
}
