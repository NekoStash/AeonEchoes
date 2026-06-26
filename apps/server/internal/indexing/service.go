package indexing

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"aeonechoes/server/internal/agent"
	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/extractor"
	"aeonechoes/server/internal/provider"
	"aeonechoes/server/internal/repository"
	"aeonechoes/server/internal/vector"
)

type VectorIndex interface {
	EnsureCollection(ctx context.Context, dimension int) error
	RecreateCollection(ctx context.Context, dimension int) error
	UpsertTextVector(ctx context.Context, pointID string, vector []float64, payload vector.PointPayload) error
	DeleteBySource(ctx context.Context, sourceID string) error
	Health(ctx context.Context) error
}

type ProviderFactory interface {
	NewEmbeddingClient(cfg domain.ProviderConfig) (provider.EmbeddingModelClient, error)
}

// Service processes index jobs into narrative knowledge and vector storage.
type Service struct {
	store     repository.AppStore
	router    *agent.ModelRouter
	providers ProviderFactory
	vectors   VectorIndex
	extractor extractor.Extractor
}

type RunResult struct {
	Processed []domain.IndexJob `json:"processed"`
	Count     int               `json:"count"`
	Error     string            `json:"error,omitempty"`
}

type RebuildVectorsResult struct {
	EmbeddingModelID    string `json:"embedding_model_id"`
	EmbeddingModelName  string `json:"embedding_model_name"`
	EmbeddingDimension  int    `json:"embedding_dimension"`
	ProjectCount        int    `json:"project_count"`
	ChapterVersionCount int    `json:"chapter_version_count"`
	JobCount            int    `json:"job_count"`
}

func NewService(store repository.AppStore, router *agent.ModelRouter, providers ProviderFactory, vectors VectorIndex, knowledgeExtractor ...extractor.Extractor) *Service {
	var ex extractor.Extractor
	if len(knowledgeExtractor) > 0 {
		ex = knowledgeExtractor[0]
	}
	return &Service{store: store, router: router, providers: providers, vectors: vectors, extractor: ex}
}

func (s *Service) RunJob(ctx context.Context, id string) (domain.IndexJob, error) {
	if strings.TrimSpace(id) == "" {
		return domain.IndexJob{}, fmt.Errorf("index job id must not be empty")
	}
	if err := s.requireBase(); err != nil {
		return domain.IndexJob{}, err
	}
	job, err := s.store.GetIndexJob(id)
	if err != nil {
		return domain.IndexJob{}, err
	}
	return s.process(ctx, job)
}

func (s *Service) RunPending(ctx context.Context, projectID string, limit int) (RunResult, error) {
	if err := s.requireBase(); err != nil {
		return RunResult{}, err
	}
	if limit <= 0 {
		limit = 10
	}
	jobs, err := s.store.ListPendingIndexJobs(projectID, limit)
	if err != nil {
		return RunResult{}, err
	}
	processed := make([]domain.IndexJob, 0, len(jobs))
	for _, job := range jobs {
		updated, err := s.process(ctx, job)
		processed = append(processed, updated)
		if err != nil {
			return RunResult{Processed: processed, Count: len(processed), Error: err.Error()}, err
		}
	}
	return RunResult{Processed: processed, Count: len(processed)}, nil
}

func (s *Service) requireBase() error {
	if s == nil || s.store == nil {
		return fmt.Errorf("indexing service store is not configured")
	}
	return nil
}

