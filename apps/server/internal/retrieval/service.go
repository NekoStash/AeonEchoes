package retrieval

import (
	"context"
	"fmt"
	"strings"

	"aeonechoes/server/internal/agent"
	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/provider"
)

type ProviderFactory interface {
	NewEmbeddingClient(cfg domain.ProviderConfig) (provider.EmbeddingModelClient, error)
}

type VectorSearcher interface {
	Search(ctx context.Context, vector []float64, projectID string, limit int) ([]domain.SemanticSearchItem, error)
}

// Service routes semantic search requests through the configured embedding model
// and then executes vector retrieval against Qdrant.
type Service struct {
	router    *agent.ModelRouter
	providers ProviderFactory
	vectors   VectorSearcher
}

func NewService(router *agent.ModelRouter, providers ProviderFactory, vectors VectorSearcher) *Service {
	return &Service{router: router, providers: providers, vectors: vectors}
}

func (s *Service) Search(ctx context.Context, req domain.SemanticSearchRequest) (domain.SemanticSearchResult, error) {
	cleanQuery := strings.TrimSpace(req.Query)
	cleanProjectID := strings.TrimSpace(req.ProjectID)
	if cleanQuery == "" {
		return domain.SemanticSearchResult{}, fmt.Errorf("semantic search query must not be empty")
	}
	if cleanProjectID == "" {
		return domain.SemanticSearchResult{}, fmt.Errorf("semantic search project_id must not be empty")
	}
	if s == nil || s.vectors == nil {
		return domain.SemanticSearchResult{}, fmt.Errorf("semantic retrieval vector search is not configured")
	}
	if s.router == nil || s.providers == nil {
		return domain.SemanticSearchResult{}, fmt.Errorf("semantic retrieval embedding routing is not configured")
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	selection, err := s.router.SelectEmbeddingModel()
	if err != nil {
		return domain.SemanticSearchResult{}, err
	}
	client, err := s.providers.NewEmbeddingClient(selection.Provider)
	if err != nil {
		return domain.SemanticSearchResult{}, err
	}
	resp, err := client.Embed(ctx, provider.EmbeddingRequest{Model: selection.Model.Name, Inputs: []string{cleanQuery}, Metadata: map[string]string{"project_id": cleanProjectID, "operation": "semantic_search"}})
	if err != nil {
		return domain.SemanticSearchResult{}, err
	}
	if len(resp.Vectors) != 1 || len(resp.Vectors[0]) == 0 {
		return domain.SemanticSearchResult{}, fmt.Errorf("embedding model returned %d vectors for one query", len(resp.Vectors))
	}
	items, err := s.vectors.Search(ctx, resp.Vectors[0], cleanProjectID, limit)
	if err != nil {
		return domain.SemanticSearchResult{}, err
	}
	return domain.SemanticSearchResult{Query: cleanQuery, ProjectID: cleanProjectID, Items: items}, nil
}
