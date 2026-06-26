package retrieval

import (
	"context"
	"fmt"
	"testing"
	"time"

	"aeonechoes/server/internal/agent"
	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/memory"
	"aeonechoes/server/internal/provider"
)

type stubEmbeddingClient struct {
	response provider.EmbeddingResponse
	err      error
	seenReq  provider.EmbeddingRequest
}

func (c *stubEmbeddingClient) Embed(ctx context.Context, req provider.EmbeddingRequest) (provider.EmbeddingResponse, error) {
	c.seenReq = req
	if c.err != nil {
		return provider.EmbeddingResponse{}, c.err
	}
	return c.response, nil
}

type stubProviderFactory struct {
	client provider.EmbeddingModelClient
	err    error
	seen   domain.ProviderConfig
}

func (f *stubProviderFactory) NewEmbeddingClient(cfg domain.ProviderConfig) (provider.EmbeddingModelClient, error) {
	f.seen = cfg
	if f.err != nil {
		return nil, f.err
	}
	return f.client, nil
}

type stubVectorSearcher struct {
	items         []domain.SemanticSearchItem
	err           error
	seenVector    []float64
	seenProjectID string
	seenLimit     int
}

func (s *stubVectorSearcher) Search(ctx context.Context, vector []float64, projectID string, limit int) ([]domain.SemanticSearchItem, error) {
	s.seenVector = append([]float64(nil), vector...)
	s.seenProjectID = projectID
	s.seenLimit = limit
	if s.err != nil {
		return nil, s.err
	}
	return s.items, nil
}

func TestServiceSearchRoutesEmbeddingAndVectorSearch(t *testing.T) {
	store := memory.NewStore()
	providerCfg, err := store.CreateProvider(domain.ProviderConfig{ID: "provider_embed", Name: "Embed Provider", Type: domain.ProviderOpenAI, Enabled: true})
	if err != nil {
		t.Fatalf("CreateProvider() error: %v", err)
	}
	_, err = store.CreateModel(domain.ModelConfig{ID: "model_embed", ProviderID: providerCfg.ID, Name: "text-embedding-3-small", Kind: domain.ModelKindEmbedding, Enabled: true, DefaultForKind: true, CreatedAt: time.Now().UTC()})
	if err != nil {
		t.Fatalf("CreateModel() error: %v", err)
	}
	embedClient := &stubEmbeddingClient{response: provider.EmbeddingResponse{Vectors: [][]float64{{0.11, 0.22, 0.33}}}}
	vectorSearcher := &stubVectorSearcher{items: []domain.SemanticSearchItem{{SourceID: "chapter_version_1", Score: 0.98, Payload: map[string]any{"project_id": "project_1"}}}}
	service := NewService(agent.NewModelRouter(store, agent.NewAgentRoleRegistry()), &stubProviderFactory{client: embedClient}, vectorSearcher)

	result, err := service.Search(context.Background(), domain.SemanticSearchRequest{Query: "  星门遗迹  ", ProjectID: "project_1", Limit: 5})
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}
	if result.Query != "星门遗迹" || result.ProjectID != "project_1" {
		t.Fatalf("unexpected result envelope: %+v", result)
	}
	if len(result.Items) != 1 || result.Items[0].SourceID != "chapter_version_1" {
		t.Fatalf("unexpected result items: %+v", result.Items)
	}
	if embedClient.seenReq.Model != "text-embedding-3-small" {
		t.Fatalf("unexpected embedding model: %+v", embedClient.seenReq)
	}
	if len(embedClient.seenReq.Inputs) != 1 || embedClient.seenReq.Inputs[0] != "星门遗迹" {
		t.Fatalf("unexpected embedding inputs: %+v", embedClient.seenReq.Inputs)
	}
	if vectorSearcher.seenProjectID != "project_1" || vectorSearcher.seenLimit != 5 {
		t.Fatalf("unexpected vector search args: project=%s limit=%d", vectorSearcher.seenProjectID, vectorSearcher.seenLimit)
	}
	if len(vectorSearcher.seenVector) != 3 || vectorSearcher.seenVector[0] != 0.11 {
		t.Fatalf("unexpected vector payload: %+v", vectorSearcher.seenVector)
	}
}

func TestServiceSearchRequiresQueryAndProjectID(t *testing.T) {
	service := NewService(nil, nil, nil)
	_, err := service.Search(context.Background(), domain.SemanticSearchRequest{Query: " ", ProjectID: "project_1"})
	if err == nil || err.Error() != "semantic search query must not be empty" {
		t.Fatalf("expected query validation error, got %v", err)
	}
	_, err = service.Search(context.Background(), domain.SemanticSearchRequest{Query: "hello", ProjectID: " "})
	if err == nil || err.Error() != "semantic search project_id must not be empty" {
		t.Fatalf("expected project validation error, got %v", err)
	}
}

func TestServiceSearchRejectsUnexpectedEmbeddingShape(t *testing.T) {
	store := memory.NewStore()
	providerCfg, err := store.CreateProvider(domain.ProviderConfig{ID: "provider_embed", Name: "Embed Provider", Type: domain.ProviderOpenAI, Enabled: true})
	if err != nil {
		t.Fatalf("CreateProvider() error: %v", err)
	}
	_, err = store.CreateModel(domain.ModelConfig{ID: "model_embed", ProviderID: providerCfg.ID, Name: "text-embedding-3-small", Kind: domain.ModelKindEmbedding, Enabled: true, DefaultForKind: true})
	if err != nil {
		t.Fatalf("CreateModel() error: %v", err)
	}
	embedClient := &stubEmbeddingClient{response: provider.EmbeddingResponse{Vectors: [][]float64{{1}, {2}}}}
	service := NewService(agent.NewModelRouter(store, agent.NewAgentRoleRegistry()), &stubProviderFactory{client: embedClient}, &stubVectorSearcher{})

	_, err = service.Search(context.Background(), domain.SemanticSearchRequest{Query: "query", ProjectID: "project_1"})
	if err == nil || err.Error() != fmt.Sprintf("embedding model returned %d vectors for one query", 2) {
		t.Fatalf("unexpected error: %v", err)
	}
}
