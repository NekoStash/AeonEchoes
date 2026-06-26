package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/memory"
	"aeonechoes/server/internal/provider"
)

type fakeWorkflowStore struct {
	projects        []domain.Project
	bibles          []domain.StoryBible
	workflows       []domain.AIWorkflow
	chapterVersions []domain.ChapterVersion
	indexJobs       []domain.IndexJob
}

type fakeContextPackRepository struct {
	bible     domain.StoryBible
	versions  []domain.ChapterVersion
	entities  []domain.Entity
	facts     []domain.Fact
	threads   []domain.PlotThread
	expansion domain.GraphExpansion
}

type fakeTextClientFactory struct {
	client *fakeTextClient
}

type fakeTextClient struct {
	responses []provider.ModelResponse
	requests  []provider.TextRequest
}

type fakeIDSource struct {
	id  string
	err error
}

func (s *fakeWorkflowStore) CreateProject(project domain.Project, bible domain.StoryBible) (domain.Project, domain.StoryBible, error) {
	project.ID = "project-1"
	bible.ID = "bible-1"
	bible.ProjectID = project.ID
	project.ActiveStoryBibleID = bible.ID
	s.projects = append(s.projects, project)
	s.bibles = append(s.bibles, bible)
	return project, bible, nil
}

func (s *fakeWorkflowStore) SaveChapterVersion(version domain.ChapterVersion) (domain.ChapterVersion, domain.IndexJob, error) {
	version.ID = "chapter-version-1"
	version.CreatedAt = time.Now().UTC()
	s.chapterVersions = append(s.chapterVersions, version)
	job := domain.IndexJob{ID: "index-job-1", ProjectID: version.ProjectID, ChapterID: version.ChapterID, ChapterVersionID: version.ID, Status: "pending", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	s.indexJobs = append(s.indexJobs, job)
	return version, job, nil
}

func (s *fakeWorkflowStore) SaveWorkflow(workflow domain.AIWorkflow) (domain.AIWorkflow, error) {
	if workflow.ID == "" {
		workflow.ID = fmt.Sprintf("workflow-%d", len(s.workflows)+1)
	}
	s.workflows = append(s.workflows, workflow)
	return workflow, nil
}

func (s *fakeWorkflowStore) ListChapterVersions(projectID, chapterID string) ([]domain.ChapterVersion, error) {
	items := make([]domain.ChapterVersion, 0)
	for _, version := range s.chapterVersions {
		if projectID != "" && version.ProjectID != projectID {
			continue
		}
		if chapterID != "" && version.ChapterID != chapterID {
			continue
		}
		items = append(items, version)
	}
	return items, nil
}

func (s *fakeWorkflowStore) ListPendingIndexJobs(projectID string, limit int) ([]domain.IndexJob, error) {
	items := make([]domain.IndexJob, 0)
	for _, job := range s.indexJobs {
		if job.Status != "pending" {
			continue
		}
		if projectID != "" && job.ProjectID != projectID {
			continue
		}
		items = append(items, job)
	}
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (r fakeContextPackRepository) ExpandGraph(projectID string, _ []string, depth int) (domain.GraphExpansion, error) {
	expansion := r.expansion
	expansion.ProjectID = projectID
	expansion.Depth = depth
	return expansion, nil
}

func (r fakeContextPackRepository) ListEntities(_ string) ([]domain.Entity, error) {
	return r.entities, nil
}

func (r fakeContextPackRepository) ListFacts(_ string) ([]domain.Fact, error) {
	return r.facts, nil
}

func (r fakeContextPackRepository) ListPlotThreads(_ string) ([]domain.PlotThread, error) {
	return r.threads, nil
}

func (r fakeContextPackRepository) ListChapterVersions(_, _ string) ([]domain.ChapterVersion, error) {
	return r.versions, nil
}

func (r fakeContextPackRepository) GetStoryBible(projectID string) (domain.StoryBible, error) {
	bible := r.bible
	if bible.ID == "" {
		bible.ID = "bible-1"
	}
	bible.ProjectID = projectID
	return bible, nil
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

func (s fakeIDSource) NewID(_ string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.id, nil
}

func TestWorkflowRunnerInitializeProjectValidatesSeed(t *testing.T) {
	runner := NewWorkflowRunner(&fakeWorkflowStore{}, nil, nil, nil)

	if _, err := runner.InitializeProject(context.Background(), domain.ProjectSeed{Premise: "失落舰队重返群星"}); err == nil {
		t.Fatalf("InitializeProject() with empty title error = nil, want error")
	}
	if _, err := runner.InitializeProject(context.Background(), domain.ProjectSeed{Title: "星海回声"}); err == nil {
		t.Fatalf("InitializeProject() with empty premise error = nil, want error")
	}
}

func TestContextPackBuilderReturnsIDSourceError(t *testing.T) {
	wantErr := errors.New("id generator unavailable")
	repo := fakeContextPackRepository{bible: domain.StoryBible{ID: "bible-1"}}
	builder := NewContextPackBuilder(repo, NewToolRuntime(repo), fakeIDSource{err: wantErr})

	pack, err := builder.Build("project-1", "chapter-1", domain.AgentRoleWriter, "brief", 1000)
	if err == nil {
		t.Fatalf("Build() error = nil, want error")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("Build() error = %v, want wrapped %v", err, wantErr)
	}
	if pack.ID != "" {
		t.Fatalf("Build() pack ID = %q, want empty", pack.ID)
	}
	if !strings.Contains(err.Error(), "generate context pack id") {
		t.Fatalf("Build() error lacks context: %v", err)
	}
}

func TestContextPackBuilderSelectionFiltersResults(t *testing.T) {
	includeWorldRules := false
	repo := fakeContextPackRepository{
		bible: domain.StoryBible{ID: "bible-1", Rules: map[string]string{"canon_policy": "必须遵守"}},
		versions: []domain.ChapterVersion{
			{ID: "cv-1", ProjectID: "project-1", ChapterID: "chapter-1", Title: "第一章", Summary: "林烬得到钥匙", Content: "林烬得到钥匙"},
			{ID: "cv-2", ProjectID: "project-1", ChapterID: "chapter-2", Title: "第二章", Summary: "苏九调查塔楼", Content: "苏九调查塔楼"},
		},
		entities: []domain.Entity{
			{ID: "char-1", ProjectID: "project-1", Name: "林烬", Type: "character", Metadata: map[string]string{"source_chapter_version_id": "cv-1"}},
			{ID: "char-2", ProjectID: "project-1", Name: "苏九", Type: "character", Metadata: map[string]string{"source_chapter_version_id": "cv-2"}},
			{ID: "item-1", ProjectID: "project-1", Name: "灰烬钥匙", Type: "item", Metadata: map[string]string{"source_chapter_version_id": "cv-1"}},
		},
		facts: []domain.Fact{
			{ID: "fact-1", ProjectID: "project-1", EntityID: "char-1", ChapterID: "chapter-1", ChapterVersionID: "cv-1", Claim: "林烬持有灰烬钥匙"},
			{ID: "fact-2", ProjectID: "project-1", EntityID: "char-2", ChapterID: "chapter-2", ChapterVersionID: "cv-2", Claim: "苏九独自调查塔楼"},
		},
		threads: []domain.PlotThread{
			{ID: "thread-1", ProjectID: "project-1", Title: "钥匙去向", OpenedChapterID: "chapter-1", Metadata: map[string]string{"source_chapter_version_id": "cv-1"}},
			{ID: "thread-2", ProjectID: "project-1", Title: "塔楼钟声", OpenedChapterID: "chapter-2", Metadata: map[string]string{"source_chapter_version_id": "cv-2"}},
		},
		expansion: domain.GraphExpansion{
			Entities: []domain.Entity{{ID: "char-1", ProjectID: "project-1", Name: "林烬", Type: "character", Metadata: map[string]string{"source_chapter_version_id": "cv-1"}}, {ID: "item-1", ProjectID: "project-1", Name: "灰烬钥匙", Type: "item", Metadata: map[string]string{"source_chapter_version_id": "cv-1"}}},
			Edges:    []domain.GraphEdge{{ID: "edge-1", ProjectID: "project-1", SourceEntityID: "char-1", TargetEntityID: "item-1", Label: "持有"}},
		},
	}
	builder := NewContextPackBuilder(repo, NewToolRuntime(repo), fakeIDSource{id: "context-pack-1"})

	pack, err := builder.BuildWithSelection("project-1", "chapter-1", domain.AgentRoleWriter, "围绕林烬和钥匙", 1000, &ContextSelection{ChapterIDs: []string{"chapter-1"}, CharacterNames: []string{"林烬"}, IncludeWorldRules: &includeWorldRules}, []string{"legacy-node-1"})
	if err != nil {
		t.Fatalf("BuildWithSelection() error: %v", err)
	}
	if len(pack.ChapterSummaries) != 1 || pack.ChapterSummaries[0].ChapterID != "chapter-1" {
		t.Fatalf("unexpected chapter summaries: %+v", pack.ChapterSummaries)
	}
	if len(pack.Entities) != 2 {
		t.Fatalf("unexpected entities: %+v", pack.Entities)
	}
	for _, entity := range pack.Entities {
		if entity.ID == "char-2" {
			t.Fatalf("unexpected unrelated entity in selection pack: %+v", pack.Entities)
		}
	}
	if len(pack.Facts) != 1 || pack.Facts[0].ID != "fact-1" {
		t.Fatalf("unexpected facts: %+v", pack.Facts)
	}
	if pack.WorldRules != nil {
		t.Fatalf("expected world rules excluded, got %+v", pack.WorldRules)
	}
	if pack.Metadata["selection_mode"] != "explicit" || pack.Metadata["selection_character_names"] != "林烬" || pack.Metadata["context_node_ids_compat"] != "legacy-node-1" {
		t.Fatalf("unexpected selection metadata: %+v", pack.Metadata)
	}
}

func TestContextPackBuilderAutoModeStillWorks(t *testing.T) {
	repo := fakeContextPackRepository{
		bible:     domain.StoryBible{ID: "bible-1", Rules: map[string]string{"canon_policy": "必须遵守"}},
		versions:  []domain.ChapterVersion{{ID: "cv-1", ProjectID: "project-1", ChapterID: "chapter-1", Title: "第一章", Summary: "摘要 1", Content: "内容 1"}},
		facts:     []domain.Fact{{ID: "fact-1", ProjectID: "project-1", ChapterID: "chapter-1", ChapterVersionID: "cv-1", Claim: "事实 1"}},
		threads:   []domain.PlotThread{{ID: "thread-1", ProjectID: "project-1", Title: "线索 1"}},
		expansion: domain.GraphExpansion{Entities: []domain.Entity{{ID: "char-1", ProjectID: "project-1", Name: "林烬", Type: "character"}}},
	}
	builder := NewContextPackBuilder(repo, NewToolRuntime(repo), fakeIDSource{id: "context-pack-2"})

	pack, err := builder.Build("project-1", "chapter-1", domain.AgentRoleWriter, "自动模式", 1000)
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}
	if pack.Metadata["selection_mode"] != "auto" {
		t.Fatalf("expected auto mode metadata, got %+v", pack.Metadata)
	}
	if len(pack.ChapterSummaries) != 1 || len(pack.Entities) != 1 || len(pack.Facts) != 1 {
		t.Fatalf("unexpected auto pack content: %+v", pack)
	}
	if pack.WorldRules["canon_policy"] != "必须遵守" {
		t.Fatalf("expected world rules in auto mode, got %+v", pack.WorldRules)
	}
}

func TestWorkflowRunnerGenerateChapterIdeaUsesPlotArchitectBrief(t *testing.T) {
	store, runner, textClient, projectID := newConfiguredWorkflowRunner(t)
	_ = store

	result, err := runner.GenerateChapterIdea(context.Background(), ChapterIdeaRequest{ProjectID: projectID, ChapterID: "chapter-7", Title: "第七章", Brief: "让主角进入灰塔", MaxOutputTokens: 777})
	if err != nil {
		t.Fatalf("GenerateChapterIdea() error: %v", err)
	}
	if result.ChapterIdea != "## 本章目标\n进入灰塔并发现钟声异常" {
		t.Fatalf("GenerateChapterIdea() chapter idea = %q", result.ChapterIdea)
	}
	if result.Workflow.Kind != "chapter_idea" || result.Workflow.Role != domain.AgentRolePlotArchitect || result.Workflow.Status != "completed" {
		t.Fatalf("GenerateChapterIdea() workflow mismatch: %+v", result.Workflow)
	}
	if result.Workflow.Output["chapter_idea"] != result.ChapterIdea {
		t.Fatalf("GenerateChapterIdea() workflow output did not persist chapter idea: %+v", result.Workflow.Output)
	}
	if len(textClient.requests) != 1 {
		t.Fatalf("GenerateChapterIdea() provider request count = %d, want 1", len(textClient.requests))
	}
	req := textClient.requests[0]
	if req.Model != "architect-explicit" {
		t.Fatalf("GenerateChapterIdea() model = %q, want explicit plot architect model", req.Model)
	}
	if req.MaxOutputTokens != 777 {
		t.Fatalf("GenerateChapterIdea() max output tokens = %d, want request override 777", req.MaxOutputTokens)
	}
	if !strings.Contains(req.SystemPrompt, "Plot Architect Agent") || !strings.Contains(req.UserPrompt, "不要写正文") {
		t.Fatalf("GenerateChapterIdea() prompt does not describe chapter idea contract: system=%q user=%q", req.SystemPrompt, req.UserPrompt)
	}
	if result.ContextPack.Role != domain.AgentRolePlotArchitect || result.ContextPack.ProjectID != projectID {
		t.Fatalf("GenerateChapterIdea() context pack mismatch: %+v", result.ContextPack)
	}
}

func TestWorkflowRunnerDraftChapterWithIdeaRoutesBothStagesAndPersistsLink(t *testing.T) {
	store, runner, textClient, projectID := newConfiguredWorkflowRunner(t)

	result, err := runner.DraftChapterWithIdea(context.Background(), DraftWithIdeaRequest{ProjectID: projectID, ChapterID: "chapter-8", Title: "第八章", Brief: "灰塔钟声引发追捕"})
	if err != nil {
		t.Fatalf("DraftChapterWithIdea() error: %v", err)
	}
	if result.ChapterIdea.ChapterIdea == "" || result.Draft.ChapterVersion.Content == "" {
		t.Fatalf("DraftChapterWithIdea() missing idea or draft: %+v", result)
	}
	if result.Draft.ContinuityAudit.Status == "" {
		t.Fatalf("DraftChapterWithIdea() continuity audit missing: %+v", result.Draft.ContinuityAudit)
	}
	if len(textClient.requests) != 2 {
		t.Fatalf("DraftChapterWithIdea() provider request count = %d, want 2", len(textClient.requests))
	}
	if textClient.requests[0].Model != "architect-explicit" {
		t.Fatalf("DraftChapterWithIdea() idea model = %q, want architect-explicit", textClient.requests[0].Model)
	}
	if textClient.requests[1].Model != "writer-explicit" {
		t.Fatalf("DraftChapterWithIdea() writer model = %q, want writer-explicit", textClient.requests[1].Model)
	}
	if !strings.Contains(textClient.requests[1].UserPrompt, "## 本章目标") {
		t.Fatalf("DraftChapterWithIdea() writer prompt did not include generated chapter idea: %q", textClient.requests[1].UserPrompt)
	}
	if result.Draft.Workflow.Input["chapter_idea_workflow_id"] != result.ChapterIdea.Workflow.ID {
		t.Fatalf("DraftChapterWithIdea() workflow input idea link = %q, want %q", result.Draft.Workflow.Input["chapter_idea_workflow_id"], result.ChapterIdea.Workflow.ID)
	}
	versions, err := store.ListChapterVersions(projectID, "chapter-8")
	if err != nil {
		t.Fatalf("ListChapterVersions() error: %v", err)
	}
	if len(versions) != 1 {
		t.Fatalf("ListChapterVersions() len = %d, want 1", len(versions))
	}
	if versions[0].Metadata["chapter_idea_workflow_id"] != result.ChapterIdea.Workflow.ID || versions[0].Metadata["chapter_idea_used"] != "true" {
		t.Fatalf("DraftChapterWithIdea() chapter version metadata missing idea link: %+v", versions[0].Metadata)
	}
}

func TestWorkflowRunnerDraftChapterRequiresIdeaWhenBriefIsOtherwiseEmpty(t *testing.T) {
	_, runner, _, projectID := newConfiguredWorkflowRunner(t)

	_, err := runner.DraftChapter(context.Background(), DraftRequest{ProjectID: projectID})
	if err == nil {
		t.Fatalf("DraftChapter() error = nil, want empty brief error")
	}
	if !strings.Contains(err.Error(), "draft brief must not be empty") {
		t.Fatalf("DraftChapter() error = %v, want empty brief validation", err)
	}
}

func TestWorkflowRunnerDraftChapterReturnsContinuityAudit(t *testing.T) {
	_, runner, _, projectID := newConfiguredWorkflowRunner(t)

	result, err := runner.DraftChapter(context.Background(), DraftRequest{ProjectID: projectID, ChapterID: "chapter-audit", Title: "审计测试", Brief: "围绕林烬推进并处理钥匙伏笔"})
	if err != nil {
		t.Fatalf("DraftChapter() error: %v", err)
	}
	if result.ContinuityAudit.Status == "" {
		t.Fatalf("DraftChapter() continuity audit missing: %+v", result.ContinuityAudit)
	}
	hasContinuityAuditStep := false
	for _, step := range result.Workflow.Steps {
		if step.Name == "continuity_audit" {
			hasContinuityAuditStep = true
			break
		}
	}
	if !hasContinuityAuditStep {
		t.Fatalf("DraftChapter() workflow steps missing continuity_audit: %+v", result.Workflow.Steps)
	}
}

func TestWorkflowRunnerSelectionFlowsIntoContextPackAndMetadata(t *testing.T) {
	store, runner, textClient, projectID := newConfiguredWorkflowRunner(t)
	selection := &ContextSelection{ChapterIDs: []string{"chapter-manual"}, CharacterNames: []string{"林烬"}}

	result, err := runner.DraftChapter(context.Background(), DraftRequest{ProjectID: projectID, ChapterID: "chapter-manual", Title: "手动测试", Brief: "围绕林烬推进", ContextSelection: selection, ContextNodeIDs: []string{"legacy-node-a"}})
	if err != nil {
		t.Fatalf("DraftChapter() error: %v", err)
	}
	if result.ContextPack.Metadata["selection_mode"] != "explicit" || result.ContextPack.Metadata["selection_character_names"] != "林烬" {
		t.Fatalf("unexpected context pack selection metadata: %+v", result.ContextPack.Metadata)
	}
	if result.Workflow.Input["context_selection.character_names"] != "林烬" {
		t.Fatalf("unexpected workflow selection input: %+v", result.Workflow.Input)
	}
	versions, err := store.ListChapterVersions(projectID, "chapter-manual")
	if err != nil {
		t.Fatalf("ListChapterVersions() error: %v", err)
	}
	if len(versions) < 2 {
		t.Fatalf("ListChapterVersions() len = %d, want at least 2", len(versions))
	}
	if versions[0].Metadata["context_selection.character_names"] != "林烬" || versions[0].Metadata["context_node_ids"] != "legacy-node-a" {
		t.Fatalf("unexpected latest chapter metadata: %+v", versions[0].Metadata)
	}
	if len(textClient.requests) != 1 {
		t.Fatalf("DraftChapter() provider request count = %d, want 1", len(textClient.requests))
	}
}

func newConfiguredWorkflowRunner(t *testing.T) (*memory.Store, *WorkflowRunner, *fakeTextClient, string) {
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
	_, err = store.CreateModel(domain.ModelConfig{ID: "provider_architect:architect-explicit", ProviderID: architectProvider.ID, Name: "architect-explicit", Kind: domain.ModelKindText, Enabled: true, MaxOutputTokens: 600, AllowedAgentRoles: []domain.AgentRole{domain.AgentRolePlotArchitect}})
	if err != nil {
		t.Fatalf("CreateModel(architect) error: %v", err)
	}
	_, err = store.CreateModel(domain.ModelConfig{ID: "provider_writer:writer-explicit", ProviderID: writerProvider.ID, Name: "writer-explicit", Kind: domain.ModelKindText, Enabled: true, MaxOutputTokens: 1400, AllowedAgentRoles: []domain.AgentRole{domain.AgentRoleWriter}})
	if err != nil {
		t.Fatalf("CreateModel(writer) error: %v", err)
	}
	_, err = store.UpsertSetting(domain.AppSetting{Scope: ModelRoutingSettingScope, Key: string(domain.AgentRolePlotArchitect), Value: map[string]any{ModelRoutingSettingValueKey: "provider_architect:architect-explicit"}})
	if err != nil {
		t.Fatalf("UpsertSetting(plot architect) error: %v", err)
	}
	_, err = store.UpsertSetting(domain.AppSetting{Scope: ModelRoutingSettingScope, Key: string(domain.AgentRoleWriter), Value: map[string]any{ModelRoutingSettingValueKey: "provider_writer:writer-explicit"}})
	if err != nil {
		t.Fatalf("UpsertSetting(writer) error: %v", err)
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
	textClient := &fakeTextClient{responses: []provider.ModelResponse{{Content: "## 本章目标\n进入灰塔并发现钟声异常"}, {Content: "林烬踏入灰塔，钟声在背后收拢。"}, {Content: "林烬把灰烬钥匙藏进袖口，走向塔门。"}}}
	router := NewModelRouter(store, NewAgentRoleRegistry())
	tools := NewToolRuntime(store)
	builder := NewContextPackBuilder(store, tools, store)
	runner := NewWorkflowRunner(store, router, builder, fakeTextClientFactory{client: textClient})
	return store, runner, textClient, project.ID
}

func TestWorkflowRunnerInitializeProjectCreatesRuleBasedGenesis(t *testing.T) {
	store := &fakeWorkflowStore{}
	runner := NewWorkflowRunner(store, nil, nil, nil)
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

	result, err := runner.InitializeProject(context.Background(), seed)
	if err != nil {
		t.Fatalf("InitializeProject() error: %v", err)
	}
	if result.Project.ID == "" {
		t.Fatalf("InitializeProject() project ID is empty")
	}
	if result.Bible.ID == "" {
		t.Fatalf("InitializeProject() story bible ID is empty")
	}
	if result.Workflow.ID == "" {
		t.Fatalf("InitializeProject() workflow ID is empty")
	}
	if result.Project.ActiveStoryBibleID != result.Bible.ID {
		t.Fatalf("InitializeProject() active story bible ID = %q, want %q", result.Project.ActiveStoryBibleID, result.Bible.ID)
	}
	if result.Workflow.ProjectID != result.Project.ID {
		t.Fatalf("InitializeProject() workflow project ID = %q, want %q", result.Workflow.ProjectID, result.Project.ID)
	}
	if result.Workflow.Status != "completed" {
		t.Fatalf("InitializeProject() workflow status = %q, want completed", result.Workflow.Status)
	}
	if len(store.projects) != 1 || len(store.bibles) != 1 || len(store.workflows) != 1 {
		t.Fatalf("InitializeProject() persisted projects/bibles/workflows = %d/%d/%d, want 1/1/1", len(store.projects), len(store.bibles), len(store.workflows))
	}
	if got := result.Project.Metadata["genesis_mode"]; got != genesisModeRuleBased {
		t.Fatalf("InitializeProject() project genesis mode = %q, want %q", got, genesisModeRuleBased)
	}
	if got := result.Workflow.Output["mode"]; got != genesisModeRuleBased {
		t.Fatalf("InitializeProject() workflow output mode = %q, want %q", got, genesisModeRuleBased)
	}
	if containsFallback(result.Project.Metadata["genesis_mode"], result.Workflow.Output["mode"]) {
		t.Fatalf("InitializeProject() genesis metadata/output contains fallback: %q / %q", result.Project.Metadata["genesis_mode"], result.Workflow.Output["mode"])
	}
}

func containsFallback(values ...string) bool {
	for _, value := range values {
		if strings.Contains(value, "fallback") {
			return true
		}
	}
	return false
}
