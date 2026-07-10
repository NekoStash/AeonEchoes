package indexing

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"aeonechoes/server/internal/agent"
	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/extractor"
	"aeonechoes/server/internal/memory"
	"aeonechoes/server/internal/provider"
	"aeonechoes/server/internal/repository"
	"aeonechoes/server/internal/vector"
)

type fakeEmbeddingClient struct {
	vectors  [][]float64
	requests []provider.EmbeddingRequest
}

func (c *fakeEmbeddingClient) Embed(ctx context.Context, req provider.EmbeddingRequest) (provider.EmbeddingResponse, error) {
	c.requests = append(c.requests, req)
	if len(c.vectors) == 0 {
		return provider.EmbeddingResponse{}, fmt.Errorf("fake embedding client vectors are not configured")
	}
	return provider.EmbeddingResponse{Vectors: c.vectors}, nil
}

type fakeProviderFactory struct {
	client provider.EmbeddingModelClient
}

func (f fakeProviderFactory) NewEmbeddingClient(cfg domain.ProviderConfig) (provider.EmbeddingModelClient, error) {
	if f.client == nil {
		return nil, fmt.Errorf("fake embedding client is not configured")
	}
	return f.client, nil
}

type fakeVectorIndex struct {
	ensuredDimensions   []int
	recreatedDimensions []int
	upserts             int
}

func (v *fakeVectorIndex) EnsureCollection(ctx context.Context, dimension int) error {
	v.ensuredDimensions = append(v.ensuredDimensions, dimension)
	return nil
}
func (v *fakeVectorIndex) RecreateCollection(ctx context.Context, dimension int) error {
	v.recreatedDimensions = append(v.recreatedDimensions, dimension)
	return nil
}
func (v *fakeVectorIndex) UpsertTextVector(ctx context.Context, pointID string, values []float64, payload vector.PointPayload) error {
	v.upserts++
	return nil
}
func (v *fakeVectorIndex) DeleteBySource(ctx context.Context, sourceID string) error { return nil }
func (v *fakeVectorIndex) Health(ctx context.Context) error                          { return nil }

func TestServiceRunJobExtractsKnowledgeAndIndexesVector(t *testing.T) {
	store := memory.NewStore()
	project, _, err := store.CreateProject(domain.Project{Title: "黑曜档案馆", Slug: "archive"}, domain.StoryBible{Title: "黑曜档案馆", Logline: "测试"})
	if err != nil {
		t.Fatalf("CreateProject() error: %v", err)
	}
	providerCfg, err := store.CreateProvider(domain.ProviderConfig{ID: "provider_embed", Name: "Embedding", Type: domain.ProviderOpenAI, Enabled: true})
	if err != nil {
		t.Fatalf("CreateProvider() error: %v", err)
	}
	_, err = store.CreateModel(domain.ModelConfig{ID: "model_embed", ProviderID: providerCfg.ID, Name: "text-embedding-3-small", Kind: domain.ModelKindEmbedding, Dimension: 3, Enabled: true, DefaultForKind: true})
	if err != nil {
		t.Fatalf("CreateModel() error: %v", err)
	}
	chapter, err := store.CreateChapter(domain.CreateChapterRequest{ProjectID: project.ID, Title: "空白目录"})
	if err != nil {
		t.Fatalf("CreateChapter() error: %v", err)
	}
	version, job, err := store.SaveChapterVersion(domain.ChapterVersion{ProjectID: project.ID, ChapterID: chapter.ID, Title: "空白目录", Content: "[[人物:林烬]] 拿起 [[物品:灰烬钥匙]]。[[关系:林烬->灰烬钥匙:持有]] [[伏笔:第三见证人|未来回收]]", AuthorRole: domain.AgentRoleWriter, IndexStatus: "pending"})
	if err != nil {
		t.Fatalf("SaveChapterVersion() error: %v", err)
	}
	vectors := &fakeVectorIndex{}
	embedClient := &fakeEmbeddingClient{vectors: [][]float64{{0.1, 0.2, 0.3}}}
	service := NewService(store, agent.NewModelRouter(store, agent.NewAgentRoleRegistry()), fakeProviderFactory{client: embedClient}, vectors, extractor.NewDeterministicExtractor())
	updated, err := service.RunJob(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("RunJob() error: %v", err)
	}
	if updated.Status != "completed" {
		t.Fatalf("expected completed job, got %+v", updated)
	}
	if vectors.upserts != 1 {
		t.Fatalf("expected one vector upsert, got %d", vectors.upserts)
	}
	if len(vectors.ensuredDimensions) != 1 || vectors.ensuredDimensions[0] != 3 {
		t.Fatalf("expected EnsureCollection dimension 3, got %+v", vectors.ensuredDimensions)
	}
	facts, err := store.ListFacts(project.ID)
	if err != nil || len(facts) != 1 {
		t.Fatalf("expected one fact, facts=%+v err=%v", facts, err)
	}
	entities, err := store.ListEntities(project.ID)
	if err != nil || len(entities) != 3 {
		t.Fatalf("expected three entities, entities=%+v err=%v", entities, err)
	}
	expansion, err := store.ExpandGraph(project.ID, nil, 2)
	if err != nil {
		t.Fatalf("ExpandGraph() error: %v", err)
	}
	if len(expansion.Edges) != 1 {
		t.Fatalf("expected one edge, got %+v", expansion.Edges)
	}
	threads, err := store.ListPlotThreads(project.ID)
	if err != nil || len(threads) != 1 {
		t.Fatalf("expected one plot thread, threads=%+v err=%v", threads, err)
	}
	indexed, err := store.GetChapterVersion(version.ID)
	if err != nil || indexed.IndexStatus != "indexed" {
		t.Fatalf("expected indexed chapter version, got %+v err=%v", indexed, err)
	}
}

