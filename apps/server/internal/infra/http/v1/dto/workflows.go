package dto

import (
	"aeonechoes/server/internal/domain"
	"time"
)

type WorkflowStepDTO struct {
	Name      string            `json:"name"`
	Status    string            `json:"status"`
	StartedAt *time.Time        `json:"started_at,omitempty"`
	EndedAt   *time.Time        `json:"ended_at,omitempty"`
	Error     string            `json:"error,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type ModelResolutionDTO struct {
	RouteKey         string              `json:"route_key"`
	ResolutionSource string              `json:"resolution_source"`
	ProviderID       string              `json:"provider_id"`
	ProviderName     string              `json:"provider_name"`
	ProviderType     domain.ProviderType `json:"provider_type"`
	ModelID          string              `json:"model_id"`
	ModelName        string              `json:"model_name"`
	ModelKind        domain.ModelKind    `json:"model_kind"`
}

type WorkflowDTO struct {
	ID              string              `json:"id"`
	ProjectID       string              `json:"project_id"`
	Kind            string              `json:"kind"`
	Role            domain.AgentRole    `json:"role"`
	Status          string              `json:"status"`
	ModelID         string              `json:"model_id,omitempty"`
	ContextPackID   string              `json:"context_pack_id,omitempty"`
	ModelResolution *ModelResolutionDTO `json:"model_resolution,omitempty"`
	Steps           []WorkflowStepDTO   `json:"steps,omitempty"`
	Input           map[string]string   `json:"input,omitempty"`
	Output          map[string]string   `json:"output,omitempty"`
	Error           string              `json:"error,omitempty"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
}

type IndexFreshnessDTO struct {
	ProjectID                     string     `json:"project_id"`
	ChapterID                     string     `json:"chapter_id,omitempty"`
	Status                        string     `json:"status"`
	LatestChapterVersionID        string     `json:"latest_chapter_version_id,omitempty"`
	LatestChapterVersionCreatedAt *time.Time `json:"latest_chapter_version_created_at,omitempty"`
	LatestIndexedChapterVersionID string     `json:"latest_indexed_chapter_version_id,omitempty"`
	LatestIndexedAt               *time.Time `json:"latest_indexed_at,omitempty"`
	PendingJobCount               int        `json:"pending_job_count"`
}

type ContinuityEvidenceRefDTO struct {
	SourceType string `json:"source_type"`
	SourceID   string `json:"source_id,omitempty"`
	Label      string `json:"label"`
	Excerpt    string `json:"excerpt,omitempty"`
}

type ContinuityIssueDTO struct {
	Type         string                     `json:"type"`
	Severity     string                     `json:"severity"`
	Message      string                     `json:"message"`
	DraftExcerpt string                     `json:"draft_excerpt"`
	Suggestion   string                     `json:"suggestion"`
	Evidence     []ContinuityEvidenceRefDTO `json:"evidence"`
}

type ContinuityAuditDTO struct {
	Status string               `json:"status"`
	Issues []ContinuityIssueDTO `json:"issues"`
}

type ChapterSummaryDTO struct {
	ChapterID        string `json:"chapter_id"`
	ChapterVersionID string `json:"chapter_version_id"`
	Title            string `json:"title"`
	Summary          string `json:"summary"`
}

type ContextPackDTO struct {
	ID               string              `json:"id"`
	ProjectID        string              `json:"project_id"`
	ChapterID        string              `json:"chapter_id,omitempty"`
	Role             domain.AgentRole    `json:"role"`
	TokenBudget      int                 `json:"token_budget"`
	Query            string              `json:"query"`
	StoryBibleID     string              `json:"story_bible_id,omitempty"`
	WorldRules       map[string]string   `json:"world_rules,omitempty"`
	Facts            []FactDTO           `json:"facts,omitempty"`
	Entities         []EntityDTO         `json:"entities,omitempty"`
	Edges            []GraphEdgeDTO      `json:"edges,omitempty"`
	PlotThreads      []PlotThreadDTO     `json:"plot_threads,omitempty"`
	ChapterSummaries []ChapterSummaryDTO `json:"chapter_summaries,omitempty"`
	ToolTrace        []string            `json:"tool_trace,omitempty"`
	Metadata         map[string]string   `json:"metadata,omitempty"`
	CreatedAt        time.Time           `json:"created_at"`
}

type ContextSelectionDTO struct {
	ChapterIDs        []string `json:"chapter_ids,omitempty"`
	CharacterIDs      []string `json:"character_ids,omitempty"`
	CharacterNames    []string `json:"character_names,omitempty"`
	IncludeWorldRules *bool    `json:"include_world_rules,omitempty"`
}

type ContextPreviewRequestDTO struct {
	ProjectID          string               `json:"project_id"`
	ChapterID          string               `json:"chapter_id,omitempty"`
	Title              string               `json:"title,omitempty"`
	Brief              string               `json:"brief,omitempty"`
	Prompt             string               `json:"prompt,omitempty"`
	ContextNodeIDs     []string             `json:"context_node_ids,omitempty"`
	ContextSelection   *ContextSelectionDTO `json:"context_selection,omitempty"`
	ReferenceSelection *ContextSelectionDTO `json:"reference_selection,omitempty"`
	StyleConstraints   []string             `json:"style_constraints,omitempty"`
	Role               domain.AgentRole     `json:"role,omitempty"`
	TokenBudget        int                  `json:"token_budget,omitempty"`
}

