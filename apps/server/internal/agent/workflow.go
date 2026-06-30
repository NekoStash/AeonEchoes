package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/provider"
)

// WorkflowStore is the repository surface required by workflow orchestration.
type WorkflowStore interface {
	NewID(prefix string) (string, error)
	CreateProject(project domain.Project, bible domain.StoryBible) (domain.Project, domain.StoryBible, error)
	SaveEntity(item domain.Entity) (domain.Entity, error)
	SaveGraphEdge(item domain.GraphEdge) (domain.GraphEdge, error)
	SavePlotThread(item domain.PlotThread) (domain.PlotThread, error)
	ListEntities(projectID string) ([]domain.Entity, error)
	ListPlotThreads(projectID string) ([]domain.PlotThread, error)
	ExpandGraph(projectID string, entityIDs []string, depth int) (domain.GraphExpansion, error)
	EnsureChapter(req domain.ChapterEnsureRequest) (domain.Chapter, error)
	GetChapter(id string) (domain.Chapter, error)
	ListChapters(projectID string) ([]domain.Chapter, error)
	SaveChapterVersion(version domain.ChapterVersion) (domain.ChapterVersion, domain.IndexJob, error)
	SaveWorkflow(workflow domain.AIWorkflow) (domain.AIWorkflow, error)
	ListChapterVersions(projectID, chapterID string) ([]domain.ChapterVersion, error)
	ListPendingIndexJobs(projectID string, limit int) ([]domain.IndexJob, error)
}

// TextClientFactory creates concrete text clients for routed provider configs.
type TextClientFactory interface {
	NewTextClient(cfg domain.ProviderConfig) (provider.TextModelClient, error)
}

// WorkflowRunner coordinates deterministic state-machine workflows.
type WorkflowRunner struct {
	store   WorkflowStore
	router  *ModelRouter
	builder *ContextPackBuilder
	clients TextClientFactory
	auditor ContinuityAuditor
}

func NewWorkflowRunner(store WorkflowStore, router *ModelRouter, builder *ContextPackBuilder, clients TextClientFactory) *WorkflowRunner {
	return &WorkflowRunner{store: store, router: router, builder: builder, clients: clients, auditor: NewRuleBasedContinuityAuditor()}
}

const genesisModeRuleBased = "rule_based_genesis"

type InitializeResult struct {
	Project  domain.Project    `json:"project"`
	Bible    domain.StoryBible `json:"story_bible"`
	Workflow domain.AIWorkflow `json:"workflow"`
}

type ContextSelection struct {
	ChapterIDs        []string `json:"chapter_ids,omitempty"`
	CharacterIDs      []string `json:"character_ids,omitempty"`
	CharacterNames    []string `json:"character_names,omitempty"`
	IncludeWorldRules *bool    `json:"include_world_rules,omitempty"`
}

type ChapterIdeaRequest struct {
	ProjectID          string            `json:"project_id"`
	ChapterID          string            `json:"chapter_id,omitempty"`
	Title              string            `json:"title,omitempty"`
	Brief              string            `json:"brief"`
	Prompt             string            `json:"prompt,omitempty"`
	ContextNodeIDs     []string          `json:"context_node_ids,omitempty"`
	ContextSelection   *ContextSelection `json:"context_selection,omitempty"`
	ReferenceSelection *ContextSelection `json:"reference_selection,omitempty"`
	StyleConstraints   []string          `json:"style_constraints,omitempty"`
	MaxOutputTokens    int               `json:"max_output_tokens,omitempty"`
}

type ChapterIdeaResult struct {
	Workflow        domain.AIWorkflow      `json:"workflow"`
	ContextPack     domain.ContextPack     `json:"context_pack"`
	ChapterIdea     string                 `json:"chapter_idea"`
	ModelResolution domain.ModelResolution `json:"model_resolution"`
	ToolTrace       []string               `json:"tool_trace,omitempty"`
}

type CharacterProfilesRequest struct {
	ProjectID        string            `json:"project_id"`
	Focus            string            `json:"focus"`
	Count            int               `json:"count"`
	Brief            string            `json:"brief"`
	ChapterID        string            `json:"chapter_id,omitempty"`
	ContextNodeIDs   []string          `json:"context_node_ids,omitempty"`
	ContextSelection *ContextSelection `json:"context_selection,omitempty"`
	MaxOutputTokens  int               `json:"max_output_tokens,omitempty"`
}

type CharacterProfilesResult struct {
	Workflow        domain.AIWorkflow                `json:"workflow"`
	ContextPack     domain.ContextPack               `json:"context_pack"`
	Characters      []domain.CharacterProfile        `json:"characters"`
	Entities        []domain.Entity                  `json:"entities,omitempty"`
	Mappings        []domain.CharacterProfileMapping `json:"mappings,omitempty"`
	ModelResolution domain.ModelResolution           `json:"model_resolution"`
	ToolTrace       []string                         `json:"tool_trace,omitempty"`
}

type DraftRequest struct {
	ProjectID             string            `json:"project_id"`
	ChapterID             string            `json:"chapter_id,omitempty"`
	Title                 string            `json:"title,omitempty"`
	Brief                 string            `json:"brief"`
	Prompt                string            `json:"prompt,omitempty"`
	ChapterIdea           string            `json:"chapter_idea,omitempty"`
	ChapterIdeaWorkflowID string            `json:"chapter_idea_workflow_id,omitempty"`
	ContextNodeIDs        []string          `json:"context_node_ids,omitempty"`
	ContextSelection      *ContextSelection `json:"context_selection,omitempty"`
	ReferenceSelection    *ContextSelection `json:"reference_selection,omitempty"`
	StyleConstraints      []string          `json:"style_constraints,omitempty"`
	Role                  domain.AgentRole  `json:"role,omitempty"`
	MaxOutputTokens       int               `json:"max_output_tokens,omitempty"`
}

type DraftResult struct {
	Workflow        domain.AIWorkflow      `json:"workflow"`
	ContextPack     domain.ContextPack     `json:"context_pack"`
	ChapterVersion  domain.ChapterVersion  `json:"chapter_version"`
	IndexJob        domain.IndexJob        `json:"index_job"`
	IndexFreshness  domain.IndexFreshness  `json:"index_freshness"`
	ModelResolution domain.ModelResolution `json:"model_resolution"`
	ContinuityAudit domain.ContinuityAudit `json:"continuity_audit"`
	ToolTrace       []string               `json:"tool_trace,omitempty"`
}

