package openai

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
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
