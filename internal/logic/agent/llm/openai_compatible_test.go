package llm

import (
	"aATA/internal/app/apperr"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestNewOpenAICompatibleClientOnlyReadsOpenAIEnv 验证客户端只读取 OPENAI_* 配置，且原样使用 URL。
func TestNewOpenAICompatibleClientOnlyReadsOpenAIEnv(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_BASE_URL", "https://api.example.com/v1/chat/completions")
	t.Setenv("LLM_API_KEY", "legacy-key")
	t.Setenv("LLM_BASE_URL", "https://legacy.example.com")
	t.Setenv("DASHSCOPE_API_KEY", "dash-key")
	t.Setenv("DASHSCOPE_BASE_URL", "https://dash.example.com")

	client := NewOpenAICompatibleClient("gpt-test")
	if client.apiKey != "test-key" {
		t.Fatalf("apiKey = %q, want %q", client.apiKey, "test-key")
	}
	if client.baseURL != "https://api.example.com/v1/chat/completions" {
		t.Fatalf("baseURL = %q", client.baseURL)
	}
}

// TestChatRequiresOpenAIEnv 验证缺少 OPENAI_* 配置时直接报错。
func TestChatRequiresOpenAIEnv(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_BASE_URL", "")

	client := NewOpenAICompatibleClient("gpt-test")
	_, err := client.Chat(context.Background(), ChatRequest{})
	if err == nil || !strings.Contains(err.Error(), "OPENAI_API_KEY") {
		t.Fatalf("Chat() error = %v, want OPENAI env error", err)
	}
}

// TestChatParsesStandardResponse 验证标准文本响应可被正确解析。
func TestChatParsesStandardResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request failed: %v", err)
		}
		if body["tool_choice"] != "auto" {
			t.Fatalf("tool_choice = %v, want auto", body["tool_choice"])
		}
		responseFormat, ok := body["response_format"].(map[string]any)
		if !ok {
			t.Fatalf("response_format = %T, want object", body["response_format"])
		}
		if responseFormat["type"] != "json_schema" {
			t.Fatalf("response_format.type = %v, want json_schema", responseFormat["type"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [{
				"message": {"role": "assistant", "content": "ok", "tool_calls": []},
				"finish_reason": "stop"
			}],
			"usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}
		}`))
	}))
	defer server.Close()

	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_BASE_URL", server.URL)

	client := NewOpenAICompatibleClient("gpt-test")
	resp, err := client.Chat(context.Background(), ChatRequest{
		ResponseFormat: &ResponseFormat{
			Type: "json_schema",
			JSONSchema: &ResponseJSONSchema{
				Name:   "final_output",
				Strict: true,
				Schema: map[string]any{"type": "object"},
			},
		},
		Tools: []ToolDefinition{{
			Type: "function",
			Function: ToolFunctionDefinition{
				Name:        "sample_tool",
				Description: "sample",
				Parameters:  map[string]any{"type": "object"},
			},
		}},
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if resp.Content != "ok" {
		t.Fatalf("Content = %q, want ok", resp.Content)
	}
	if resp.Usage.TotalTokens != 15 {
		t.Fatalf("TotalTokens = %d, want 15", resp.Usage.TotalTokens)
	}
}

// TestChatParsesToolCallResponse 验证标准 tool-calling 响应可被正确解析。
func TestChatParsesToolCallResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [{
				"message": {
					"role": "assistant",
					"content": null,
					"tool_calls": [{
						"id": "call_1",
						"type": "function",
						"function": {
							"name": "sample_tool",
							"arguments": "{\"student_id\":\"1\"}"
						}
					}]
				},
				"finish_reason": "tool_calls"
			}],
			"usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}
		}`))
	}))
	defer server.Close()

	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_BASE_URL", server.URL)

	client := NewOpenAICompatibleClient("gpt-test")
	resp, err := client.Chat(context.Background(), ChatRequest{})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d, want 1", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Function.Name != "sample_tool" {
		t.Fatalf("ToolCalls[0].Function.Name = %q", resp.ToolCalls[0].Function.Name)
	}
}

// TestChatReturnsTypedProviderError 验证上游 401 错误会被分类为可直接返回的用户错误。
func TestChatReturnsTypedProviderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{
			"error": {
				"message": "Invalid API key provided",
				"type": "invalid_request_error",
				"code": "invalid_api_key"
			}
		}`))
	}))
	defer server.Close()

	t.Setenv("OPENAI_API_KEY", "bad-key")
	t.Setenv("OPENAI_BASE_URL", server.URL)

	client := NewOpenAICompatibleClient("gpt-test")
	_, err := client.Chat(context.Background(), ChatRequest{})
	if err == nil {
		t.Fatalf("Chat() error = nil, want provider error")
	}

	appErr, ok := apperr.As(err)
	if !ok {
		t.Fatalf("Chat() error = %T, want *apperr.Error", err)
	}
	if appErr.Kind != apperr.KindUser {
		t.Fatalf("Kind = %q, want %q", appErr.Kind, apperr.KindUser)
	}
	if appErr.Code != "invalid_api_key" {
		t.Fatalf("Code = %q, want invalid_api_key", appErr.Code)
	}
	if appErr.HTTPStatus != http.StatusUnauthorized {
		t.Fatalf("HTTPStatus = %d, want %d", appErr.HTTPStatus, http.StatusUnauthorized)
	}
}

// TestNormalizeContentRejectsLegacyArray 验证旧 content 数组形态会被直接拒绝。
func TestNormalizeContentRejectsLegacyArray(t *testing.T) {
	_, err := normalizeContent([]any{map[string]any{"text": "legacy"}})
	if err == nil || !strings.Contains(err.Error(), "期望 string 或 null") {
		t.Fatalf("normalizeContent() error = %v", err)
	}
}

// TestNormalizeArgumentsRejectsObject 验证非字符串 arguments 会被直接拒绝。
func TestNormalizeArgumentsRejectsObject(t *testing.T) {
	_, err := normalizeArguments(map[string]any{"student_id": "1"})
	if err == nil || !strings.Contains(err.Error(), "期望 string 或 null") {
		t.Fatalf("normalizeArguments() error = %v", err)
	}
}
