package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"aeonechoes/server/internal/agent"
	"aeonechoes/server/internal/config"
	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/indexing"
	"aeonechoes/server/internal/provider"
	"aeonechoes/server/internal/repository"
	"aeonechoes/server/internal/retrieval"
	"aeonechoes/server/internal/skills"
	"aeonechoes/server/internal/tooling"
)

// Server owns HTTP routing and request handlers.
type Server struct {
	cfg               config.Config
	store             repository.AppStore
	providers         provider.ProviderFactory
	workflow          *agent.WorkflowRunner
	agentRuntime      *agent.Runtime
	skillService      *skills.Service
	toolRegistry      *tooling.Registry
	mcpDefaultTimeout time.Duration
	indexing          *indexing.Service
	retrieval         *retrieval.Service
	indexWake         indexing.WakeNotifier
	logger            *slog.Logger
}

func NewServer(cfg config.Config, store repository.AppStore, providers provider.ProviderFactory, workflow *agent.WorkflowRunner, indexingService *indexing.Service, retrievalService *retrieval.Service, indexWake indexing.WakeNotifier, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{cfg: cfg, store: store, providers: providers, workflow: workflow, mcpDefaultTimeout: cfg.MCPDefaultTimeout, indexing: indexingService, retrieval: retrievalService, indexWake: indexWake, logger: logger}
}

func (s *Server) ConfigureAgents(runtime *agent.Runtime, skillService *skills.Service, toolRegistry *tooling.Registry, mcpDefaultTimeout time.Duration) {
	s.agentRuntime = runtime
	s.skillService = skillService
	s.toolRegistry = toolRegistry
	if mcpDefaultTimeout > 0 {
		s.mcpDefaultTimeout = mcpDefaultTimeout
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/providers", s.listProviders)
	mux.HandleFunc("POST /api/providers", s.createProvider)
	mux.HandleFunc("GET /api/providers/{id}", s.getProvider)
	mux.HandleFunc("PUT /api/providers/{id}", s.updateProvider)
	mux.HandleFunc("DELETE /api/providers/{id}", s.deleteProvider)
	mux.HandleFunc("POST /api/providers/{id}/refresh-models", s.refreshProviderModels)
	mux.HandleFunc("GET /api/models", s.listModels)
	mux.HandleFunc("POST /api/models", s.createModel)
	mux.HandleFunc("GET /api/models/{id}", s.getModel)
	mux.HandleFunc("PUT /api/models/{id}", s.updateModel)
	mux.HandleFunc("DELETE /api/models/{id}", s.deleteModel)
	mux.HandleFunc("GET /api/projects", s.listProjects)
	mux.HandleFunc("GET /api/projects/{projectID}", s.getProject)
	mux.HandleFunc("POST /api/projects/initialize", s.initializeProject)
	mux.HandleFunc("POST /api/projects/seed/optimize", s.optimizeProjectSeed)
	mux.HandleFunc("GET /api/projects/{projectID}/story-bible", s.getStoryBible)
	mux.HandleFunc("PUT /api/projects/{projectID}/story-bible", s.updateStoryBible)
	mux.HandleFunc("POST /api/projects/{projectID}/characters/sync", s.syncCharacters)
	mux.HandleFunc("GET /api/projects/{projectID}/chapters", s.listChapters)
	mux.HandleFunc("POST /api/projects/{projectID}/chapters/ensure", s.ensureChapter)
	mux.HandleFunc("POST /api/projects/{projectID}/chapter-versions", s.createChapterVersion)
	mux.HandleFunc("GET /api/projects/{projectID}/chapter-versions", s.listChapterVersions)
	mux.HandleFunc("GET /api/workflows", s.listWorkflows)
	mux.HandleFunc("GET /api/workflows/{id}", s.getWorkflow)
	mux.HandleFunc("GET /api/settings", s.listSettings)
	mux.HandleFunc("PUT /api/settings/{scope}/{key}", s.upsertSetting)
	mux.HandleFunc("POST /api/retrieval/semantic-search", s.semanticSearch)
	mux.HandleFunc("GET /api/system/status", s.systemStatus)
	mux.HandleFunc("GET /api/graph/expand", s.expandGraph)
	mux.HandleFunc("GET /api/agents", s.listAgents)
	mux.HandleFunc("POST /api/agents", s.createAgent)
	mux.HandleFunc("GET /api/agents/{id}", s.getAgent)
	mux.HandleFunc("PUT /api/agents/{id}", s.updateAgent)
	mux.HandleFunc("DELETE /api/agents/{id}", s.deleteAgent)
	mux.HandleFunc("POST /api/agents/{id}/runs", s.runAgent)
	mux.HandleFunc("GET /api/agent-runs", s.listAgentRuns)
	mux.HandleFunc("GET /api/agent-runs/{id}", s.getAgentRun)
	mux.HandleFunc("GET /api/skills", s.listSkills)
	mux.HandleFunc("POST /api/skills", s.createSkill)
	mux.HandleFunc("GET /api/skills/{id}", s.getSkill)
	mux.HandleFunc("PUT /api/skills/{id}", s.updateSkill)
	mux.HandleFunc("DELETE /api/skills/{id}", s.deleteSkill)
	mux.HandleFunc("PUT /api/skills/{id}/enabled", s.setSkillEnabled)
	mux.HandleFunc("GET /api/skills/sources", s.listSkillSources)
	mux.HandleFunc("POST /api/skills/sources/default/scan", s.scanDefaultSkillSource)
	mux.HandleFunc("POST /api/skills/sources/{id}/scan", s.scanSkillSource)
	mux.HandleFunc("GET /api/mcp/servers", s.listMCPServers)
	mux.HandleFunc("POST /api/mcp/servers", s.createMCPServer)
	mux.HandleFunc("GET /api/mcp/servers/{id}", s.getMCPServer)
	mux.HandleFunc("PUT /api/mcp/servers/{id}", s.updateMCPServer)
	mux.HandleFunc("DELETE /api/mcp/servers/{id}", s.deleteMCPServer)
	mux.HandleFunc("PUT /api/mcp/servers/{id}/enabled", s.setMCPServerEnabled)
	mux.HandleFunc("POST /api/mcp/servers/{id}/test", s.testMCPServer)
	mux.HandleFunc("POST /api/mcp/servers/{id}/refresh-tools", s.refreshMCPTools)
	mux.HandleFunc("GET /api/mcp/servers/{id}/tools", s.listMCPServerTools)
	mux.HandleFunc("GET /api/tools/catalog", s.listToolCatalog)
	mux.HandleFunc("PUT /api/tools/catalog/{id}/enabled", s.setToolEnabled)
	mux.HandleFunc("GET /api/tools/invocations", s.listToolInvocations)
	mux.HandleFunc("GET /api/index/jobs", s.listIndexJobs)
	mux.HandleFunc("POST /api/index/jobs/{id}/run", s.runIndexJob)
	mux.HandleFunc("POST /api/index/run-pending", s.runPendingIndexJobs)
	mux.HandleFunc("POST /api/index/rebuild-vectors", s.rebuildVectors)
	return loggingMiddleware(s.logger, corsMiddleware(s.cfg.CORSAllowedOrigins, jsonMiddleware(mux)))
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "time": time.Now().UTC(), "qdrant_configured": s.cfg.QdrantURL != "", "postgres_configured": s.cfg.PostgresDSN != ""})
}