func (s *Service) RebuildVectors(ctx context.Context) (RebuildVectorsResult, error) {
	if err := s.requireBase(); err != nil {
		return RebuildVectorsResult{}, err
	}
	if s.vectors == nil {
		return RebuildVectorsResult{}, fmt.Errorf("qdrant vector index is not configured")
	}
	selection, client, err := s.selectEmbeddingClient()
	if err != nil {
		return RebuildVectorsResult{}, err
	}
	result := RebuildVectorsResult{
		EmbeddingModelID:   selection.Model.ID,
		EmbeddingModelName: selection.Model.Name,
	}
	result.EmbeddingDimension, err = s.probeEmbeddingDimension(ctx, selection, client)
	if err != nil {
		return RebuildVectorsResult{}, err
	}
	projects, err := s.store.ListProjects()
	if err != nil {
		return RebuildVectorsResult{}, err
	}
	result.ProjectCount = len(projects)
	versionsByProject := make(map[string][]domain.ChapterVersion, len(projects))
	for _, project := range projects {
		versions, err := s.store.ListChapterVersions(project.ID, "")
		if err != nil {
			return RebuildVectorsResult{}, err
		}
		versionsByProject[project.ID] = versions
		result.ChapterVersionCount += len(versions)
	}
	if err := s.vectors.RecreateCollection(ctx, result.EmbeddingDimension); err != nil {
		return RebuildVectorsResult{}, err
	}
	for _, project := range projects {
		for _, version := range versionsByProject[project.ID] {
			if _, err := s.store.UpdateChapterVersionIndexStatus(version.ID, "pending"); err != nil {
				return RebuildVectorsResult{}, err
			}
			if _, err := s.store.CreateIndexJob(domain.IndexJob{
				ProjectID:        version.ProjectID,
				ChapterID:        version.ChapterID,
				ChapterVersionID: version.ID,
				Kind:             "chapter_reindex",
				Status:           "pending",
				Payload: map[string]string{
					"trigger":              "vector_rebuild",
					"embedding_model_id":   selection.Model.ID,
					"embedding_model_name": selection.Model.Name,
					"embedding_dimension":  strconv.Itoa(result.EmbeddingDimension),
				},
			}); err != nil {
				return RebuildVectorsResult{}, err
			}
			result.JobCount++
		}
	}
	return result, nil
}

func (s *Service) selectEmbeddingClient() (agent.ModelSelection, provider.EmbeddingModelClient, error) {
	if s.router == nil || s.providers == nil {
		return agent.ModelSelection{}, nil, fmt.Errorf("embedding model routing is not configured")
	}
	selection, err := s.router.SelectEmbeddingModel()
	if err != nil {
		return agent.ModelSelection{}, nil, err
	}
	client, err := s.providers.NewEmbeddingClient(selection.Provider)
	if err != nil {
		return agent.ModelSelection{}, nil, err
	}
	return selection, client, nil
}

func (s *Service) probeEmbeddingDimension(ctx context.Context, selection agent.ModelSelection, client provider.EmbeddingModelClient) (int, error) {
	resp, err := client.Embed(ctx, provider.EmbeddingRequest{
		Model:  selection.Model.Name,
		Inputs: []string{"vector dimension probe"},
		Metadata: map[string]string{
			"trigger":            "vector_rebuild_probe",
			"embedding_model_id": selection.Model.ID,
		},
	})
	if err != nil {
		return 0, err
	}
	return embeddingDimensionFromResponse(selection, resp, "probe")
}

func embeddingDimensionFromResponse(selection agent.ModelSelection, resp provider.EmbeddingResponse, operation string) (int, error) {
	if len(resp.Vectors) != 1 || len(resp.Vectors[0]) == 0 {
		return 0, fmt.Errorf("embedding model returned %d vectors for one %s input", len(resp.Vectors), operation)
	}
	actualDimension := len(resp.Vectors[0])
	if selection.Model.Dimension > 0 && selection.Model.Dimension != actualDimension {
		return 0, fmt.Errorf("embedding model %q dimension mismatch: configured %d, provider returned %d", selection.Model.ID, selection.Model.Dimension, actualDimension)
	}
	return actualDimension, nil
}

