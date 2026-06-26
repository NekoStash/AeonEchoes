package domain

import "time"

// ProviderType identifies the upstream model provider protocol.
type ProviderType string

const (
	ProviderOpenAIResponses ProviderType = "openai-responses"
	ProviderOpenAI          ProviderType = "openai"
	ProviderAnthropic       ProviderType = "anthropic"
	ProviderGemini          ProviderType = "gemini"
)

func (p ProviderType) Valid() bool {
	switch p {
	case ProviderOpenAIResponses, ProviderOpenAI, ProviderAnthropic, ProviderGemini:
		return true
	default:
		return false
	}
}

// ModelKind separates generation models from embedding models so routing is deterministic.
type ModelKind string

const (
	ModelKindText      ModelKind = "text"
	ModelKindEmbedding ModelKind = "embedding"
)

func (k ModelKind) Valid() bool {
	switch k {
	case ModelKindText, ModelKindEmbedding:
		return true
	default:
		return false
	}
}

// AppSetting stores JSON-friendly operator settings under a scoped key.
type AppSetting struct {
	Scope     string         `json:"scope"`
	Key       string         `json:"key"`
	Value     map[string]any `json:"value"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// SemanticSearchRequest asks the backend to embed a query and search vector context.
type SemanticSearchRequest struct {
	Query     string            `json:"query"`
	ProjectID string            `json:"project_id"`
	Limit     int               `json:"limit,omitempty"`
	Filters   map[string]string `json:"filters,omitempty"`
}

// SemanticSearchItem is one vector search hit with Qdrant payload fields preserved.
type SemanticSearchItem struct {
	SourceID string         `json:"source_id"`
	Score    float64        `json:"score"`
	Payload  map[string]any `json:"payload,omitempty"`
}

// SemanticSearchResult is returned by retrieval APIs and agent tools.
type SemanticSearchResult struct {
	Query     string               `json:"query"`
	ProjectID string               `json:"project_id"`
	Items     []SemanticSearchItem `json:"items"`
}

// SystemStatus summarizes configured infrastructure and queue health for operators.
type SystemStatus struct {
	Status             string    `json:"status"`
	PostgresConfigured bool      `json:"postgres_configured"`
	QdrantConfigured   bool      `json:"qdrant_configured"`
	ProviderCount      int       `json:"provider_count"`
	ModelCount         int       `json:"model_count"`
	PendingJobsCount   int       `json:"pending_jobs_count"`
	CheckedAt          time.Time `json:"checked_at"`
}

// BackendStatus is kept as a compatibility alias for older callers.
type BackendStatus = SystemStatus

// AgentRole is a logical writing role. Roles route to models and tools; they are not separate context dumps.
type AgentRole string

const (
	AgentRoleGenesisOptimizer AgentRole = "genesis-optimizer"
	AgentRolePlotArchitect    AgentRole = "plot-architect"
	AgentRoleWorldBuilder     AgentRole = "world-builder"
	AgentRoleCharacterKeeper  AgentRole = "character-keeper"
	AgentRoleContinuityAudit  AgentRole = "continuity-auditor"
	AgentRoleWriter           AgentRole = "writer"
	AgentRoleEditor           AgentRole = "editor"
	AgentRoleFactExtractor    AgentRole = "fact-extractor"
	AgentRoleGraphCurator     AgentRole = "graph-curator"
)

// ProviderConfig is an administrator-controlled provider connection.
type ProviderConfig struct {
	ID                       string            `json:"id"`
	Name                     string            `json:"name"`
	Type                     ProviderType      `json:"type"`
	BaseURL                  string            `json:"base_url"`
	APIKey                   string            `json:"api_key,omitempty"`
	APIKeyEnv                string            `json:"api_key_env,omitempty"`
	Enabled                  bool              `json:"enabled"`
	TraceEnabled             bool              `json:"trace_enabled"`
	TraceRetentionDays       int               `json:"trace_retention_days"`
	DefaultRequestTimeoutSec int               `json:"default_request_timeout_sec"`
	Metadata                 map[string]string `json:"metadata,omitempty"`
	CreatedAt                time.Time         `json:"created_at"`
	UpdatedAt                time.Time         `json:"updated_at"`
	LastModelRefreshAt       *time.Time        `json:"last_model_refresh_at,omitempty"`
}

// ModelConfig declares a model made available to routing.
type ModelConfig struct {
	ID                string            `json:"id"`
	ProviderID        string            `json:"provider_id"`
	ProviderType      ProviderType      `json:"provider_type"`
	Name              string            `json:"name"`
	DisplayName       string            `json:"display_name"`
	Kind              ModelKind         `json:"kind"`
	ContextWindow     int               `json:"context_window"`
	MaxOutputTokens   int               `json:"max_output_tokens"`
	Dimension         int               `json:"dimension,omitempty"`
	SupportsTools     bool              `json:"supports_tools"`
	SupportsStreaming bool              `json:"supports_streaming"`
	DefaultForKind    bool              `json:"default_for_kind"`
	Enabled           bool              `json:"enabled"`
	CostInputPerMTok  float64           `json:"cost_input_per_mtok,omitempty"`
	CostOutputPerMTok float64           `json:"cost_output_per_mtok,omitempty"`
	RoutingWeight     int               `json:"routing_weight"`
	AllowedAgentRoles []AgentRole       `json:"allowed_agent_roles,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
	LastSeenAt        *time.Time        `json:"last_seen_at,omitempty"`
}