type DraftWithIdeaRequest struct {
	ProjectID            string            `json:"project_id"`
	ChapterID            string            `json:"chapter_id,omitempty"`
	Title                string            `json:"title,omitempty"`
	Brief                string            `json:"brief"`
	Prompt               string            `json:"prompt,omitempty"`
	ContextNodeIDs       []string          `json:"context_node_ids,omitempty"`
	ContextSelection     *ContextSelection `json:"context_selection,omitempty"`
	ReferenceSelection   *ContextSelection `json:"reference_selection,omitempty"`
	StyleConstraints     []string          `json:"style_constraints,omitempty"`
	MaxIdeaOutputTokens  int               `json:"max_idea_output_tokens,omitempty"`
	MaxDraftOutputTokens int               `json:"max_draft_output_tokens,omitempty"`
}

type DraftWithIdeaResult struct {
	ChapterIdea     ChapterIdeaResult      `json:"chapter_idea"`
	Draft           DraftResult            `json:"draft"`
	ModelResolution domain.ModelResolution `json:"model_resolution"`
}

type ContextSelectionPreviewRequest struct {
	ProjectID          string            `json:"project_id"`
	ChapterID          string            `json:"chapter_id,omitempty"`
	Title              string            `json:"title,omitempty"`
	Brief              string            `json:"brief,omitempty"`
	Prompt             string            `json:"prompt,omitempty"`
	ContextNodeIDs     []string          `json:"context_node_ids,omitempty"`
	ContextSelection   *ContextSelection `json:"context_selection,omitempty"`
	ReferenceSelection *ContextSelection `json:"reference_selection,omitempty"`
	StyleConstraints   []string          `json:"style_constraints,omitempty"`
	Role               domain.AgentRole  `json:"role,omitempty"`
	TokenBudget        int               `json:"token_budget,omitempty"`
}

type ContextSelectionPreviewResult struct {
	ContextPack     domain.ContextPack     `json:"context_pack"`
	Summary         string                 `json:"summary"`
	EstimatedTokens int                    `json:"estimated_tokens"`
	IndexFreshness  domain.IndexFreshness  `json:"index_freshness"`
	ModelResolution domain.ModelResolution `json:"model_resolution"`
}

func (r *WorkflowRunner) InitializeProject(ctx context.Context, seed domain.ProjectSeed) (InitializeResult, error) {
	if r == nil || r.store == nil {
		return InitializeResult{}, fmt.Errorf("workflow runner is not configured")
	}
	if strings.TrimSpace(seed.Title) == "" {
		return InitializeResult{}, fmt.Errorf("project seed title must not be empty")
	}
	if strings.TrimSpace(seed.Premise) == "" {
		return InitializeResult{}, fmt.Errorf("project seed premise must not be empty")
	}
	workflow := domain.AIWorkflow{ProjectID: "pending", Kind: "genesis", Role: domain.AgentRoleGenesisOptimizer, Status: "running", Input: map[string]string{"title": seed.Title, "premise": seed.Premise}, CreatedAt: nowUTC(), UpdatedAt: nowUTC()}
	workflow.Steps = append(workflow.Steps, stepDone("validate_seed"))
	bible := deterministicBible(seed)
	workflow.Steps = append(workflow.Steps, stepDone("rule_based_story_bible"))
	project := domain.Project{Title: seed.Title, Slug: slugify(seed.Title), Seed: seed, Status: "active", Metadata: map[string]string{"genesis_mode": genesisModeRuleBased}}
	createdProject, createdBible, err := r.store.CreateProject(project, bible)
	if err != nil {
		return InitializeResult{}, err
	}
	workflow.ProjectID = createdProject.ID
	workflow.Status = "completed"
	workflow.Output = map[string]string{"story_bible_id": createdBible.ID, "mode": genesisModeRuleBased}
	workflow.Steps = append(workflow.Steps, stepDone("persist_project"))
	workflow, err = r.store.SaveWorkflow(workflow)
	if err != nil {
		return InitializeResult{}, err
	}
	_ = ctx
	return InitializeResult{Project: createdProject, Bible: createdBible, Workflow: workflow}, nil
}

func (r *WorkflowRunner) PreviewContextSelection(ctx context.Context, req ContextSelectionPreviewRequest) (ContextSelectionPreviewResult, error) {
	if r == nil || r.store == nil || r.router == nil || r.builder == nil {
		return ContextSelectionPreviewResult{}, fmt.Errorf("workflow runner is not fully configured")
	}
	if strings.TrimSpace(req.ProjectID) == "" {
		return ContextSelectionPreviewResult{}, fmt.Errorf("context selection preview project_id must not be empty")
	}
	selection := effectiveContextSelection(req.ContextSelection, req.ReferenceSelection)
	role := req.Role
	if role == "" {
		role = domain.AgentRoleWriter
	}
	query := joinNonEmpty([]string{req.Brief, req.Prompt})
	if query == "" {
		query = firstText(strings.TrimSpace(req.Title), req.ChapterID, "context selection preview")
	}
	pack, selectionModel, modelResolution, err := r.buildPackAndResolveModel(req.ProjectID, req.ChapterID, role, query, firstPositive(req.TokenBudget, 6000), selection, req.ContextNodeIDs)
	if err != nil {
		return ContextSelectionPreviewResult{}, err
	}
	freshness, err := r.computeIndexFreshness(req.ProjectID, req.ChapterID)
	if err != nil {
		return ContextSelectionPreviewResult{}, err
	}
	_ = ctx
	return ContextSelectionPreviewResult{ContextPack: pack, Summary: summarizeContextPack(pack), EstimatedTokens: estimateContextPackTokens(pack), IndexFreshness: freshness, ModelResolution: withResolvedIDs(modelResolution, selectionModel, pack)}, nil
}

