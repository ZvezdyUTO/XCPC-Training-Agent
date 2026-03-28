package context

import (
	"aATA/internal/logic/agent"
	agentmodel "aATA/internal/logic/agent/model"
	"encoding/json"
	"fmt"
)

const recentConversationLimit = 6

// buildBaseMessages 构造与本次任务稳定相关的基础消息：
// system prompt、project memory、路径规则和任务输入。
func buildBaseMessages(input agent.Input, bundle Bundle) []agentmodel.Message {
	messages := []agentmodel.Message{
		{
			Role:    "system",
			Content: systemPrompt(),
		},
	}

	if bundle.Project != "" {
		messages = append(messages, agentmodel.Message{
			Role:    "system",
			Content: "Project Memory:\n" + bundle.Project,
		})
	}

	for _, rule := range bundle.Rules {
		if rule.Content == "" {
			continue
		}
		messages = append(messages, agentmodel.Message{
			Role:    "system",
			Content: fmt.Sprintf("Path Rule (%s):\n%s", rule.Name, rule.Content),
		})
	}

	inputJSON, _ := json.MarshalIndent(input, "", "  ")
	messages = append(messages, agentmodel.Message{
		Role:    "user",
		Content: fmt.Sprintf("当前任务输入如下：\n%s", string(inputJSON)),
	})

	return messages
}

// buildRequestMessages 组合本轮模型调用真正需要的消息序列。
func buildRequestMessages(baseMessages []agentmodel.Message, snapshot *sessionSnapshot, conversation []agentmodel.Message) []agentmodel.Message {
	out := make([]agentmodel.Message, 0, len(baseMessages)+1+recentConversationLimit)
	if len(baseMessages) == 0 {
		return out
	}

	out = append(out, baseMessages[:len(baseMessages)-1]...)
	out = append(out, agentmodel.Message{
		Role:    "system",
		Content: buildSnapshotMessage(snapshot),
	})
	out = append(out, baseMessages[len(baseMessages)-1])
	out = append(out, recentConversation(conversation)...)
	return out
}

// systemPrompt 定义运行时要求模型遵守的最小系统约束。
func systemPrompt() string {
	return `
你是 XCPC 集训队训练分析智能体。

规则：
1. 当信息不足时，优先调用已提供的工具获取数据。
2. 不要虚构工具、数据或结论。
3. 当证据充分时，直接输出最终结果。
4. 最终输出必须是一个合法 JSON 对象，不要输出 Markdown，不要输出额外说明。

最终 JSON 结构：
{
  "decision_type": string,
  "focus_students": string[],
  "confidence": number,
  "report": string,
  "metrics": object
}
`
}

// buildSnapshotMessage 将当前 session snapshot 序列化成 system message。
func buildSnapshotMessage(snapshot *sessionSnapshot) string {
	if snapshot == nil {
		snapshot = &sessionSnapshot{}
	}
	body, _ := json.MarshalIndent(snapshot, "", "  ")
	return "Session Snapshot:\n" + string(body)
}

// recentConversation 截取最近若干条对话，避免上下文无限增长。
func recentConversation(messages []agentmodel.Message) []agentmodel.Message {
	if len(messages) <= recentConversationLimit {
		return append([]agentmodel.Message(nil), messages...)
	}
	return append([]agentmodel.Message(nil), messages[len(messages)-recentConversationLimit:]...)
}
