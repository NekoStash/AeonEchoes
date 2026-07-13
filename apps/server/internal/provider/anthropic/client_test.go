package anthropic

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

func TestStreamAccumulatesTextUsageAndToolInput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		events := []string{
			`{"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"claude-test","content":[],"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":3,"output_tokens":1}}}`,
			`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
			`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hel"}}`,
			`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"lo"}}`,
			`{"type":"content_block_stop","index":0}`,
			`{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_1","name":"search","input":{}}}`,
			`{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"q\":\"x\"}"}}`,
			`{"type":"content_block_stop","index":1}`,
			`{"type":"message_delta","delta":{"stop_reason":"tool_use","stop_sequence":null},"usage":{"output_tokens":4}}`,
			`{"type":"message_stop"}`,
		}
		for _, event := range events {
			_, _ = fmt.Fprintf(w, "event: message\ndata: %s\n\n", event)
		}
	}))
	defer server.Close()

	client, err := (Factory{HTTPClient: server.Client()}).NewTextClient(domain.ProviderConfig{Type: domain.ProviderAnthropic, BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewTextClient() error: %v", err)
	}
	events, err := client.Stream(context.Background(), provider.TextRequest{Model: "claude-test", UserPrompt: "hello", Tools: []provider.ToolSpec{{Name: "search"}}})
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
	if deltas != "hello" || final == nil || final.Content != "hello" || final.FinishReason != "tool_use" || final.Usage.TotalTokens != 7 {
		t.Fatalf("deltas=%q final=%+v", deltas, final)
	}
	if len(final.ToolCalls) != 1 || final.ToolCalls[0].ID != "toolu_1" || final.ToolCalls[0].Name != "search" || string(final.ToolCalls[0].Arguments) != `{"q":"x"}` {
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
		_, _ = w.Write([]byte(`{"id":"msg_tools","type":"message","role":"assistant","model":"claude-test","content":[{"type":"text","text":"done"}],"stop_reason":"end_turn","stop_sequence":null,"usage":{"input_tokens":2,"output_tokens":1}}`))
	}))
	defer server.Close()

	client, err := (Factory{HTTPClient: server.Client()}).NewTextClient(domain.ProviderConfig{Type: domain.ProviderAnthropic, BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewTextClient() error: %v", err)
	}
	_, err = client.Generate(context.Background(), provider.TextRequest{
		Model: "claude-test",
		Messages: []provider.Message{
			{Role: "user", Content: "查找角色"},
			{
				Role: "assistant",
				ToolCalls: []provider.ToolCall{{
					ID:        "toolu_1",
					Type:      "tool_use",
					Name:      "character.search",
					Arguments: json.RawMessage(`{"project_id":"p1","query":"林烬"}`),
				}},
			},
			{Role: "tool", Name: "character.search", ToolCallID: "toolu_1", Content: `{"count":0}`},
		},
		MaxOutputTokens: 64,
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	raw, err := json.Marshal(seen["messages"])
	if err != nil {
		t.Fatalf("marshal messages: %v", err)
	}
	payload := string(raw)
	if !strings.Contains(payload, `"type":"tool_use"`) || !strings.Contains(payload, `"id":"toolu_1"`) {
		t.Fatalf("tool_use missing from payload: %s", payload)
	}
	if !strings.Contains(payload, `"type":"tool_result"`) || !strings.Contains(payload, `"tool_use_id":"toolu_1"`) {
		t.Fatalf("tool_result missing from payload: %s", payload)
	}
	if strings.Contains(payload, "Tool result for") || strings.Contains(payload, "Assistant requested tool calls") {
		t.Fatalf("payload still uses text fallback history: %s", payload)
	}
}

func TestAnthropicMessagesRejectsToolWithoutCallID(t *testing.T) {
	_, err := anthropicMessages(provider.TextRequest{
		Messages: []provider.Message{{Role: "tool", Name: "character.search", Content: `{}`}},
	})
	if err == nil || !strings.Contains(err.Error(), "tool_call_id") {
		t.Fatalf("error = %v, want tool_call_id", err)
	}
}