func (s *Server) listProjects(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListProjects()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) getProject(w http.ResponseWriter, r *http.Request) {
	item, err := s.store.GetProject(r.PathValue("projectID"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) listWorkflows(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListWorkflows(r.URL.Query().Get("project_id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) getWorkflow(w http.ResponseWriter, r *http.Request) {
	item, err := s.store.GetWorkflow(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) listSettings(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListSettings(r.URL.Query().Get("scope"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) upsertSetting(w http.ResponseWriter, r *http.Request) {
	var input domain.AppSetting
	if !decodeRequest(w, r, &input) {
		return
	}
	input.Scope = firstNonEmpty(r.PathValue("scope"), input.Scope)
	input.Key = firstNonEmpty(r.PathValue("key"), input.Key)
	updated, err := s.store.UpsertSetting(input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) semanticSearch(w http.ResponseWriter, r *http.Request) {
	if s.retrieval == nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Errorf("semantic retrieval is not configured"))
		return
	}
	var input domain.SemanticSearchRequest
	if !decodeRequest(w, r, &input) {
		return
	}
	result, err := s.retrieval.Search(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) systemStatus(w http.ResponseWriter, r *http.Request) {
	providers, err := s.store.ListProviders()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	models, err := s.store.ListModels()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	jobs, err := s.store.ListPendingIndexJobs("", 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, domain.SystemStatus{Status: "ok", PostgresConfigured: s.cfg.PostgresDSN != "", QdrantConfigured: s.cfg.QdrantURL != "", ProviderCount: len(providers), ModelCount: len(models), PendingJobsCount: len(jobs), CheckedAt: time.Now().UTC()})
}

func (s *Server) listProviders(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListProviders()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, providerResponses(items))
}

func (s *Server) createProvider(w http.ResponseWriter, r *http.Request) {
	var input providerConfigRequest
	if !decodeRequest(w, r, &input) {
		return
	}
	created, err := s.store.CreateProvider(input.toDomain())
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, providerResponse(created))
}

func (s *Server) getProvider(w http.ResponseWriter, r *http.Request) {
	item, err := s.store.GetProvider(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, providerResponse(item))
}

func (s *Server) updateProvider(w http.ResponseWriter, r *http.Request) {
	var input providerConfigRequest
	if !decodeRequest(w, r, &input) {
		return
	}
	id := r.PathValue("id")
	existing, err := s.store.GetProvider(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	updated, err := s.store.UpdateProvider(id, input.applyTo(existing))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, providerResponse(updated))
}

func (s *Server) deleteProvider(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteProvider(r.PathValue("id")); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) refreshProviderModels(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.store.GetProvider(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	client, err := s.providers.NewModelListClient(cfg)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	infos, err := client.ListModels(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	models := make([]domain.ModelConfig, 0, len(infos))
	now := time.Now().UTC()
	for _, info := range infos {
		id := fmt.Sprintf("%s:%s", cfg.ID, info.ID)
		discovered := discoveredModelConfig(cfg, info, now)
		model, err := s.store.GetModel(id)
		if err == nil {
			model = mergeDiscoveredModel(model, discovered, info)
			model, err = s.store.UpdateModel(id, model)
		} else if strings.Contains(err.Error(), "not found") {
			model, err = s.store.CreateModel(discovered)
		}
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		models = append(models, model)
	}
	if err := s.store.TouchProviderModelRefresh(cfg.ID); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	refreshedProvider, err := s.store.GetProvider(cfg.ID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"models": models, "count": len(models), "provider": providerResponse(refreshedProvider)})
}

func discoveredModelConfig(cfg domain.ProviderConfig, info provider.ModelInfo, seenAt time.Time) domain.ModelConfig {
	return domain.ModelConfig{
		ID:                fmt.Sprintf("%s:%s", cfg.ID, info.ID),
		ProviderID:        cfg.ID,
		ProviderType:      cfg.Type,
		Name:              info.ID,
		DisplayName:       firstNonEmpty(info.DisplayName, info.Name, info.ID),
		Kind:              info.Kind,
		ContextWindow:     info.ContextWindow,
		MaxOutputTokens:   info.MaxOutputTokens,
		Dimension:         info.Dimension,
		SupportsTools:     info.SupportsTools,
		SupportsStreaming: info.SupportsStream,
		Enabled:           true,
		RoutingWeight:     100,
		LastSeenAt:        &seenAt,
	}
}

func mergeDiscoveredModel(existing, discovered domain.ModelConfig, info provider.ModelInfo) domain.ModelConfig {
	existing.ProviderID = discovered.ProviderID
	existing.ProviderType = discovered.ProviderType
	existing.Name = discovered.Name
	existing.DisplayName = discovered.DisplayName
	existing.Kind = discovered.Kind
	if discovered.ContextWindow > 0 {
		existing.ContextWindow = discovered.ContextWindow
	}
	if discovered.MaxOutputTokens > 0 {
		existing.MaxOutputTokens = discovered.MaxOutputTokens
	}
	if discovered.Dimension > 0 {
		existing.Dimension = discovered.Dimension
	}
	if info.SupportsToolsKnown {
		existing.SupportsTools = discovered.SupportsTools
	}
	if info.SupportsStreamKnown {
		existing.SupportsStreaming = discovered.SupportsStreaming
	}
	existing.LastSeenAt = discovered.LastSeenAt
	return existing
}

func (s *Server) listModels(w http.ResponseWriter, r *http.Request) {
	kind := domain.ModelKind(r.URL.Query().Get("kind"))
	if kind != "" {
		if !kind.Valid() {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid model kind %q", kind))
			return
		}
		items, err := s.store.ListModelsByKind(kind)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
		return
	}
	items, err := s.store.ListModels()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) createModel(w http.ResponseWriter, r *http.Request) {
	var input domain.ModelConfig
	if !decodeRequest(w, r, &input) {
		return
	}
	created, err := s.store.CreateModel(input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) getModel(w http.ResponseWriter, r *http.Request) {
	item, err := s.store.GetModel(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) updateModel(w http.ResponseWriter, r *http.Request) {
	var input domain.ModelConfig
	if !decodeRequest(w, r, &input) {
		return
	}
	updated, err := s.store.UpdateModel(r.PathValue("id"), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) deleteModel(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteModel(r.PathValue("id")); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) initializeProject(w http.ResponseWriter, r *http.Request) {
	var seed domain.ProjectSeed
	if !decodeRequest(w, r, &seed) {
		return
	}
	result, err := s.workflow.InitializeProject(r.Context(), seed)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) optimizeProjectSeed(w http.ResponseWriter, r *http.Request) {
	var seed domain.ProjectSeed
	if !decodeRequest(w, r, &seed) {
		return
	}
	if strings.TrimSpace(seed.Title) == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("project seed title must not be empty"))
		return
	}
	if strings.TrimSpace(seed.Premise) == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("project seed premise must not be empty"))
		return
	}
	if seed.Language == "" {
		seed.Language = "zh-CN"
	}
	if seed.TargetChapters <= 0 {
		seed.TargetChapters = 12
	}
	if seed.Metadata == nil {
		seed.Metadata = map[string]string{}
	}
	seed.Metadata["optimized_prompt"] = buildOptimizedPrompt(seed)
	writeJSON(w, http.StatusOK, seed)
}

func (s *Server) getStoryBible(w http.ResponseWriter, r *http.Request) {
	bible, err := s.store.GetStoryBible(r.PathValue("projectID"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, bible)
}

func (s *Server) updateStoryBible(w http.ResponseWriter, r *http.Request) {
	var bible domain.StoryBible
	if !decodeRequest(w, r, &bible) {
		return
	}
	updated, err := s.store.UpdateStoryBible(r.PathValue("projectID"), bible)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) syncCharacters(w http.ResponseWriter, r *http.Request) {
	var input characterSyncRequest
	if !decodeRequest(w, r, &input) {
		return
	}
	input.ProjectID = firstNonEmpty(r.PathValue("projectID"), input.ProjectID)
	result, err := syncStoryBibleCharacters(s.store, input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) listChapters(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListChapters(r.PathValue("projectID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) ensureChapter(w http.ResponseWriter, r *http.Request) {
	var input domain.ChapterEnsureRequest
	if !decodeRequest(w, r, &input) {
		return
	}
	input.ProjectID = firstNonEmpty(r.PathValue("projectID"), input.ProjectID)
	chapter, err := s.store.EnsureChapter(input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, chapter)
}

func (s *Server) createChapterVersion(w http.ResponseWriter, r *http.Request) {
	var input domain.ChapterVersion
	if !decodeRequest(w, r, &input) {
		return
	}
	input.ProjectID = r.PathValue("projectID")
	created, job, err := s.store.SaveChapterVersion(input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.notifyIndexWorker()
	writeJSON(w, http.StatusCreated, map[string]any{"chapter_version": created, "index_job": job})
}

func (s *Server) listChapterVersions(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListChapterVersions(r.PathValue("projectID"), r.URL.Query().Get("chapter_id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) expandGraph(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	ids := splitCSV(r.URL.Query().Get("entity_ids"))
	depth := 1
	if raw := r.URL.Query().Get("depth"); raw != "" {
		_, err := fmt.Sscanf(raw, "%d", &depth)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("depth must be an integer"))
			return
		}
	}
	expansion, err := s.store.ExpandGraph(projectID, ids, depth)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, expansion)
}

func (s *Server) generateChapterIdea(w http.ResponseWriter, r *http.Request) {
	if s.workflow == nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Errorf("workflow runner is not configured"))
		return
	}
	var input agent.ChapterIdeaRequest
	if !decodeRequest(w, r, &input) {
		return
	}
	result, err := s.workflow.GenerateChapterIdea(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) generateCharacterProfiles(w http.ResponseWriter, r *http.Request) {
	if s.workflow == nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Errorf("workflow runner is not configured"))
		return
	}
	var input agent.CharacterProfilesRequest
	if !decodeRequest(w, r, &input) {
		return
	}
	result, err := s.workflow.GenerateCharacterProfiles(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) previewContextSelection(w http.ResponseWriter, r *http.Request) {
	if s.workflow == nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Errorf("workflow runner is not configured"))
		return
	}
	var input agent.ContextSelectionPreviewRequest
	if !decodeRequest(w, r, &input) {
		return
	}
	result, err := s.workflow.PreviewContextSelection(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) draft(w http.ResponseWriter, r *http.Request) {
	if s.workflow == nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Errorf("workflow runner is not configured"))
		return
	}
	var input agent.DraftRequest
	if !decodeRequest(w, r, &input) {
		return
	}
	result, err := s.workflow.DraftChapter(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.notifyIndexWorker()
	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) draftWithIdea(w http.ResponseWriter, r *http.Request) {
	if s.workflow == nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Errorf("workflow runner is not configured"))
		return
	}
	var input agent.DraftWithIdeaRequest
	if !decodeRequest(w, r, &input) {
		return
	}
	result, err := s.workflow.DraftChapterWithIdea(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.notifyIndexWorker()
	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) listIndexJobs(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limit := 0
	if raw := strings.TrimSpace(query.Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, fmt.Errorf("limit must be a positive integer"))
			return
		}
		limit = parsed
	}
	items, err := s.store.ListIndexJobs(repository.IndexJobFilter{ProjectID: query.Get("project_id"), Status: query.Get("status"), Limit: limit})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) runIndexJob(w http.ResponseWriter, r *http.Request) {
	if s.indexing == nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Errorf("indexing service is not configured"))
		return
	}
	job, err := s.indexing.RunJob(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (s *Server) runPendingIndexJobs(w http.ResponseWriter, r *http.Request) {
	if s.indexing == nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Errorf("indexing service is not configured"))
		return
	}
	limit := 10
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, fmt.Errorf("limit must be a positive integer"))
			return
		}
		limit = parsed
	}
	result, err := s.indexing.RunPending(r.Context(), r.URL.Query().Get("project_id"), limit)
	if err != nil {
		if result.Count > 0 || len(result.Processed) > 0 {
			writeJSON(w, http.StatusOK, result)
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) rebuildVectors(w http.ResponseWriter, r *http.Request) {
	if s.indexing == nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Errorf("indexing service is not configured"))
		return
	}
	result, err := s.indexing.RebuildVectors(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) notifyIndexWorker() {
	if s == nil || s.indexWake == nil {
		return
	}
	s.indexWake.Notify()
}