func TestServiceRebuildVectorsRecreatesCollectionAndRequeuesAllChapterVersions(t *testing.T) {
	store := memory.NewStore()
	providerCfg, err := store.CreateProvider(domain.ProviderConfig{ID: "provider_embed", Name: "Embedding", Type: domain.ProviderOpenAI, Enabled: true})
	if err != nil {
		t.Fatalf("CreateProvider() error: %v", err)
	}
	model, err := store.CreateModel(domain.ModelConfig{ID: "model_embed", ProviderID: providerCfg.ID, Name: "text-embedding-3-small", Kind: domain.ModelKindEmbedding, Dimension: 3, Enabled: true, DefaultForKind: true})
	if err != nil {
		t.Fatalf("CreateModel() error: %v", err)
	}
	projectA, _, err := store.CreateProject(domain.Project{Title: "A", Slug: "a"}, domain.StoryBible{Title: "A", Logline: "测试"})
	if err != nil {
		t.Fatalf("CreateProject(A) error: %v", err)
	}
	projectB, _, err := store.CreateProject(domain.Project{Title: "B", Slug: "b"}, domain.StoryBible{Title: "B", Logline: "测试"})
	if err != nil {
		t.Fatalf("CreateProject(B) error: %v", err)
	}
	chapterA, err := store.CreateChapter(domain.CreateChapterRequest{ProjectID: projectA.ID, Title: "A1"})
	if err != nil {
		t.Fatalf("CreateChapter(A) error: %v", err)
	}
	chapterB, err := store.CreateChapter(domain.CreateChapterRequest{ProjectID: projectB.ID, Title: "B1"})
	if err != nil {
		t.Fatalf("CreateChapter(B) error: %v", err)
	}
	versionA, _, err := store.SaveChapterVersion(domain.ChapterVersion{ProjectID: projectA.ID, ChapterID: chapterA.ID, Title: "A1", Content: "内容 A1", AuthorRole: domain.AgentRoleWriter, IndexStatus: "indexed"})
	if err != nil {
		t.Fatalf("SaveChapterVersion(A) error: %v", err)
	}
	versionB, _, err := store.SaveChapterVersion(domain.ChapterVersion{ProjectID: projectB.ID, ChapterID: chapterB.ID, Title: "B1", Content: "内容 B1", AuthorRole: domain.AgentRoleWriter, IndexStatus: "failed"})
	if err != nil {
		t.Fatalf("SaveChapterVersion(B) error: %v", err)
	}
	vectors := &fakeVectorIndex{}
	embedClient := &fakeEmbeddingClient{vectors: [][]float64{{0.1, 0.2, 0.3}}}
	service := NewService(store, agent.NewModelRouter(store, agent.NewAgentRoleRegistry()), fakeProviderFactory{client: embedClient}, vectors)

	result, err := service.RebuildVectors(context.Background())
	if err != nil {
		t.Fatalf("RebuildVectors() error: %v", err)
	}
	if result.EmbeddingModelID != model.ID || result.EmbeddingModelName != model.Name || result.EmbeddingDimension != 3 {
		t.Fatalf("unexpected rebuild result model info: %+v", result)
	}
	if result.ProjectCount != 2 || result.ChapterVersionCount != 2 || result.JobCount != 2 {
		t.Fatalf("unexpected rebuild result counts: %+v", result)
	}
	if len(vectors.recreatedDimensions) != 1 || vectors.recreatedDimensions[0] != 3 {
		t.Fatalf("expected RecreateCollection dimension 3, got %+v", vectors.recreatedDimensions)
	}
	if len(embedClient.requests) != 1 {
		t.Fatalf("expected one probe request, got %d", len(embedClient.requests))
	}
	if len(embedClient.requests[0].Inputs) != 1 || embedClient.requests[0].Inputs[0] != "vector dimension probe" {
		t.Fatalf("unexpected probe request: %+v", embedClient.requests[0])
	}
	updatedA, err := store.GetChapterVersion(versionA.ID)
	if err != nil {
		t.Fatalf("GetChapterVersion(A) error: %v", err)
	}
	updatedB, err := store.GetChapterVersion(versionB.ID)
	if err != nil {
		t.Fatalf("GetChapterVersion(B) error: %v", err)
	}
	if updatedA.IndexStatus != "pending" || updatedB.IndexStatus != "pending" {
		t.Fatalf("expected chapter versions reset to pending, got A=%q B=%q", updatedA.IndexStatus, updatedB.IndexStatus)
	}
	jobs, err := store.ListIndexJobs(repository.IndexJobFilter{})
	if err != nil {
		t.Fatalf("ListIndexJobs() error: %v", err)
	}
	var rebuildJobs []domain.IndexJob
	for _, job := range jobs {
		if job.Payload["trigger"] == "vector_rebuild" {
			rebuildJobs = append(rebuildJobs, job)
		}
	}
	if len(rebuildJobs) != 2 {
		t.Fatalf("expected 2 rebuild jobs, got %+v", rebuildJobs)
	}
	for _, job := range rebuildJobs {
		if job.Kind != "chapter_reindex" || job.Status != "pending" {
			t.Fatalf("unexpected rebuild job state: %+v", job)
		}
		if job.Payload["embedding_model_id"] != model.ID || job.Payload["embedding_model_name"] != model.Name || job.Payload["embedding_dimension"] != strconv.Itoa(result.EmbeddingDimension) {
			t.Fatalf("unexpected rebuild job payload: %+v", job.Payload)
		}
	}
}