func (r *WorkflowRunner) GenerateChapterIdea(ctx context.Context, req ChapterIdeaRequest) (ChapterIdeaResult, error) {
	if r == nil || r.store == nil || r.router == nil || r.builder == nil || r.clients == nil {
		return ChapterIdeaResult{}, fmt.Errorf("workflow runner is not fully configured")
	}
	if strings.TrimSpace(req.ProjectID) == "" {
		return ChapterIdeaResult{}, fmt.Errorf("chapter idea project_id must not be empty")
	}
	req.Brief = normalizeChapterIdeaBrief(req)
	if strings.TrimSpace(req.Brief) == "" {
		return ChapterIdeaResult{}, fmt.Errorf("chapter idea brief must not be empty")
	}
	selection := effectiveContextSelection(req.ContextSelection, req.ReferenceSelection)
	role := domain.AgentRolePlotArchitect
	workflow := domain.AIWorkflow{ProjectID: req.ProjectID, Kind: "chapter_idea", Role: role, Status: "running", Input: chapterIdeaWorkflowInput(req), CreatedAt: nowUTC(), UpdatedAt: nowUTC()}
	workflow.Steps = append(workflow.Steps, stepDone("retrieve_context"))
	pack, selectionModel, modelResolution, err := r.buildPackAndResolveModel(req.ProjectID, req.ChapterID, role, req.Brief, 6000, selection, req.ContextNodeIDs)
	if err != nil {
		return ChapterIdeaResult{}, r.failWorkflow(workflow, err)
	}
	client, err := r.clients.NewTextClient(selectionModel.Provider)
	if err != nil {
		return ChapterIdeaResult{}, r.failWorkflow(workflow, err)
	}
	workflow.ModelID = selectionModel.Model.ID
	workflow.ContextPackID = pack.ID
	workflow.ModelResolution = &modelResolution
	workflow.Steps = append(workflow.Steps, stepDoneWithMetadata("route_chapter_idea_model", modelResolutionMetadata(modelResolution)))
	promptBytes, err := json.Marshal(pack)
	if err != nil {
		return ChapterIdeaResult{}, r.failWorkflow(workflow, fmt.Errorf("marshal context pack: %w", err))
	}
	loopResult, err := r.generateWithTools(ctx, client, selectionModel, provider.TextRequest{
		Model:           selectionModel.Model.Name,
		SystemPrompt:    chapterIdeaSystemPrompt(),
		UserPrompt:      chapterIdeaUserPrompt(req, string(promptBytes)),
		MaxOutputTokens: firstPositive(req.MaxOutputTokens, selectionModel.Model.MaxOutputTokens, 1200),
		Temperature:     0.45,
	})
	if err != nil {
		return ChapterIdeaResult{}, r.failWorkflow(workflow, err)
	}
	chapterIdea := strings.TrimSpace(loopResult.Response.Content)
	if chapterIdea == "" {
		err := fmt.Errorf("chapter idea model returned empty content")
		return ChapterIdeaResult{}, r.failWorkflow(workflow, err)
	}
	workflow.Status = "completed"
	workflow.Output = mergeWorkflowOutput(map[string]string{"chapter_idea": chapterIdea, "tool_trace": strings.Join(loopResult.Trace, "\n")}, modelResolution)
	workflow.Steps = append(workflow.Steps, stepDoneWithMetadata("chapter_idea_generate", map[string]string{"tool_trace_count": strconv.Itoa(len(loopResult.Trace))}))
	workflow, err = r.store.SaveWorkflow(workflow)
	if err != nil {
		return ChapterIdeaResult{}, err
	}
	return ChapterIdeaResult{Workflow: workflow, ContextPack: pack, ChapterIdea: chapterIdea, ModelResolution: modelResolution, ToolTrace: loopResult.Trace}, nil
}

func (r *WorkflowRunner) GenerateCharacterProfiles(ctx context.Context, req CharacterProfilesRequest) (CharacterProfilesResult, error) {
	if r == nil || r.store == nil || r.router == nil || r.builder == nil || r.clients == nil {
		return CharacterProfilesResult{}, fmt.Errorf("workflow runner is not fully configured")
	}
	if strings.TrimSpace(req.ProjectID) == "" {
		return CharacterProfilesResult{}, fmt.Errorf("character profiles project_id must not be empty")
	}
	req.Focus = strings.TrimSpace(req.Focus)
	req.Brief = strings.TrimSpace(req.Brief)
	if req.Focus == "" {
		return CharacterProfilesResult{}, fmt.Errorf("character profiles focus must not be empty")
	}
	if req.Brief == "" {
		return CharacterProfilesResult{}, fmt.Errorf("character profiles brief must not be empty")
	}
	if req.Count <= 0 {
		return CharacterProfilesResult{}, fmt.Errorf("character profiles count must be a positive integer")
	}
	if req.Count > 12 {
		return CharacterProfilesResult{}, fmt.Errorf("character profiles count must not exceed 12")
	}
	selection := effectiveContextSelection(req.ContextSelection, nil)
	role := domain.AgentRoleCharacterKeeper
	workflow := domain.AIWorkflow{ProjectID: req.ProjectID, Kind: "character_profiles", Role: role, Status: "running", Input: characterProfilesWorkflowInput(req), CreatedAt: nowUTC(), UpdatedAt: nowUTC()}
	workflow.Steps = append(workflow.Steps, stepDone("retrieve_context"))
	query := normalizeCharacterProfilesBrief(req)
	pack, selectionModel, modelResolution, err := r.buildPackAndResolveModel(req.ProjectID, req.ChapterID, role, query, 6000, selection, req.ContextNodeIDs)
	if err != nil {
		return CharacterProfilesResult{}, r.failWorkflow(workflow, err)
	}
	client, err := r.clients.NewTextClient(selectionModel.Provider)
	if err != nil {
		return CharacterProfilesResult{}, r.failWorkflow(workflow, err)
	}
	workflow.ModelID = selectionModel.Model.ID
	workflow.ContextPackID = pack.ID
	workflow.ModelResolution = &modelResolution
	workflow.Steps = append(workflow.Steps, stepDoneWithMetadata("route_character_keeper_model", modelResolutionMetadata(modelResolution)))
	promptBytes, err := json.Marshal(pack)
	if err != nil {
		return CharacterProfilesResult{}, r.failWorkflow(workflow, fmt.Errorf("marshal context pack: %w", err))
	}
	loopResult, err := r.generateWithTools(ctx, client, selectionModel, provider.TextRequest{
		Model:           selectionModel.Model.Name,
		SystemPrompt:    characterProfilesSystemPrompt(),
		UserPrompt:      characterProfilesUserPrompt(req, string(promptBytes)),
		MaxOutputTokens: firstPositive(req.MaxOutputTokens, selectionModel.Model.MaxOutputTokens, 1600),
		Temperature:     0.55,
	})
	if err != nil {
		return CharacterProfilesResult{}, r.failWorkflow(workflow, err)
	}
	characters, err := parseCharacterProfilesResponse(loopResult.Response.Content, req.Count)
	if err != nil {
		return CharacterProfilesResult{}, r.failWorkflow(workflow, err)
	}
	entities, mappings, err := characterProfileToolResults(characters, loopResult.ToolCalls)
	if err != nil {
		return CharacterProfilesResult{}, r.failWorkflow(workflow, err)
	}
	workflow.Status = "completed"
	workflow.Output = mergeWorkflowOutput(map[string]string{"character_count": strconv.Itoa(len(characters)), "tool_trace": strings.Join(loopResult.Trace, "\n")}, modelResolution)
	workflow.Steps = append(workflow.Steps, stepDone("character_profiles_generate"), stepDoneWithMetadata("character_profiles_persist", map[string]string{"mapping_count": strconv.Itoa(len(mappings)), "tool_trace_count": strconv.Itoa(len(loopResult.Trace))}))
	workflow, err = r.store.SaveWorkflow(workflow)
	if err != nil {
		return CharacterProfilesResult{}, err
	}
	return CharacterProfilesResult{Workflow: workflow, ContextPack: pack, Characters: characters, Entities: entities, Mappings: mappings, ModelResolution: modelResolution, ToolTrace: loopResult.Trace}, nil
}

