package context

import (
	"aATA/internal/logic/agent"
	"fmt"
)

const (
	maxSnapshotItems     = 4
	maxToolResultItems   = 8
	maxDetailedToolItems = 4
)

// newSessionSnapshot 为当前任务初始化一个空白的运行快照。
func newSessionSnapshot(input agent.Input) Snapshot {
	return Snapshot{
		Goal:           input.Query, // 总体目标
		ConfirmedFacts: []string{},  // 从工具获取的已确认事实
		DoneItems:      []string{},  // 已完成工作
		TodoItems:      []string{},  // 需要继续关注的评估事项
		Artifacts:      []string{},  // 运行过程标记（tool name）
	}
}

// applyToolResultToSnapshot 将工具调用结果压缩进快照，供下一轮推理参考。
func applyToolResultToSnapshot(snapshot *Snapshot, patch ToolResultPatch) {
	if snapshot == nil || patch.ToolName == "" {
		return
	}

	if patch.Success {
		snapshot.DoneItems = appendLimited(snapshot.DoneItems, fmt.Sprintf("已调用工具 %s", patch.ToolName))
		snapshot.ConfirmedFacts = appendLimited(snapshot.ConfirmedFacts, fmt.Sprintf("工具 %s 已返回可用数据", patch.ToolName))
		snapshot.Artifacts = appendLimited(snapshot.Artifacts, "tool:"+patch.ToolName)
		return
	}

	// 失败了重新评估结果
	snapshot.TodoItems = appendLimited(snapshot.TodoItems, fmt.Sprintf("需要重新评估工具 %s 的调用结果", patch.ToolName))
}

// appendToolResultSummary 维护最近若干次工具结果记录。
// 最新的若干次保留完整结果，更早的结果仅保留摘要。
func appendToolResultSummary(items []ToolResultSummary, patch ToolResultPatch) []ToolResultSummary {
	if patch.ToolName == "" {
		return items
	}

	items = append(items, ToolResultSummary{
		ToolName: patch.ToolName,
		Success:  patch.Success,
		Result:   patch.Result,
		Summary:  summarizeToolResult(patch.ToolName, patch),
	})
	if len(items) > maxToolResultItems {
		items = append([]ToolResultSummary(nil), items[len(items)-maxToolResultItems:]...)
	}
	for i := 0; i < len(items)-maxDetailedToolItems; i++ {
		items[i].Result = nil
	}
	return items
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
	if len(items) <= maxSnapshotItems {
		return items
	}
	return append([]string(nil), items[len(items)-maxSnapshotItems:]...)
}