func TestServiceRebuildVectorsFailsWhenConfiguredDimensionDiffersFromProvider(t *testing.T) {
	store := memory.NewStore()
	providerCfg, err := store.CreateProvider(domain.ProviderConfig{ID: "provider_embed", Name: "Embedding", Type: domain.ProviderOpenAI, Enabled: true})
	if err != nil {
		t.Fatalf("CreateProvider() error: %v", err)
	}
	_, err = store.CreateModel(domain.ModelConfig{ID: "model_embed", ProviderID: providerCfg.ID, Name: "text-embedding-3-small", Kind: domain.ModelKindEmbedding, Dimension: 4, Enabled: true, DefaultForKind: true})
	if err != nil {
		t.Fatalf("CreateModel() error: %v", err)
	}
	vectors := &fakeVectorIndex{}
	embedClient := &fakeEmbeddingClient{vectors: [][]float64{{0.1, 0.2, 0.3}}}
	service := NewService(store, agent.NewModelRouter(store, agent.NewAgentRoleRegistry()), fakeProviderFactory{client: embedClient}, vectors)

	_, err = service.RebuildVectors(context.Background())
	if err == nil {
		t.Fatalf("RebuildVectors() expected error")
	}
	if err.Error() != "embedding model \"model_embed\" dimension mismatch: configured 4, provider returned 3" {
		t.Fatalf("unexpected RebuildVectors() error: %v", err)
	}
	if len(vectors.recreatedDimensions) != 0 {
		t.Fatalf("expected no collection recreation on dimension mismatch, got %+v", vectors.recreatedDimensions)
	}
}