type characterSyncRequest struct {
	ProjectID    string                    `json:"project_id,omitempty"`
	StoryBibleID string                    `json:"story_bible_id,omitempty"`
	Source       string                    `json:"source,omitempty"`
	Characters   []domain.CharacterProfile `json:"characters"`
	Metadata     map[string]string         `json:"metadata,omitempty"`
}

type characterSyncResult struct {
	ProjectID    string                           `json:"project_id"`
	StoryBibleID string                           `json:"story_bible_id,omitempty"`
	Characters   []domain.Entity                  `json:"characters"`
	Mappings     []domain.CharacterProfileMapping `json:"mappings"`
}

func syncStoryBibleCharacters(store repository.AppStore, input characterSyncRequest) (characterSyncResult, error) {
	if store == nil {
		return characterSyncResult{}, fmt.Errorf("character sync store is not configured")
	}
	projectID := strings.TrimSpace(input.ProjectID)
	if projectID == "" {
		return characterSyncResult{}, fmt.Errorf("character sync project_id must not be empty")
	}
	if _, err := store.GetProject(projectID); err != nil {
		return characterSyncResult{}, err
	}
	if len(input.Characters) == 0 {
		return characterSyncResult{}, fmt.Errorf("character sync characters must not be empty")
	}
	characters, err := normalizeCharacterSyncProfiles(input.Characters)
	if err != nil {
		return characterSyncResult{}, err
	}
	requestedNames := characterNameSet(characters)
	existing, err := store.ListEntities(projectID)
	if err != nil {
		return characterSyncResult{}, err
	}
	byName := make(map[string]domain.Entity, len(existing))
	for _, entity := range existing {
		nameKey := normalizedCharacterName(entity.Name)
		if nameKey == "" {
			continue
		}
		if entity.Type != "" && entity.Type != "character" {
			if _, requested := requestedNames[nameKey]; requested {
				return characterSyncResult{}, fmt.Errorf("character sync name %q conflicts with existing non-character entity %q of type %q", entity.Name, entity.ID, entity.Type)
			}
			continue
		}
		if previous, ok := byName[nameKey]; ok {
			return characterSyncResult{}, fmt.Errorf("character sync found duplicate existing character name %q for entities %q and %q", entity.Name, previous.ID, entity.ID)
		}
		byName[nameKey] = entity
	}
	result := characterSyncResult{ProjectID: projectID, StoryBibleID: strings.TrimSpace(input.StoryBibleID), Characters: make([]domain.Entity, 0, len(characters)), Mappings: make([]domain.CharacterProfileMapping, 0, len(characters))}
	for _, profile := range characters {
		nameKey := normalizedCharacterName(profile.Name)
		existingEntity, exists := byName[nameKey]
		entity := characterProfileEntity(projectID, profile, existingEntity, input)
		saved, err := store.SaveEntity(entity)
		if err != nil {
			return characterSyncResult{}, err
		}
		action := "created"
		if exists {
			action = "updated"
		}
		byName[nameKey] = saved
		result.Characters = append(result.Characters, saved)
		result.Mappings = append(result.Mappings, domain.CharacterProfileMapping{Name: saved.Name, EntityID: saved.ID, Action: action})
	}
	return result, nil
}

