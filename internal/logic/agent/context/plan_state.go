package context

import (
	agentllm "aATA/internal/logic/agent/llm"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	planLabelState  = "PLAN_STATE:"
	planLabelUpdate = "PLAN_UPDATE:"

	planStatusWaiting = "waiting"
	planStatusRunning = "running"
	planStatusDone    = "done"
	planStatusFailed  = "failed"

	planUpdateAppend             = "append"
	planUpdateInsertAfterCurrent = "insert_after_current"
	planUpdateDrop               = "drop"

	maxPlanSteps = 6
)

type planUpdate struct {
	Action     string `json:"action"`
	TargetStep int    `json:"target_step"`
	Title      string `json:"title"`
}

// applyAssistantTurnToSnapshot 解析 assistant 返回中的计划协议块，并把合法结果写入快照。
// 该函数只处理 PLAN_STATE / PLAN_UPDATE，不负责工具推进或最终输出校验。
func applyAssistantTurnToSnapshot(snapshot *Snapshot, message agentllm.Message) AssistantTurnOutcome {
	outcome := AssistantTurnOutcome{Message: message}
	if snapshot == nil {
		return outcome
	}

	content := strings.TrimSpace(message.Content)
	if content == "" {
		return outcome
	}

	planState, hasPlanState, err := parsePlanState(content)
	if err != nil && hasPlanState {
		outcome.Error = err
		return outcome
	}
	if err == nil && hasPlanState {
		snapshot.PlanState = planState
		outcome.HasPlanDirective = true
		outcome.Message.Content = ""
		return outcome
	}

	update, hasPlanUpdate, err := parsePlanUpdate(content)
	if err != nil && hasPlanUpdate {
		outcome.Error = err
		return outcome
	}
	if err == nil && hasPlanUpdate {
		mergePlanUpdate(&snapshot.PlanState, update)
		outcome.HasPlanDirective = true
		outcome.Message.Content = ""
		return outcome
	}

	return outcome
}

// advancePlanAfterToolResult 根据真实工具执行结果推进当前计划状态。
// 成功时当前 running 步骤会完成并推进到下一个 waiting；失败时当前步骤直接标记 failed。
func advancePlanAfterToolResult(plan *PlanState, success bool) {
	if plan == nil || !plan.Initialized || len(plan.Steps) == 0 {
		return
	}

	currentIndex := findRunningStep(plan)
	if currentIndex < 0 {
		return
	}

	if success {
		plan.Steps[currentIndex].Status = planStatusDone
		nextIndex := findNextWaiting(plan, currentIndex+1)
		if nextIndex >= 0 {
			plan.Steps[nextIndex].Status = planStatusRunning
			plan.CurrentStep = plan.Steps[nextIndex].Index
			return
		}

		plan.CurrentStep = 0
		return
	}

	plan.Steps[currentIndex].Status = planStatusFailed
	plan.CurrentStep = 0
}

// preparePlanForFinalization 在最终 JSON 收尾前判断计划是否已进入可收尾状态。
// 这里不修改计划本身，只负责告诉 runtime 是否应该进入最后一轮 JSON 输出。
func preparePlanForFinalization(plan *PlanState) bool {
	if plan == nil || !plan.Initialized {
		return false
	}

	runningIndex := findRunningStep(plan)
	waitingIndex := findNextWaiting(plan, 0)

	switch {
	case runningIndex >= 0 && waitingIndex < 0:
		return true
	case runningIndex < 0 && waitingIndex < 0:
		return true
	default:
		return false
	}
}

// completePlanAfterFinalization 在最终 JSON 输出成功后提交最后一步计划状态。
// 若当前仍有一个 running 收尾步骤，则将其标记为 done。
func completePlanAfterFinalization(plan *PlanState) {
	if plan == nil || !plan.Initialized {
		return
	}
	runningIndex := findRunningStep(plan)
	if runningIndex < 0 {
		return
	}
	plan.Steps[runningIndex].Status = planStatusDone
	plan.CurrentStep = 0
}

// completePlanAfterDirectOutput 在中间轮直接接受最终 JSON 时强制结束计划。
// 这里把当前 running 和剩余 waiting 一并标记为 done，表示本次计划已被最终答案提前收束。
func completePlanAfterDirectOutput(plan *PlanState) {
	if plan == nil || !plan.Initialized {
		return
	}

	for i := range plan.Steps {
		switch plan.Steps[i].Status {
		case planStatusRunning, planStatusWaiting:
			plan.Steps[i].Status = planStatusDone
		}
	}
	plan.CurrentStep = 0
}

func parsePlanState(raw string) (PlanState, bool, error) {
	jsonBody, ok := extractLabeledJSON(raw, planLabelState)
	if !ok {
		return PlanState{}, false, nil
	}

	var plan PlanState
	if err := json.Unmarshal([]byte(jsonBody), &plan); err != nil {
		return PlanState{}, true, err
	}
	normalized, err := normalizePlanState(plan)
	if err != nil {
		return PlanState{}, true, err
	}
	return normalized, true, nil
}

