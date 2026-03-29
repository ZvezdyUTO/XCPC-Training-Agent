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
1. 当信息不足时，优先调用已提供的工具获取数据。
2. 不要虚构工具、数据或结论。
3. 当证据充分时，直接输出最终结果。
4. 最终输出必须是一个合法 JSON 对象，不要输出 Markdown，不要输出额外说明，不要使用代码块包裹 JSON。

最终 JSON 结构：
{
  "decision_type": string,
  "focus_students": string[],
  "confidence": number,
  "report": string,
  "metrics": object
}
`, time.Now().Format("2006-01-02"))
}

// buildContextStateMessage 将当前上下文状态序列化成 system message。
func buildContextStateMessage(state *State) string {
	body, _ := json.MarshalIndent(map[string]any{
		"snapshot":     state.Snapshot,
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