type ContextPreviewResponseDTO struct {
	ContextPack     ContextPackDTO     `json:"context_pack"`
	Summary         string             `json:"summary"`
	EstimatedTokens int                `json:"estimated_tokens"`
	IndexFreshness  IndexFreshnessDTO  `json:"index_freshness"`
	ModelResolution ModelResolutionDTO `json:"model_resolution"`
}

type ChapterIdeaRequestDTO struct {
	ProjectID          string               `json:"project_id"`
	ChapterID          string               `json:"chapter_id,omitempty"`
	Title              string               `json:"title,omitempty"`
	Brief              string               `json:"brief"`
	Prompt             string               `json:"prompt,omitempty"`
	ContextNodeIDs     []string             `json:"context_node_ids,omitempty"`
	ContextSelection   *ContextSelectionDTO `json:"context_selection,omitempty"`
	ReferenceSelection *ContextSelectionDTO `json:"reference_selection,omitempty"`
	StyleConstraints   []string             `json:"style_constraints,omitempty"`
	MaxOutputTokens    int                  `json:"max_output_tokens,omitempty"`
}

type ChapterIdeaResponseDTO struct {
	Workflow        WorkflowDTO        `json:"workflow"`
	ContextPack     ContextPackDTO     `json:"context_pack"`
	ChapterIdea     string             `json:"chapter_idea"`
	ModelResolution ModelResolutionDTO `json:"model_resolution"`
	ToolTrace       []string           `json:"tool_trace,omitempty"`
}

type CharacterProfilesRequestDTO struct {
	ProjectID        string               `json:"project_id"`
	Focus            string               `json:"focus"`
	Count            int                  `json:"count"`
	Brief            string               `json:"brief"`
	ChapterID        string               `json:"chapter_id,omitempty"`
	ContextNodeIDs   []string             `json:"context_node_ids,omitempty"`
	ContextSelection *ContextSelectionDTO `json:"context_selection,omitempty"`
	MaxOutputTokens  int                  `json:"max_output_tokens,omitempty"`
}

type CharacterProfilesResponseDTO struct {
	Workflow        WorkflowDTO                  `json:"workflow"`
	ContextPack     ContextPackDTO               `json:"context_pack"`
	Characters      []StoryBibleCharacterDTO     `json:"characters"`
	Entities        []EntityDTO                  `json:"entities,omitempty"`
	Mappings        []CharacterProfileMappingDTO `json:"mappings,omitempty"`
	ModelResolution ModelResolutionDTO           `json:"model_resolution"`
	ToolTrace       []string                     `json:"tool_trace,omitempty"`
}

type DraftRequestDTO struct {
	ProjectID             string               `json:"project_id"`
	ChapterID             string               `json:"chapter_id,omitempty"`
	Title                 string               `json:"title,omitempty"`
	Brief                 string               `json:"brief"`
	Prompt                string               `json:"prompt,omitempty"`
	ChapterIdea           string               `json:"chapter_idea,omitempty"`
	ChapterIdeaWorkflowID string               `json:"chapter_idea_workflow_id,omitempty"`
	ContextNodeIDs        []string             `json:"context_node_ids,omitempty"`
	ContextSelection      *ContextSelectionDTO `json:"context_selection,omitempty"`
	ReferenceSelection    *ContextSelectionDTO `json:"reference_selection,omitempty"`
	StyleConstraints      []string             `json:"style_constraints,omitempty"`
	Role                  domain.AgentRole     `json:"role,omitempty"`
	MaxOutputTokens       int                  `json:"max_output_tokens,omitempty"`
}

type DraftResponseDTO struct {
	Workflow        WorkflowDTO        `json:"workflow"`
	ContextPack     ContextPackDTO     `json:"context_pack"`
	ChapterVersion  ChapterVersionDTO  `json:"chapter_version"`
	IndexJob        IndexJobDTO        `json:"index_job"`
	IndexFreshness  IndexFreshnessDTO  `json:"index_freshness"`
	ModelResolution ModelResolutionDTO `json:"model_resolution"`
	ContinuityAudit ContinuityAuditDTO `json:"continuity_audit"`
	ToolTrace       []string           `json:"tool_trace,omitempty"`
}

type DraftWithIdeaRequestDTO struct {
	ProjectID            string               `json:"project_id"`
	ChapterID            string               `json:"chapter_id,omitempty"`
	Title                string               `json:"title,omitempty"`
	Brief                string               `json:"brief"`
	Prompt               string               `json:"prompt,omitempty"`
	ContextNodeIDs       []string             `json:"context_node_ids,omitempty"`
	ContextSelection     *ContextSelectionDTO `json:"context_selection,omitempty"`
	ReferenceSelection   *ContextSelectionDTO `json:"reference_selection,omitempty"`
	StyleConstraints     []string             `json:"style_constraints,omitempty"`
	MaxIdeaOutputTokens  int                  `json:"max_idea_output_tokens,omitempty"`
	MaxDraftOutputTokens int                  `json:"max_draft_output_tokens,omitempty"`
}

type DraftWithIdeaResponseDTO struct {
	ChapterIdea     ChapterIdeaResponseDTO `json:"chapter_idea"`
	Draft           DraftResponseDTO       `json:"draft"`
	ModelResolution ModelResolutionDTO     `json:"model_resolution"`
}
