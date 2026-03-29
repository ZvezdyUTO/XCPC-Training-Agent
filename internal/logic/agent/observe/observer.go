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
	spanID  string
	eventID string
	req     agentllm.ChatRequest
}

// toolAttempt 记录一次尚未闭合的工具调用尝试。
type toolAttempt struct {
	spanID     string
	eventID    string
	toolName   string
	toolCallID string
}

// traceObserver 把 runtime 的高层事件翻译为 sink 可存储的 event/span。
type traceObserver struct {
	trace        Sink
	modelName    string
	lastEventID  string
	lastSpanID   string
	modelAttempt *modelAttempt
	toolAttempt  *toolAttempt
}

// RunStarted 记录本次运行的起点和已注册工具集合。
func (o *traceObserver) RunStarted(input agent.Input, toolNames []string) {
	runStartedID := o.trace.Record(0, agent.EventRunStarted, "", map[string]any{
		"query":  input.Query,
		"params": input.Params,
	})

	names := append([]string(nil), toolNames...)
	sort.Strings(names)

	o.lastEventID = o.trace.Record(0, agent.EventToolsRegistered, runStartedID, map[string]any{
		"status":      "success",
		"entity_type": "registry",
		"entity_name": "agent_tools",
		"summary":     fmt.Sprintf("已注册 %d 个工具", len(names)),
		"tool_names":  names,
		"tool_count":  len(names),
	})
}

// ModelStarted 记录一次模型调用开始事件，并打开对应 span。
func (o *traceObserver) ModelStarted(step int, req agentllm.ChatRequest) {
	spanID := o.trace.StartSpan(step, agent.SpanModelCall, o.lastSpanID, map[string]any{
		"entity_type": "model",
		"entity_name": o.modelName,
		"status":      "started",
		"summary":     "开始调用模型",
	})

	eventID := o.trace.Record(step, agent.EventModelCalled, o.lastEventID, map[string]any{
		"status":        "started",
		"entity_type":   "model",
		"entity_name":   o.modelName,
		"summary":       "开始调用模型",
		"messages":      req.Messages,
		"message_count": len(req.Messages),
		"tool_count":    len(req.Tools),
		"context_chars": buildRequestContextSize(req),
	})

	o.modelAttempt = &modelAttempt{
		spanID:  spanID,
		eventID: eventID,
		req:     req,
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
		"entity_type":   "model",
		"entity_name":   o.modelName,
		"summary":       summary,
		"latency_ms":    completion.LatencyMs,
		"finish_reason": completion.FinishReason,
		"parse_ok":      parseOK,
		"input_tokens":  completion.Usage.PromptTokens,
		"output_tokens": completion.Usage.CompletionTokens,
		"total_tokens":  completion.Usage.TotalTokens,
	})

	payload := map[string]any{
		"status":        "success",
		"entity_type":   "model",
		"entity_name":   o.modelName,
		"summary":       summary,
		"content":       completion.Content,
		"tool_calls":    completion.ToolCalls,
		"raw_response":  completion.RawResponse,
		"parse_ok":      parseOK,
		"latency_ms":    completion.LatencyMs,
		"finish_reason": completion.FinishReason,
		"input_tokens":  completion.Usage.PromptTokens,
		"output_tokens": completion.Usage.CompletionTokens,
		"total_tokens":  completion.Usage.TotalTokens,
	}
	if parseErr != nil {
		payload["parse_error"] = parseErr.Error()
	}

	o.lastSpanID = o.modelAttempt.spanID
	o.lastEventID = o.trace.Record(step, agent.EventModelReturned, o.modelAttempt.eventID, payload)
	o.modelAttempt = nil
}

// ToolStarted 记录一次工具调用开始事件，并打开对应 span。
func (o *traceObserver) ToolStarted(step int, name string, args string, toolCallID string) {
	spanID := o.trace.StartSpan(step, agent.SpanToolCall, o.lastSpanID, map[string]any{
		"entity_type": "tool",
		"entity_name": name,
		"status":      "started",
		"summary":     "开始调用工具",
	})

	eventID := o.trace.Record(step, agent.EventToolCalled, o.lastEventID, map[string]any{
		"status":        "started",
		"entity_type":   "tool",
		"entity_name":   name,
		"summary":       "开始调用工具",
		"tool_name":     name,
		"tool_call_id":  toolCallID,
		"arguments":     tryDecodeJSON(args),
		"arguments_raw": args,
	})

	o.toolAttempt = &toolAttempt{
		spanID:     spanID,
		eventID:    eventID,
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
		"entity_type":    "tool",
		"entity_name":    o.toolAttempt.toolName,
		"summary":        summary,
		"tool_name":      o.toolAttempt.toolName,
		"tool_call_id":   o.toolAttempt.toolCallID,
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
		"entity_type":    "tool",
		"entity_name":    o.toolAttempt.toolName,
		"summary":        summary,
		"error":          errorString(err),
		"latency_ms":     latencyMs,
		"result_summary": resultSummary,
	})

	o.lastSpanID = o.toolAttempt.spanID
	o.lastEventID = o.trace.Record(step, agent.EventToolReturned, o.toolAttempt.eventID, payload)
	o.toolAttempt = nil
}

// RunFinished 记录本次运行成功完成。
func (o *traceObserver) RunFinished(step int, output map[string]any) {
	o.trace.Record(step, agent.EventRunFinished, o.lastEventID, map[string]any{
		"status":       "success",
		"entity_type":  "run",
		"entity_name":  "agent_run",
		"summary":      "运行成功完成",
		"final_output": output,
	})
}

// RunFailed 记录本次运行在某个阶段失败，同时尽量关闭未完成的模型 span。
func (o *traceObserver) RunFailed(step int, stage string, err error, extra map[string]any) {
	if o.modelAttempt != nil {
		o.trace.FinishSpan(o.modelAttempt.spanID, "error", map[string]any{
			"entity_type": "model",
			"entity_name": o.modelName,
			"summary":     "模型调用失败",
			"error":       err.Error(),
		})
		o.lastSpanID = o.modelAttempt.spanID
		o.lastEventID = o.modelAttempt.eventID
		o.modelAttempt = nil
	}

	payload := map[string]any{
		"status":      "error",
		"entity_type": "run",
		"entity_name": "agent_run",
		"stage":       stage,
		"error":       err.Error(),
		"summary":     fmt.Sprintf("运行失败，阶段：%s", stage),
	}
	for k, v := range extra {
		payload[k] = v
	}
	o.trace.Record(step, agent.EventRunFailed, o.lastEventID, payload)
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
