package openairesponses

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/provider"
)

func TestGenerateUsesResponsesEndpointAndParsesOutput(t *testing.T) {
	var seenPath string
	var seenPayload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&seenPayload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_1","model":"gpt-5-test","status":"completed","output":[{"id":"msg_1","type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"正文","annotations":[]}]}],"usage":{"input_tokens":3,"output_tokens":5,"total_tokens":8}}`))
	}))
	defer server.Close()

	factory := Factory{HTTPClient: server.Client()}
	clientIface, err := factory.NewTextClient(domain.ProviderConfig{Type: domain.ProviderOpenAIResponses, BaseURL: server.URL, APIKey: "test-key", Enabled: true})
	if err != nil {
		t.Fatalf("NewTextClient: %v", err)
	}
	resp, err := clientIface.Generate(context.Background(), provider.TextRequest{Model: "gpt-5-test", SystemPrompt: "sys", UserPrompt: "写一章", MaxOutputTokens: 128})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if seenPath != "/v1/responses" {
		t.Fatalf("expected /v1/responses, got %s", seenPath)
	}
	if seenPayload["model"] != "gpt-5-test" || seenPayload["instructions"] != "sys" {
		t.Fatalf("unexpected payload: %+v", seenPayload)
	}
	if resp.Content != "正文" || resp.Usage.TotalTokens != 8 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestEmbeddingUsesEmbeddingsEndpoint(t *testing.T) {
	var seenPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"text-embedding-test","data":[{"embedding":[0.1,0.2]}],"usage":{"prompt_tokens":2,"total_tokens":2}}`))
	}))
	defer server.Close()
	factory := Factory{HTTPClient: server.Client()}
	clientIface, err := factory.NewEmbeddingClient(domain.ProviderConfig{Type: domain.ProviderOpenAIResponses, BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewEmbeddingClient: %v", err)
	}
	resp, err := clientIface.Embed(context.Background(), provider.EmbeddingRequest{Model: "text-embedding-test", Inputs: []string{"hello"}})
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if seenPath != "/v1/embeddings" {
		t.Fatalf("expected /v1/embeddings, got %s", seenPath)
	}
	if len(resp.Vectors) != 1 || len(resp.Vectors[0]) != 2 {
		t.Fatalf("unexpected vectors: %+v", resp.Vectors)
	}
}