func (r *WorkflowRunner) DraftChapterWithIdea(ctx context.Context, req DraftWithIdeaRequest) (DraftWithIdeaResult, error) {
	ideaResult, err := r.GenerateChapterIdea(ctx, ChapterIdeaRequest{
		ProjectID:          req.ProjectID,
		ChapterID:          req.ChapterID,
		Title:              req.Title,
		Brief:              req.Brief,
		Prompt:             req.Prompt,
		ContextNodeIDs:     req.ContextNodeIDs,
		ContextSelection:   req.ContextSelection,
		ReferenceSelection: req.ReferenceSelection,
		StyleConstraints:   req.StyleConstraints,
		MaxOutputTokens:    req.MaxIdeaOutputTokens,
	})
	if err != nil {
		return DraftWithIdeaResult{}, err
	}
	draftResult, err := r.DraftChapter(ctx, DraftRequest{
		ProjectID:             req.ProjectID,
		ChapterID:             req.ChapterID,
		Title:                 req.Title,
		Brief:                 req.Brief,
		Prompt:                req.Prompt,
		ChapterIdea:           ideaResult.ChapterIdea,
		ChapterIdeaWorkflowID: ideaResult.Workflow.ID,
		ContextNodeIDs:        req.ContextNodeIDs,
		ContextSelection:      req.ContextSelection,
		ReferenceSelection:    req.ReferenceSelection,
		StyleConstraints:      req.StyleConstraints,
		MaxOutputTokens:       req.MaxDraftOutputTokens,
	})
	if err != nil {
		return DraftWithIdeaResult{ChapterIdea: ideaResult}, err
	}
	return DraftWithIdeaResult{ChapterIdea: ideaResult, Draft: draftResult, ModelResolution: draftResult.ModelResolution}, nil
}

func (r *WorkflowRunner) DraftChapter(ctx context.Context, req DraftRequest) (DraftResult, error) {
	if r == nil || r.store == nil || r.router == nil || r.builder == nil || r.clients == nil || r.auditor == nil {
		return DraftResult{}, fmt.Errorf("workflow runner is not fully configured")
	}
	if strings.TrimSpace(req.ProjectID) == "" {
		return DraftResult{}, fmt.Errorf("draft project_id must not be empty")
	}
	req.Brief = normalizeDraftBrief(req)
	if strings.TrimSpace(req.Brief) == "" {
		return DraftResult{}, fmt.Errorf("draft brief must not be empty")
	}
	selection := effectiveContextSelection(req.ContextSelection, req.ReferenceSelection)
	role := req.Role
	if role == "" {
		role = domain.AgentRoleWriter
	}
	workflow := domain.AIWorkflow{ProjectID: req.ProjectID, Kind: "chapter_draft", Role: role, Status: "running", Input: draftWorkflowInput(req), CreatedAt: nowUTC(), UpdatedAt: nowUTC()}
	workflow.Steps = append(workflow.Steps, stepDone("retrieve_context"))
	pack, selectionModel, modelResolution, err := r.buildPackAndResolveModel(req.ProjectID, req.ChapterID, role, req.Brief, 6000, selection, req.ContextNodeIDs)
	if err != nil {
		return DraftResult{}, r.failWorkflow(workflow, err)
	}
	client, err := r.clients.NewTextClient(selectionModel.Provider)
	if err != nil {
		return DraftResult{}, r.failWorkflow(workflow, err)
	}
	workflow.ModelID = selectionModel.Model.ID
	workflow.ContextPackID = pack.ID
	workflow.ModelResolution = &modelResolution
	workflow.Steps = append(workflow.Steps, stepDoneWithMetadata("route_writer_model", modelResolutionMetadata(modelResolution)))
	promptBytes, err := json.Marshal(pack)
	if err != nil {
		return DraftResult{}, r.failWorkflow(workflow, fmt.Errorf("marshal context pack: %w", err))
	}
	loopResult, err := r.generateWithTools(ctx, client, selectionModel, provider.TextRequest{
		Model:           selectionModel.Model.Name,
		SystemPrompt:    writerSystemPrompt(),
		UserPrompt:      fmt.Sprintf("章节写作简报：%s\n\n上下文包 JSON：%s", req.Brief, string(promptBytes)),
		MaxOutputTokens: firstPositive(req.MaxOutputTokens, selectionModel.Model.MaxOutputTokens, 1800),
		Temperature:     0.7,
	})
	if err != nil {
		return DraftResult{}, r.failWorkflow(workflow, err)
	}
	modelResp := loopResult.Response
	if strings.TrimSpace(modelResp.Content) == "" {
		err := fmt.Errorf("writer model returned empty content")
		return DraftResult{}, r.failWorkflow(workflow, err)
	}
	workflow.Steps = append(workflow.Steps, stepDone("writer_generate"))
	continuityAudit, err := r.auditor.Audit(ContinuityAuditInput{Title: req.Title, Brief: req.Brief, ChapterIdea: req.ChapterIdea, Draft: modelResp.Content, ContextPack: pack})
	if err != nil {
		return DraftResult{}, r.failWorkflow(workflow, err)
	}
	workflow.Steps = append(workflow.Steps, stepDoneWithMetadata("continuity_audit", continuityAuditMetadata(continuityAudit)), stepDone("extractor_deferred_to_index_job"))
	workflow, err = r.store.SaveWorkflow(workflow)
	if err != nil {
		return DraftResult{}, err
	}
	version, job, err := r.store.SaveChapterVersion(domain.ChapterVersion{ProjectID: req.ProjectID, ChapterID: req.ChapterID, Title: firstText(req.Title, "未命名章节"), Content: modelResp.Content, Summary: trimToRunes(modelResp.Content, 320), AuthorRole: role, SourceWorkflowID: workflow.ID, IndexStatus: "pending", Metadata: draftChapterMetadata(req, selection)})
	if err != nil {
		return DraftResult{}, r.failWorkflow(workflow, err)
	}
	freshness, err := r.computeIndexFreshness(req.ProjectID, version.ChapterID)
	if err != nil {
		return DraftResult{}, r.failWorkflow(workflow, err)
	}
	workflow.Status = "completed"
	workflow.Output = mergeWorkflowOutput(map[string]string{"chapter_version_id": version.ID, "index_job_id": job.ID, "tool_trace": strings.Join(loopResult.Trace, "\n")}, modelResolution)
	workflow.Steps = append(workflow.Steps, stepDone("persist_chapter_version_and_index_job"))
	workflow, err = r.store.SaveWorkflow(workflow)
	if err != nil {
		return DraftResult{}, err
	}
	return DraftResult{Workflow: workflow, ContextPack: pack, ChapterVersion: version, IndexJob: job, IndexFreshness: freshness, ModelResolution: modelResolution, ContinuityAudit: continuityAudit, ToolTrace: loopResult.Trace}, nil
}

