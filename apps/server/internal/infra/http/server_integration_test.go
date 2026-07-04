package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aeonechoes/server/internal/agent"
	"aeonechoes/server/internal/config"
	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/indexing"
	httpapi "aeonechoes/server/internal/infra/http"
	"aeonechoes/server/internal/memory"
	"aeonechoes/server/internal/provider"
	"aeonechoes/server/internal/providerregistry"
	"aeonechoes/server/internal/skills"
	"aeonechoes/server/internal/tooling"
	"aeonechoes/server/internal/vector"
)

type integrationFakeEmbeddingClient struct {
	vectors [][]float64
}

func (c *integrationFakeEmbeddingClient) Embed(ctx context.Context, req provider.EmbeddingRequest) (provider.EmbeddingResponse, error) {
	if len(c.vectors) == 0 {
		return provider.EmbeddingResponse{}, fmt.Errorf("integration fake embedding vectors are not configured")
	}
	return provider.EmbeddingResponse{Vectors: c.vectors}, nil
}

type integrationFakeProviderFactory struct {
	client provider.EmbeddingModelClient
}

func (f integrationFakeProviderFactory) NewEmbeddingClient(cfg domain.ProviderConfig) (provider.EmbeddingModelClient, error) {
	if f.client == nil {
		return nil, fmt.Errorf("integration fake embedding client is not configured")
	}
	return f.client, nil
}

type integrationFakeVectorIndex struct {
	recreatedDimensions []int
}

func (v *integrationFakeVectorIndex) EnsureCollection(ctx context.Context, dimension int) error {
	return nil
}
func (v *integrationFakeVectorIndex) RecreateCollection(ctx context.Context, dimension int) error {
	v.recreatedDimensions = append(v.recreatedDimensions, dimension)
	return nil
}
func (v *integrationFakeVectorIndex) UpsertTextVector(ctx context.Context, pointID string, values []float64, payload vector.PointPayload) error {
	return nil
}
func (v *integrationFakeVectorIndex) DeleteBySource(ctx context.Context, sourceID string) error {
	return nil
}
func (v *integrationFakeVectorIndex) Health(ctx context.Context) error { return nil }

type integrationWakeNotifier struct {
	count int
}

func (n *integrationWakeNotifier) Notify() {
	n.count++
}

func TestHandlerProjectSmokePaths(t *testing.T) {
	handler := newSmokeTestHandler(t)

	healthResponse := sendJSON(t, handler, http.MethodGet, "/api/health", nil)
	assertStatus(t, healthResponse, http.StatusOK)
	var health struct {
		Status string `json:"status"`
	}
	decodeJSON(t, healthResponse, &health)
	if health.Status != "ok" {
		t.Fatalf("GET /api/health status = %q, want ok", health.Status)
	}

	invalidInitializeResponse := sendJSON(t, handler, http.MethodPost, "/api/projects/initialize", domain.ProjectSeed{Premise: "失落舰队重返群星"})
	assertStatus(t, invalidInitializeResponse, http.StatusBadRequest)

	seed := domain.ProjectSeed{
		Title:          "星海回声",
		Premise:        "远航者寻找失落文明",
		Genre:          "科幻",
		Tone:           "辽阔",
		Audience:       "青年读者",
		Language:       "zh-CN",
		Setting:        "边境星域",
		Themes:         []string{"记忆", "归途"},
		MainCharacters: []string{"林烬"},
	}
	initializeResponse := sendJSON(t, handler, http.MethodPost, "/api/projects/initialize", seed)
	assertStatus(t, initializeResponse, http.StatusCreated)
	var initialized struct {
		Project    domain.Project    `json:"project"`
		StoryBible domain.StoryBible `json:"story_bible"`
		Workflow   domain.AIWorkflow `json:"workflow"`
	}
	decodeJSON(t, initializeResponse, &initialized)
	if initialized.Project.ID == "" {
		t.Fatalf("POST /api/projects/initialize project.id is empty")
	}
	if initialized.StoryBible.ID == "" {
		t.Fatalf("POST /api/projects/initialize story_bible.id is empty")
	}
	if initialized.Workflow.Output["mode"] != "rule_based_genesis" {
		t.Fatalf("POST /api/projects/initialize workflow.output.mode = %q, want rule_based_genesis", initialized.Workflow.Output["mode"])
	}

	projectsResponse := sendJSON(t, handler, http.MethodGet, "/api/projects", nil)
	assertStatus(t, projectsResponse, http.StatusOK)
	var projects []domain.Project
	decodeJSON(t, projectsResponse, &projects)
	if !containsProject(projects, initialized.Project.ID) {
		t.Fatalf("GET /api/projects did not include initialized project %q: %#v", initialized.Project.ID, projects)
	}

	bibleResponse := sendJSON(t, handler, http.MethodGet, "/api/projects/"+initialized.Project.ID+"/story-bible", nil)
	assertStatus(t, bibleResponse, http.StatusOK)
	var bible domain.StoryBible
	decodeJSON(t, bibleResponse, &bible)
	if bible.ProjectID != initialized.Project.ID {
		t.Fatalf("GET /api/projects/{id}/story-bible project_id = %q, want %q", bible.ProjectID, initialized.Project.ID)
	}
}

