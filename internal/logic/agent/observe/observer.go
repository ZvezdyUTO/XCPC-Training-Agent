package observe

import (
	"aATA/internal/logic/agent"
	agentllm "aATA/internal/logic/agent/llm"
	"aATA/internal/logic/agent/tooling"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// TraceObserverFactory 把底层 Sink 包装成 runtime 可直接使用的 ObserverFactory。
type TraceObserverFactory struct {
	trace Sink
}

// NewTraceObserverFactory 创建基于给定 sink 的 observer 工厂。
func NewTraceObserverFactory(trace Sink) TraceObserverFactory {
	if trace == nil {
		trace = NoopSink{}
	}
	return TraceObserverFactory{trace: trace}
}

// New 为本次运行创建 observer，并捕获模型标识等稳定元信息。
func (f TraceObserverFactory) New(llmClient agentllm.Client, input agent.Input, toolNames []string) Observer {
	return &traceObserver{
		trace:     f.trace,
		modelName: modelName(llmClient),
	}
}

// modelAttempt 记录一次尚未闭合的模型调用尝试。
type modelAttempt struct {
	spanID string
	req    agentllm.ChatRequest
}

// toolAttempt 记录一次尚未闭合的工具调用尝试。
type toolAttempt struct {
	spanID     string
	toolName   string
	toolCallID string
}

// traceObserver 把 runtime 的高层事件翻译为 sink 可存储的 event/span。
type traceObserver struct {
	trace        Sink
	modelName    string
	modelAttempt *modelAttempt
	toolAttempt  *toolAttempt
}

// RunStarted 记录本次运行的起点和已注册工具集合。
func (o *traceObserver) RunStarted(input agent.Input, toolNames []string) {
	o.trace.Record(0, agent.EventRunStarted, "", map[string]any{
		"summary": "开始运行",
		"query":   input.Query,
		"params":  input.Params,
	})

	names := append([]string(nil), toolNames...)
	sort.Strings(names)

	o.trace.Record(0, agent.EventToolsRegistered, "", map[string]any{
		"status":     "success",
		"summary":    fmt.Sprintf("已注册 %d 个工具", len(names)),
		"tool_names": names,
		"tool_count": len(names),
	})
}

// ModelStarted 记录一次模型调用开始事件，并打开对应 span。
func (o *traceObserver) ModelStarted(step int, req agentllm.ChatRequest) {
	spanID := o.trace.StartSpan(step, agent.SpanModelCall, "", map[string]any{
		"model_name": o.modelName,
		"status":     "started",
		"summary":    "开始调用模型",
	})

	o.trace.Record(step, agent.EventModelCalled, "", map[string]any{
		"status":        "started",
		"summary":       "开始调用模型",
		"model_name":    o.modelName,
		"messages":      req.Messages,
		"message_count": len(req.Messages),
		"tool_count":    len(req.Tools),
		"context_chars": buildRequestContextSize(req),
	})

	o.modelAttempt = &modelAttempt{
		spanID: spanID,
		req:    req,
	}
}

// ModelFinished 关闭模型调用 span，并把返回内容写入 trace。
func (o *traceObserver) ModelFinished(step int, completion *agentllm.ChatCompletion, parseErr error) {
	if o.modelAttempt == nil {
		return
	}

	parseOK := parseErr == nil
	summary := buildModelReturnSummary(completion, parseErr)

	o.trace.FinishSpan(o.modelAttempt.spanID, "success", map[string]any{
		"model_name":    o.modelName,
		"summary":       summary,
		"latency_ms":    completion.LatencyMs,
		"finish_reason": completion.FinishReason,
		"parse_ok":      parseOK,
		"input_tokens":  completion.Usage.PromptTokens,
		"output_tokens": completion.Usage.CompletionTokens,
		"total_tokens":  completion.Usage.TotalTokens,
	})

	payload := map[string]any{
		"status":             "success",
		"summary":            summary,
		"model_name":         o.modelName,
		"content":            completion.Content,
		"content_preview":    preview(completion.Content, 240),
		"tool_calls":         completion.ToolCalls,
		"tool_calls_summary": buildToolCallSummary(completion.ToolCalls),
		"raw_response":       completion.RawResponse,
		"parse_ok":           parseOK,
		"latency_ms":         completion.LatencyMs,
		"finish_reason":      completion.FinishReason,
		"input_tokens":       completion.Usage.PromptTokens,
		"output_tokens":      completion.Usage.CompletionTokens,
		"total_tokens":       completion.Usage.TotalTokens,
	}
	if parseErr != nil {
		payload["parse_error"] = parseErr.Error()
	}

	o.trace.Record(step, agent.EventModelReturned, "", payload)
	o.modelAttempt = nil
}

// ToolStarted 记录一次工具调用开始事件，并打开对应 span。
func (o *traceObserver) ToolStarted(step int, name string, args string, toolCallID string) {
	spanID := o.trace.StartSpan(step, agent.SpanToolCall, "", map[string]any{
		"tool_name": name,
		"status":    "started",
		"summary":   "开始调用工具",
	})

	o.trace.Record(step, agent.EventToolCalled, "", map[string]any{
		"status":            "started",
		"summary":           "开始调用工具",
		"tool_name":         name,
		"arguments":         tryDecodeJSON(args),
		"arguments_raw":     args,
		"arguments_summary": buildArgumentSummary(args),
	})

	o.toolAttempt = &toolAttempt{
		spanID:     spanID,
		toolName:   name,
		toolCallID: toolCallID,
	}
}

// ToolFinished 关闭工具 span，并记录工具返回摘要或错误。
func (o *traceObserver) ToolFinished(step int, result tooling.CallResult, err error, latencyMs int64) {
	if o.toolAttempt == nil {
		return
	}

	status := "success"
	summary := "工具调用成功"
	resultSummary := buildToolResultSummary(result.Result)
	payload := map[string]any{
		"status":         status,
		"summary":        summary,
		"tool_name":      o.toolAttempt.toolName,
		"latency_ms":     latencyMs,
		"result_summary": resultSummary,
	}

	if err != nil {
		status = "error"
		summary = "工具调用失败"
		payload["status"] = status
		payload["summary"] = summary
		payload["error"] = err.Error()
		payload["error_code"] = "工具调用失败"
	} else {
		payload["result"] = result.Result
	}

	o.trace.FinishSpan(o.toolAttempt.spanID, status, map[string]any{
		"tool_name":      o.toolAttempt.toolName,
		"summary":        summary,
		"error":          errorString(err),
		"latency_ms":     latencyMs,
		"result_summary": resultSummary,
	})

	o.trace.Record(step, agent.EventToolReturned, "", payload)
	o.toolAttempt = nil
}

// RunFinished 记录本次运行成功完成，并把最终计划状态一并写入 trace。
func (o *traceObserver) RunFinished(step int, output map[string]any, planState any) {
	o.trace.Record(step, agent.EventRunFinished, "", map[string]any{
		"status":       "success",
		"summary":      "运行成功完成",
		"final_output": output,
		"plan_state":   planState,
	})
}

// RunFailed 记录本次运行在某个阶段失败，同时尽量关闭未完成的模型 span。
func (o *traceObserver) RunFailed(step int, stage string, err error, extra map[string]any) {
	if o.modelAttempt != nil {
		o.trace.FinishSpan(o.modelAttempt.spanID, "error", map[string]any{
			"model_name": o.modelName,
			"summary":    "模型调用失败",
			"error":      err.Error(),
		})
		o.modelAttempt = nil
	}

	payload := map[string]any{
		"status":  "error",
		"stage":   stage,
		"error":   err.Error(),
		"summary": fmt.Sprintf("运行失败，阶段：%s", stage),
	}
	for k, v := range extra {
		payload[k] = v
	}
	o.trace.Record(step, agent.EventRunFailed, "", payload)
}

// Result 导出 observer 当前持有的 trace。
func (o *traceObserver) Result() agent.RunTrace {
	return o.trace.Result()
}

// modelName 以可选方式从模型客户端中提取用于 trace 展示的模型名。
func modelName(client agentllm.Client) string {
	descriptor, ok := client.(agentllm.Descriptor)
	if !ok {
		return "未知模型"
	}
	return descriptor.ModelName()
}

// buildToolResultSummary 生成适合写入 trace 的工具结果摘要。
func buildToolResultSummary(result any) map[string]any {
	if result == nil {
		return map[string]any{"type": "nil"}
	}

	switch v := result.(type) {
	case map[string]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return map[string]any{
			"type":      "object",
			"key_count": len(keys),
			"keys":      keys,
		}
	default:
		b, _ := json.Marshal(v)
		return map[string]any{
			"type":         fmt.Sprintf("%T", result),
			"result_chars": len(b),
		}
	}
}

// buildToolCallSummary 为模型返回中的工具调用生成可读摘要。
// summary 模式只依赖这份精简信息，不再要求前端先啃完整 tool_calls。
func buildToolCallSummary(calls []agentllm.ToolCall) []map[string]any {
	if len(calls) == 0 {
		return nil
	}

	out := make([]map[string]any, 0, len(calls))
	for _, call := range calls {
		out = append(out, map[string]any{
			"tool_name": call.Function.Name,
		})
	}
	return out
}

// buildArgumentSummary 把工具参数压成适合 summary 模式展示的简单对象。
// 这里只返回结构化后的简要参数，不保留原始字符串。
func buildArgumentSummary(args string) any {
	parsed := tryDecodeJSON(args)
	if parsed == nil {
		return preview(args, 200)
	}
	return parsed
}

// buildModelReturnSummary 生成一条可读性更好的模型返回摘要。
func buildModelReturnSummary(resp *agentllm.ChatCompletion, parseErr error) string {
	if resp == nil {
		return "模型返回为空"
	}
	if len(resp.ToolCalls) > 0 {
		if len(resp.ToolCalls) == 1 {
			return fmt.Sprintf("模型决定调用工具 %s", resp.ToolCalls[0].Function.Name)
		}
		return fmt.Sprintf("模型决定并行调用 %d 个工具", len(resp.ToolCalls))
	}
	if parseErr != nil {
		return "模型返回了无法解析的最终答案"
	}
	return "模型生成了最终答案"
}

// buildRequestContextSize 粗略估算本轮请求上下文的字符规模。
func buildRequestContextSize(req agentllm.ChatRequest) int {
	size := 0
	for _, msg := range req.Messages {
		size += len(msg.Content)
		for _, call := range msg.ToolCalls {
			size += len(call.Function.Name)
			size += len(call.Function.Arguments)
		}
	}
	return size
}

// tryDecodeJSON 尝试把工具参数解码为结构化对象，失败时退回原字符串。
func tryDecodeJSON(raw string) any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}
	}

	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return raw
	}
	return v
}

// errorString 在需要字符串字段时安全提取错误内容。
func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