func normalizeCharacterSyncProfiles(input []domain.CharacterProfile) ([]domain.CharacterProfile, error) {
	characters := make([]domain.CharacterProfile, 0, len(input))
	seen := map[string]struct{}{}
	for i, character := range input {
		character.Name = strings.TrimSpace(character.Name)
		character.Role = strings.TrimSpace(character.Role)
		character.Desire = strings.TrimSpace(character.Desire)
		character.Wound = strings.TrimSpace(character.Wound)
		character.Secret = strings.TrimSpace(character.Secret)
		character.Summary = strings.TrimSpace(character.Summary)
		if character.Name == "" {
			return nil, fmt.Errorf("character sync characters[%d].name must not be empty", i)
		}
		if character.Role == "" {
			return nil, fmt.Errorf("character sync characters[%d].role must not be empty", i)
		}
		if character.Desire == "" {
			return nil, fmt.Errorf("character sync characters[%d].desire must not be empty", i)
		}
		if character.Wound == "" {
			return nil, fmt.Errorf("character sync characters[%d].wound must not be empty", i)
		}
		nameKey := normalizedCharacterName(character.Name)
		if _, ok := seen[nameKey]; ok {
			return nil, fmt.Errorf("character sync duplicate character name %q", character.Name)
		}
		seen[nameKey] = struct{}{}
		characters = append(characters, character)
	}
	return characters, nil
}

