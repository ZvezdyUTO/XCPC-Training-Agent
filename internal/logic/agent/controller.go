/*
事件循环核心
Run(ctx, AgentInput) (FinalOutput, Trace, error)

循环：
build prompt
llm.complete
parse + validate
if call_tool → registry.call → append tool result → continue
if finish → return

Trace 由旁路系统负责记录事件流，controller 只发出关键运行事件
*/

package agent

import (
	"aATA/internal/llm"
	"aATA/internal/logic/agenttrace"
	"context"
	"encoding/json"
	"errors"
	"strings"
)

type Controller struct {
	LLM      llm.Client
	Registry *Registry
	Trace    agenttrace.Sink
}

func NewController(
	llmClient llm.Client,
	registry *Registry,
	traceSink agenttrace.Sink,
) *Controller {
	if traceSink == nil {
		traceSink = agenttrace.NoopSink{}
	}

	return &Controller{
		LLM:      llmClient,
		Registry: registry,
		Trace:    traceSink,
	}
}

// Run 核心事件循环逻辑
func (c *Controller) Run(ctx context.Context, input AgentInput) (map[string]interface{}, agenttrace.RunTrace, error) {

	state := AgentState{
		Input: input,
	}

	runStartedID := c.Trace.Record(0, agenttrace.EventRunStarted, "", map[string]any{
		"query":  input.Query,
		"params": input.Params,
	})

	toolNames := make([]string, 0, len(c.Registry.List()))
	for _, tool := range c.Registry.List() {
		toolNames = append(toolNames, tool.Name())
	}
	lastEventID := c.Trace.Record(0, agenttrace.EventToolsRegistered, runStartedID, map[string]any{
		"tool_names": toolNames,
		"tool_count": len(toolNames),
	})

	// 防止无限循环
	for state.Step = 0; state.Step < 10; state.Step++ {

		prompt := buildPrompt(state, c.Registry) //注入提示词
		modelCalledID := c.Trace.Record(state.Step, agenttrace.EventModelCalled, lastEventID, map[string]any{
			"model_name":              modelName(c.LLM),
			"prompt":                  prompt,
			"prompt_length":           len(prompt),
			"history_tool_result_cnt": len(state.ToolResults),
		})

		raw, err := c.LLM.Complete(ctx, prompt) // 调用 LLM

		if err != nil {
			c.Trace.Record(state.Step, agenttrace.EventRunFailed, modelCalledID, map[string]any{
				"stage": "model_called",
				"error": err.Error(),
			})
			return nil, c.Trace.Result(), err
		}

		var resp LLMResponse

		parsedSummary := map[string]any{}
		cleaned := cleanLLMOutput(raw)
		if err := json.Unmarshal([]byte(cleaned), &resp); err != nil {
			// 第一次失败，要求 LLM 重新输出纯 JSON
			repairPrompt := prompt + "\nYour previous output was invalid JSON. Please output strictly valid JSON only."
			c.Trace.Record(state.Step, agenttrace.EventModelReturned, modelCalledID, map[string]any{
				"raw":         raw,
				"cleaned":     cleaned,
				"parse_ok":    false,
				"repairing":   true,
				"parse_error": err.Error(),
			})

			repairModelCalledID := c.Trace.Record(state.Step, agenttrace.EventModelCalled, modelCalledID, map[string]any{
				"model_name":              modelName(c.LLM),
				"prompt":                  repairPrompt,
				"prompt_length":           len(repairPrompt),
				"history_tool_result_cnt": len(state.ToolResults),
				"repair_attempt":          true,
			})

			raw2, err2 := c.LLM.Complete(ctx, repairPrompt)
			if err2 != nil {
				c.Trace.Record(state.Step, agenttrace.EventRunFailed, repairModelCalledID, map[string]any{
					"stage": "model_repair_called",
					"error": err2.Error(),
				})
				return nil, c.Trace.Result(), err2
			}

			// 第二次失败直接返回
			cleaned2 := cleanLLMOutput(raw2)
			if err := json.Unmarshal([]byte(cleaned2), &resp); err != nil {
				c.Trace.Record(state.Step, agenttrace.EventModelReturned, repairModelCalledID, map[string]any{
					"raw":         raw2,
					"cleaned":     cleaned2,
					"parse_ok":    false,
					"parse_error": err.Error(),
				})
				finalErr := errors.New("LLM produced invalid JSON twice")
				c.Trace.Record(state.Step, agenttrace.EventRunFailed, repairModelCalledID, map[string]any{
					"stage": "model_returned",
					"error": finalErr.Error(),
				})
				return nil, c.Trace.Result(), finalErr
			}
			parsedSummary = llmResponseSummary(&resp)
			lastEventID = c.Trace.Record(state.Step, agenttrace.EventModelReturned, repairModelCalledID, map[string]any{
				"raw":      raw2,
				"cleaned":  cleaned2,
				"parse_ok": true,
				"response": parsedSummary,
				"repaired": true,
			})
		} else {
			parsedSummary = llmResponseSummary(&resp)
			lastEventID = c.Trace.Record(state.Step, agenttrace.EventModelReturned, modelCalledID, map[string]any{
				"raw":      raw,
				"cleaned":  cleaned,
				"parse_ok": true,
				"response": parsedSummary,
			})
		}

		// Agent 只允许 call_tool 和 finish，若输出其它内容直接报错
		if resp.Action != "call_tool" && resp.Action != "finish" {
			err := errors.New("invalid action")
			c.Trace.Record(state.Step, agenttrace.EventRunFailed, lastEventID, map[string]any{
				"stage":  "validate_action",
				"error":  err.Error(),
				"action": resp.Action,
			})
			return nil, c.Trace.Result(), err
		}

		if resp.Action == "call_tool" {

			// 执行工具
			toolCalledID := c.Trace.Record(state.Step, agenttrace.EventToolCalled, lastEventID, map[string]any{
				"tool_name": resp.ToolName,
				"arguments": resp.Arguments,
			})
			rawArgs, _ := json.Marshal(resp.Arguments)
			result, err := c.Registry.Call(ctx, resp.ToolName, rawArgs)
			if err != nil { // 若工具调用失败，将错误注入到 state 中，交给 LLM 自己修正
				c.Trace.Record(state.Step, agenttrace.EventToolReturned, toolCalledID, map[string]any{
					"tool_name": resp.ToolName,
					"status":    "error",
					"error":     err.Error(),
				})
				state.ToolResults = append(state.ToolResults, ToolResult{
					ToolName: resp.ToolName,
					Result:   map[string]string{"error": err.Error()},
				})
				lastEventID = toolCalledID
				continue
			}

			lastEventID = c.Trace.Record(state.Step, agenttrace.EventToolReturned, toolCalledID, map[string]any{
				"tool_name": resp.ToolName,
				"status":    "success",
				"result":    result,
			})
			state.ToolResults = append(state.ToolResults, ToolResult{
				ToolName: resp.ToolName,
				Result:   result,
			})
			continue
		}

		if resp.FinalOutput == nil {
			err := errors.New("missing final_output")
			c.Trace.Record(state.Step, agenttrace.EventRunFailed, lastEventID, map[string]any{
				"stage": "validate_final_output",
				"error": err.Error(),
			})
			return nil, c.Trace.Result(), err
		}

		// 终止机制，事件循环调用完成
		if resp.Action == "finish" {
			if resp.FinalOutput == nil {
				err := errors.New("missing final_output")
				c.Trace.Record(state.Step, agenttrace.EventRunFailed, lastEventID, map[string]any{
					"stage": "finish_validate",
					"error": err.Error(),
				})
				return nil, c.Trace.Result(), err
			}

			if resp.FinalOutput.DecisionType == "" {
				err := errors.New("missing decision_type")
				c.Trace.Record(state.Step, agenttrace.EventRunFailed, lastEventID, map[string]any{
					"stage": "finish_validate",
					"error": err.Error(),
				})
				return nil, c.Trace.Result(), err
			}

			if resp.FinalOutput.Report == "" {
				err := errors.New("missing report")
				c.Trace.Record(state.Step, agenttrace.EventRunFailed, lastEventID, map[string]any{
					"stage": "finish_validate",
					"error": err.Error(),
				})
				return nil, c.Trace.Result(), err
			}

			final := structToMap(resp.FinalOutput)
			c.Trace.Record(state.Step, agenttrace.EventRunFinished, lastEventID, map[string]any{
				"final_output": final,
			})
			return final, c.Trace.Result(), nil
		}
	}

	err := errors.New("max steps reached")
	c.Trace.Record(state.Step, agenttrace.EventRunFailed, lastEventID, map[string]any{
		"stage": "loop_guard",
		"error": err.Error(),
	})
	return nil, c.Trace.Result(), err
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

func modelName(client llm.Client) string {
	descriptor, ok := client.(llm.Descriptor)
	if !ok {
		return "unknown"
	}
	return descriptor.ModelName()
}

func llmResponseSummary(resp *LLMResponse) map[string]any {
	if resp == nil {
		return map[string]any{}
	}

	summary := map[string]any{
		"action":    resp.Action,
		"tool_name": resp.ToolName,
		"reasoning": resp.Reasoning,
	}

	if resp.FinalOutput != nil {
		summary["final_output"] = map[string]any{
			"decision_type":      resp.FinalOutput.DecisionType,
			"focus_students_cnt": len(resp.FinalOutput.FocusStudents),
			"metrics_cnt":        len(resp.FinalOutput.Metrics),
			"report_length":      len(resp.FinalOutput.Report),
			"confidence":         resp.FinalOutput.Confidence,
		}
	}

	if len(resp.Arguments) > 0 {
		summary["arguments"] = resp.Arguments
	}

	return summary
}