func (r *WorkflowRunner) generateWithTools(ctx context.Context, client provider.TextModelClient, selection ModelSelection, req provider.TextRequest) (ToolLoopResult, error) {
	if !selection.Model.SupportsTools {
		return ToolLoopResult{}, fmt.Errorf("selected model %q does not support tools", selection.Model.ID)
	}
	req.Tools = NarrativeToolSpecs()
	return RunToolLoop(ctx, client, req, NewToolExecutor(r.store), defaultToolLoopMaxRounds)
}

func characterProfileToolResults(characters []domain.CharacterProfile, records []ToolExecutionRecord) ([]domain.Entity, []domain.CharacterProfileMapping, error) {
	if len(records) == 0 {
		return nil, nil, fmt.Errorf("character profiles require character.search and character.upsert tool calls before final JSON")
	}
	searchedByName := map[string]struct{}{}
	upsertedByName := map[string]domain.CharacterProfileMapping{}
	upsertEntities := map[string]domain.Entity{}
	for _, record := range records {
		switch record.Name {
		case "character.search":
			query, err := toolRecordStringArgument(record, "query")
			if err != nil {
				return nil, nil, err
			}
			if query != "" {
				searchedByName[strings.ToLower(query)] = struct{}{}
			}
		case "character.upsert":
			argsName, err := toolRecordStringArgument(record, "name")
			if err != nil {
				return nil, nil, err
			}
			if argsName == "" {
				return nil, nil, fmt.Errorf("character.upsert tool call missing name argument")
			}
			if _, ok := searchedByName[strings.ToLower(argsName)]; !ok {
				return nil, nil, fmt.Errorf("character.upsert for %q executed before matching character.search", argsName)
			}
			entity, action, err := decodeCharacterUpsertRecord(record)
			if err != nil {
				return nil, nil, err
			}
			if entity.Type != "character" {
				return nil, nil, fmt.Errorf("character.upsert tool result for %q returned entity type %q", argsName, entity.Type)
			}
			if entity.ID == "" {
				return nil, nil, fmt.Errorf("character.upsert tool result for %q returned empty entity id", argsName)
			}
			resultName := strings.TrimSpace(firstText(entity.Name, argsName))
			if resultName == "" {
				return nil, nil, fmt.Errorf("character.upsert tool result returned empty character name")
			}
			key := strings.ToLower(resultName)
			upsertedByName[key] = domain.CharacterProfileMapping{Name: resultName, EntityID: entity.ID, Action: action}
			upsertEntities[key] = entity
		}
	}
	entities := make([]domain.Entity, 0, len(characters))
	mappings := make([]domain.CharacterProfileMapping, 0, len(characters))
	for _, profile := range characters {
		name := strings.TrimSpace(profile.Name)
		if name == "" {
			return nil, nil, fmt.Errorf("character profile name must not be empty")
		}
		key := strings.ToLower(name)
		if _, ok := searchedByName[key]; !ok {
			return nil, nil, fmt.Errorf("character %q declared in final JSON without prior character.search tool call", name)
		}
		mapping, ok := upsertedByName[key]
		if !ok {
			return nil, nil, fmt.Errorf("character %q declared in final JSON without matching character.upsert tool result", name)
		}
		entities = append(entities, upsertEntities[key])
		mappings = append(mappings, mapping)
	}
	return entities, mappings, nil
}

func toolRecordStringArgument(record ToolExecutionRecord, key string) (string, error) {
	if len(record.Arguments) == 0 {
		return "", fmt.Errorf("tool %s record missing arguments", record.Name)
	}
	var args map[string]any
	if err := json.Unmarshal(record.Arguments, &args); err != nil {
		return "", fmt.Errorf("decode tool %s arguments: %w", record.Name, err)
	}
	value, ok := args[key]
	if !ok || value == nil {
		return "", nil
	}
	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("tool %s argument %s must be a string", record.Name, key)
	}
	return strings.TrimSpace(text), nil
}

