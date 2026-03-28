package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type OpenAICompatibleClient struct {
	apiKey  string
	baseURL string
	model   string
}

func NewOpenAICompatibleClient(model string) *OpenAICompatibleClient {
	return &OpenAICompatibleClient{
		apiKey: firstNonEmpty(
			os.Getenv("LLM_API_KEY"),
			os.Getenv("OPENAI_API_KEY"),
			os.Getenv("DASHSCOPE_API_KEY"),
		),
		baseURL: normalizeChatCompletionsURL(firstNonEmpty(
			os.Getenv("LLM_BASE_URL"),
			os.Getenv("OPENAI_BASE_URL"),
			os.Getenv("DASHSCOPE_BASE_URL"),
		)),
		model: model,
	}
}

func NewAliyunQwenClient(model string) *OpenAICompatibleClient {
	return NewOpenAICompatibleClient(model)
}

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

func (c *OpenAICompatibleClient) Chat(ctx context.Context, req ChatRequest) (*ChatCompletion, error) {
	startedAt := time.Now()

	if c.apiKey == "" || c.baseURL == "" {
		return nil, errors.New("缺少 LLM_API_KEY/OPENAI_API_KEY/DASHSCOPE_API_KEY 或 LLM_BASE_URL/OPENAI_BASE_URL/DASHSCOPE_BASE_URL 配置")
	}

	body := map[string]any{
		"model":    c.model,
		"messages": req.Messages,
		"stream":   false,
	}

	if req.Temperature != nil {
		body["temperature"] = *req.Temperature
	} else {
		body["temperature"] = 0
	}

	if len(req.Tools) > 0 {
		body["tools"] = req.Tools
		body["tool_choice"] = "auto"
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
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("LLM 请求失败：状态码=%d，响应=%s", resp.StatusCode, string(respBody))
	}

	var result chatCompletionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析 LLM 响应失败：%w", err)
	}

	if len(result.Choices) == 0 {
		if result.Error != nil {
			return nil, fmt.Errorf("LLM 未返回可用结果：%v", result.Error)
		}
		return nil, errors.New("LLM 未返回可用结果")
	}

	message, err := normalizeMessage(result.Choices[0].Message)
	if err != nil {
		return nil, err
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

func normalizeContent(raw any) (string, error) {
	switch v := raw.(type) {
	case nil:
		return "", nil
	case string:
		return v, nil
	case []any:
		var builder strings.Builder
		for _, item := range v {
			part, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if text, _ := part["text"].(string); text != "" {
				builder.WriteString(text)
			}
		}
		return builder.String(), nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("解析消息 content 失败：%w", err)
		}
		return string(b), nil
	}
}

func normalizeArguments(raw any) (string, error) {
	switch v := raw.(type) {
	case nil:
		return "{}", nil
	case string:
		return v, nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("解析 tool arguments 失败：%w", err)
		}
		return string(b), nil
	}
}

func normalizeChatCompletionsURL(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimRight(raw, "/")
	if raw == "" {
		return raw
	}
	if strings.HasSuffix(raw, "/chat/completions") {
		return raw
	}
	return raw + "/chat/completions"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
