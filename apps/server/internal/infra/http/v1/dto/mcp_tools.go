package dto

import (
	"aeonechoes/server/internal/domain"
	"time"
)

type McpServerDTO struct {
	ID                string                 `json:"id"`
	ProjectID         string                 `json:"project_id,omitempty"`
	Name              string                 `json:"name"`
	Transport         domain.MCPTransport    `json:"transport"`
	Status            domain.MCPServerStatus `json:"status"`
	Enabled           bool                   `json:"enabled"`
	Command           string                 `json:"command,omitempty"`
	Args              []string               `json:"args,omitempty"`
	URL               string                 `json:"url,omitempty"`
	Headers           map[string]string      `json:"headers,omitempty"`
	SecretHeadersHint []string               `json:"secret_headers_hint,omitempty"`
	Env               map[string]string      `json:"env,omitempty"`
	SecretEnvHint     []string               `json:"secret_env_hint,omitempty"`
	TimeoutSec        int                    `json:"timeout_sec,omitempty"`
	Metadata          map[string]string      `json:"metadata,omitempty"`
	LastSeenAt        *time.Time             `json:"last_seen_at,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

type McpServerRequestDTO struct {
	ID            string                 `json:"id,omitempty"`
	ProjectID     string                 `json:"project_id,omitempty"`
	Name          string                 `json:"name"`
	Transport     domain.MCPTransport    `json:"transport"`
	Status        domain.MCPServerStatus `json:"status,omitempty"`
	Enabled       bool                   `json:"enabled"`
	Command       string                 `json:"command,omitempty"`
	Args          []string               `json:"args,omitempty"`
	URL           string                 `json:"url,omitempty"`
	Headers       map[string]string      `json:"headers,omitempty"`
	SecretHeaders map[string]string      `json:"secret_headers,omitempty"`
	Env           map[string]string      `json:"env,omitempty"`
	SecretEnv     map[string]string      `json:"secret_env,omitempty"`
	TimeoutSec    int                    `json:"timeout_sec,omitempty"`
	Metadata      map[string]string      `json:"metadata,omitempty"`
}

type McpServerTestDTO struct {
	OK     bool         `json:"ok"`
	Server McpServerDTO `json:"server"`
}

type McpToolRefreshDTO struct {
	Tools       []ToolDefinitionDTO `json:"tools"`
	Count       int                 `json:"count"`
	Unavailable int                 `json:"unavailable"`
}

type ToolDefinitionDTO struct {
	ID          string                    `json:"id"`
	ProjectID   string                    `json:"project_id,omitempty"`
	Name        string                    `json:"name"`
	DisplayName string                    `json:"display_name,omitempty"`
	Description string                    `json:"description,omitempty"`
	Kind        domain.ToolDefinitionKind `json:"kind"`
	Status      domain.ToolStatus         `json:"status"`
	MCPServerID string                    `json:"mcp_server_id,omitempty"`
	SourceID    string                    `json:"source_id,omitempty"`
	SkillID     string                    `json:"skill_id,omitempty"`
	InputSchema map[string]any            `json:"input_schema,omitempty"`
	Metadata    map[string]string         `json:"metadata,omitempty"`
	CreatedAt   time.Time                 `json:"created_at"`
	UpdatedAt   time.Time                 `json:"updated_at"`
}

type ToolInvocationDTO struct {
	ID          string                      `json:"id"`
	AgentRunID  string                      `json:"agent_run_id,omitempty"`
	AgentID     string                      `json:"agent_id,omitempty"`
	ProjectID   string                      `json:"project_id,omitempty"`
	ToolID      string                      `json:"tool_id,omitempty"`
	ToolName    string                      `json:"tool_name"`
	Status      domain.ToolInvocationStatus `json:"status"`
	Arguments   map[string]any              `json:"arguments,omitempty"`
	Result      map[string]any              `json:"result,omitempty"`
	Error       string                      `json:"error,omitempty"`
	StartedAt   *time.Time                  `json:"started_at,omitempty"`
	CompletedAt *time.Time                  `json:"completed_at,omitempty"`
	CreatedAt   time.Time                   `json:"created_at"`
	UpdatedAt   time.Time                   `json:"updated_at"`
}