func characterNameSet(characters []domain.CharacterProfile) map[string]struct{} {
	set := make(map[string]struct{}, len(characters))
	for _, character := range characters {
		set[normalizedCharacterName(character.Name)] = struct{}{}
	}
	return set
}

func characterProfileEntity(projectID string, profile domain.CharacterProfile, existing domain.Entity, input characterSyncRequest) domain.Entity {
	entity := existing
	entity.ProjectID = projectID
	entity.Name = profile.Name
	entity.Type = "character"
	entity.Summary = characterProfileSummary(profile)
	if entity.Status == "" {
		entity.Status = "active"
	}
	if entity.Importance <= 0 {
		entity.Importance = characterImportance(profile.Role)
	}
	traits := map[string]string{}
	for key, value := range entity.Traits {
		traits[key] = value
	}
	traits["role"] = profile.Role
	traits["desire"] = profile.Desire
	traits["wound"] = profile.Wound
	if profile.Secret != "" {
		traits["secret"] = profile.Secret
	} else {
		delete(traits, "secret")
	}
	if profile.Summary != "" {
		traits["summary"] = profile.Summary
	}
	entity.Traits = traits
	metadata := map[string]string{}
	for key, value := range entity.Metadata {
		metadata[key] = value
	}
	for key, value := range input.Metadata {
		metadata["sync_"+key] = value
	}
	if strings.TrimSpace(input.StoryBibleID) != "" {
		metadata["story_bible_id"] = strings.TrimSpace(input.StoryBibleID)
	}
	if strings.TrimSpace(input.Source) != "" {
		metadata["source"] = strings.TrimSpace(input.Source)
	}
	metadata["source_layer"] = "story_bible"
	metadata["character_profile_json"] = mustMarshalCharacterProfile(profile)
	entity.Metadata = metadata
	return entity
}

