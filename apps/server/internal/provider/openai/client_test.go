package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/provider"
)

func TestStreamAggregatesContentUsageAndToolFragments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		chunks := []string{
			`{"id":"chat_1","object":"chat.completion.chunk","created":1,"model":"gpt-test","choices":[{"index":0,"delta":{"role":"assistant","content":"hel","tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"search","arguments":"{\"q\":"}}]},"finish_reason":null}]}`,
			`{"id":"chat_1","object":"chat.completion.chunk","created":1,"model":"gpt-test","choices":[{"index":0,"delta":{"content":"lo","tool_calls":[{"index":0,"function":{"arguments":"\"x\"}"}}]},"finish_reason":"tool_calls"}]}`,
			`{"id":"chat_1","object":"chat.completion.chunk","created":1,"model":"gpt-test","choices":[],"usage":{"prompt_tokens":3,"completion_tokens":2,"total_tokens":5}}`,
		}
		for _, chunk := range chunks {
			_, _ = fmt.Fprintf(w, "data: %s\n\n", chunk)
		}
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	client, err := (Factory{HTTPClient: server.Client()}).NewTextClient(domain.ProviderConfig{Type: domain.ProviderOpenAI, BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewTextClient() error: %v", err)
	}
	events, err := client.Stream(context.Background(), provider.TextRequest{Model: "gpt-test", UserPrompt: "hello", Tools: []provider.ToolSpec{{Name: "search"}}})
	if err != nil {
		t.Fatalf("Stream() error: %v", err)
	}
	var deltas string
	var final *provider.ModelResponse
	for event := range events {
		if event.Error != "" {
			t.Fatalf("stream event error: %s", event.Error)
		}
		deltas += event.Delta
		if event.Response != nil {
			final = event.Response
		}
	}
	if deltas != "hello" {
		t.Fatalf("deltas = %q", deltas)
	}
	if final == nil || final.Content != "hello" || final.FinishReason != "tool_calls" || final.Usage.TotalTokens != 5 {
		t.Fatalf("final = %+v", final)
	}
	if len(final.ToolCalls) != 1 || final.ToolCalls[0].ID != "call_1" || final.ToolCalls[0].Name != "search" || string(final.ToolCalls[0].Arguments) != `{"q":"x"}` {
		t.Fatalf("tool calls = %+v", final.ToolCalls)
	}
}

func TestGenerateSerializesNativeToolHistory(t *testing.T) {
	var seen map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&seen); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chat_tools","object":"chat.completion","created":1,"model":"gpt-test","choices":[{"index":0,"message":{"role":"assistant","content":"done"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
	}))
	defer server.Close()

	client, err := (Factory{HTTPClient: server.Client()}).NewTextClient(domain.ProviderConfig{Type: domain.ProviderOpenAI, BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewTextClient() error: %v", err)
	}
	_, err = client.Generate(context.Background(), provider.TextRequest{
		Model: "gpt-test",
		Messages: []provider.Message{
			{Role: "user", Content: "查找角色"},
			{
				Role:    "assistant",
				Content: "先调用工具",
				ToolCalls: []provider.ToolCall{{
					ID:        "call_1",
					Type:      "function",
					Name:      "character.search",
					Arguments: json.RawMessage(`{"project_id":"p1","query":"林烬"}`),
				}},
			},
			{Role: "tool", Name: "character.search", ToolCallID: "call_1", Content: `{"count":0}`},
		},
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	raw, err := json.Marshal(seen["messages"])
	if err != nil {
		t.Fatalf("marshal messages: %v", err)
	}
	payload := string(raw)
	if !strings.Contains(payload, `"role":"assistant"`) || !strings.Contains(payload, `"tool_calls"`) || !strings.Contains(payload, `"call_1"`) {
		t.Fatalf("assistant tool_calls missing from payload: %s", payload)
	}
	if !strings.Contains(payload, `"role":"tool"`) || !strings.Contains(payload, `"tool_call_id":"call_1"`) {
		t.Fatalf("tool result missing from payload: %s", payload)
	}
	if strings.Contains(payload, "Tool result for") || strings.Contains(payload, "Assistant requested tool calls") {
		t.Fatalf("payload still uses text fallback history: %s", payload)
	}
}

func TestOpenAIChatMessagesRejectsToolWithoutCallID(t *testing.T) {
	_, err := openAIChatMessages(provider.TextRequest{
		Messages: []provider.Message{{Role: "tool", Name: "character.search", Content: `{}`}},
	})
	if err == nil || !strings.Contains(err.Error(), "tool_call_id") {
		t.Fatalf("error = %v, want tool_call_id", err)
	}
}