func decodeCharacterUpsertRecord(record ToolExecutionRecord) (domain.Entity, string, error) {
	var result struct {
		Action string        `json:"action"`
		Entity domain.Entity `json:"entity"`
	}
	if len(record.Result) == 0 {
		return domain.Entity{}, "", fmt.Errorf("character.upsert tool result is empty")
	}
	if err := json.Unmarshal(record.Result, &result); err != nil {
		return domain.Entity{}, "", fmt.Errorf("decode character.upsert tool result: %w", err)
	}
	action := strings.TrimSpace(result.Action)
	if action == "" {
		return domain.Entity{}, "", fmt.Errorf("character.upsert tool result missing action")
	}
	return result.Entity, action, nil
}

func (r *WorkflowRunner) buildPackAndResolveModel(projectID, chapterID string, role domain.AgentRole, query string, tokenBudget int, selection *ContextSelection, contextNodeIDs []string) (domain.ContextPack, ModelSelection, domain.ModelResolution, error) {
	pack, err := r.builder.BuildWithSelection(projectID, chapterID, role, query, tokenBudget, selection, contextNodeIDs)
	if err != nil {
		return domain.ContextPack{}, ModelSelection{}, domain.ModelResolution{}, err
	}
	selectionModel, err := r.router.SelectTextModel(role)
	if err != nil {
		return domain.ContextPack{}, ModelSelection{}, domain.ModelResolution{}, err
	}
	return pack, selectionModel, withResolvedIDs(buildModelResolution(selectionModel), selectionModel, pack), nil
}

func (r *WorkflowRunner) computeIndexFreshness(projectID, chapterID string) (domain.IndexFreshness, error) {
	versions, err := r.store.ListChapterVersions(projectID, chapterID)
	if err != nil {
		return domain.IndexFreshness{}, err
	}
	pendingJobs, err := r.store.ListPendingIndexJobs(projectID, 0)
	if err != nil {
		return domain.IndexFreshness{}, err
	}
	freshness := domain.IndexFreshness{ProjectID: projectID, ChapterID: chapterID, Status: "missing", PendingJobCount: countRelevantPendingJobs(pendingJobs, chapterID)}
	if len(versions) == 0 {
		return freshness, nil
	}
	latest := versions[0]
	freshness.LatestChapterVersionID = latest.ID
	freshness.LatestChapterVersionCreatedAt = &latest.CreatedAt
	for _, version := range versions {
		if version.IndexStatus != "indexed" {
			continue
		}
		freshness.LatestIndexedChapterVersionID = version.ID
		freshness.LatestIndexedAt = &version.CreatedAt
		break
	}
	switch {
	case latest.IndexStatus == "indexed" && freshness.PendingJobCount == 0:
		freshness.Status = "fresh"
	case freshness.LatestIndexedChapterVersionID != "":
		freshness.Status = "stale"
	default:
		freshness.Status = "pending"
	}
	return freshness, nil
}

func countRelevantPendingJobs(jobs []domain.IndexJob, chapterID string) int {
	count := 0
	for _, job := range jobs {
		if strings.TrimSpace(chapterID) != "" && job.ChapterID != chapterID {
			continue
		}
		count++
	}
	return count
}

func buildModelResolution(selection ModelSelection) domain.ModelResolution {
	return domain.ModelResolution{
		RouteKey:         selection.RouteKey,
		ResolutionSource: selection.ResolutionSource,
		ProviderID:       selection.Provider.ID,
		ProviderName:     selection.Provider.Name,
		ProviderType:     selection.Provider.Type,
		ModelID:          selection.Model.ID,
		ModelName:        selection.Model.Name,
		ModelKind:        selection.Model.Kind,
	}
}

func withResolvedIDs(resolution domain.ModelResolution, selection ModelSelection, pack domain.ContextPack) domain.ModelResolution {
	resolution.ModelID = selection.Model.ID
	resolution.ModelName = selection.Model.Name
	if resolution.RouteKey == "" {
		resolution.RouteKey = selection.RouteKey
	}
	if pack.ID != "" && resolution.ResolutionSource == "" {
		resolution.ResolutionSource = selection.ResolutionSource
	}
	return resolution
}

func modelResolutionMetadata(resolution domain.ModelResolution) map[string]string {
	return map[string]string{
		"route_key":         resolution.RouteKey,
		"resolution_source": resolution.ResolutionSource,
		"provider_id":       resolution.ProviderID,
		"provider_name":     resolution.ProviderName,
		"provider_type":     string(resolution.ProviderType),
		"model_id":          resolution.ModelID,
		"model_name":        resolution.ModelName,
		"model_kind":        string(resolution.ModelKind),
	}
}

func mergeWorkflowOutput(output map[string]string, resolution domain.ModelResolution) map[string]string {
	merged := map[string]string{}
	for key, value := range output {
		merged[key] = value
	}
	for key, value := range modelResolutionMetadata(resolution) {
		merged["resolved_"+key] = value
	}
	return merged
}

func summarizeContextPack(pack domain.ContextPack) string {
	parts := []string{
		fmt.Sprintf("章节摘要 %d 条", len(pack.ChapterSummaries)),
		fmt.Sprintf("实体 %d 个", len(pack.Entities)),
		fmt.Sprintf("事实 %d 条", len(pack.Facts)),
		fmt.Sprintf("情节线 %d 条", len(pack.PlotThreads)),
	}
	if len(pack.WorldRules) > 0 {
		parts = append(parts, fmt.Sprintf("世界规则 %d 条", len(pack.WorldRules)))
	}
	return strings.Join(parts, "，")
}

func estimateContextPackTokens(pack domain.ContextPack) int {
	payload, err := json.Marshal(pack)
	if err != nil {
		return 0
	}
	return len([]rune(string(payload))) / 4
}

func (r *WorkflowRunner) failWorkflow(workflow domain.AIWorkflow, cause error) error {
	if cause == nil {
		cause = fmt.Errorf("workflow failed without cause")
	}
	workflow.Status = "failed"
	workflow.Error = cause.Error()
	if _, err := r.store.SaveWorkflow(workflow); err != nil {
		return fmt.Errorf("%w; save failed workflow state: %v", cause, err)
	}
	return cause
}

