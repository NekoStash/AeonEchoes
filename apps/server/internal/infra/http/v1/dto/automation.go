package dto

import (
	"aeonechoes/server/internal/domain"
	"time"
)

type AgentConfigDTO struct {
	ID             string            `json:"id"`
	ProjectID      string            `json:"project_id,omitempty"`
	Name           string            `json:"name"`
	Description    string            `json:"description,omitempty"`
	Role           domain.AgentRole  `json:"role,omitempty"`
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

type AgentRunDTO struct {
	ID                string                `json:"id"`
	AgentID           string                `json:"agent_id"`
	ProjectID         string                `json:"project_id,omitempty"`
	Status            domain.AgentRunStatus `json:"status"`
	Input             map[string]any        `json:"input,omitempty"`
	Output            map[string]any        `json:"output,omitempty"`
	Error             string                `json:"error,omitempty"`
	ToolInvocationIDs []string              `json:"tool_invocation_ids,omitempty"`
	StartedAt         *time.Time            `json:"started_at,omitempty"`
	CompletedAt       *time.Time            `json:"completed_at,omitempty"`
	CreatedAt         time.Time             `json:"created_at"`
	UpdatedAt         time.Time             `json:"updated_at"`
}

type AgentRunRequestDTO struct {
	ProjectID        string               `json:"project_id,omitempty"`
	TaskType         string               `json:"task_type,omitempty"`
	Input            map[string]any       `json:"input,omitempty"`
	ContextSelection *ContextSelectionDTO `json:"context_selection,omitempty"`
	MaxOutputTokens  int                  `json:"max_output_tokens,omitempty"`
}

type AgentRunResultDTO struct {
	Run             AgentRunDTO        `json:"run"`
	Content         string             `json:"content"`
	ToolTrace       []string           `json:"tool_trace,omitempty"`
	ModelResolution ModelResolutionDTO `json:"model_resolution"`
}

type SkillSourceDTO struct {
	ID         string                 `json:"id"`
	ProjectID  string                 `json:"project_id,omitempty"`
	Name       string                 `json:"name"`
	Type       domain.SkillSourceType `json:"type"`
	Path       string                 `json:"path,omitempty"`
	InlineText string                 `json:"inline_text,omitempty"`
	Enabled    bool                   `json:"enabled"`
	Metadata   map[string]string      `json:"metadata,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

type SkillDTO struct {
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

type SkillScanResultDTO struct {
	SourceID  string    `json:"source_id"`
	Path      string    `json:"path"`
	Created   int       `json:"created"`
	Updated   int       `json:"updated"`
	Deleted   int       `json:"deleted"`
	Unchanged int       `json:"unchanged"`
	Errors    []string  `json:"errors,omitempty"`
	ScannedAt time.Time `json:"scanned_at"`
}

type InlineSkillRequestDTO struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Content     string            `json:"content,omitempty"`
	Enabled     *bool             `json:"enabled,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	SourceID    string            `json:"source_id,omitempty"`
	ProjectID   string            `json:"project_id,omitempty"`
	Path        string            `json:"path,omitempty"`
}