func characterProfileSummary(profile domain.CharacterProfile) string {
	if strings.TrimSpace(profile.Summary) != "" {
		return strings.TrimSpace(profile.Summary)
	}
	parts := []string{
		fmt.Sprintf("角色定位：%s", profile.Role),
		fmt.Sprintf("欲望：%s", profile.Desire),
		fmt.Sprintf("创伤：%s", profile.Wound),
	}
	if strings.TrimSpace(profile.Secret) != "" {
		parts = append(parts, fmt.Sprintf("秘密：%s", profile.Secret))
	}
	return strings.Join(parts, "；") + "。"
}

func characterImportance(role string) int {
	role = strings.ToLower(strings.TrimSpace(role))
	if strings.Contains(role, "主角") || strings.Contains(role, "protagonist") || strings.Contains(role, "main") {
		return 100
	}
	if strings.Contains(role, "反派") || strings.Contains(role, "antagonist") || strings.Contains(role, "主要") || strings.Contains(role, "配角") {
		return 80
	}
	return 60
}

func mustMarshalCharacterProfile(profile domain.CharacterProfile) string {
	payload, err := json.Marshal(profile)
	if err != nil {
		panic(fmt.Sprintf("marshal character profile: %v", err))
	}
	return string(payload)
}

func normalizedCharacterName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

