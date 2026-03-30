package context

import (
	"aATA/internal/logic/agent"
	agentllm "aATA/internal/logic/agent/llm"
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

	manager.ApplyAssistantTurn(state, agentllm.Message{
		Role: "assistant",
		Content: `PLAN_STATE:
{
  "current_step": 1,
  "steps": [
    {"index": 1, "title": "查询训练记录", "status": "running"},
    {"index": 2, "title": "查询比赛记录", "status": "waiting"}
  ]
}`,
	})

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
	if len(state.Snapshot.ToolSummaries) != 1 {
		t.Fatalf("Snapshot.ToolSummaries len = %d, want 1", len(state.Snapshot.ToolSummaries))
	}
	if got := state.ToolResults[0].Summary["tool"]; got != "training_summary" {
		t.Fatalf("ToolResults[0].Summary[tool] = %v", got)
	}
	if got := state.ToolResults[0].Result.(map[string]any)["student_id"]; got != "1" {
		t.Fatalf("ToolResults[0].Result[student_id] = %v", got)
	}
	if state.Snapshot.PlanState.CurrentStep != 2 {
		t.Fatalf("Snapshot.PlanState.CurrentStep = %d, want 2", state.Snapshot.PlanState.CurrentStep)
	}
	if state.Snapshot.PlanState.Steps[0].Status != planStatusDone {
		t.Fatalf("first step status = %q, want done", state.Snapshot.PlanState.Steps[0].Status)
	}
	if state.Snapshot.PlanState.Steps[1].Status != planStatusRunning {
		t.Fatalf("second step status = %q, want running", state.Snapshot.PlanState.Steps[1].Status)
	}
}

// TestDefaultManagerApplyAssistantTurn 验证初始计划会被解析进快照，并从会话消息中剥离原始计划文本。
func TestDefaultManagerApplyAssistantTurn(t *testing.T) {
	manager := NewManager("")
	state, err := manager.Open(t.Context(), agent.Input{
		Query: "分析本周训练情况",
	})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	outcome := manager.ApplyAssistantTurn(state, agentllm.Message{
		Role: "assistant",
		Content: `PLAN_STATE:
{
  "current_step": 1,
  "steps": [
    {"index": 1, "title": "查询训练记录", "status": "running"},
    {"index": 2, "title": "查询比赛记录", "status": "waiting"},
    {"index": 3, "title": "整理最终结论", "status": "waiting"}
  ]
}`,
	})

	if !outcome.HasPlanDirective {
		t.Fatalf("HasPlanDirective = false, want true")
	}
	if outcome.Message.Content != "" {
		t.Fatalf("sanitized content = %q, want empty", outcome.Message.Content)
	}
	if !state.Snapshot.PlanState.Initialized {
		t.Fatalf("plan state should be initialized")
	}
	if state.Snapshot.PlanState.CurrentStep != 1 {
		t.Fatalf("current step = %d, want 1", state.Snapshot.PlanState.CurrentStep)
	}
	if len(state.Snapshot.PlanState.Steps) != 3 {
		t.Fatalf("steps len = %d, want 3", len(state.Snapshot.PlanState.Steps))
	}
}

// TestDefaultManagerPrepareFinalization 验证仅剩最后一个 running 收尾步骤时，可以进入最终 JSON 收尾阶段。
func TestDefaultManagerPrepareFinalization(t *testing.T) {
	manager := NewManager("")
	state, err := manager.Open(t.Context(), agent.Input{
		Query: "分析本周训练情况",
	})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	manager.ApplyAssistantTurn(state, agentllm.Message{
		Role: "assistant",
		Content: `PLAN_STATE:
{
  "current_step": 2,
  "steps": [
    {"index": 1, "title": "查询训练记录", "status": "done"},
    {"index": 2, "title": "整理最终结论", "status": "running"}
  ]
}`,
	})

	if !manager.PrepareFinalization(state) {
		t.Fatalf("PrepareFinalization() = false, want true")
	}
	if state.Snapshot.PlanState.CurrentStep != 2 {
		t.Fatalf("current step = %d, want 2 before completion", state.Snapshot.PlanState.CurrentStep)
	}
	if state.Snapshot.PlanState.Steps[1].Status != planStatusRunning {
		t.Fatalf("last step status = %q, want running before completion", state.Snapshot.PlanState.Steps[1].Status)
	}

	manager.CompleteFinalization(state)

	if state.Snapshot.PlanState.CurrentStep != 0 {
		t.Fatalf("current step = %d, want 0", state.Snapshot.PlanState.CurrentStep)
	}
	if state.Snapshot.PlanState.Steps[1].Status != planStatusDone {
		t.Fatalf("last step status = %q, want done", state.Snapshot.PlanState.Steps[1].Status)
	}
}