// ProjectSeed is the human supplied genesis prompt for a novel project.
type ProjectSeed struct {
	Title          string            `json:"title"`
	Premise        string            `json:"premise"`
	Genre          string            `json:"genre"`
	Tone           string            `json:"tone"`
	Audience       string            `json:"audience"`
	Language       string            `json:"language"`
	Setting        string            `json:"setting"`
	Themes         []string          `json:"themes,omitempty"`
	MainCharacters []string          `json:"main_characters,omitempty"`
	Constraints    []string          `json:"constraints,omitempty"`
	TargetChapters int               `json:"target_chapters"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// Project is the top-level novel workspace.
type Project struct {
	ID                 string            `json:"id"`
	Title              string            `json:"title"`
	Slug               string            `json:"slug"`
	Status             string            `json:"status"`
	Seed               ProjectSeed       `json:"seed"`
	ActiveStoryBibleID string            `json:"active_story_bible_id,omitempty"`
	DefaultWorldlineID string            `json:"default_worldline_id,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
}

// StoryBible is a versioned canon document derived by Genesis Optimizer.
type StoryBible struct {
	ID                string            `json:"id"`
	ProjectID         string            `json:"project_id"`
	Version           int               `json:"version"`
	Title             string            `json:"title"`
	Logline           string            `json:"logline"`
	Synopsis          string            `json:"synopsis"`
	Genre             string            `json:"genre"`
	Tone              string            `json:"tone"`
	Audience          string            `json:"audience"`
	Language          string            `json:"language"`
	Themes            []string          `json:"themes,omitempty"`
	Rules             map[string]string `json:"rules,omitempty"`
	WorldlineIDs      []string          `json:"worldline_ids,omitempty"`
	EntityIDs         []string          `json:"entity_ids,omitempty"`
	PlotThreadIDs     []string          `json:"plot_thread_ids,omitempty"`
	SourceSeed        ProjectSeed       `json:"source_seed"`
	GenesisWorkflowID string            `json:"genesis_workflow_id,omitempty"`
	Approved          bool              `json:"approved"`
	CreatedAt         time.Time         `json:"created_at"`
}

