/*
事件循环核心
Run(ctx, AgentInput) (FinalOutput, Trace, error)

循环：
build prompt
llm.complete
parse + validate
if call_tool → registry.call → append tool result → continue
if finish → return

控制：
maxSteps=5
maxToolCallsPerTool（例如每个工具最多 2 次）

Trace 里记录：
每一步 reasoning
调了哪个工具
工具结果摘要
*/

package agent

import (
	"aATA/internal/llm"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type Controller struct {
	LLM      llm.Client
	Registry *Registry
}

func NewController(
	llmClient llm.Client,
	registry *Registry,
) *Controller {

	return &Controller{
		LLM:      llmClient,
		Registry: registry,
	}
}

// Run 核心事件循环逻辑
func (c *Controller) Run(ctx context.Context, input AgentInput) (map[string]interface{}, []string, error) {

	state := AgentState{
		Input: input,
	}

	// 防止无限循环
	for state.Step = 0; state.Step < 10; state.Step++ {

		prompt := buildPrompt(state, c.Registry) //注入提示词
		fmt.Println("发送：", prompt)
		raw, err := c.LLM.Complete(ctx, prompt) // 调用 LLM
		if err != nil {
			return nil, state.ReasoningLog, err
		}
		fmt.Println("接收：", raw)

		var resp LLMResponse

		cleaned := cleanLLMOutput(raw)
		if err := json.Unmarshal([]byte(cleaned), &resp); err != nil {
			// 第一次失败，要求 LLM 重新输出纯 JSON
			repairPrompt := prompt + "\nYour previous output was invalid JSON. Please output strictly valid JSON only."
			raw2, err2 := c.LLM.Complete(ctx, repairPrompt)
			if err2 != nil {
				return nil, state.ReasoningLog, err2
			}

			// 第二次失败直接返回
			cleaned2 := cleanLLMOutput(raw2)
			if err := json.Unmarshal([]byte(cleaned2), &resp); err != nil {
				fmt.Println("第二次绑定出现问题，问题为", err)
				return nil, state.ReasoningLog, errors.New("LLM produced invalid JSON twice")
			}
		}

		// Agent 只允许 call_tool 和 finish，若输出其它内容直接报错
		if resp.Action != "call_tool" && resp.Action != "finish" {
			return nil, state.ReasoningLog, errors.New("invalid action")
		}

		// 记录思维轨迹
		state.ReasoningLog = append(
			state.ReasoningLog,
			fmt.Sprintf("Step %d: %s", state.Step, resp.Reasoning),
		)

		if resp.Action == "call_tool" {

			// 执行工具
			fmt.Println("工具名称：", resp.ToolName)
			rawArgs, _ := json.Marshal(resp.Arguments)
			result, err := c.Registry.Call(ctx, resp.ToolName, rawArgs)
			if err != nil { // 若工具调用失败，将错误注入到 state 中，交给 LLM 自己修正
				fmt.Println("工具调用失败！", err)
				state.ToolResults = append(state.ToolResults, ToolResult{
					ToolName: resp.ToolName,
					Result:   map[string]string{"error": err.Error()},
				})
				continue
			}

			fmt.Println("工具调用成功", result)
			state.ToolResults = append(state.ToolResults, ToolResult{
				ToolName: resp.ToolName,
				Result:   result,
			})
			continue
		}

		if resp.FinalOutput == nil {
			return nil, state.ReasoningLog, errors.New("missing final_output")
		}

		// 终止机制，事件循环调用完成
		if resp.Action == "finish" {
			if resp.FinalOutput == nil {
				return nil, state.ReasoningLog, errors.New("missing final_output")
			}

			if resp.FinalOutput.DecisionType == "" {
				return nil, state.ReasoningLog, errors.New("missing decision_type")
			}

			if resp.FinalOutput.Report == "" {
				return nil, state.ReasoningLog, errors.New("missing report")
			}

			return structToMap(resp.FinalOutput), state.ReasoningLog, nil
		}
	}

	return nil, state.ReasoningLog, errors.New("max steps reached")
}

// cleanLLMOutput 清洗 LLM 回复
func cleanLLMOutput(raw string) string {
	raw = strings.TrimSpace(raw)

	// 如果是 ```json 包装
	if strings.HasPrefix(raw, "```") {
		lines := strings.Split(raw, "\n")
		if len(lines) >= 3 {
			raw = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	raw = strings.TrimSpace(raw)

	// 截取第一个 { 到最后一个 }
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		raw = raw[start : end+1]
	}

	return raw
}

// structToMap 将回复格式转为 map
func structToMap(f *FinalOutput) map[string]interface{} {
	return map[string]interface{}{
		"decision_type":  f.DecisionType,
		"focus_students": f.FocusStudents,
		"confidence":     f.Confidence,
		"report":         f.Report,
		"metrics":        f.Metrics,
	}
}
