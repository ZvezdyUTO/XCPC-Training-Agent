package context

import (
	"aATA/internal/logic/agent"
	"fmt"
)

const maxSessionItems = 8

// sessionSnapshot 是注入给模型的轻量运行状态摘要。
// 它只保留对下一轮推理有帮助的信息，不试图完整复刻历史。
type sessionSnapshot struct {
	Goal           string   `json:"goal"`
	ConfirmedFacts []string `json:"confirmed_facts"`
	DoneItems      []string `json:"done_items"`
	TodoItems      []string `json:"todo_items"`
	Artifacts      []string `json:"artifacts"`
}

// newSessionSnapshot 为当前任务初始化一个空白的运行快照。
func newSessionSnapshot(input agent.Input) *sessionSnapshot {
	return &sessionSnapshot{
		Goal:           input.Query,
		ConfirmedFacts: []string{},
		DoneItems:      []string{},
		TodoItems:      []string{},
		Artifacts:      []string{},
	}
}

// recordToolResult 将工具调用结果压缩进快照，供下一轮推理参考。
func (s *sessionSnapshot) recordToolResult(toolName string, success bool) {
	if toolName == "" {
		return
	}

	if success {
		s.DoneItems = appendLimited(s.DoneItems, fmt.Sprintf("已调用工具 %s", toolName))
		s.ConfirmedFacts = appendLimited(s.ConfirmedFacts, fmt.Sprintf("工具 %s 已返回可用数据", toolName))
		s.Artifacts = appendLimited(s.Artifacts, "tool:"+toolName)
		return
	}

	s.TodoItems = appendLimited(s.TodoItems, fmt.Sprintf("需要重新评估工具 %s 的调用结果", toolName))
}

// appendLimited 维护一个去重且长度有上限的字符串列表。
func appendLimited(items []string, value string) []string {
	if value == "" {
		return items
	}
	for _, item := range items {
		if item == value {
			return items
		}
	}

	items = append(items, value)
	if len(items) <= maxSessionItems {
		return items
	}
	return append([]string(nil), items[len(items)-maxSessionItems:]...)
}
