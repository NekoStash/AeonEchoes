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

func TestHandlerCharacterProfilesEndpointReturnsWorkflowContextAndCharacters(t *testing.T) {
	store, workflow, projectID := newWorkflowBackedServerFixture(t)
	workflow = newCharacterProfileWorkflowForStore(t, store)
	server := httpapi.NewServer(config.Config{Host: "127.0.0.1", Port: 1, DataDir: t.TempDir(), DefaultProviderTimeout: time.Second}, store, providerregistry.New(nil, time.Second), workflow, nil, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	response := sendJSON(t, server.Handler(), http.MethodPost, "/api/ai/character-profiles", map[string]any{
		"project_id": projectID,
		"focus":      "主角完整设定",
		"count":      1,
		"brief":      "补全能支撑第一卷的主角人物弧",
	})
	assertStatus(t, response, http.StatusCreated)
	var body struct {
		Workflow        domain.AIWorkflow                `json:"workflow"`
		ContextPack     domain.ContextPack               `json:"context_pack"`
		Characters      []domain.CharacterProfile        `json:"characters"`
		Entities        []domain.Entity                  `json:"entities"`
		Mappings        []domain.CharacterProfileMapping `json:"mappings"`
		ModelResolution domain.ModelResolution           `json:"model_resolution"`
		ToolTrace       []string                         `json:"tool_trace"`
	}
	decodeJSON(t, response, &body)
	if body.Workflow.Kind != "character_profiles" || body.Workflow.Role != domain.AgentRoleCharacterKeeper || body.Workflow.Status != "completed" {
		t.Fatalf("unexpected character profile workflow: %+v", body.Workflow)
	}
	if body.ContextPack.Role != domain.AgentRoleCharacterKeeper || body.ContextPack.ID == "" {
		t.Fatalf("unexpected character profile context pack: %+v", body.ContextPack)
	}
	if body.ModelResolution.ModelID == "" || body.ModelResolution.RouteKey != string(domain.AgentRoleCharacterKeeper) {
		t.Fatalf("character profile model resolution missing: %+v", body.ModelResolution)
	}
	if len(body.Characters) != 1 || body.Characters[0].Name != "林烬" || body.Characters[0].Role != "主角" || body.Characters[0].Desire == "" || body.Characters[0].Wound == "" || body.Characters[0].Secret == "" || body.Characters[0].Summary == "" {
		t.Fatalf("character profile response missing complete profile: %+v", body.Characters)
	}
	if len(body.Entities) != 1 || body.Entities[0].ProjectID != projectID || body.Entities[0].Type != "character" || body.Entities[0].Name != "林烬" || body.Entities[0].Summary != body.Characters[0].Summary || body.Entities[0].Traits["secret"] != body.Characters[0].Secret {
		t.Fatalf("character profile response missing saved entity: entities=%+v characters=%+v", body.Entities, body.Characters)
	}
	if len(body.Mappings) != 1 || body.Mappings[0].Name != "林烬" || body.Mappings[0].EntityID != body.Entities[0].ID || body.Mappings[0].Action == "" {
		t.Fatalf("character profile response missing entity mapping: mappings=%+v entities=%+v", body.Mappings, body.Entities)
	}
	if len(body.ToolTrace) != 2 || !strings.Contains(strings.Join(body.ToolTrace, "\n"), "character.search") || !strings.Contains(strings.Join(body.ToolTrace, "\n"), "character.upsert") {
		t.Fatalf("character profile response missing tool trace: %+v", body.ToolTrace)
	}
	savedEntities, err := store.ListEntities(projectID)
	if err != nil {
		t.Fatalf("ListEntities() error: %v", err)
	}
	if !containsEntityID(savedEntities, body.Mappings[0].EntityID) {
		t.Fatalf("character profile entity was not persisted through tool call: id=%q entities=%+v", body.Mappings[0].EntityID, savedEntities)
	}
}

func TestHandlerPreviewReturnsFreshnessAndModelResolution(t *testing.T) {
	store, workflow, projectID := newWorkflowBackedServerFixture(t)
	_ = store
	server := httpapi.NewServer(config.Config{Host: "127.0.0.1", Port: 1, DataDir: t.TempDir(), DefaultProviderTimeout: time.Second}, store, providerregistry.New(nil, time.Second), workflow, nil, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	response := sendJSON(t, server.Handler(), http.MethodPost, "/api/ai/context-selection/preview", map[string]any{
		"project_id": projectID,
		"chapter_id": "chapter-manual",
		"brief":      "围绕林烬推进",
		"context_selection": map[string]any{
			"chapter_ids":     []string{"chapter-manual"},
			"character_names": []string{"林烬"},
		},
	})
	assertStatus(t, response, http.StatusOK)
	var body struct {
		ContextPack struct {
			ID string `json:"id"`
		} `json:"context_pack"`
		Summary         string                 `json:"summary"`
		EstimatedTokens int                    `json:"estimated_tokens"`
		IndexFreshness  domain.IndexFreshness  `json:"index_freshness"`
		ModelResolution domain.ModelResolution `json:"model_resolution"`
	}
	decodeJSON(t, response, &body)
	if body.ContextPack.ID == "" {
		t.Fatalf("preview context_pack.id is empty")
	}
	if body.Summary == "" || body.EstimatedTokens <= 0 {
		t.Fatalf("preview summary/tokens invalid: %+v", body)
	}
	if body.IndexFreshness.Status == "" {
		t.Fatalf("preview freshness missing: %+v", body.IndexFreshness)
	}
	if body.ModelResolution.ModelID == "" || body.ModelResolution.ProviderID == "" {
		t.Fatalf("preview model_resolution missing: %+v", body.ModelResolution)
	}
}

func TestHandlerDraftNotifiesWorkerAndReturnsFreshnessAndModelResolution(t *testing.T) {
	store, workflow, projectID := newWorkflowBackedServerFixture(t)
	wake := &integrationWakeNotifier{}
	server := httpapi.NewServer(config.Config{Host: "127.0.0.1", Port: 1, DataDir: t.TempDir(), DefaultProviderTimeout: time.Second}, store, providerregistry.New(nil, time.Second), workflow, nil, nil, wake, slog.New(slog.NewTextHandler(io.Discard, nil)))

	response := sendJSON(t, server.Handler(), http.MethodPost, "/api/ai/draft", map[string]any{
		"project_id": projectID,
		"chapter_id": "chapter-manual",
		"title":      "手动测试",
		"brief":      "围绕林烬推进",
	})
	assertStatus(t, response, http.StatusCreated)
	var body struct {
		IndexFreshness  domain.IndexFreshness  `json:"index_freshness"`
		ModelResolution domain.ModelResolution `json:"model_resolution"`
		ContinuityAudit domain.ContinuityAudit `json:"continuity_audit"`
	}
	decodeJSON(t, response, &body)
	if wake.count != 1 {
		t.Fatalf("wake count = %d, want 1", wake.count)
	}
	if body.IndexFreshness.Status == "" {
		t.Fatalf("draft freshness missing: %+v", body.IndexFreshness)
	}
	if body.ModelResolution.ModelID == "" || body.ModelResolution.ProviderID == "" {
		t.Fatalf("draft model_resolution missing: %+v", body.ModelResolution)
	}
	if body.ContinuityAudit.Status == "" {
		t.Fatalf("draft continuity_audit missing: %+v", body.ContinuityAudit)
	}
}

func TestHandlerDraftWithIdeaReturnsDraftContinuityAudit(t *testing.T) {
	store, workflow, projectID := newWorkflowBackedServerFixture(t)
	wake := &integrationWakeNotifier{}
	server := httpapi.NewServer(config.Config{Host: "127.0.0.1", Port: 1, DataDir: t.TempDir(), DefaultProviderTimeout: time.Second}, store, providerregistry.New(nil, time.Second), workflow, nil, nil, wake, slog.New(slog.NewTextHandler(io.Discard, nil)))

	response := sendJSON(t, server.Handler(), http.MethodPost, "/api/ai/draft-with-idea", map[string]any{
		"project_id": projectID,
		"chapter_id": "chapter-manual",
		"title":      "手动测试",
		"brief":      "围绕林烬推进",
	})
	assertStatus(t, response, http.StatusCreated)
	var body struct {
		Draft struct {
			ContinuityAudit domain.ContinuityAudit `json:"continuity_audit"`
		} `json:"draft"`
	}
	decodeJSON(t, response, &body)
	if wake.count != 1 {
		t.Fatalf("wake count = %d, want 1", wake.count)
	}
	if body.Draft.ContinuityAudit.Status == "" {
		t.Fatalf("draft-with-idea draft.continuity_audit missing: %+v", body.Draft.ContinuityAudit)
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

func mustRawJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal raw json: %v", err)
	}
	return payload
}
