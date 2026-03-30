package context

import (
	"aATA/internal/logic/agent"
)

const (
	maxToolSummaryItems = 6
	maxToolResultItems  = 2
)

// newSessionSnapshot 为当前任务初始化一个空白的运行快照。
func newSessionSnapshot(input agent.Input) Snapshot {
	return Snapshot{
		Goal:          input.Query,
		PlanState:     PlanState{},
		ToolSummaries: []ToolSummary{},
	}
}

// applyToolResultToSnapshot 将工具调用结果压缩进快照，并同步推进计划状态。
// 快照只保留历史摘要；近期的全量工具结果由 State.ToolResults 单独维护。
func applyToolResultToSnapshot(snapshot *Snapshot, patch ToolResultPatch) {
	if snapshot == nil || patch.ToolName == "" {
		return
	}

	snapshot.ToolSummaries = appendToolSummary(snapshot.ToolSummaries, patch)
	advancePlanAfterToolResult(&snapshot.PlanState, patch.Success)
}

// appendToolResultSummary 维护最近若干次工具结果记录。
// 当前策略只保留最近 2 次全量结果，避免动态状态无界膨胀。
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
	return items
}

// appendToolSummary 把工具结果摘要压入快照，供后续轮次快速复用。
func appendToolSummary(items []ToolSummary, patch ToolResultPatch) []ToolSummary {
	if patch.ToolName == "" {
		return items
	}

	items = append(items, ToolSummary{
		ToolName: patch.ToolName,
		Success:  patch.Success,
		Summary:  summarizeToolResult(patch.ToolName, patch),
	})
	if len(items) > maxToolSummaryItems {
		items = append([]ToolSummary(nil), items[len(items)-maxToolSummaryItems:]...)
	}
	return items
}