type providerConfigRequest struct {
	ID                       string              `json:"id"`
	Name                     string              `json:"name"`
	Type                     domain.ProviderType `json:"type"`
	ProviderType             domain.ProviderType `json:"provider_type"`
	BaseURL                  string              `json:"base_url"`
	APIKey                   *string             `json:"api_key,omitempty"`
	APIKeyEnv                *string             `json:"api_key_env,omitempty"`
	APIKeyHint               string              `json:"api_key_hint,omitempty"`
	Enabled                  *bool               `json:"enabled"`
	Streaming                *bool               `json:"streaming,omitempty"`
	TraceEnabled             *bool               `json:"trace_enabled"`
	TraceRetentionDays       int                 `json:"trace_retention_days"`
	DefaultRequestTimeoutSec int                 `json:"default_request_timeout_sec"`
	DefaultModelID           *string             `json:"default_model_id,omitempty"`
	Metadata                 map[string]string   `json:"metadata,omitempty"`
	Status                   string              `json:"status,omitempty"`
	LastCheckedAt            *time.Time          `json:"last_checked_at,omitempty"`
	LastModelRefreshAt       *time.Time          `json:"last_model_refresh_at,omitempty"`
	CreatedAt                time.Time           `json:"created_at,omitempty"`
	UpdatedAt                time.Time           `json:"updated_at,omitempty"`
}

func (p providerConfigRequest) toDomain() domain.ProviderConfig {
	providerType := p.Type
	if providerType == "" {
		providerType = p.ProviderType
	}
	cfg := domain.ProviderConfig{ID: p.ID, Name: p.Name, Type: providerType, BaseURL: p.BaseURL, Enabled: true, TraceRetentionDays: p.TraceRetentionDays, DefaultRequestTimeoutSec: p.DefaultRequestTimeoutSec, LastModelRefreshAt: p.LastModelRefreshAt}
	if p.Enabled != nil {
		cfg.Enabled = *p.Enabled
	}
	if p.TraceEnabled != nil {
		cfg.TraceEnabled = *p.TraceEnabled
	}
	if p.APIKey != nil {
		cfg.APIKey = *p.APIKey
	}
	cfg.APIKeyEnv = ""
	cfg.Metadata = p.normalizedMetadata(nil)
	return cfg
}

func (p providerConfigRequest) applyTo(existing domain.ProviderConfig) domain.ProviderConfig {
	providerType := p.Type
	if providerType == "" {
		providerType = p.ProviderType
	}
	if providerType != "" {
		existing.Type = providerType
	}
	if strings.TrimSpace(p.Name) != "" {
		existing.Name = p.Name
	}
	if strings.TrimSpace(p.BaseURL) != "" {
		existing.BaseURL = p.BaseURL
	}
	if p.APIKey != nil {
		existing.APIKey = *p.APIKey
	}
	existing.APIKeyEnv = ""
	if p.Enabled != nil {
		existing.Enabled = *p.Enabled
	}
	if p.TraceEnabled != nil {
		existing.TraceEnabled = *p.TraceEnabled
	}
	if p.TraceRetentionDays > 0 {
		existing.TraceRetentionDays = p.TraceRetentionDays
	}
	if p.DefaultRequestTimeoutSec > 0 {
		existing.DefaultRequestTimeoutSec = p.DefaultRequestTimeoutSec
	}
	existing.Metadata = p.normalizedMetadata(existing.Metadata)
	if p.LastModelRefreshAt != nil {
		existing.LastModelRefreshAt = p.LastModelRefreshAt
	}
	return existing
}

