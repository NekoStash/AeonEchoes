package gemini

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/provider"
)

func TestStreamAggregatesMultipleGenerateContentChunks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, ":streamGenerateContent") {
			t.Fatalf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		chunks := []string{
			`{"responseId":"gem_1","modelVersion":"gemini-test","candidates":[{"index":0,"content":{"role":"model","parts":[{"text":"hel"}]}}]}`,
			`{"responseId":"gem_1","modelVersion":"gemini-test","candidates":[{"index":0,"content":{"role":"model","parts":[{"text":"lo"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":2,"candidatesTokenCount":2,"totalTokenCount":4}}`,
		}
		for _, chunk := range chunks {
			_, _ = fmt.Fprintf(w, "data: %s\n\n", chunk)
		}
	}))
	defer server.Close()

	client, err := (Factory{HTTPClient: server.Client()}).NewTextClient(domain.ProviderConfig{Type: domain.ProviderGemini, BaseURL: server.URL, APIKey: "test-key"})
	if err != nil {
		t.Fatalf("NewTextClient() error: %v", err)
	}
	events, err := client.Stream(context.Background(), provider.TextRequest{Model: "gemini-test", UserPrompt: "hello"})
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
	if deltas != "hello" || final == nil || final.Content != "hello" || final.FinishReason != "STOP" || final.Usage.TotalTokens != 4 {
		t.Fatalf("deltas=%q final=%+v", deltas, final)
	}
}

func TestStreamKeepsAnonymousSameNameCallsWithDifferentArguments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		chunks := []string{
			`{"responseId":"gem_tools","modelVersion":"gemini-test","candidates":[{"index":0,"content":{"role":"model","parts":[{"functionCall":{"name":"search","args":{"q":"alpha"}}}]}}]}`,
			`{"responseId":"gem_tools","modelVersion":"gemini-test","candidates":[{"index":0,"content":{"role":"model","parts":[{"functionCall":{"name":"search","args":{"q":"alpha"}}},{"functionCall":{"name":"search","args":{"q":"beta"}}}]},"finishReason":"STOP"}]}`,
		}
		for _, chunk := range chunks {
			_, _ = fmt.Fprintf(w, "data: %s\n\n", chunk)
		}
	}))
	defer server.Close()

	client, err := (Factory{HTTPClient: server.Client()}).NewTextClient(domain.ProviderConfig{Type: domain.ProviderGemini, BaseURL: server.URL, APIKey: "test-key"})
	if err != nil {
		t.Fatalf("NewTextClient() error: %v", err)
	}
	events, err := client.Stream(context.Background(), provider.TextRequest{Model: "gemini-test", UserPrompt: "use tools"})
	if err != nil {
		t.Fatalf("Stream() error: %v", err)
	}
	var final *provider.ModelResponse
	for event := range events {
		if event.Error != "" {
			t.Fatalf("stream event error: %s", event.Error)
		}
		if event.Response != nil {
			final = event.Response
		}
	}
	if final == nil || len(final.ToolCalls) != 2 {
		t.Fatalf("final tool calls = %+v", final)
	}
	if final.ToolCalls[0].Name != "search" || string(final.ToolCalls[0].Arguments) != `{"q":"alpha"}` {
		t.Fatalf("first tool call = %+v", final.ToolCalls[0])
	}
	if final.ToolCalls[1].Name != "search" || string(final.ToolCalls[1].Arguments) != `{"q":"beta"}` {
		t.Fatalf("second tool call = %+v", final.ToolCalls[1])
	}
}