func parsePlanUpdate(raw string) (planUpdate, bool, error) {
	jsonBody, ok := extractLabeledJSON(raw, planLabelUpdate)
	if !ok {
		return planUpdate{}, false, nil
	}

	var update planUpdate
	if err := json.Unmarshal([]byte(jsonBody), &update); err != nil {
		return planUpdate{}, true, err
	}
	update.Action = strings.TrimSpace(update.Action)
	update.Title = strings.TrimSpace(update.Title)
	if err := validatePlanUpdate(update); err != nil {
		return planUpdate{}, true, err
	}
	return update, true, nil
}

func extractLabeledJSON(raw, label string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, label) {
		return "", false
	}
	return strings.TrimSpace(strings.TrimPrefix(trimmed, label)), true
}

func normalizePlanState(input PlanState) (PlanState, error) {
	if len(input.Steps) == 0 {
		return PlanState{}, fmt.Errorf("PLAN_STATE 缺少 steps")
	}
	if len(input.Steps) > maxPlanSteps {
		return PlanState{}, fmt.Errorf("PLAN_STATE steps 超过上限")
	}

	runningCount := 0
	currentStep := 0
	normalized := PlanState{
		Initialized: true,
		Version:     1,
		Steps:       make([]PlanStep, 0, len(input.Steps)),
	}

	for i, step := range input.Steps {
		title := strings.TrimSpace(step.Title)
		if title == "" {
			return PlanState{}, fmt.Errorf("PLAN_STATE 存在空标题步骤")
		}
		status := strings.TrimSpace(step.Status)
		if !isValidPlanStatus(status) {
			return PlanState{}, fmt.Errorf("PLAN_STATE 存在非法步骤状态")
		}
		index := i + 1
		normalized.Steps = append(normalized.Steps, PlanStep{
			Index:  index,
			Title:  title,
			Status: status,
		})
		if status == planStatusRunning {
			runningCount++
			currentStep = index
		}
	}

	if runningCount != 1 {
		return PlanState{}, fmt.Errorf("PLAN_STATE 必须且只能有一个 running 步骤")
	}
	if input.CurrentStep != currentStep {
		return PlanState{}, fmt.Errorf("PLAN_STATE current_step 与 running 步骤不一致")
	}

	normalized.CurrentStep = currentStep
	return normalized, nil
}

func validatePlanUpdate(update planUpdate) error {
	switch update.Action {
	case planUpdateAppend, planUpdateInsertAfterCurrent:
		if update.Title == "" {
			return fmt.Errorf("PLAN_UPDATE 缺少 title")
		}
	case planUpdateDrop:
		if update.TargetStep <= 0 {
			return fmt.Errorf("PLAN_UPDATE drop 缺少 target_step")
		}
	default:
		return fmt.Errorf("PLAN_UPDATE action 非法")
	}
	return nil
}

func mergePlanUpdate(plan *PlanState, update planUpdate) {
	if plan == nil || !plan.Initialized {
		return
	}

	switch update.Action {
	case planUpdateAppend:
		if len(plan.Steps) >= maxPlanSteps {
			return
		}
		status := planStatusWaiting
		if findRunningStep(plan) < 0 {
			status = planStatusRunning
		}
		plan.Steps = append(plan.Steps, PlanStep{
			Title:  update.Title,
			Status: status,
		})
	case planUpdateInsertAfterCurrent:
		if len(plan.Steps) >= maxPlanSteps {
			return
		}
		insertPos := len(plan.Steps)
		if runningIndex := findRunningStep(plan); runningIndex >= 0 {
			insertPos = runningIndex + 1
		}
		status := planStatusWaiting
		if findRunningStep(plan) < 0 {
			status = planStatusRunning
		}
		step := PlanStep{
			Title:  update.Title,
			Status: status,
		}
		plan.Steps = append(plan.Steps, PlanStep{})
		copy(plan.Steps[insertPos+1:], plan.Steps[insertPos:])
		plan.Steps[insertPos] = step
	case planUpdateDrop:
		dropIndex := update.TargetStep - 1
		if dropIndex < 0 || dropIndex >= len(plan.Steps) {
			return
		}
		if plan.Steps[dropIndex].Status == planStatusRunning {
			return
		}
		plan.Steps = append(plan.Steps[:dropIndex], plan.Steps[dropIndex+1:]...)
	}

	normalizePlanIndexes(plan)
}

func normalizePlanIndexes(plan *PlanState) {
	if plan == nil {
		return
	}

	currentStep := 0
	for i := range plan.Steps {
		plan.Steps[i].Index = i + 1
		if plan.Steps[i].Status == planStatusRunning {
			currentStep = plan.Steps[i].Index
		}
	}
	plan.CurrentStep = currentStep
	plan.Version++
}

func findRunningStep(plan *PlanState) int {
	for i, step := range plan.Steps {
		if step.Status == planStatusRunning {
			return i
		}
	}
	return -1
}

func findNextWaiting(plan *PlanState, from int) int {
	for i := from; i < len(plan.Steps); i++ {
		if plan.Steps[i].Status == planStatusWaiting {
			return i
		}
	}
	return -1
}

func isValidPlanStatus(status string) bool {
	switch status {
	case planStatusWaiting, planStatusRunning, planStatusDone, planStatusFailed:
		return true
	default:
		return false
	}
}