func (p providerConfigRequest) normalizedMetadata(existing map[string]string) map[string]string {
	metadata := map[string]string{}
	if p.Metadata != nil {
		for key, value := range p.Metadata {
			metadata[key] = value
		}
	} else {
		for key, value := range existing {
			metadata[key] = value
		}
	}
	if p.Streaming != nil {
		metadata["streaming"] = strconv.FormatBool(*p.Streaming)
	}
	if p.DefaultModelID != nil {
		if strings.TrimSpace(*p.DefaultModelID) != "" {
			metadata["default_model_id"] = strings.TrimSpace(*p.DefaultModelID)
		} else {
			delete(metadata, "default_model_id")
		}
	}
	return metadata
}

func providerResponses(items []domain.ProviderConfig) []map[string]any {
	responses := make([]map[string]any, 0, len(items))
	for _, item := range items {
		responses = append(responses, providerResponse(item))
	}
	return responses
}

func providerResponse(item domain.ProviderConfig) map[string]any {
	metadata := map[string]string{}
	for key, value := range item.Metadata {
		metadata[key] = value
	}
	response := map[string]any{
		"id":                          item.ID,
		"name":                        item.Name,
		"type":                        item.Type,
		"provider_type":               item.Type,
		"base_url":                    item.BaseURL,
		"api_key_hint":                providerAPIKeyHint(item),
		"enabled":                     item.Enabled,
		"streaming":                   metadata["streaming"] == "true",
		"trace_enabled":               item.TraceEnabled,
		"trace_retention_days":        item.TraceRetentionDays,
		"default_request_timeout_sec": item.DefaultRequestTimeoutSec,
		"metadata":                    metadata,
		"created_at":                  item.CreatedAt,
		"updated_at":                  item.UpdatedAt,
		"last_model_refresh_at":       item.LastModelRefreshAt,
		"last_checked_at":             item.LastModelRefreshAt,
		"default_model_id":            metadata["default_model_id"],
		"status":                      providerStatus(item),
	}
	return response
}

func providerAPIKeyHint(item domain.ProviderConfig) string {
	if strings.TrimSpace(item.APIKey) != "" {
		return "configured"
	}
	return ""
}

func providerStatus(item domain.ProviderConfig) string {
	if !item.Enabled {
		return "offline"
	}
	if item.LastModelRefreshAt != nil {
		return "online"
	}
	return "unknown"
}

func buildOptimizedPrompt(seed domain.ProjectSeed) string {
	parts := []string{
		fmt.Sprintf("标题：%s", seed.Title),
		fmt.Sprintf("核心设定：%s", seed.Premise),
		fmt.Sprintf("类型 / 语气 / 读者：%s / %s / %s", firstNonEmpty(seed.Genre, "未分类"), firstNonEmpty(seed.Tone, "稳健、清晰"), firstNonEmpty(seed.Audience, "通用读者")),
		fmt.Sprintf("舞台：%s", firstNonEmpty(seed.Setting, "待扩展")),
	}
	if len(seed.Themes) > 0 {
		parts = append(parts, "主题："+strings.Join(seed.Themes, "、"))
	}
	if len(seed.MainCharacters) > 0 {
		parts = append(parts, "关键角色："+strings.Join(seed.MainCharacters, "、"))
	}
	if len(seed.Constraints) > 0 {
		parts = append(parts, "约束："+strings.Join(seed.Constraints, "；"))
	}
	return strings.Join(parts, "\n")
}

func decodeRequest(w http.ResponseWriter, r *http.Request, out any) bool {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		if errors.Is(err, http.ErrBodyReadAfterClose) {
			writeError(w, http.StatusBadRequest, fmt.Errorf("request body is not readable: %w", err))
			return false
		}
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid JSON request body: %w", err))
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Default().Error("encode HTTP response failed", "error", err)
	}
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{"error": err.Error(), "status": status})
}

func jsonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

func corsMiddleware(allowedOrigins []string, next http.Handler) http.Handler {
	allowed := map[string]bool{}
	for _, origin := range allowedOrigins {
		if strings.TrimSpace(origin) != "" {
			allowed[strings.TrimSpace(origin)] = true
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && (allowed[origin] || allowed["*"]) {
			if allowed["*"] {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept,Authorization,Content-Type")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("http request", "method", r.Method, "path", r.URL.Path, "duration_ms", time.Since(start).Milliseconds())
	})
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
