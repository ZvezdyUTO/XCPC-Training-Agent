package context

import (
	"aATA/internal/logic/agent"
	"strings"
	"testing"
	"time"
)

// TestDefaultManagerOpenAndBuildMessages 验证上下文能初始化状态并产出带快照的消息列表。
func TestDefaultManagerOpenAndBuildMessages(t *testing.T) {
	manager := NewManager("")
	state, err := manager.Open(t.Context(), agent.Input{
		Query: "分析本周训练情况",
	})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	if state.Snapshot.Goal != "分析本周训练情况" {
		t.Fatalf("Snapshot.Goal = %q", state.Snapshot.Goal)
	}

	messages := manager.BuildMessages(state, nil)
	if len(messages) < 3 {
		t.Fatalf("messages len = %d, want >= 3", len(messages))
	}
	if messages[1].Role != "system" {
		t.Fatalf("messages[1].Role = %q, want system", messages[1].Role)
	}
	if !strings.Contains(messages[0].Content, time.Now().Format("2006-01-02")) {
		t.Fatalf("system prompt should contain current date, got %q", messages[0].Content)
	}
}

// TestDefaultManagerApplyToolResult 验证工具结果补丁会同时更新快照和工具结果列表。
func TestDefaultManagerApplyToolResult(t *testing.T) {
	manager := NewManager("")
	state, err := manager.Open(t.Context(), agent.Input{
		Query: "分析本周训练情况",
	})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	manager.ApplyToolResult(state, ToolResultPatch{
		ToolName: "training_summary",
		Success:  true,
		Result: map[string]any{
			"student_id": "1",
			"score":      100,
		},
	})

	if len(state.ToolResults) != 1 {
		t.Fatalf("ToolResults len = %d, want 1", len(state.ToolResults))
	}
	if len(state.Snapshot.DoneItems) != 1 {
		t.Fatalf("Snapshot.DoneItems len = %d, want 1", len(state.Snapshot.DoneItems))
	}
	if got := state.ToolResults[0].Summary["tool"]; got != "training_summary" {
		t.Fatalf("ToolResults[0].Summary[tool] = %v", got)
	}
	if got := state.ToolResults[0].Result.(map[string]any)["student_id"]; got != "1" {
		t.Fatalf("ToolResults[0].Result[student_id] = %v", got)
	}
}