// Worldline tracks canon variants without mixing facts across timelines.
type Worldline struct {
	ID        string            `json:"id"`
	ProjectID string            `json:"project_id"`
	Name      string            `json:"name"`
	Summary   string            `json:"summary"`
	Canonical bool              `json:"canonical"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// Entity is a graph node for character, place, object, faction or concept.
type Entity struct {
	ID          string            `json:"id"`
	ProjectID   string            `json:"project_id"`
	WorldlineID string            `json:"worldline_id,omitempty"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Aliases     []string          `json:"aliases,omitempty"`
	Summary     string            `json:"summary"`
	Traits      map[string]string `json:"traits,omitempty"`
	Importance  int               `json:"importance"`
	Status      string            `json:"status"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// Fact is an atomic canonical assertion that can be re-indexed and audited.
type Fact struct {
	ID               string            `json:"id"`
	ProjectID        string            `json:"project_id"`
	WorldlineID      string            `json:"worldline_id,omitempty"`
	EntityID         string            `json:"entity_id,omitempty"`
	ChapterID        string            `json:"chapter_id,omitempty"`
	ChapterVersionID string            `json:"chapter_version_id,omitempty"`
	Claim            string            `json:"claim"`
	Source           string            `json:"source"`
	Confidence       float64           `json:"confidence"`
	Status           string            `json:"status"`
	EmbeddingRef     string            `json:"embedding_ref,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

// GraphEdge connects entities and stores evidence, enabling graph expansion tools.
type GraphEdge struct {
	ID              string            `json:"id"`
	ProjectID       string            `json:"project_id"`
	WorldlineID     string            `json:"worldline_id,omitempty"`
	SourceEntityID  string            `json:"source_entity_id"`
	TargetEntityID  string            `json:"target_entity_id"`
	Type            string            `json:"type"`
	Label           string            `json:"label"`
	Weight          float64           `json:"weight"`
	EvidenceFactIDs []string          `json:"evidence_fact_ids,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// PlotThread tracks unresolved narrative promises and arcs.
type PlotThread struct {
	ID               string            `json:"id"`
	ProjectID        string            `json:"project_id"`
	WorldlineID      string            `json:"worldline_id,omitempty"`
	Title            string            `json:"title"`
	Summary          string            `json:"summary"`
	Status           string            `json:"status"`
	Priority         int               `json:"priority"`
	RelatedEntityIDs []string          `json:"related_entity_ids,omitempty"`
	OpenedChapterID  string            `json:"opened_chapter_id,omitempty"`
	ClosedChapterID  string            `json:"closed_chapter_id,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

// Chapter is a stable chapter identity; versions carry mutable content.
type Chapter struct {
	ID        string            `json:"id"`
	ProjectID string            `json:"project_id"`
	Number    int               `json:"number"`
	Title     string            `json:"title"`
	Status    string            `json:"status"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// ChapterVersion is immutable content saved after a user or AI write.
type ChapterVersion struct {
	ID               string            `json:"id"`
	ProjectID        string            `json:"project_id"`
	ChapterID        string            `json:"chapter_id"`
	Version          int               `json:"version"`
	Title            string            `json:"title"`
	Content          string            `json:"content"`
	Summary          string            `json:"summary"`
	AuthorRole       AgentRole         `json:"author_role"`
	SourceWorkflowID string            `json:"source_workflow_id,omitempty"`
	IndexStatus      string            `json:"index_status"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
}

// IndexJob requests re-indexing, fact extraction and graph refresh after content changes.
type IndexJob struct {
	ID               string            `json:"id"`
	ProjectID        string            `json:"project_id"`
	ChapterID        string            `json:"chapter_id,omitempty"`
	ChapterVersionID string            `json:"chapter_version_id,omitempty"`
	Kind             string            `json:"kind"`
	Status           string            `json:"status"`
	Attempts         int               `json:"attempts"`
	Error            string            `json:"error,omitempty"`
	Payload          map[string]string `json:"payload,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	ScheduledAt      *time.Time        `json:"scheduled_at,omitempty"`
	StartedAt        *time.Time        `json:"started_at,omitempty"`
	CompletedAt      *time.Time        `json:"completed_at,omitempty"`
}

// WorkflowStep captures a deterministic state transition in an AI workflow.
type WorkflowStep struct {
	Name      string            `json:"name"`
	Status    string            `json:"status"`
	StartedAt *time.Time        `json:"started_at,omitempty"`
	EndedAt   *time.Time        `json:"ended_at,omitempty"`
	Error     string            `json:"error,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// ModelResolution describes the resolved route/provider/model used by a workflow run.
type ModelResolution struct {
	RouteKey         string       `json:"route_key"`
	ResolutionSource string       `json:"resolution_source"`
	ProviderID       string       `json:"provider_id"`
	ProviderName     string       `json:"provider_name"`
	ProviderType     ProviderType `json:"provider_type"`
	ModelID          string       `json:"model_id"`
	ModelName        string       `json:"model_name"`
	ModelKind        ModelKind    `json:"model_kind"`
}

// IndexFreshness describes whether the latest chapter versions are already indexed.
type IndexFreshness struct {
	ProjectID                     string     `json:"project_id"`
	ChapterID                     string     `json:"chapter_id,omitempty"`
	Status                        string     `json:"status"`
	LatestChapterVersionID        string     `json:"latest_chapter_version_id,omitempty"`
	LatestChapterVersionCreatedAt *time.Time `json:"latest_chapter_version_created_at,omitempty"`
	LatestIndexedChapterVersionID string     `json:"latest_indexed_chapter_version_id,omitempty"`
	LatestIndexedAt               *time.Time `json:"latest_indexed_at,omitempty"`
	PendingJobCount               int        `json:"pending_job_count"`
}

// ContinuityEvidenceRef points at canon material used by deterministic continuity checks.
type ContinuityEvidenceRef struct {
	SourceType string `json:"source_type"`
	SourceID   string `json:"source_id,omitempty"`
	Label      string `json:"label"`
	Excerpt    string `json:"excerpt,omitempty"`
}

// ContinuityIssue describes one deterministic continuity finding for a generated draft.
type ContinuityIssue struct {
	Type         string                  `json:"type"`
	Severity     string                  `json:"severity"`
	Message      string                  `json:"message"`
	DraftExcerpt string                  `json:"draft_excerpt"`
	Suggestion   string                  `json:"suggestion"`
	Evidence     []ContinuityEvidenceRef `json:"evidence"`
}

// ContinuityAudit summarizes deterministic continuity checks for one generated draft.
type ContinuityAudit struct {
	Status string            `json:"status"`
	Issues []ContinuityIssue `json:"issues"`
}

// AIWorkflow tracks long-running logical agent work.
type AIWorkflow struct {
	ID              string            `json:"id"`
	ProjectID       string            `json:"project_id"`
	Kind            string            `json:"kind"`
	Role            AgentRole         `json:"role"`
	Status          string            `json:"status"`
	ModelID         string            `json:"model_id,omitempty"`
	ContextPackID   string            `json:"context_pack_id,omitempty"`
	ModelResolution *ModelResolution  `json:"model_resolution,omitempty"`
	Steps           []WorkflowStep    `json:"steps,omitempty"`
	Input           map[string]string `json:"input,omitempty"`
	Output          map[string]string `json:"output,omitempty"`
	Error           string            `json:"error,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// AIRun records one concrete provider request/response summary.
type AIRun struct {
	ID             string            `json:"id"`
	WorkflowID     string            `json:"workflow_id"`
	ProviderID     string            `json:"provider_id"`
	ModelID        string            `json:"model_id"`
	Role           AgentRole         `json:"role"`
	Status         string            `json:"status"`
	PromptTokens   int               `json:"prompt_tokens"`
	OutputTokens   int               `json:"output_tokens"`
	TotalTokens    int               `json:"total_tokens"`
	LatencyMillis  int64             `json:"latency_millis"`
	Error          string            `json:"error,omitempty"`
	TraceObjectRef string            `json:"trace_object_ref,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
}

// ChapterSummary is a compact retrieval result for context packs.
type ChapterSummary struct {
	ChapterID        string `json:"chapter_id"`
	ChapterVersionID string `json:"chapter_version_id"`
	Title            string `json:"title"`
	Summary          string `json:"summary"`
}

// ContextPack contains selected facts, graph nodes and summaries for a role-specific model call.
type ContextPack struct {
	ID               string            `json:"id"`
	ProjectID        string            `json:"project_id"`
	ChapterID        string            `json:"chapter_id,omitempty"`
	Role             AgentRole         `json:"role"`
	TokenBudget      int               `json:"token_budget"`
	Query            string            `json:"query"`
	StoryBibleID     string            `json:"story_bible_id,omitempty"`
	WorldRules       map[string]string `json:"world_rules,omitempty"`
	Facts            []Fact            `json:"facts,omitempty"`
	Entities         []Entity          `json:"entities,omitempty"`
	Edges            []GraphEdge       `json:"edges,omitempty"`
	PlotThreads      []PlotThread      `json:"plot_threads,omitempty"`
	ChapterSummaries []ChapterSummary  `json:"chapter_summaries,omitempty"`
	ToolTrace        []string          `json:"tool_trace,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
}

// GraphExpansion is returned by novel graph expansion tools.
type GraphExpansion struct {
	ProjectID string      `json:"project_id"`
	Depth     int         `json:"depth"`
	Entities  []Entity    `json:"entities"`
	Edges     []GraphEdge `json:"edges"`
	Facts     []Fact      `json:"facts"`
}