func (s *Service) process(ctx context.Context, job domain.IndexJob) (domain.IndexJob, error) {
	if strings.TrimSpace(job.ChapterVersionID) == "" {
		failed, err := s.store.UpdateIndexJobStatus(job.ID, "failed", "index job has no chapter_version_id")
		if err != nil {
			return domain.IndexJob{}, err
		}
		return failed, fmt.Errorf("index job %q has no chapter_version_id", job.ID)
	}
	running, err := s.store.UpdateIndexJobStatus(job.ID, "running", "")
	if err != nil {
		return domain.IndexJob{}, err
	}
	version, err := s.store.GetChapterVersion(running.ChapterVersionID)
	if err != nil {
		return s.failJob(running, err)
	}
	if s.extractor != nil {
		if err := s.extractAndPersist(version); err != nil {
			return s.failJob(running, err)
		}
	}
	if s.vectors == nil {
		return s.failJob(running, fmt.Errorf("qdrant vector index is not configured; deterministic knowledge extraction completed before vector indexing failed"))
	}
	selection, client, err := s.selectEmbeddingClient()
	if err != nil {
		return s.failJob(running, err)
	}
	resp, err := client.Embed(ctx, provider.EmbeddingRequest{Model: selection.Model.Name, Inputs: []string{version.Content}, Metadata: map[string]string{"chapter_version_id": version.ID, "project_id": version.ProjectID}})
	if err != nil {
		return s.failJob(running, err)
	}
	dimension, err := embeddingDimensionFromResponse(selection, resp, "chapter")
	if err != nil {
		return s.failJob(running, err)
	}
	if err := s.vectors.EnsureCollection(ctx, dimension); err != nil {
		return s.failJob(running, err)
	}
	payload := vector.PointPayload{ProjectID: version.ProjectID, ChapterID: version.ChapterID, ChapterVersionID: version.ID, ContentType: "chapter_version", SourceID: version.ID, CanonStatus: version.IndexStatus}
	if err := s.vectors.UpsertTextVector(ctx, version.ID, resp.Vectors[0], payload); err != nil {
		return s.failJob(running, err)
	}
	if _, err := s.store.UpdateChapterVersionIndexStatus(version.ID, "indexed"); err != nil {
		return s.failJob(running, err)
	}
	completed, err := s.store.UpdateIndexJobStatus(running.ID, "completed", "")
	if err != nil {
		return domain.IndexJob{}, err
	}
	return completed, nil
}

func (s *Service) extractAndPersist(version domain.ChapterVersion) error {
	result, err := s.extractor.ExtractChapter(version)
	if err != nil {
		return fmt.Errorf("extract chapter knowledge: %w", err)
	}
	existingEntities, err := s.store.ListEntities(version.ProjectID)
	if err != nil {
		return err
	}
	entityIDsByNameType := map[string]string{}
	for _, item := range existingEntities {
		entityIDsByNameType[entityKey(item.Name, item.Type)] = item.ID
	}
	for _, entity := range result.Entities {
		key := entityKey(entity.Name, entity.Type)
		if _, exists := entityIDsByNameType[key]; exists {
			continue
		}
		saved, err := s.store.SaveEntity(entity)
		if err != nil {
			return err
		}
		entityIDsByNameType[key] = saved.ID
	}
	for _, fact := range result.Facts {
		if _, err := s.store.SaveFact(fact); err != nil {
			return err
		}
	}
	for _, thread := range result.PlotThreads {
		if _, err := s.store.SavePlotThread(thread); err != nil {
			return err
		}
	}
	for _, edge := range result.Edges {
		sourceName := firstText(edge.Metadata["source_entity_name"], edge.SourceEntityID)
		targetName := firstText(edge.Metadata["target_entity_name"], edge.TargetEntityID)
		sourceID := findEntityID(entityIDsByNameType, sourceName)
		targetID := findEntityID(entityIDsByNameType, targetName)
		if sourceID == "" || targetID == "" {
			return fmt.Errorf("cannot resolve extracted relation %q -> %q", sourceName, targetName)
		}
		edge.SourceEntityID = sourceID
		edge.TargetEntityID = targetID
		if _, err := s.store.SaveGraphEdge(edge); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) failJob(job domain.IndexJob, cause error) (domain.IndexJob, error) {
	message := "unknown indexing failure"
	if cause != nil {
		message = cause.Error()
	}
	failed, err := s.store.UpdateIndexJobStatus(job.ID, "failed", message)
	if err != nil {
		return domain.IndexJob{}, err
	}
	if job.ChapterVersionID != "" {
		if _, updateErr := s.store.UpdateChapterVersionIndexStatus(job.ChapterVersionID, "failed"); updateErr != nil {
			return domain.IndexJob{}, fmt.Errorf("mark chapter version %q index failed after job %q failure: %w", job.ChapterVersionID, job.ID, updateErr)
		}
	}
	return failed, cause
}

func entityKey(name, entityType string) string {
	return strings.ToLower(strings.TrimSpace(entityType)) + ":" + strings.TrimSpace(name)
}

func findEntityID(items map[string]string, name string) string {
	name = strings.TrimSpace(name)
	for key, id := range items {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) == 2 && parts[1] == name {
			return id
		}
	}
	return ""
}

func firstText(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
