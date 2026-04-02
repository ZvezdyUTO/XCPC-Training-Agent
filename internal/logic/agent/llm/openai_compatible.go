package llm

import (
	"aATA/internal/app/apperr"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// OpenAICompatibleClient 负责与遵循 OpenAI Chat Completions 协议的模型服务交互。
type OpenAICompatibleClient struct {
	apiKey  string
	baseURL string
	model   string
}

// NewOpenAICompatibleClient 根据当前统一约定的 OPENAI_* 环境变量创建客户端。
func NewOpenAICompatibleClient(model string) *OpenAICompatibleClient {
	return &OpenAICompatibleClient{
		apiKey:  strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
		baseURL: strings.TrimSpace(os.Getenv("OPENAI_BASE_URL")),
		model:   model,
	}
}

// ModelName 返回当前客户端配置的模型名称。
func (c *OpenAICompatibleClient) ModelName() string {
	return c.model
}

type chatCompletionResponse struct {
	Choices []struct {
		Message      responseMessage `json:"message"`
		FinishReason string          `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error any `json:"error"`
}

// openAIErrorPayload 是 OpenAI Chat Completions 标准错误响应结构。
type openAIErrorPayload struct {
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"`
	} `json:"error"`
}

type responseMessage struct {
	Role      string             `json:"role"`
	Content   any                `json:"content"`
	ToolCalls []responseToolCall `json:"tool_calls"`
}

type responseToolCall struct {
	ID       string               `json:"id"`
	Type     string               `json:"type"`
	Function responseFunctionCall `json:"function"`
}

type responseFunctionCall struct {
	Name      string `json:"name"`
	Arguments any    `json:"arguments"`
}

// Chat 发送一次非流式 chat completion 请求，并把 provider 响应规范化成内部结构。
func (c *OpenAICompatibleClient) Chat(ctx context.Context, req ChatRequest) (*ChatCompletion, error) {
	startedAt := time.Now()

	if c.apiKey == "" || c.baseURL == "" {
		return nil, apperr.New(apperr.KindConfig, "openai_config_missing", "缺少 OPENAI_API_KEY 或 OPENAI_BASE_URL 配置", 500)
	}
	if err := validateBaseURL(c.baseURL); err != nil {
		return nil, err
	}

	body := map[string]any{
		"model":    c.model,
		"messages": req.Messages,
		"stream":   false,
	}

	if req.Temperature != nil {
		body["temperature"] = *req.Temperature
	}

	if len(req.Tools) > 0 {
		body["tools"] = req.Tools
		body["tool_choice"] = "auto"
	}
	if req.ResponseFormat != nil {
		body["response_format"] = req.ResponseFormat
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL,
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, apperr.Wrap(
			apperr.New(apperr.KindUpstream, "llm_request_failed", "LLM 请求失败", 502),
			err,
		)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apperr.Wrap(
			apperr.New(apperr.KindUpstream, "llm_response_read_failed", "读取 LLM 响应失败", 502),
			err,
		)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, buildProviderError(resp.StatusCode, respBody)
	}

	var result chatCompletionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, apperr.Wrap(
			apperr.New(apperr.KindUpstream, "llm_response_invalid_json", "解析 LLM 响应失败", 502),
			err,
		)
	}

	if len(result.Choices) == 0 {
		if result.Error != nil {
			return nil, apperr.New(apperr.KindUpstream, "llm_no_choices", fmt.Sprintf("LLM 未返回可用结果：%v", result.Error), 502)
		}
		return nil, apperr.New(apperr.KindUpstream, "llm_no_choices", "LLM 未返回可用结果", 502)
	}

	message, err := normalizeMessage(result.Choices[0].Message)
	if err != nil {
		return nil, apperr.Wrap(
			apperr.New(apperr.KindUpstream, "llm_response_invalid_shape", "LLM 响应格式不符合当前协议", 502),
			err,
		)
	}

	return &ChatCompletion{
		Message:      message,
		Content:      message.Content,
		ToolCalls:    message.ToolCalls,
		FinishReason: result.Choices[0].FinishReason,
		LatencyMs:    time.Since(startedAt).Milliseconds(),
		Usage: CompletionUsage{
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: result.Usage.CompletionTokens,
			TotalTokens:      result.Usage.TotalTokens,
		},
		RawResponse: string(respBody),
	}, nil
}

// validateBaseURL 只校验当前运行必需的 URL 形态，配置非法时直接返回明确错误。
func validateBaseURL(raw string) error {
	parsed, err := url.ParseRequestURI(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return apperr.New(apperr.KindConfig, "openai_base_url_invalid", "OPENAI_BASE_URL 配置非法", 500)
	}
	return nil
}

// normalizeMessage 将 provider 原始消息格式统一成内部 Message 结构。
// 这里按当前 OpenAI Chat Completions 协议做严格解析，异常形态直接报错。
func normalizeMessage(msg responseMessage) (Message, error) {
	content, err := normalizeContent(msg.Content)
	if err != nil {
		return Message{}, err
	}

	toolCalls := make([]ToolCall, 0, len(msg.ToolCalls))
	for _, call := range msg.ToolCalls {
		arguments, err := normalizeArguments(call.Function.Arguments)
		if err != nil {
			return Message{}, err
		}

		toolCalls = append(toolCalls, ToolCall{
			ID:   call.ID,
			Type: call.Type,
			Function: FunctionCall{
				Name:      call.Function.Name,
				Arguments: arguments,
			},
		})
	}

	return Message{
		Role:      msg.Role,
		Content:   content,
		ToolCalls: toolCalls,
	}, nil
}

// normalizeContent 只接受当前协议中的 string 或 null 内容。
func normalizeContent(raw any) (string, error) {
	switch v := raw.(type) {
	case nil:
		return "", nil
	case string:
		return v, nil
	default:
		return "", fmt.Errorf("解析消息 content 失败：期望 string 或 null，实际为 %T", raw)
	}
}

// normalizeArguments 只接受当前协议中的 string 或 null 参数。
func normalizeArguments(raw any) (string, error) {
	switch v := raw.(type) {
	case nil:
		return "", fmt.Errorf("解析 tool arguments 失败：期望 string，实际为 null")
	case string:
		return v, nil
	default:
		return "", fmt.Errorf("解析 tool arguments 失败：期望 string 或 null，实际为 %T", raw)
	}
}

// buildProviderError 将 OpenAI 风格错误响应转换为统一错误对象。
// 这里只区分“可直接返回给用户的 4xx 错误”和“上游/内部故障”两大类。
func buildProviderError(statusCode int, respBody []byte) error {
	var payload openAIErrorPayload
	kind := apperr.KindUpstream
	httpStatus := http.StatusBadGateway
	code := "llm_upstream_error"
	message := fmt.Sprintf("LLM 请求失败，状态码=%d", statusCode)

	if statusCode >= http.StatusBadRequest && statusCode < http.StatusInternalServerError {
		kind = apperr.KindUser
		httpStatus = statusCode
		code = "llm_request_invalid"
	}

	if err := json.Unmarshal(respBody, &payload); err == nil && payload.Error != nil {
		if providerCode := normalizeProviderCode(payload.Error.Code); providerCode != "" {
			code = providerCode
		}
		if providerMessage := strings.TrimSpace(payload.Error.Message); providerMessage != "" {
			message = providerMessage
		}
	}

	return apperr.New(kind, code, message, httpStatus)
}

// normalizeProviderCode 将 provider 错误码统一转成字符串。
func normalizeProviderCode(raw any) string {
	switch v := raw.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}