func TestHandlerRebuildVectorsEndpoint(t *testing.T) {
	store := memory.NewStore()
	providerCfg, err := store.CreateProvider(domain.ProviderConfig{ID: "provider_embed", Name: "Embedding", Type: domain.ProviderOpenAI, Enabled: true})
	if err != nil {
		t.Fatalf("CreateProvider() error: %v", err)
	}
	model, err := store.CreateModel(domain.ModelConfig{ID: "model_embed", ProviderID: providerCfg.ID, Name: "text-embedding-3-small", Kind: domain.ModelKindEmbedding, Dimension: 3, Enabled: true, DefaultForKind: true})
	if err != nil {
		t.Fatalf("CreateModel() error: %v", err)
	}
	project, _, err := store.CreateProject(domain.Project{Title: "重建测试", Slug: "rebuild"}, domain.StoryBible{Title: "重建测试", Logline: "测试"})
	if err != nil {
		t.Fatalf("CreateProject() error: %v", err)
	}
	_, _, err = store.SaveChapterVersion(domain.ChapterVersion{ProjectID: project.ID, ChapterID: "chapter_1", Title: "第一章", Content: "测试内容", IndexStatus: "indexed"})
	if err != nil {
		t.Fatalf("SaveChapterVersion() error: %v", err)
	}
	vectors := &integrationFakeVectorIndex{}
	indexingService := indexing.NewService(store, agent.NewModelRouter(store, agent.NewAgentRoleRegistry()), integrationFakeProviderFactory{client: &integrationFakeEmbeddingClient{vectors: [][]float64{{0.1, 0.2, 0.3}}}}, vectors)
	server := httpapi.NewServer(config.Config{Host: "127.0.0.1", Port: 1, DataDir: t.TempDir(), DefaultProviderTimeout: time.Second}, store, providerregistry.New(nil, time.Second), nil, indexingService, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	response := sendJSON(t, server.Handler(), http.MethodPost, "/api/index/rebuild-vectors", map[string]any{})
	assertStatus(t, response, http.StatusOK)
	var body struct {
		EmbeddingModelID    string `json:"embedding_model_id"`
		EmbeddingModelName  string `json:"embedding_model_name"`
		EmbeddingDimension  int    `json:"embedding_dimension"`
		ProjectCount        int    `json:"project_count"`
		ChapterVersionCount int    `json:"chapter_version_count"`
		JobCount            int    `json:"job_count"`
	}
	decodeJSON(t, response, &body)
	if body.EmbeddingModelID != model.ID || body.EmbeddingModelName != model.Name || body.EmbeddingDimension != 3 {
		t.Fatalf("unexpected rebuild vectors response model info: %+v", body)
	}
	if body.ProjectCount != 1 || body.ChapterVersionCount != 1 || body.JobCount != 1 {
		t.Fatalf("unexpected rebuild vectors response counts: %+v", body)
	}
	if len(vectors.recreatedDimensions) != 1 || vectors.recreatedDimensions[0] != 3 {
		t.Fatalf("expected collection recreated with dimension 3, got %+v", vectors.recreatedDimensions)
	}
}

func TestHandlerCharacterSyncUpsertsStoryBibleProfiles(t *testing.T) {
	store := memory.NewStore()
	project, bible, err := store.CreateProject(domain.Project{Title: "角色同步", Slug: "characters", Status: "active"}, domain.StoryBible{Title: "角色同步", Logline: "测试"})
	if err != nil {
		t.Fatalf("CreateProject() error: %v", err)
	}
	existing, err := store.SaveEntity(domain.Entity{ProjectID: project.ID, Name: "林烬", Type: "character", Summary: "旧摘要", Traits: map[string]string{"role": "旧定位", "secret": "旧秘密"}, Metadata: map[string]string{"kept": "true"}})
	if err != nil {
		t.Fatalf("SaveEntity(existing) error: %v", err)
	}
	server := httpapi.NewServer(config.Config{Host: "127.0.0.1", Port: 1, DataDir: t.TempDir(), DefaultProviderTimeout: time.Second}, store, providerregistry.New(nil, time.Second), nil, nil, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	response := sendJSON(t, server.Handler(), http.MethodPost, "/api/projects/"+project.ID+"/characters/sync", map[string]any{
		"story_bible_id": bible.ID,
		"source":         "story_bible_editor",
		"characters": []map[string]any{
			{"name": "林烬", "role": "主角", "desire": "找回失落舰队真相", "wound": "曾在撤离中放弃同伴", "summary": "背负旧债的远航者。"},
			{"name": "苏九", "role": "主要配角", "desire": "破解灰塔钟声", "wound": "家族因灰塔蒙冤", "secret": "她能听懂钟声里的坐标"},
		},
	})
	assertStatus(t, response, http.StatusOK)
	var body struct {
		ProjectID    string          `json:"project_id"`
		StoryBibleID string          `json:"story_bible_id"`
		Characters   []domain.Entity `json:"characters"`
		Mappings     []struct {
			Name     string `json:"name"`
			EntityID string `json:"entity_id"`
			Action   string `json:"action"`
		} `json:"mappings"`
	}
	decodeJSON(t, response, &body)
	if body.ProjectID != project.ID || body.StoryBibleID != bible.ID {
		t.Fatalf("unexpected sync envelope: %+v", body)
	}
	if len(body.Characters) != 2 || len(body.Mappings) != 2 {
		t.Fatalf("sync returned characters/mappings len = %d/%d, want 2/2", len(body.Characters), len(body.Mappings))
	}
	if body.Mappings[0].EntityID != existing.ID || body.Mappings[0].Action != "updated" {
		t.Fatalf("existing character was not updated by stable name: %+v", body.Mappings[0])
	}
	if body.Characters[0].Type != "character" || body.Characters[0].Traits["desire"] == "" || body.Characters[0].Traits["wound"] == "" {
		t.Fatalf("updated character missing canonical profile traits: %+v", body.Characters[0])
	}
	if _, ok := body.Characters[0].Traits["secret"]; ok {
		t.Fatalf("updated character retained empty/old secret trait: %+v", body.Characters[0].Traits)
	}
	if strings.Contains(body.Characters[0].Summary, "秘密：") {
		t.Fatalf("updated character summary contains empty secret segment: %q", body.Characters[0].Summary)
	}
	if body.Characters[0].Metadata["kept"] != "true" || body.Characters[0].Metadata["story_bible_id"] != bible.ID || body.Characters[0].Metadata["character_profile_json"] == "" {
		t.Fatalf("updated character metadata was not preserved/enriched: %+v", body.Characters[0].Metadata)
	}
	if body.Characters[1].Traits["secret"] == "" || !strings.Contains(body.Characters[1].Summary, "秘密：") {
		t.Fatalf("new character with secret missing secret trait/summary: %+v", body.Characters[1])
	}
	if body.Mappings[1].Action != "created" || body.Characters[1].ID == "" {
		t.Fatalf("new character was not created: character=%+v mapping=%+v", body.Characters[1], body.Mappings[1])
	}
}

func TestHandlerLegacyAIEndpointsAreRemoved(t *testing.T) {
	handler := newSmokeTestHandler(t)
	paths := []string{
		"/api/ai/character-profiles",
		"/api/ai/context-selection/preview",
		"/api/ai/draft",
		"/api/ai/draft-with-idea",
		"/api/ai/chapter-idea",
	}
	for _, path := range paths {
		response := sendJSON(t, handler, http.MethodPost, path, map[string]any{"project_id": "project_removed"})
		assertStatus(t, response, http.StatusNotFound)
	}
}

func TestHandlerAgentCRUDAndRunListing(t *testing.T) {
	handler, store := newAgentTestHandler(t)
	project, _, err := store.CreateProject(domain.Project{Title: "智能体项目", Slug: "agents"}, domain.StoryBible{Title: "智能体项目", Logline: "测试智能体"})
	if err != nil {
		t.Fatalf("CreateProject() error: %v", err)
	}

	createResponse := sendJSON(t, handler, http.MethodPost, "/api/agents", map[string]any{
		"project_id":    project.ID,
		"name":          "主写作代理",
		"description":   "用于验证智能体配置",
		"role":          domain.AgentRoleWriter,
		"enabled":       true,
		"system_prompt": "只输出正文",
		"tool_ids":      []string{"builtin:character.search"},
	})
	assertStatus(t, createResponse, http.StatusCreated)
	var created domain.AgentConfig
	decodeJSON(t, createResponse, &created)
	if created.ID == "" || created.ProjectID != project.ID || !created.Enabled || created.Role != domain.AgentRoleWriter {
		t.Fatalf("created agent config invalid: %+v", created)
	}

	updateResponse := sendJSON(t, handler, http.MethodPut, "/api/agents/"+created.ID, map[string]any{
		"project_id":    project.ID,
		"name":          "主写作代理（暂停）",
		"description":   created.Description,
		"role":          domain.AgentRoleEditor,
		"enabled":       false,
		"system_prompt": created.SystemPrompt,
	})
	assertStatus(t, updateResponse, http.StatusOK)
	var updated domain.AgentConfig
	decodeJSON(t, updateResponse, &updated)
	if updated.ID != created.ID || updated.Enabled || updated.Role != domain.AgentRoleEditor {
		t.Fatalf("updated agent config invalid: %+v", updated)
	}

	listResponse := sendJSON(t, handler, http.MethodGet, "/api/agents?project_id="+project.ID+"&enabled=false", nil)
	assertStatus(t, listResponse, http.StatusOK)
	var agents []domain.AgentConfig
	decodeJSON(t, listResponse, &agents)
	if len(agents) != 1 || agents[0].ID != created.ID {
		t.Fatalf("filtered agents = %+v, want updated agent", agents)
	}

	run, err := store.CreateAgentRun(domain.AgentRun{AgentID: created.ID, ProjectID: project.ID, Status: domain.AgentRunStatusCompleted, Input: map[string]any{"brief": "验证"}, Output: map[string]any{"text": "完成"}})
	if err != nil {
		t.Fatalf("CreateAgentRun() error: %v", err)
	}
	runsResponse := sendJSON(t, handler, http.MethodGet, "/api/agent-runs?agent_id="+created.ID+"&status=completed", nil)
	assertStatus(t, runsResponse, http.StatusOK)
	var runs []domain.AgentRun
	decodeJSON(t, runsResponse, &runs)
	if len(runs) != 1 || runs[0].ID != run.ID {
		t.Fatalf("filtered agent runs = %+v, want %q", runs, run.ID)
	}
}

func TestHandlerAgentSkillsAndToolToggles(t *testing.T) {
	handler, store := newAgentTestHandler(t)
	disabled := false
	createResponse := sendJSON(t, handler, http.MethodPost, "/api/skills", map[string]any{
		"name":        "Style Guard",
		"description": "限制叙事风格",
		"content":     "保持第三人称有限视角。",
		"enabled":     disabled,
		"metadata":    map[string]string{"origin": "inline"},
	})
	assertStatus(t, createResponse, http.StatusCreated)
	var skill domain.Skill
	decodeJSON(t, createResponse, &skill)
	if skill.ID == "" || skill.SourceID == "" || skill.Enabled {
		t.Fatalf("created inline skill invalid: %+v", skill)
	}

	enableResponse := sendJSON(t, handler, http.MethodPut, "/api/skills/"+skill.ID+"/enabled", map[string]any{"enabled": true})
	assertStatus(t, enableResponse, http.StatusOK)
	var enabledSkill domain.Skill
	decodeJSON(t, enableResponse, &enabledSkill)
	if !enabledSkill.Enabled {
		t.Fatalf("enabled skill is still disabled: %+v", enabledSkill)
	}

	tool, err := store.UpsertToolDefinition(domain.ToolDefinition{Name: "style.guard", Kind: domain.ToolDefinitionSkill, Status: domain.ToolStatusActive, SkillID: skill.ID})
	if err != nil {
		t.Fatalf("UpsertToolDefinition() error: %v", err)
	}
	disableToolResponse := sendJSON(t, handler, http.MethodPut, "/api/tools/catalog/"+tool.ID+"/enabled", map[string]any{"enabled": false})
	assertStatus(t, disableToolResponse, http.StatusOK)
	var disabledTool domain.ToolDefinition
	decodeJSON(t, disableToolResponse, &disabledTool)
	if disabledTool.Status != domain.ToolStatusDisabled {
		t.Fatalf("disabled tool status = %q, want disabled", disabledTool.Status)
	}

	catalogResponse := sendJSON(t, handler, http.MethodGet, "/api/tools/catalog?kind=skill&status=disabled", nil)
	assertStatus(t, catalogResponse, http.StatusOK)
	var catalog []domain.ToolDefinition
	decodeJSON(t, catalogResponse, &catalog)
	if len(catalog) != 1 || catalog[0].ID != tool.ID {
		t.Fatalf("filtered tool catalog = %+v, want %q", catalog, tool.ID)
	}
}

func TestHandlerAgentMCPServerSecretsAreHiddenAndToggleStatus(t *testing.T) {
	handler, _ := newAgentTestHandler(t)
	createResponse := sendJSON(t, handler, http.MethodPost, "/api/mcp/servers", map[string]any{
		"name":           "Local MCP",
		"transport":      domain.MCPTransportStdio,
		"enabled":        true,
		"command":        "node",
		"args":           []string{"server.js"},
		"secret_env":     map[string]string{"TOKEN": "secret-token"},
		"secret_headers": map[string]string{"Authorization": "Bearer secret"},
	})
	assertStatus(t, createResponse, http.StatusCreated)
	var created map[string]any
	decodeJSON(t, createResponse, &created)
	id, _ := created["id"].(string)
	if id == "" || created["secret_env"] != nil || created["secret_headers"] != nil {
		t.Fatalf("mcp server response exposed secrets or missed id: %+v", created)
	}
	if !containsAnyString(created["secret_env_hint"], "TOKEN") || !containsAnyString(created["secret_headers_hint"], "Authorization") {
		t.Fatalf("mcp server response missing secret hints: %+v", created)
	}

	disableResponse := sendJSON(t, handler, http.MethodPut, "/api/mcp/servers/"+id+"/enabled", map[string]any{"enabled": false})
	assertStatus(t, disableResponse, http.StatusOK)
	var disabledMCP map[string]any
	decodeJSON(t, disableResponse, &disabledMCP)
	if disabledMCP["enabled"] != false || disabledMCP["status"] != string(domain.MCPServerStatusDisabled) {
		t.Fatalf("disabled mcp response invalid: %+v", disabledMCP)
	}
}

func TestHandlerIndexJobsFiltersAndLimit(t *testing.T) {
	store := memory.NewStore()
	projectA, _, err := store.CreateProject(domain.Project{Title: "索引项目 A", Slug: "index-a"}, domain.StoryBible{Title: "索引项目 A", Logline: "A"})
	if err != nil {
		t.Fatalf("CreateProject(A) error: %v", err)
	}
	projectB, _, err := store.CreateProject(domain.Project{Title: "索引项目 B", Slug: "index-b"}, domain.StoryBible{Title: "索引项目 B", Logline: "B"})
	if err != nil {
		t.Fatalf("CreateProject(B) error: %v", err)
	}
	jobs := []domain.IndexJob{
		{ProjectID: projectA.ID, ChapterID: "chapter-a", Kind: "chapter_version", Status: "pending"},
		{ProjectID: projectA.ID, ChapterID: "chapter-a", Kind: "chapter_version", Status: "failed", Error: "embed failed"},
		{ProjectID: projectB.ID, ChapterID: "chapter-b", Kind: "chapter_version", Status: "completed"},
	}
	for _, job := range jobs {
		if _, err := store.CreateIndexJob(job); err != nil {
			t.Fatalf("CreateIndexJob() error: %v", err)
		}
	}
	server := httpapi.NewServer(config.Config{Host: "127.0.0.1", Port: 1, DataDir: t.TempDir(), DefaultProviderTimeout: time.Second}, store, providerregistry.New(nil, time.Second), nil, nil, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	allResponse := sendJSON(t, server.Handler(), http.MethodGet, "/api/index/jobs", nil)
	assertStatus(t, allResponse, http.StatusOK)
	var allJobs []domain.IndexJob
	decodeJSON(t, allResponse, &allJobs)
	if len(allJobs) != 3 {
		t.Fatalf("GET /api/index/jobs len = %d, want 3", len(allJobs))
	}

	filteredResponse := sendJSON(t, server.Handler(), http.MethodGet, "/api/index/jobs?project_id="+projectA.ID+"&status=failed&limit=1", nil)
	assertStatus(t, filteredResponse, http.StatusOK)
	var filteredJobs []domain.IndexJob
	decodeJSON(t, filteredResponse, &filteredJobs)
	if len(filteredJobs) != 1 || filteredJobs[0].ProjectID != projectA.ID || filteredJobs[0].Status != "failed" {
		t.Fatalf("filtered index jobs = %+v, want one failed job for project A", filteredJobs)
	}

	invalidLimitResponse := sendJSON(t, server.Handler(), http.MethodGet, "/api/index/jobs?limit=not-a-number", nil)
	assertStatus(t, invalidLimitResponse, http.StatusBadRequest)
}

func TestHandlerProviderAPIKeyEnvIsIgnoredAndHidden(t *testing.T) {
	store := memory.NewStore()
	server := httpapi.NewServer(config.Config{Host: "127.0.0.1", Port: 1, DataDir: t.TempDir(), DefaultProviderTimeout: time.Second}, store, providerregistry.New(nil, time.Second), nil, nil, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	createResponse := sendJSON(t, server.Handler(), http.MethodPost, "/api/providers", map[string]any{
		"id":          "provider_env_ignored",
		"name":        "Env Ignored",
		"type":        "openai",
		"base_url":    "https://example.invalid/v1",
		"api_key_env": "AEON_TEST_KEY",
		"enabled":     true,
	})
	assertStatus(t, createResponse, http.StatusCreated)
	var created map[string]any
	decodeJSON(t, createResponse, &created)
	if _, ok := created["api_key_env"]; ok {
		t.Fatalf("provider response exposed api_key_env: %+v", created)
	}
	if created["api_key_hint"] != "" {
		t.Fatalf("provider api_key_hint = %q, want empty when only api_key_env was submitted", created["api_key_hint"])
	}
	createdProvider, err := store.GetProvider("provider_env_ignored")
	if err != nil {
		t.Fatalf("GetProvider(created) error: %v", err)
	}
	if createdProvider.APIKeyEnv != "" {
		t.Fatalf("created provider APIKeyEnv = %q, want empty", createdProvider.APIKeyEnv)
	}

	legacyProvider, err := store.CreateProvider(domain.ProviderConfig{ID: "legacy_env", Name: "Legacy Env", Type: domain.ProviderOpenAI, BaseURL: "https://example.invalid/v1", APIKeyEnv: "OLD_ENV", Enabled: true})
	if err != nil {
		t.Fatalf("CreateProvider(legacy) error: %v", err)
	}
	updateResponse := sendJSON(t, server.Handler(), http.MethodPut, "/api/providers/"+legacyProvider.ID, map[string]any{
		"name":     legacyProvider.Name,
		"type":     legacyProvider.Type,
		"base_url": legacyProvider.BaseURL,
		"api_key":  "new-key",
		"enabled":  true,
	})
	assertStatus(t, updateResponse, http.StatusOK)
	var updated map[string]any
	decodeJSON(t, updateResponse, &updated)
	if _, ok := updated["api_key_env"]; ok {
		t.Fatalf("updated provider response exposed api_key_env: %+v", updated)
	}
	if updated["api_key_hint"] != "configured" {
		t.Fatalf("updated provider api_key_hint = %q, want configured", updated["api_key_hint"])
	}
	updatedProvider, err := store.GetProvider(legacyProvider.ID)
	if err != nil {
		t.Fatalf("GetProvider(updated) error: %v", err)
	}
	if updatedProvider.APIKeyEnv != "" || updatedProvider.APIKey != "new-key" {
		t.Fatalf("updated provider credentials = api_key_env:%q api_key:%q, want env cleared and key saved", updatedProvider.APIKeyEnv, updatedProvider.APIKey)
	}
}

func newSmokeTestHandler(t *testing.T) http.Handler {
	t.Helper()
	store := memory.NewStore()
	providers := providerregistry.New(nil, time.Second)
	workflow := agent.NewWorkflowRunner(store, nil, nil, providers)
	server := httpapi.NewServer(config.Config{
		Host:                   "127.0.0.1",
		Port:                   1,
		DataDir:                t.TempDir(),
		DefaultProviderTimeout: time.Second,
	}, store, providers, workflow, nil, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	return server.Handler()
}

func newAgentTestHandler(t *testing.T) (http.Handler, *memory.Store) {
	t.Helper()
	store := memory.NewStore()
	providers := providerregistry.New(nil, time.Second)
	toolRegistry := tooling.NewRegistry(store, store)
	if err := toolRegistry.SeedBuiltinTools(context.Background()); err != nil {
		t.Fatalf("SeedBuiltinTools() error: %v", err)
	}
	cfg := config.Config{Host: "127.0.0.1", Port: 1, DataDir: t.TempDir(), DefaultProviderTimeout: time.Second, MCPDefaultTimeout: time.Second}
	server := httpapi.NewServer(cfg, store, providers, nil, nil, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	server.ConfigureAgents(nil, skills.NewService(store, t.TempDir()), toolRegistry, time.Second)
	return server.Handler(), store
}

func newWorkflowBackedServerFixture(t *testing.T) (*memory.Store, *agent.WorkflowRunner, string) {
	t.Helper()
	store := memory.NewStore()
	project, _, err := store.CreateProject(domain.Project{Title: "星海回声", Slug: "xinghai", Status: "active"}, domain.StoryBible{Title: "星海回声", Logline: "远航者寻找失落文明", Rules: map[string]string{"canon_policy": "必须守住连续性", "context_policy": "仅使用上下文包"}})
	if err != nil {
		t.Fatalf("CreateProject() error: %v", err)
	}
	architectProvider, err := store.CreateProvider(domain.ProviderConfig{ID: "provider_architect", Name: "Architect Provider", Type: domain.ProviderOpenAI, Enabled: true})
	if err != nil {
		t.Fatalf("CreateProvider(architect) error: %v", err)
	}
	writerProvider, err := store.CreateProvider(domain.ProviderConfig{ID: "provider_writer", Name: "Writer Provider", Type: domain.ProviderAnthropic, Enabled: true})
	if err != nil {
		t.Fatalf("CreateProvider(writer) error: %v", err)
	}
	characterProvider, err := store.CreateProvider(domain.ProviderConfig{ID: "provider_character", Name: "Character Provider", Type: domain.ProviderOpenAI, Enabled: true})
	if err != nil {
		t.Fatalf("CreateProvider(character) error: %v", err)
	}
	_, err = store.CreateModel(domain.ModelConfig{ID: "provider_architect:architect-explicit", ProviderID: architectProvider.ID, Name: "architect-explicit", Kind: domain.ModelKindText, Enabled: true, SupportsTools: true, MaxOutputTokens: 600, AllowedAgentRoles: []domain.AgentRole{domain.AgentRolePlotArchitect}})
	if err != nil {
		t.Fatalf("CreateModel(architect) error: %v", err)
	}
	_, err = store.CreateModel(domain.ModelConfig{ID: "provider_writer:writer-explicit", ProviderID: writerProvider.ID, Name: "writer-explicit", Kind: domain.ModelKindText, Enabled: true, SupportsTools: true, MaxOutputTokens: 1400, AllowedAgentRoles: []domain.AgentRole{domain.AgentRoleWriter}})
	if err != nil {
		t.Fatalf("CreateModel(writer) error: %v", err)
	}
	_, err = store.CreateModel(domain.ModelConfig{ID: "provider_character:character-explicit", ProviderID: characterProvider.ID, Name: "character-explicit", Kind: domain.ModelKindText, Enabled: true, SupportsTools: true, MaxOutputTokens: 900, AllowedAgentRoles: []domain.AgentRole{domain.AgentRoleCharacterKeeper}})
	if err != nil {
		t.Fatalf("CreateModel(character) error: %v", err)
	}
	_, err = store.UpsertSetting(domain.AppSetting{Scope: agent.ModelRoutingSettingScope, Key: string(domain.AgentRolePlotArchitect), Value: map[string]any{agent.ModelRoutingSettingValueKey: "provider_architect:architect-explicit"}})
	if err != nil {
		t.Fatalf("UpsertSetting(plot architect) error: %v", err)
	}
	_, err = store.UpsertSetting(domain.AppSetting{Scope: agent.ModelRoutingSettingScope, Key: string(domain.AgentRoleWriter), Value: map[string]any{agent.ModelRoutingSettingValueKey: "provider_writer:writer-explicit"}})
	if err != nil {
		t.Fatalf("UpsertSetting(writer) error: %v", err)
	}
	_, err = store.UpsertSetting(domain.AppSetting{Scope: agent.ModelRoutingSettingScope, Key: string(domain.AgentRoleCharacterKeeper), Value: map[string]any{agent.ModelRoutingSettingValueKey: "provider_character:character-explicit"}})
	if err != nil {
		t.Fatalf("UpsertSetting(character) error: %v", err)
	}
	seedWorkflow, err := store.SaveWorkflow(domain.AIWorkflow{ProjectID: project.ID, Kind: "chapter_seed", Role: domain.AgentRoleWriter, Status: "completed"})
	if err != nil {
		t.Fatalf("SaveWorkflow(seed) error: %v", err)
	}
	charLin, err := store.SaveEntity(domain.Entity{ProjectID: project.ID, Name: "林烬", Type: "character", Metadata: map[string]string{"source_chapter_version_id": "chapter_version_seed_1"}})
	if err != nil {
		t.Fatalf("SaveEntity(林烬) error: %v", err)
	}
	charSu, err := store.SaveEntity(domain.Entity{ProjectID: project.ID, Name: "苏九", Type: "character", Metadata: map[string]string{"source_chapter_version_id": "chapter_version_seed_2"}})
	if err != nil {
		t.Fatalf("SaveEntity(苏九) error: %v", err)
	}
	itemKey, err := store.SaveEntity(domain.Entity{ProjectID: project.ID, Name: "灰烬钥匙", Type: "item", Metadata: map[string]string{"source_chapter_version_id": "chapter_version_seed_1"}})
	if err != nil {
		t.Fatalf("SaveEntity(灰烬钥匙) error: %v", err)
	}
	chapterOne, _, err := store.SaveChapterVersion(domain.ChapterVersion{ID: "chapter_version_seed_1", ProjectID: project.ID, ChapterID: "chapter-manual", Title: "前章", Content: "林烬得到灰烬钥匙", Summary: "林烬得到灰烬钥匙", AuthorRole: domain.AgentRoleWriter, SourceWorkflowID: seedWorkflow.ID, IndexStatus: "indexed"})
	if err != nil {
		t.Fatalf("SaveChapterVersion(前章) error: %v", err)
	}
	chapterTwo, _, err := store.SaveChapterVersion(domain.ChapterVersion{ID: "chapter_version_seed_2", ProjectID: project.ID, ChapterID: "chapter-other", Title: "旁章", Content: "苏九调查灰塔", Summary: "苏九调查灰塔", AuthorRole: domain.AgentRoleWriter, SourceWorkflowID: seedWorkflow.ID, IndexStatus: "indexed"})
	if err != nil {
		t.Fatalf("SaveChapterVersion(旁章) error: %v", err)
	}
	if _, err := store.SaveFact(domain.Fact{ProjectID: project.ID, EntityID: charLin.ID, ChapterID: chapterOne.ChapterID, ChapterVersionID: chapterOne.ID, Claim: "林烬持有灰烬钥匙", Source: chapterOne.ID, Confidence: 1, Status: "accepted"}); err != nil {
		t.Fatalf("SaveFact(林烬) error: %v", err)
	}
	if _, err := store.SaveFact(domain.Fact{ProjectID: project.ID, EntityID: charSu.ID, ChapterID: chapterTwo.ChapterID, ChapterVersionID: chapterTwo.ID, Claim: "苏九独自调查灰塔", Source: chapterTwo.ID, Confidence: 1, Status: "accepted"}); err != nil {
		t.Fatalf("SaveFact(苏九) error: %v", err)
	}
	if _, err := store.SaveGraphEdge(domain.GraphEdge{ProjectID: project.ID, SourceEntityID: charLin.ID, TargetEntityID: itemKey.ID, Type: "owns", Label: "持有"}); err != nil {
		t.Fatalf("SaveGraphEdge() error: %v", err)
	}
	if _, err := store.SavePlotThread(domain.PlotThread{ProjectID: project.ID, Title: "钥匙伏笔", OpenedChapterID: chapterOne.ChapterID, RelatedEntityIDs: []string{charLin.ID, itemKey.ID}, Metadata: map[string]string{"source_chapter_version_id": chapterOne.ID}}); err != nil {
		t.Fatalf("SavePlotThread(钥匙伏笔) error: %v", err)
	}
	if _, err := store.SavePlotThread(domain.PlotThread{ProjectID: project.ID, Title: "塔楼异动", OpenedChapterID: chapterTwo.ChapterID, RelatedEntityIDs: []string{charSu.ID}, Metadata: map[string]string{"source_chapter_version_id": chapterTwo.ID}}); err != nil {
		t.Fatalf("SavePlotThread(塔楼异动) error: %v", err)
	}
	textClient := &fakeTextClient{responses: []provider.ModelResponse{{Content: "## 本章目标\n进入灰塔并发现钟声异常"}, {Content: "林烬踏入灰塔，钟声在背后收拢。"}, {Content: "林烬把灰烬钥匙藏进袖口，走向塔门。"}, {Content: `{"characters":[{"name":"林烬","role":"主角","desire":"找回失落舰队真相","wound":"曾在撤离中放弃同伴","secret":"他携带舰队核心坐标","summary":"背负旧债的远航者。"}]}`}}}
	router := agent.NewModelRouter(store, agent.NewAgentRoleRegistry())
	tools := agent.NewToolRuntime(store)
	builder := agent.NewContextPackBuilder(store, tools, store)
	workflow := agent.NewWorkflowRunner(store, router, builder, fakeTextClientFactory{client: textClient})
	return store, workflow, project.ID
}

func newCharacterProfileWorkflowForStore(t *testing.T, store *memory.Store) *agent.WorkflowRunner {
	t.Helper()
	projects, err := store.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects() error: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("ListProjects() len = %d, want 1", len(projects))
	}
	projectID := projects[0].ID
	textClient := &fakeTextClient{responses: []provider.ModelResponse{
		{ToolCalls: []provider.ToolCall{
			{ID: "call-search-lin", Name: "character.search", Arguments: mustRawJSON(t, map[string]any{"project_id": projectID, "query": "林烬", "limit": 5})},
			{ID: "call-upsert-lin", Name: "character.upsert", Arguments: mustRawJSON(t, map[string]any{"project_id": projectID, "name": "林烬", "summary": "背负旧债的远航者。", "traits": map[string]any{"role": "主角", "desire": "找回失落舰队真相", "wound": "曾在撤离中放弃同伴", "secret": "他携带舰队核心坐标"}, "metadata": map[string]any{"source": "ai_character_profiles"}})},
		}},
		{Content: `{"characters":[{"name":"林烬","role":"主角","desire":"找回失落舰队真相","wound":"曾在撤离中放弃同伴","secret":"他携带舰队核心坐标","summary":"背负旧债的远航者。"}]}`},
	}}
	router := agent.NewModelRouter(store, agent.NewAgentRoleRegistry())
	tools := agent.NewToolRuntime(store)
	builder := agent.NewContextPackBuilder(store, tools, store)
	return agent.NewWorkflowRunner(store, router, builder, fakeTextClientFactory{client: textClient})
}

type fakeTextClientFactory struct {
	client *fakeTextClient
}

type fakeTextClient struct {
	responses []provider.ModelResponse
	requests  []provider.TextRequest
}

func (f fakeTextClientFactory) NewTextClient(_ domain.ProviderConfig) (provider.TextModelClient, error) {
	if f.client == nil {
		return nil, fmt.Errorf("fake text client is nil")
	}
	return f.client, nil
}

func (c *fakeTextClient) Generate(_ context.Context, req provider.TextRequest) (provider.ModelResponse, error) {
	c.requests = append(c.requests, req)
	if len(c.responses) == 0 {
		return provider.ModelResponse{}, fmt.Errorf("missing fake text response")
	}
	resp := c.responses[0]
	c.responses = c.responses[1:]
	return resp, nil
}

func (c *fakeTextClient) Stream(ctx context.Context, req provider.TextRequest) (<-chan provider.StreamEvent, error) {
	resp, err := c.Generate(ctx, req)
	return provider.StreamSingleEvent(ctx, resp, err)
}

func sendJSON(t *testing.T, handler http.Handler, method string, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader = http.NoBody
	if body != nil {
		payload := bytes.Buffer{}
		if err := json.NewEncoder(&payload).Encode(body); err != nil {
			t.Fatalf("encode request body: %v", err)
		}
		reader = &payload
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}

func assertStatus(t *testing.T, response *httptest.ResponseRecorder, want int) {
	t.Helper()
	if response.Code != want {
		t.Fatalf("HTTP status = %d, want %d; body: %s", response.Code, want, response.Body.String())
	}
}

func decodeJSON(t *testing.T, response *httptest.ResponseRecorder, out any) {
	t.Helper()
	if err := json.NewDecoder(response.Body).Decode(out); err != nil {
		t.Fatalf("decode response body %q: %v", response.Body.String(), err)
	}
}

func containsProject(projects []domain.Project, projectID string) bool {
	for _, project := range projects {
		if project.ID == projectID {
			return true
		}
	}
	return false
}

func containsEntityID(entities []domain.Entity, entityID string) bool {
	for _, entity := range entities {
		if entity.ID == entityID {
			return true
		}
	}
	return false
}

func containsAnyString(value any, want string) bool {
	items, ok := value.([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		if text, ok := item.(string); ok && text == want {
			return true
		}
	}
	return false
}

func mustRawJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal raw json: %v", err)
	}
	return payload
}
