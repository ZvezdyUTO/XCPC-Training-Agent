package runtime

import (
	"encoding/json"
	"errors"
	"strings"
)

// finalOutput 是 runtime 期待模型返回的最终 JSON 结构。
type finalOutput struct {
	DecisionType  string                 `json:"decision_type"`
	FocusStudents []string               `json:"focus_students"`
	Confidence    float64                `json:"confidence"`
	Report        string                 `json:"report"`
	Metrics       map[string]interface{} `json:"metrics"`
}

// finishOutput 负责解析并校验最终答案，同时补齐对外输出所需的默认字段。
func finishOutput(raw string) (map[string]any, error) {
	final, err := parseFinalOutput(raw)
	if err != nil {
		return nil, err
	}
	if final.DecisionType == "" {
		return nil, errors.New("缺少 decision_type")
	}
	if final.Report == "" {
		return nil, errors.New("缺少 report")
	}
	if final.FocusStudents == nil {
		final.FocusStudents = []string{}
	}
	if final.Metrics == nil {
		final.Metrics = map[string]any{}
	}
	return map[string]any{
		"decision_type":  final.DecisionType,
		"focus_students": final.FocusStudents,
		"confidence":     final.Confidence,
		"report":         final.Report,
		"metrics":        final.Metrics,
	}, nil
}

// parseFinalOutput 仅负责把模型输出解码成最终结果结构。
func parseFinalOutput(raw string) (finalOutput, error) {
	var final finalOutput
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &final); err != nil {
		return finalOutput{}, err
	}
	return final, nil
}
