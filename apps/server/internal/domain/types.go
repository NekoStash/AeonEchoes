package domain

import (
	"fmt"
	"strings"
	"time"
)

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

// Valid reports whether the role is part of the supported agent role contract.
func (r AgentRole) Valid() bool {
	switch r {
	case AgentRoleGenesisOptimizer, AgentRolePlotArchitect, AgentRoleWorldBuilder, AgentRoleCharacterKeeper, AgentRoleContinuityAudit, AgentRoleWriter, AgentRoleEditor, AgentRoleFactExtractor, AgentRoleGraphCurator:
		return true
	default:
		return false
	}
}

// SkillSourceType identifies how skill material is supplied to the catalog.
type SkillSourceType string

const (
	SkillSourceInlineText SkillSourceType = "inline_text"
	SkillSourceDirectory  SkillSourceType = "directory"
)

func (t SkillSourceType) Valid() bool {
	switch t {
	case SkillSourceInlineText, SkillSourceDirectory:
		return true
	default:
		return false
	}
}

// MCPTransport declares the transport protocol used to reach an MCP server.
type MCPTransport string

const (
	MCPTransportStdio          MCPTransport = "stdio"
	MCPTransportStreamableHTTP MCPTransport = "streamable_http"
	MCPTransportSSE            MCPTransport = "sse"
)

func (t MCPTransport) Valid() bool {
	switch t {
	case MCPTransportStdio, MCPTransportStreamableHTTP, MCPTransportSSE:
		return true
	default:
		return false
	}
}

// MCPServerStatus describes the last known availability of an MCP server.
type MCPServerStatus string

const (
	MCPServerStatusOnline   MCPServerStatus = "online"
	MCPServerStatusOffline  MCPServerStatus = "offline"
	MCPServerStatusDisabled MCPServerStatus = "disabled"
	MCPServerStatusFailed   MCPServerStatus = "failed"
	MCPServerStatusUnknown  MCPServerStatus = "unknown"
)

func (s MCPServerStatus) Valid() bool {
	switch s {
	case MCPServerStatusOnline, MCPServerStatusOffline, MCPServerStatusDisabled, MCPServerStatusFailed, MCPServerStatusUnknown:
		return true
	default:
		return false
	}
}

// ToolDefinitionKind separates builtin, MCP-backed and skill-backed tool definitions.
type ToolDefinitionKind string

const (
	ToolDefinitionBuiltin ToolDefinitionKind = "builtin"
	ToolDefinitionMCP     ToolDefinitionKind = "mcp"
	ToolDefinitionSkill   ToolDefinitionKind = "skill"
)

func (k ToolDefinitionKind) Valid() bool {
	switch k {
	case ToolDefinitionBuiltin, ToolDefinitionMCP, ToolDefinitionSkill:
		return true
	default:
		return false
	}
}

// ToolStatus is the catalog availability state for a tool definition.
type ToolStatus string

const (
	ToolStatusActive      ToolStatus = "active"
	ToolStatusDisabled    ToolStatus = "disabled"
	ToolStatusUnavailable ToolStatus = "unavailable"
)

func (s ToolStatus) Valid() bool {
	switch s {
	case ToolStatusActive, ToolStatusDisabled, ToolStatusUnavailable:
		return true
	default:
		return false
	}
}

// ToolInvocationStatus tracks one persisted tool call attempt.
type ToolInvocationStatus string

const (
	ToolInvocationStatusRunning   ToolInvocationStatus = "running"
	ToolInvocationStatusSucceeded ToolInvocationStatus = "succeeded"
	ToolInvocationStatusFailed    ToolInvocationStatus = "failed"
)

func (s ToolInvocationStatus) Valid() bool {
	switch s {
	case ToolInvocationStatusRunning, ToolInvocationStatusSucceeded, ToolInvocationStatusFailed:
		return true
	default:
		return false
	}
}

// AgentRunStatus tracks the lifecycle of an agent run.
type AgentRunStatus string

const (
	AgentRunStatusRunning   AgentRunStatus = "running"
	AgentRunStatusCompleted AgentRunStatus = "completed"
	AgentRunStatusFailed    AgentRunStatus = "failed"
)

func (s AgentRunStatus) Valid() bool {
	switch s {
	case AgentRunStatusRunning, AgentRunStatusCompleted, AgentRunStatusFailed:
		return true
	default:
		return false
	}
}

