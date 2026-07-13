package agent

import (
	"context"
	"encoding/json"
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

func (s *fakeWorkflowStore) NewID(prefix string) (string, error) {
	return fmt.Sprintf("%s-%d", prefix, len(s.workflows)+len(s.projects)+len(s.chapterVersions)+1), nil
}

func (s *fakeWorkflowStore) SaveEntity(item domain.Entity) (domain.Entity, error) {
	return item, nil
}

func (s *fakeWorkflowStore) SaveGraphEdge(item domain.GraphEdge) (domain.GraphEdge, error) {
	return item, nil
}

func (s *fakeWorkflowStore) SavePlotThread(item domain.PlotThread) (domain.PlotThread, error) {
	return item, nil
}

func (s *fakeWorkflowStore) ListEntities(projectID string) ([]domain.Entity, error) {
	return nil, nil
}

func (s *fakeWorkflowStore) ListPlotThreads(projectID string) ([]domain.PlotThread, error) {
	return nil, nil
}

func (s *fakeWorkflowStore) ExpandGraph(projectID string, entityIDs []string, depth int) (domain.GraphExpansion, error) {
	return domain.GraphExpansion{ProjectID: projectID, Depth: depth}, nil
}

func (s *fakeWorkflowStore) GetChapter(id string) (domain.Chapter, error) {
	return domain.Chapter{ID: id, ProjectID: "project-1", Number: 1}, nil
}

func (s *fakeWorkflowStore) ListChapters(projectID string) ([]domain.Chapter, error) {
	return nil, nil
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

func (r fakeContextPackRepository) ListChapters(projectID string) ([]domain.Chapter, error) {
	chapters := make([]domain.Chapter, 0)
	seen := map[string]struct{}{}
	for _, version := range r.versions {
		if version.ProjectID != "" && version.ProjectID != projectID {
			continue
		}
		if _, ok := seen[version.ChapterID]; ok {
			continue
		}
		seen[version.ChapterID] = struct{}{}
		chapters = append(chapters, domain.Chapter{ID: version.ChapterID, ProjectID: projectID, Number: len(chapters) + 1, Title: version.Title, Status: domain.ChapterStatusDrafting})
	}
	return chapters, nil
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

func TestContextPackBuilderExplicitExcludeCurrentChapterDoesNotFallbackToAuto(t *testing.T) {
	includeWorldRules := true
	repo := fakeContextPackRepository{
		bible: domain.StoryBible{ID: "bible-1", Rules: map[string]string{"canon_policy": "必须遵守"}},
		versions: []domain.ChapterVersion{
			{ID: "cv-1", ProjectID: "project-1", ChapterID: "chapter-1", Title: "第一章", Summary: "前章摘要", Content: "前章正文"},
			{ID: "cv-2", ProjectID: "project-1", ChapterID: "chapter-2", Title: "第二章", Summary: "本章旧稿摘要", Content: "本章旧稿正文不应进入重写上下文"},
		},
		facts: []domain.Fact{
			{ID: "fact-1", ProjectID: "project-1", ChapterID: "chapter-1", ChapterVersionID: "cv-1", Claim: "前章事实"},
			{ID: "fact-2", ProjectID: "project-1", ChapterID: "chapter-2", ChapterVersionID: "cv-2", Claim: "本章旧事实"},
		},
		threads:   []domain.PlotThread{{ID: "thread-1", ProjectID: "project-1", Title: "线索", OpenedChapterID: "chapter-1"}},
		expansion: domain.GraphExpansion{},
	}
	builder := NewContextPackBuilder(repo, NewToolRuntime(repo), fakeIDSource{id: "context-pack-rewrite"})

	// Rewrite current chapter: only previous chapter_ids, no current chapter.
	pack, err := builder.BuildWithSelection("project-1", "chapter-2", domain.AgentRoleWriter, "重写本章", 1000, &ContextSelection{
		ChapterIDs:        []string{"chapter-1"},
		IncludeWorldRules: &includeWorldRules,
	}, nil)
	if err != nil {
		t.Fatalf("BuildWithSelection(rewrite with previous) error: %v", err)
	}
	if pack.Metadata["selection_mode"] != "explicit" {
		t.Fatalf("expected explicit selection mode, got %+v", pack.Metadata)
	}
	if len(pack.ChapterSummaries) != 1 || pack.ChapterSummaries[0].ChapterID != "chapter-1" {
		t.Fatalf("rewrite pack should only include previous chapter, got %+v", pack.ChapterSummaries)
	}
	for _, summary := range pack.ChapterSummaries {
		if summary.ChapterID == "chapter-2" {
			t.Fatalf("current chapter leaked into rewrite context summaries: %+v", pack.ChapterSummaries)
		}
	}

	// Rewrite with no chapter context at all (world rules only) must not auto-inject current chapter.
	pack, err = builder.BuildWithSelection("project-1", "chapter-2", domain.AgentRoleWriter, "纯重写", 1000, &ContextSelection{
		IncludeWorldRules: &includeWorldRules,
	}, nil)
	if err != nil {
		t.Fatalf("BuildWithSelection(rewrite without chapters) error: %v", err)
	}
	if pack.Metadata["selection_mode"] != "explicit" {
		t.Fatalf("expected explicit selection mode for empty chapter rewrite, got %+v", pack.Metadata)
	}
	if len(pack.ChapterSummaries) != 0 {
		t.Fatalf("rewrite without chapter_ids must not include any chapter summaries, got %+v", pack.ChapterSummaries)
	}
	if pack.WorldRules["canon_policy"] != "必须遵守" {
		t.Fatalf("expected world rules preserved for rewrite, got %+v", pack.WorldRules)
	}
}

func TestWorkflowRunnerGenerateChapterIdeaUsesPlotArchitectBrief(t *testing.T) {
	store, runner, textClient, projectID := newConfiguredWorkflowRunner(t)
	chapter, err := store.CreateChapter(domain.CreateChapterRequest{ProjectID: projectID, Number: 7, Title: "第七章"})
	if err != nil {
		t.Fatalf("CreateChapter(第七章) error: %v", err)
	}

	result, err := runner.GenerateChapterIdea(context.Background(), ChapterIdeaRequest{ProjectID: projectID, ChapterID: chapter.ID, Title: "第七章", Brief: "让主角进入灰塔", MaxOutputTokens: 777})
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
	if !strings.Contains(req.SystemPrompt, "Plot Architect Agent") ||
		!strings.Contains(req.SystemPrompt, "character.search") ||
		!strings.Contains(req.SystemPrompt, "character.upsert") ||
		!strings.Contains(req.SystemPrompt, "event.search") ||
		!strings.Contains(req.SystemPrompt, "event.upsert") ||
		!strings.Contains(req.SystemPrompt, "plot_thread.search") ||
		!strings.Contains(req.SystemPrompt, "plot_thread.upsert") ||
		!strings.Contains(req.SystemPrompt, "relationship.upsert") ||
		!strings.Contains(req.SystemPrompt, "伏笔处理") ||
		!requestContainsText(req, "不要写正文") ||
		!requestContainsText(req, "project_id："+projectID) {
		t.Fatalf("GenerateChapterIdea() prompt does not describe chapter idea contract: system=%q user=%q messages=%+v", req.SystemPrompt, req.UserPrompt, req.Messages)
	}
	if result.ContextPack.Role != domain.AgentRolePlotArchitect || result.ContextPack.ProjectID != projectID {
		t.Fatalf("GenerateChapterIdea() context pack mismatch: %+v", result.ContextPack)
	}
}

func TestWorkflowRunnerDraftChapterWithIdeaRoutesBothStagesAndPersistsLink(t *testing.T) {
	store, runner, textClient, projectID := newConfiguredWorkflowRunner(t)
	chapter, err := store.CreateChapter(domain.CreateChapterRequest{ProjectID: projectID, Number: 8, Title: "第八章"})
	if err != nil {
		t.Fatalf("CreateChapter() error: %v", err)
	}

	result, err := runner.DraftChapterWithIdea(context.Background(), DraftWithIdeaRequest{ProjectID: projectID, ChapterID: chapter.ID, Title: "第八章", Brief: "灰塔钟声引发追捕"})
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
	if !requestContainsText(textClient.requests[1], "## 本章目标") {
		t.Fatalf("DraftChapterWithIdea() writer prompt did not include generated chapter idea: user=%q messages=%+v", textClient.requests[1].UserPrompt, textClient.requests[1].Messages)
	}
	writerReq := textClient.requests[1]
	if !strings.Contains(writerReq.SystemPrompt, "Writer Agent") ||
		!strings.Contains(writerReq.SystemPrompt, "character.search") ||
		!strings.Contains(writerReq.SystemPrompt, "character.upsert") ||
		!strings.Contains(writerReq.SystemPrompt, "plot_thread.upsert") ||
		!strings.Contains(writerReq.SystemPrompt, "纯正文本身") ||
		!strings.Contains(writerReq.SystemPrompt, "Markdown") ||
		!requestContainsText(writerReq, "project_id："+projectID) {
		t.Fatalf("DraftChapterWithIdea() writer prompt missing tool contract: system=%q user=%q", writerReq.SystemPrompt, writerReq.UserPrompt)
	}
	if result.Draft.Workflow.Input["chapter_idea_workflow_id"] != result.ChapterIdea.Workflow.ID {
		t.Fatalf("DraftChapterWithIdea() workflow input idea link = %q, want %q", result.Draft.Workflow.Input["chapter_idea_workflow_id"], result.ChapterIdea.Workflow.ID)
	}
	versions, err := store.ListChapterVersions(projectID, chapter.ID)
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

func TestWorkflowRunnerGenerateCharacterProfilesUsesCharacterKeeperToolLoopAndParsesProfiles(t *testing.T) {
	store, runner, textClient, projectID := newConfiguredWorkflowRunner(t)
	textClient.responses = []provider.ModelResponse{
		{ToolCalls: []provider.ToolCall{
			{ID: "call-search-lin", Name: "character.search", Arguments: mustRawJSON(t, map[string]any{"project_id": projectID, "query": "林烬", "limit": 5})},
			{ID: "call-upsert-lin", Name: "character.upsert", Arguments: mustRawJSON(t, map[string]any{"project_id": projectID, "name": "林烬", "summary": "背负旧债的远航者。", "traits": map[string]any{"role": "主角", "desire": "找回失落舰队的真相", "wound": "曾在撤离中放弃同伴", "secret": "他携带舰队核心坐标"}, "metadata": map[string]any{"source": "ai_character_profiles"}})},
		}},
		{Content: `{"characters":[{"name":"林烬","role":"主角","desire":"找回失落舰队的真相","wound":"曾在撤离中放弃同伴","secret":"他携带舰队核心坐标","summary":"背负旧债的远航者。"}]}`},
	}

	result, err := runner.GenerateCharacterProfiles(context.Background(), CharacterProfilesRequest{ProjectID: projectID, Focus: "主角完整设定", Count: 1, Brief: "补全能支撑第一卷的主角人物弧", MaxOutputTokens: 888})
	if err != nil {
		t.Fatalf("GenerateCharacterProfiles() error: %v", err)
	}
	if result.Workflow.Kind != "character_profiles" || result.Workflow.Role != domain.AgentRoleCharacterKeeper || result.Workflow.Status != "completed" {
		t.Fatalf("GenerateCharacterProfiles() workflow mismatch: %+v", result.Workflow)
	}
	if result.Workflow.Output["character_count"] != "1" {
		t.Fatalf("GenerateCharacterProfiles() workflow character_count = %q, want 1", result.Workflow.Output["character_count"])
	}
	if len(result.Characters) != 1 {
		t.Fatalf("GenerateCharacterProfiles() characters len = %d, want 1", len(result.Characters))
	}
	character := result.Characters[0]
	if character.Name != "林烬" || character.Role != "主角" || character.Desire == "" || character.Wound == "" || character.Secret == "" {
		t.Fatalf("GenerateCharacterProfiles() character missing required profile fields: %+v", character)
	}
	if len(textClient.requests) < 2 {
		t.Fatalf("GenerateCharacterProfiles() provider request count = %d, want at least 2", len(textClient.requests))
	}
	req := textClient.requests[0]
	if req.Model != "character-explicit" {
		t.Fatalf("GenerateCharacterProfiles() model = %q, want character-explicit", req.Model)
	}
	if req.MaxOutputTokens != 888 {
		t.Fatalf("GenerateCharacterProfiles() max output tokens = %d, want 888", req.MaxOutputTokens)
	}
	if len(req.Tools) == 0 {
		t.Fatalf("GenerateCharacterProfiles() first request missing tools")
	}
	if !strings.Contains(req.SystemPrompt, "Character Keeper Agent") || !strings.Contains(req.SystemPrompt, "严格 JSON") || !strings.Contains(req.SystemPrompt, "character.search") || !strings.Contains(req.SystemPrompt, "character.upsert") || !requestContainsText(req, "主角完整设定") || !requestContainsText(req, "可直接写入 Story Bible") {
		t.Fatalf("GenerateCharacterProfiles() prompt does not describe character tool contract: system=%q messages=%+v", req.SystemPrompt, req.Messages)
	}
	if !requestHasToolResult(textClient.requests[1], "character.search") || !requestHasToolResult(textClient.requests[1], "character.upsert") {
		t.Fatalf("GenerateCharacterProfiles() second request missing tool result history: %+v", textClient.requests[1].Messages)
	}
	entities, err := store.ListEntities(projectID)
	if err != nil {
		t.Fatalf("ListEntities() error: %v", err)
	}
	var saved domain.Entity
	for _, entity := range entities {
		if entity.Name == "林烬" && entity.Type == "character" {
			saved = entity
			break
		}
	}
	if saved.ID == "" || saved.Summary != "背负旧债的远航者。" || saved.Traits["secret"] != "他携带舰队核心坐标" {
		t.Fatalf("GenerateCharacterProfiles() did not persist character through tool call: %+v all=%+v", saved, entities)
	}
	if len(result.Entities) != 1 || result.Entities[0].ID != saved.ID {
		t.Fatalf("GenerateCharacterProfiles() result entities not extracted from tool result: %+v saved=%+v", result.Entities, saved)
	}
	if len(result.Mappings) != 1 || result.Mappings[0].EntityID != saved.ID || result.Mappings[0].Action == "" {
		t.Fatalf("GenerateCharacterProfiles() result mappings mismatch: %+v saved=%+v", result.Mappings, saved)
	}
	if len(result.ToolTrace) != 2 || !strings.Contains(strings.Join(result.ToolTrace, "\n"), "character.upsert") {
		t.Fatalf("GenerateCharacterProfiles() tool trace mismatch: %+v", result.ToolTrace)
	}
	if result.ContextPack.Role != domain.AgentRoleCharacterKeeper || result.ModelResolution.RouteKey != string(domain.AgentRoleCharacterKeeper) {
		t.Fatalf("GenerateCharacterProfiles() context/model role mismatch: pack=%+v resolution=%+v", result.ContextPack, result.ModelResolution)
	}
}

func TestWorkflowRunnerGenerateCharacterProfilesRejectsInvalidJSON(t *testing.T) {
	_, runner, textClient, projectID := newConfiguredWorkflowRunner(t)
	textClient.responses = []provider.ModelResponse{
		{ToolCalls: []provider.ToolCall{
			{ID: "call-search-lin", Name: "character.search", Arguments: mustRawJSON(t, map[string]any{"project_id": projectID, "query": "林烬", "limit": 5})},
			{ID: "call-upsert-lin", Name: "character.upsert", Arguments: mustRawJSON(t, map[string]any{"project_id": projectID, "name": "林烬", "summary": "背负旧债的远航者。", "traits": map[string]any{"role": "主角"}})},
		}},
		{Content: `{"characters":[{"name":"林烬","role":"主角"}]}`},
	}

	_, err := runner.GenerateCharacterProfiles(context.Background(), CharacterProfilesRequest{ProjectID: projectID, Focus: "主角完整设定", Count: 1, Brief: "补全主角"})
	if err == nil {
		t.Fatalf("GenerateCharacterProfiles() error = nil, want validation error")
	}
	if !strings.Contains(err.Error(), "desire must not be empty") {
		t.Fatalf("GenerateCharacterProfiles() error = %v, want missing desire validation", err)
	}
}

func TestWorkflowRunnerGenerateCharacterProfilesRejectsFinalJSONWithoutToolCall(t *testing.T) {
	_, runner, textClient, projectID := newConfiguredWorkflowRunner(t)
	textClient.responses = []provider.ModelResponse{{Content: `{"characters":[{"name":"林烬","role":"主角","desire":"找回失落舰队的真相","wound":"曾在撤离中放弃同伴","secret":"他携带舰队核心坐标","summary":"背负旧债的远航者。"}]}`}}

	_, err := runner.GenerateCharacterProfiles(context.Background(), CharacterProfilesRequest{ProjectID: projectID, Focus: "主角完整设定", Count: 1, Brief: "补全主角"})
	if err == nil {
		t.Fatalf("GenerateCharacterProfiles() error = nil, want missing tool call error")
	}
	if !strings.Contains(err.Error(), "require character.search and character.upsert tool calls") {
		t.Fatalf("GenerateCharacterProfiles() error = %v, want missing tool call validation", err)
	}
}

func TestWorkflowRunnerGenerateCharacterProfilesRejectsMissingUpsertForFinalCharacter(t *testing.T) {
	_, runner, textClient, projectID := newConfiguredWorkflowRunner(t)
	textClient.responses = []provider.ModelResponse{
		{ToolCalls: []provider.ToolCall{{ID: "call-search-lin", Name: "character.search", Arguments: mustRawJSON(t, map[string]any{"project_id": projectID, "query": "林烬", "limit": 5})}}},
		{Content: `{"characters":[{"name":"林烬","role":"主角","desire":"找回失落舰队的真相","wound":"曾在撤离中放弃同伴","secret":"他携带舰队核心坐标","summary":"背负旧债的远航者。"}]}`},
	}

	_, err := runner.GenerateCharacterProfiles(context.Background(), CharacterProfilesRequest{ProjectID: projectID, Focus: "主角完整设定", Count: 1, Brief: "补全主角"})
	if err == nil {
		t.Fatalf("GenerateCharacterProfiles() error = nil, want missing upsert error")
	}
	if !strings.Contains(err.Error(), "without matching character.upsert tool result") {
		t.Fatalf("GenerateCharacterProfiles() error = %v, want missing upsert validation", err)
	}
}

func TestWorkflowRunnerDraftChapterRequiresIdeaWhenBriefIsOtherwiseEmpty(t *testing.T) {
	_, runner, _, projectID := newConfiguredWorkflowRunner(t)

	_, err := runner.DraftChapter(context.Background(), DraftRequest{ProjectID: projectID})
	if err == nil {
		t.Fatalf("DraftChapter() error = nil, want empty chapter_id error")
	}
	if !strings.Contains(err.Error(), "draft chapter_id must not be empty") {
		t.Fatalf("DraftChapter() error = %v, want empty chapter_id validation", err)
	}
}

func TestWorkflowRunnerDraftChapterRejectsMissingChapterBeforeModelCall(t *testing.T) {
	_, runner, textClient, projectID := newConfiguredWorkflowRunner(t)

	_, err := runner.DraftChapter(context.Background(), DraftRequest{ProjectID: projectID, ChapterID: "missing", Brief: "不应调用模型"})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("DraftChapter() error = %v, want missing chapter", err)
	}
	if len(textClient.requests) != 0 {
		t.Fatalf("DraftChapter() called model before chapter validation: %d requests", len(textClient.requests))
	}
}

func TestWorkflowRunnerDraftChapterReturnsContinuityAudit(t *testing.T) {
	store, runner, _, projectID := newConfiguredWorkflowRunner(t)
	chapter, err := store.CreateChapter(domain.CreateChapterRequest{ProjectID: projectID, Title: "审计测试"})
	if err != nil {
		t.Fatalf("CreateChapter() error: %v", err)
	}

	result, err := runner.DraftChapter(context.Background(), DraftRequest{ProjectID: projectID, ChapterID: chapter.ID, Title: "审计测试", Brief: "围绕林烬推进并处理钥匙伏笔"})
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

func TestNarrativeToolSpecsDoNotExposeChapterCreation(t *testing.T) {
	for _, spec := range NarrativeToolSpecs() {
		if spec.Name == "chapter.ensure" || spec.Name == "chapter.create" {
			t.Fatalf("agent tool %q must not expose chapter creation", spec.Name)
		}
	}
	_, err := NewToolExecutor(memory.NewStore()).Execute(context.Background(), provider.ToolCall{Name: "chapter.ensure", Arguments: mustRawJSON(t, map[string]any{"project_id": "project"})})
	if err == nil || !strings.Contains(err.Error(), "not whitelisted") {
		t.Fatalf("chapter.ensure execution error = %v, want not whitelisted", err)
	}
}

func TestToolLoopExecutesGraphToolsAndReturnsResults(t *testing.T) {
	store := memory.NewStore()
	project, _, err := store.CreateProject(domain.Project{Title: "工具循环", Slug: "tools", Status: "active"}, domain.StoryBible{Title: "工具循环", Logline: "测试"})
	if err != nil {
		t.Fatalf("CreateProject() error: %v", err)
	}
	anchor, err := store.SaveEntity(domain.Entity{ProjectID: project.ID, Name: "主线开始", Type: "time_node", Metadata: map[string]string{"chronology_key": "0100"}})
	if err != nil {
		t.Fatalf("SaveEntity(anchor) error: %v", err)
	}
	calls := []provider.ToolCall{
		{ID: "call-character", Name: "character.upsert", Arguments: mustRawJSON(t, map[string]any{"project_id": project.ID, "name": "林烬", "summary": "背负旧债的远航者", "traits": map[string]any{"role": "主角"}})},
		{ID: "call-event", Name: "event.upsert", Arguments: mustRawJSON(t, map[string]any{"project_id": project.ID, "name": "撤离夜", "summary": "林烬在撤离中失去同伴", "chronology_key": "0001"})},
		{ID: "call-prequel", Name: "timeline.node.create_before", Arguments: mustRawJSON(t, map[string]any{"project_id": project.ID, "anchor_id": anchor.ID, "name": "前传节点", "summary": "灾难发生前的静夜"})},
	}
	textClient := &fakeTextClient{responses: []provider.ModelResponse{{ToolCalls: calls}, {Content: "完成工具写入"}}}
	result, err := RunToolLoop(context.Background(), textClient, provider.TextRequest{Model: "fake", UserPrompt: "写入图谱", Tools: NarrativeToolSpecs()}, NewToolExecutor(store), 4)
	if err != nil {
		t.Fatalf("RunToolLoop() error: %v", err)
	}
	if result.Response.Content != "完成工具写入" {
		t.Fatalf("final response content = %q", result.Response.Content)
	}
	if len(textClient.requests) != 2 {
		t.Fatalf("provider request count = %d, want 2", len(textClient.requests))
	}
	if len(textClient.requests[1].Messages) < 5 {
		t.Fatalf("second request missing tool result history: %+v", textClient.requests[1].Messages)
	}
	entities, err := store.ListEntities(project.ID)
	if err != nil {
		t.Fatalf("ListEntities() error: %v", err)
	}
	var characterID, eventID, prequelID string
	for _, entity := range entities {
		switch entity.Name {
		case "林烬":
			characterID = entity.ID
			if entity.Type != "character" {
				t.Fatalf("林烬 type = %q, want character", entity.Type)
			}
		case "撤离夜":
			eventID = entity.ID
			if entity.Type != "event" || entity.Metadata["chronology_key"] != "0001" {
				t.Fatalf("event entity mismatch: %+v", entity)
			}
		case "前传节点":
			prequelID = entity.ID
			if entity.Type != "time_node" || entity.Metadata["time_scope"] != "prequel" || !strings.Contains(entity.Metadata["chronology_key"], "prequel") {
				t.Fatalf("prequel node metadata mismatch: %+v", entity)
			}
		}
	}
	if characterID == "" || eventID == "" || prequelID == "" {
		t.Fatalf("missing saved entities character=%q event=%q prequel=%q all=%+v", characterID, eventID, prequelID, entities)
	}
	relationCall := provider.ToolCall{ID: "call-relation", Name: "relationship.upsert", Arguments: mustRawJSON(t, map[string]any{"project_id": project.ID, "source_id": characterID, "target_id": eventID, "type": "involved_in", "label": "卷入"})}
	if _, err := NewToolExecutor(store).Execute(context.Background(), relationCall); err != nil {
		t.Fatalf("relationship Execute() error: %v", err)
	}
	expansion, err := store.ExpandGraph(project.ID, []string{characterID}, 1)
	if err != nil {
		t.Fatalf("ExpandGraph() error: %v", err)
	}
	if len(expansion.Edges) == 0 {
		t.Fatalf("expected relationship edge in expansion: %+v", expansion)
	}
	for _, record := range result.ToolCalls {
		if len(record.Result) == 0 || !json.Valid(record.Result) {
			t.Fatalf("tool record %s result is not valid JSON: %q", record.Name, string(record.Result))
		}
	}
}

func TestWorkflowRunnerSelectionFlowsIntoContextPackAndMetadata(t *testing.T) {
	store, runner, textClient, projectID := newConfiguredWorkflowRunner(t)
	chapters, err := store.ListChapters(projectID)
	if err != nil || len(chapters) == 0 {
		t.Fatalf("ListChapters() chapters=%+v err=%v", chapters, err)
	}
	chapterID := chapters[0].ID
	selection := &ContextSelection{ChapterIDs: []string{chapterID}, CharacterNames: []string{"林烬"}}

	result, err := runner.DraftChapter(context.Background(), DraftRequest{ProjectID: projectID, ChapterID: chapterID, Title: "手动测试", Brief: "围绕林烬推进", ContextSelection: selection, ContextNodeIDs: []string{"legacy-node-a"}})
	if err != nil {
		t.Fatalf("DraftChapter() error: %v", err)
	}
	if result.ContextPack.Metadata["selection_mode"] != "explicit" || result.ContextPack.Metadata["selection_character_names"] != "林烬" {
		t.Fatalf("unexpected context pack selection metadata: %+v", result.ContextPack.Metadata)
	}
	if result.Workflow.Input["context_selection.character_names"] != "林烬" {
		t.Fatalf("unexpected workflow selection input: %+v", result.Workflow.Input)
	}
	versions, err := store.ListChapterVersions(projectID, chapterID)
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
	_, err = store.UpsertSetting(domain.AppSetting{Scope: ModelRoutingSettingScope, Key: string(domain.AgentRolePlotArchitect), Value: map[string]any{ModelRoutingSettingValueKey: "provider_architect:architect-explicit"}})
	if err != nil {
		t.Fatalf("UpsertSetting(plot architect) error: %v", err)
	}
	_, err = store.UpsertSetting(domain.AppSetting{Scope: ModelRoutingSettingScope, Key: string(domain.AgentRoleWriter), Value: map[string]any{ModelRoutingSettingValueKey: "provider_writer:writer-explicit"}})
	if err != nil {
		t.Fatalf("UpsertSetting(writer) error: %v", err)
	}
	_, err = store.UpsertSetting(domain.AppSetting{Scope: ModelRoutingSettingScope, Key: string(domain.AgentRoleCharacterKeeper), Value: map[string]any{ModelRoutingSettingValueKey: "provider_character:character-explicit"}})
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
	chapterManual, err := store.CreateChapter(domain.CreateChapterRequest{ProjectID: project.ID, Number: 1, Title: "前章"})
	if err != nil {
		t.Fatalf("CreateChapter(前章) error: %v", err)
	}
	chapterOther, err := store.CreateChapter(domain.CreateChapterRequest{ProjectID: project.ID, Number: 2, Title: "旁章"})
	if err != nil {
		t.Fatalf("CreateChapter(旁章) error: %v", err)
	}
	chapterOne, _, err := store.SaveChapterVersion(domain.ChapterVersion{ID: "chapter_version_seed_1", ProjectID: project.ID, ChapterID: chapterManual.ID, Title: "前章", Content: "林烬得到灰烬钥匙", Summary: "林烬得到灰烬钥匙", AuthorRole: domain.AgentRoleWriter, SourceWorkflowID: seedWorkflow.ID, IndexStatus: "indexed"})
	if err != nil {
		t.Fatalf("SaveChapterVersion(前章) error: %v", err)
	}
	chapterTwo, _, err := store.SaveChapterVersion(domain.ChapterVersion{ID: "chapter_version_seed_2", ProjectID: project.ID, ChapterID: chapterOther.ID, Title: "旁章", Content: "苏九调查灰塔", Summary: "苏九调查灰塔", AuthorRole: domain.AgentRoleWriter, SourceWorkflowID: seedWorkflow.ID, IndexStatus: "indexed"})
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
	textClient := &fakeTextClient{responses: []provider.ModelResponse{{Content: "## 本章目标\n进入灰塔并发现钟声异常"}, {Content: "林烬踏入灰塔，钟声在背后收拢。"}, {Content: "林烬把灰烬钥匙藏进袖口，走向塔门。"}, {Content: `{"characters":[{"name":"林烬","role":"主角","desire":"找回失落舰队的真相","wound":"曾在撤离中放弃同伴","secret":"他携带舰队核心坐标","summary":"背负旧债的远航者。"}]}`}}}
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

func mustRawJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal JSON fixture: %v", err)
	}
	return payload
}

func requestContainsText(req provider.TextRequest, needle string) bool {
	if strings.Contains(req.UserPrompt, needle) || strings.Contains(req.SystemPrompt, needle) {
		return true
	}
	for _, message := range req.Messages {
		if strings.Contains(message.Content, needle) {
			return true
		}
	}
	return false
}

func requestHasToolResult(req provider.TextRequest, toolName string) bool {
	for _, message := range req.Messages {
		if message.Role == "tool" && message.Name == toolName && strings.TrimSpace(message.Content) != "" {
			return true
		}
	}
	return false
}

func containsFallback(values ...string) bool {
	for _, value := range values {
		if strings.Contains(value, "fallback") {
			return true
		}
	}
	return false
}