func normalizeChapterIdeaBrief(req ChapterIdeaRequest) string {
	parts := []string{req.Brief, req.Prompt}
	if strings.TrimSpace(req.Title) != "" {
		parts = append(parts, "章节标题："+strings.TrimSpace(req.Title))
	}
	parts = appendBriefControls(parts, req.StyleConstraints, req.ContextNodeIDs)
	return joinNonEmpty(parts)
}

func normalizeDraftBrief(req DraftRequest) string {
	parts := []string{req.Brief, req.Prompt}
	if strings.TrimSpace(req.ChapterIdea) != "" {
		parts = append(parts, "章节方案：\n"+strings.TrimSpace(req.ChapterIdea))
	}
	parts = appendBriefControls(parts, req.StyleConstraints, req.ContextNodeIDs)
	return joinNonEmpty(parts)
}

func normalizeCharacterProfilesBrief(req CharacterProfilesRequest) string {
	return joinNonEmpty([]string{
		"角色需求焦点：" + strings.TrimSpace(req.Focus),
		"生成数量：" + strconv.Itoa(req.Count),
		"角色生成简报：" + strings.TrimSpace(req.Brief),
	})
}

func appendBriefControls(parts []string, styleConstraints []string, contextNodeIDs []string) []string {
	if len(styleConstraints) > 0 {
		parts = append(parts, "风格约束："+strings.Join(styleConstraints, "、"))
	}
	if len(contextNodeIDs) > 0 {
		parts = append(parts, "参考节点："+strings.Join(contextNodeIDs, "、"))
	}
	return parts
}

func joinNonEmpty(parts []string) string {
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			cleaned = append(cleaned, strings.TrimSpace(part))
		}
	}
	return strings.Join(cleaned, "\n")
}

func chapterIdeaWorkflowInput(req ChapterIdeaRequest) map[string]string {
	input := map[string]string{"brief": req.Brief, "chapter_id": req.ChapterID}
	if strings.TrimSpace(req.Title) != "" {
		input["title"] = strings.TrimSpace(req.Title)
	}
	appendSelectionInput(input, effectiveContextSelection(req.ContextSelection, req.ReferenceSelection), req.ContextNodeIDs)
	return input
}

func characterProfilesWorkflowInput(req CharacterProfilesRequest) map[string]string {
	input := map[string]string{
		"focus": strings.TrimSpace(req.Focus),
		"brief": strings.TrimSpace(req.Brief),
		"count": strconv.Itoa(req.Count),
	}
	if strings.TrimSpace(req.ChapterID) != "" {
		input["chapter_id"] = strings.TrimSpace(req.ChapterID)
	}
	appendSelectionInput(input, req.ContextSelection, req.ContextNodeIDs)
	return input
}

func draftWorkflowInput(req DraftRequest) map[string]string {
	input := map[string]string{"brief": req.Brief, "chapter_id": req.ChapterID}
	if strings.TrimSpace(req.ChapterIdea) != "" {
		input["chapter_idea"] = strings.TrimSpace(req.ChapterIdea)
	}
	if strings.TrimSpace(req.ChapterIdeaWorkflowID) != "" {
		input["chapter_idea_workflow_id"] = strings.TrimSpace(req.ChapterIdeaWorkflowID)
	}
	appendSelectionInput(input, effectiveContextSelection(req.ContextSelection, req.ReferenceSelection), req.ContextNodeIDs)
	return input
}

func draftChapterMetadata(req DraftRequest, selection *ContextSelection) map[string]string {
	metadata := map[string]string{}
	if strings.TrimSpace(req.ChapterIdeaWorkflowID) != "" {
		metadata["chapter_idea_workflow_id"] = strings.TrimSpace(req.ChapterIdeaWorkflowID)
	}
	if strings.TrimSpace(req.ChapterIdea) != "" {
		metadata["chapter_idea_used"] = "true"
	}
	appendSelectionInput(metadata, selection, req.ContextNodeIDs)
	if len(metadata) == 0 {
		return nil
	}
	return metadata
}

func appendSelectionInput(target map[string]string, selection *ContextSelection, contextNodeIDs []string) {
	if selection != nil {
		if len(selection.ChapterIDs) > 0 {
			target["context_selection.chapter_ids"] = strings.Join(selection.ChapterIDs, ",")
		}
		if len(selection.CharacterIDs) > 0 {
			target["context_selection.character_ids"] = strings.Join(selection.CharacterIDs, ",")
		}
		if len(selection.CharacterNames) > 0 {
			target["context_selection.character_names"] = strings.Join(selection.CharacterNames, ",")
		}
		target["context_selection.include_world_rules"] = strconv.FormatBool(shouldIncludeWorldRules(selection))
	}
	if len(contextNodeIDs) > 0 {
		target["context_node_ids"] = strings.Join(contextNodeIDs, ",")
	}
}

func effectiveContextSelection(primary, compatibility *ContextSelection) *ContextSelection {
	if primary != nil {
		return primary
	}
	return compatibility
}

func deterministicBible(seed domain.ProjectSeed) domain.StoryBible {
	language := firstText(seed.Language, "zh-CN")
	genre := firstText(seed.Genre, "未分类")
	tone := firstText(seed.Tone, "稳健、清晰")
	audience := firstText(seed.Audience, "通用读者")
	themes := seed.Themes
	if len(themes) == 0 {
		themes = []string{"成长", "选择", "代价"}
	}
	rules := map[string]string{
		"context_policy":      "所有 Agent 使用检索得到的 ContextPack，不直接塞入完整小说上下文。",
		"canon_policy":        "章节版本保存后必须创建索引任务，用于事实抽取、向量重索引和图谱刷新。",
		"style_guidance":      fmt.Sprintf("类型：%s；语气：%s；目标读者：%s。", genre, tone, audience),
		"continuity_guidance": "新内容必须尊重 StoryBible、世界线、实体事实和未闭合情节线。",
	}
	synopsis := fmt.Sprintf("《%s》讲述：%s", seed.Title, seed.Premise)
	if strings.TrimSpace(seed.Setting) != "" {
		synopsis += " 故事舞台设定为：" + seed.Setting
	}
	if len(seed.MainCharacters) > 0 {
		synopsis += " 关键角色包括：" + strings.Join(seed.MainCharacters, "、") + "。"
	}
	return domain.StoryBible{Title: seed.Title, Logline: seed.Premise, Synopsis: synopsis, Genre: genre, Tone: tone, Audience: audience, Language: language, Themes: themes, Rules: rules, SourceSeed: seed, Approved: false}
}