// AgentConfig stores an agent configuration, including selected skills and tools.
type AgentConfig struct {
	ID             string            `json:"id"`
	ProjectID      string            `json:"project_id,omitempty"`
	Name           string            `json:"name"`
	Description    string            `json:"description,omitempty"`
	Role           AgentRole         `json:"role,omitempty"`
	ModelID        string            `json:"model_id,omitempty"`
	Enabled        bool              `json:"enabled"`
	SystemPrompt   string            `json:"system_prompt,omitempty"`
	SkillIDs       []string          `json:"skill_ids,omitempty"`
	ToolIDs        []string          `json:"tool_ids,omitempty"`
	MCPServerIDs   []string          `json:"mcp_server_ids,omitempty"`
	MemoryPolicy   map[string]any    `json:"memory_policy,omitempty"`
	RuntimeOptions map[string]any    `json:"runtime_options,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

func (cfg AgentConfig) Valid() error {
	if strings.TrimSpace(cfg.Name) == "" {
		return fmt.Errorf("agent config name must not be empty")
	}
	return nil
}

// AgentRun records one agent execution and its input/output snapshots.
type AgentRun struct {
	ID                string         `json:"id"`
	AgentID           string         `json:"agent_id"`
	ProjectID         string         `json:"project_id,omitempty"`
	Status            AgentRunStatus `json:"status"`
	Input             map[string]any `json:"input,omitempty"`
	Output            map[string]any `json:"output,omitempty"`
	Error             string         `json:"error,omitempty"`
	ToolInvocationIDs []string       `json:"tool_invocation_ids,omitempty"`
	StartedAt         *time.Time     `json:"started_at,omitempty"`
	CompletedAt       *time.Time     `json:"completed_at,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

func (run AgentRun) Valid() error {
	if strings.TrimSpace(run.AgentID) == "" {
		return fmt.Errorf("agent run agent_id must not be empty")
	}
	if !run.Status.Valid() {
		return fmt.Errorf("agent run status %q is invalid", run.Status)
	}
	return nil
}

// SkillSource stores the source from which one or more skills are produced.
type SkillSource struct {
	ID         string            `json:"id"`
	ProjectID  string            `json:"project_id,omitempty"`
	Name       string            `json:"name"`
	Type       SkillSourceType   `json:"type"`
	Path       string            `json:"path,omitempty"`
	InlineText string            `json:"inline_text,omitempty"`
	Enabled    bool              `json:"enabled"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

func (source SkillSource) Valid() error {
	if strings.TrimSpace(source.Name) == "" {
		return fmt.Errorf("skill source name must not be empty")
	}
	if !source.Type.Valid() {
		return fmt.Errorf("skill source type %q is invalid", source.Type)
	}
	if source.Type == SkillSourceDirectory && strings.TrimSpace(source.Path) == "" {
		return fmt.Errorf("directory skill source path must not be empty")
	}
	return nil
}

// Skill is a normalized skill entry available to agents.
type Skill struct {
	ID          string            `json:"id"`
	ProjectID   string            `json:"project_id,omitempty"`
	SourceID    string            `json:"source_id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Content     string            `json:"content,omitempty"`
	Path        string            `json:"path,omitempty"`
	Enabled     bool              `json:"enabled"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

func (skill Skill) Valid() error {
	if strings.TrimSpace(skill.SourceID) == "" {
		return fmt.Errorf("skill source_id must not be empty")
	}
	if strings.TrimSpace(skill.Name) == "" {
		return fmt.Errorf("skill name must not be empty")
	}
	return nil
}

// MCPServerConfig stores one MCP server connection definition.
type MCPServerConfig struct {
	ID            string            `json:"id"`
	ProjectID     string            `json:"project_id,omitempty"`
	Name          string            `json:"name"`
	Transport     MCPTransport      `json:"transport"`
	Status        MCPServerStatus   `json:"status"`
	Enabled       bool              `json:"enabled"`
	Command       string            `json:"command,omitempty"`
	Args          []string          `json:"args,omitempty"`
	URL           string            `json:"url,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	SecretHeaders map[string]string `json:"secret_headers,omitempty"`
	Env           map[string]string `json:"env,omitempty"`
	SecretEnv     map[string]string `json:"secret_env,omitempty"`
	TimeoutSec    int               `json:"timeout_sec,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	LastSeenAt    *time.Time        `json:"last_seen_at,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

func (cfg MCPServerConfig) Valid() error {
	if strings.TrimSpace(cfg.Name) == "" {
		return fmt.Errorf("mcp server name must not be empty")
	}
	if !cfg.Transport.Valid() {
		return fmt.Errorf("mcp transport %q is invalid", cfg.Transport)
	}
	if !cfg.Status.Valid() {
		return fmt.Errorf("mcp server status %q is invalid", cfg.Status)
	}
	switch cfg.Transport {
	case MCPTransportStdio:
		if strings.TrimSpace(cfg.Command) == "" {
			return fmt.Errorf("stdio mcp server command must not be empty")
		}
	case MCPTransportStreamableHTTP, MCPTransportSSE:
		if strings.TrimSpace(cfg.URL) == "" {
			return fmt.Errorf("%s mcp server url must not be empty", cfg.Transport)
		}
	}
	return nil
}

// ToolDefinition is a catalog entry that can be exposed to agent runs.
type ToolDefinition struct {
	ID          string             `json:"id"`
	ProjectID   string             `json:"project_id,omitempty"`
	Name        string             `json:"name"`
	DisplayName string             `json:"display_name,omitempty"`
	Description string             `json:"description,omitempty"`
	Kind        ToolDefinitionKind `json:"kind"`
	Status      ToolStatus         `json:"status"`
	MCPServerID string             `json:"mcp_server_id,omitempty"`
	SourceID    string             `json:"source_id,omitempty"`
	SkillID     string             `json:"skill_id,omitempty"`
	InputSchema map[string]any     `json:"input_schema,omitempty"`
	Metadata    map[string]string  `json:"metadata,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

func (tool ToolDefinition) Valid() error {
	if strings.TrimSpace(tool.Name) == "" {
		return fmt.Errorf("tool definition name must not be empty")
	}
	if !tool.Kind.Valid() {
		return fmt.Errorf("tool definition kind %q is invalid", tool.Kind)
	}
	if !tool.Status.Valid() {
		return fmt.Errorf("tool status %q is invalid", tool.Status)
	}
	if tool.Kind == ToolDefinitionMCP && strings.TrimSpace(tool.MCPServerID) == "" {
		return fmt.Errorf("mcp tool definition mcp_server_id must not be empty")
	}
	if tool.Kind == ToolDefinitionSkill && strings.TrimSpace(tool.SourceID) == "" && strings.TrimSpace(tool.SkillID) == "" {
		return fmt.Errorf("skill tool definition source_id or skill_id must not be empty")
	}
	return nil
}

// ToolInvocation records one tool call snapshot without depending on catalog retention.
type ToolInvocation struct {
	ID          string               `json:"id"`
	AgentRunID  string               `json:"agent_run_id,omitempty"`
	AgentID     string               `json:"agent_id,omitempty"`
	ProjectID   string               `json:"project_id,omitempty"`
	ToolID      string               `json:"tool_id,omitempty"`
	ToolName    string               `json:"tool_name"`
	Status      ToolInvocationStatus `json:"status"`
	Arguments   map[string]any       `json:"arguments,omitempty"`
	Result      map[string]any       `json:"result,omitempty"`
	Error       string               `json:"error,omitempty"`
	StartedAt   *time.Time           `json:"started_at,omitempty"`
	CompletedAt *time.Time           `json:"completed_at,omitempty"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
}

func (invocation ToolInvocation) Valid() error {
	if strings.TrimSpace(invocation.ToolName) == "" {
		return fmt.Errorf("tool invocation tool_name must not be empty")
	}
	if !invocation.Status.Valid() {
		return fmt.Errorf("tool invocation status %q is invalid", invocation.Status)
	}
	return nil
}

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

// CharacterProfile is a Story Bible-ready character profile that can be generated by agents
// and synchronized into the canonical Entity graph. Secret is optional and omitted when unknown.
type CharacterProfile struct {
	Name    string `json:"name"`
	Role    string `json:"role"`
	Desire  string `json:"desire"`
	Wound   string `json:"wound"`
	Secret  string `json:"secret,omitempty"`
	Summary string `json:"summary,omitempty"`
}

// CharacterProfileMapping records how a generated profile was synchronized into the entity graph.
type CharacterProfileMapping struct {
	Name     string `json:"name"`
	EntityID string `json:"entity_id"`
	Action   string `json:"action"`
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

// ChapterStatus is the canonical lifecycle state shared by chapter plans and persisted chapters.
type ChapterStatus string

const (
	ChapterStatusPlanned   ChapterStatus = "planned"
	ChapterStatusDrafting  ChapterStatus = "drafting"
	ChapterStatusReviewing ChapterStatus = "reviewing"
	ChapterStatusLocked    ChapterStatus = "locked"
)

// Valid reports whether the status is part of the supported chapter lifecycle contract.
func (s ChapterStatus) Valid() bool {
	switch s {
	case ChapterStatusPlanned, ChapterStatusDrafting, ChapterStatusReviewing, ChapterStatusLocked:
		return true
	default:
		return false
	}
}

// Chapter is a stable chapter identity; versions carry mutable content.
type Chapter struct {
	ID        string            `json:"id"`
	ProjectID string            `json:"project_id"`
	Number    int               `json:"number"`
	Title     string            `json:"title"`
	Status    ChapterStatus     `json:"status"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// CreateChapterRequest describes an explicit stable chapter identity creation.
type CreateChapterRequest struct {
	ProjectID string            `json:"project_id"`
	Number    int               `json:"number,omitempty"`
	Title     string            `json:"title"`
	Status    ChapterStatus     `json:"status,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// UpdateChapterRequest describes changes to an existing project-owned chapter.
type UpdateChapterRequest struct {
	ProjectID string             `json:"project_id"`
	ChapterID string             `json:"chapter_id"`
	Number    *int               `json:"number,omitempty"`
	Title     *string            `json:"title,omitempty"`
	Status    *ChapterStatus     `json:"status,omitempty"`
	Metadata  *map[string]string `json:"metadata,omitempty"`
}

// ChapterVersion is immutable content saved after a user or AI write.
type ChapterVersion struct {
	ID               string            `json:"id"`
	ProjectID        string            `json:"project_id"`
	ChapterID        string            `json:"chapter_id"`
	ParentVersionID  string            `json:"parent_version_id,omitempty"`
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
	ProjectID   string      `json:"project_id"`
	Depth       int         `json:"depth"`
	Entities    []Entity    `json:"entities"`
	Edges       []GraphEdge `json:"edges"`
	Facts       []Fact      `json:"facts"`
	GeneratedAt time.Time   `json:"generated_at"`
}