func chapterIdeaSystemPrompt() string {
	return "你是 AI 小说创作平台中的 Plot Architect Agent。你的任务是为当前单章生成章节想法/章节方案，而不是写正文。必须只使用提供的 ContextPack 与用户意图，输出 Markdown 半结构化 brief，至少包含：本章目标、承接与铺垫、场景节拍、冲突/转折、角色状态变化、伏笔处理、写作注意。"
}

func chapterIdeaUserPrompt(req ChapterIdeaRequest, contextPackJSON string) string {
	return fmt.Sprintf("章节：%s\n章节方案输入：%s\n\n上下文包 JSON：%s\n\n请输出可直接交给 Writer Agent 续写正文的单章方案，不要写正文。", firstText(req.Title, req.ChapterID, "未命名章节"), req.Brief, contextPackJSON)
}

func writerSystemPrompt() string {
	return "你是 AI 小说创作平台中的 Writer Agent。只使用提供的 ContextPack 写作，不假设完整小说上下文；如上下文不足，应在正文中保持克制，不新增破坏连续性的事实。如果写作简报包含章节方案，必须以该方案为主要结构依据，同时保持正文自然流畅。"
}

func characterProfilesSystemPrompt() string {
	return "你是 AI 小说创作平台中的 Character Keeper Agent。你的任务是围绕当前项目主角/配角需求生成可直接写入 Story Bible 的角色设定，并保持与提供的 ContextPack 一致。你必须使用工具完成真实数据库同步：创建或更新每个角色前，必须先调用 character.search 按角色名查重；确认角色档案后，必须调用 character.upsert 保存该角色；如果你明确描述角色关系，必须调用 relationship.upsert 保存关系。所有需要写入数据库的角色都完成工具调用并读取工具结果后，最终仍必须输出严格 JSON 对象，不要输出 Markdown 或解释。JSON 结构为 {\"characters\":[{\"name\":\"\",\"role\":\"\",\"desire\":\"\",\"wound\":\"\",\"secret\":\"\",\"summary\":\"\"}]}。每个角色必须包含 name、role、desire、wound、secret，可包含 summary；最终 JSON 中的每个角色必须已经有对应的 character.upsert 工具结果。"
}

func characterProfilesUserPrompt(req CharacterProfilesRequest, contextPackJSON string) string {
	return fmt.Sprintf("角色需求焦点：%s\n生成数量：%d\n角色生成简报：%s\n\n上下文包 JSON：%s\n\n请生成可直接写入 Story Bible 的角色设定；如果需求是主角完整设定，必须生成完整主角档案；如果需求是配角或群像，则生成互相区分且能支撑当前项目冲突的角色。流程要求：1）对每个待创建或更新角色先调用 character.search，query 使用角色名，查重并读取工具结果；2）再调用 character.upsert 保存角色，traits 至少写入 role、desire、wound、secret，metadata 可记录来源；3）如输出中明确两个角色关系，调用 relationship.upsert 保存；4）完成全部工具调用后再输出严格 JSON，且 JSON 中只包含已通过 character.upsert 成功保存的角色。", req.Focus, req.Count, req.Brief, contextPackJSON)
}

func parseCharacterProfilesResponse(content string, expectedCount int) ([]domain.CharacterProfile, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("character profiles model returned empty content")
	}
	var envelope struct {
		Characters []domain.CharacterProfile `json:"characters"`
	}
	if err := json.Unmarshal([]byte(content), &envelope); err != nil {
		return nil, fmt.Errorf("decode character profiles JSON: %w", err)
	}
	if len(envelope.Characters) == 0 {
		return nil, fmt.Errorf("character profiles response must include at least one character")
	}
	if expectedCount > 0 && len(envelope.Characters) != expectedCount {
		return nil, fmt.Errorf("character profiles response returned %d characters, want %d", len(envelope.Characters), expectedCount)
	}
	characters := make([]domain.CharacterProfile, 0, len(envelope.Characters))
	seen := map[string]struct{}{}
	for i, character := range envelope.Characters {
		character = normalizeCharacterProfile(character)
		if character.Name == "" {
			return nil, fmt.Errorf("character profiles response character[%d].name must not be empty", i)
		}
		if character.Role == "" {
			return nil, fmt.Errorf("character profiles response character[%d].role must not be empty", i)
		}
		if character.Desire == "" {
			return nil, fmt.Errorf("character profiles response character[%d].desire must not be empty", i)
		}
		if character.Wound == "" {
			return nil, fmt.Errorf("character profiles response character[%d].wound must not be empty", i)
		}
		if character.Secret == "" {
			return nil, fmt.Errorf("character profiles response character[%d].secret must not be empty", i)
		}
		key := strings.ToLower(character.Name)
		if _, ok := seen[key]; ok {
			return nil, fmt.Errorf("character profiles response contains duplicate character name %q", character.Name)
		}
		seen[key] = struct{}{}
		characters = append(characters, character)
	}
	return characters, nil
}

func normalizeCharacterProfile(character domain.CharacterProfile) domain.CharacterProfile {
	character.Name = strings.TrimSpace(character.Name)
	character.Role = strings.TrimSpace(character.Role)
	character.Desire = strings.TrimSpace(character.Desire)
	character.Wound = strings.TrimSpace(character.Wound)
	character.Secret = strings.TrimSpace(character.Secret)
	character.Summary = strings.TrimSpace(character.Summary)
	return character
}

func stepDone(name string) domain.WorkflowStep {
	t := nowUTC()
	return domain.WorkflowStep{Name: name, Status: "completed", StartedAt: &t, EndedAt: &t}
}

func stepDoneWithMetadata(name string, metadata map[string]string) domain.WorkflowStep {
	step := stepDone(name)
	step.Metadata = metadata
	return step
}

func nowUTC() time.Time { return time.Now().UTC() }

func slugify(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range lower {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if r > 127 {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func firstPositive(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 1
}
