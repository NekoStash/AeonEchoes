package dto

import (
	"aeonechoes/server/internal/domain"
	"time"
)

type ProviderDTO struct {
	ID                       string              `json:"id"`
	Name                     string              `json:"name"`
	Type                     domain.ProviderType `json:"type"`
	BaseURL                  string              `json:"base_url"`
	APIKeyHint               string              `json:"api_key_hint,omitempty"`
	Enabled                  bool                `json:"enabled"`
	Streaming                bool                `json:"streaming"`
	TraceEnabled             bool                `json:"trace_enabled"`
	TraceRetentionDays       int                 `json:"trace_retention_days"`
	DefaultRequestTimeoutSec int                 `json:"default_request_timeout_sec"`
	DefaultModelID           string              `json:"default_model_id,omitempty"`
	Metadata                 map[string]string   `json:"metadata,omitempty"`
	Status                   string              `json:"status"`
	LastCheckedAt            *time.Time          `json:"last_checked_at,omitempty"`
	LastModelRefreshAt       *time.Time          `json:"last_model_refresh_at,omitempty"`
	CreatedAt                time.Time           `json:"created_at"`
	UpdatedAt                time.Time           `json:"updated_at"`
}

type ProviderRequestDTO struct {
	ID                       string              `json:"id,omitempty"`
	Name                     string              `json:"name"`
	Type                     domain.ProviderType `json:"type"`
	BaseURL                  string              `json:"base_url"`
	APIKey                   *string             `json:"api_key,omitempty"`
	Enabled                  *bool               `json:"enabled,omitempty"`
	Streaming                *bool               `json:"streaming,omitempty"`
	TraceEnabled             *bool               `json:"trace_enabled,omitempty"`
	TraceRetentionDays       int                 `json:"trace_retention_days,omitempty"`
	DefaultRequestTimeoutSec int                 `json:"default_request_timeout_sec,omitempty"`
	DefaultModelID           *string             `json:"default_model_id,omitempty"`
	Metadata                 map[string]string   `json:"metadata,omitempty"`
}

type ProviderModelRefreshDTO struct {
	Models   []ModelDTO  `json:"models"`
	Count    int         `json:"count"`
	Provider ProviderDTO `json:"provider"`
}

type ModelDTO struct {
	ID                string              `json:"id"`
	ProviderID        string              `json:"provider_id"`
	ProviderType      domain.ProviderType `json:"provider_type"`
	Name              string              `json:"name"`
	DisplayName       string              `json:"display_name"`
	Kind              domain.ModelKind    `json:"kind"`
	ContextWindow     int                 `json:"context_window"`
	MaxOutputTokens   int                 `json:"max_output_tokens"`
	Dimension         int                 `json:"dimension,omitempty"`
	SupportsTools     bool                `json:"supports_tools"`
	SupportsStreaming bool                `json:"supports_streaming"`
	DefaultForKind    bool                `json:"default_for_kind"`
	Enabled           bool                `json:"enabled"`
	CostInputPerMTok  float64             `json:"cost_input_per_mtok,omitempty"`
	CostOutputPerMTok float64             `json:"cost_output_per_mtok,omitempty"`
	RoutingWeight     int                 `json:"routing_weight"`
	AllowedAgentRoles []domain.AgentRole  `json:"allowed_agent_roles,omitempty"`
	Metadata          map[string]string   `json:"metadata,omitempty"`
	CreatedAt         time.Time           `json:"created_at"`
	UpdatedAt         time.Time           `json:"updated_at"`
	LastSeenAt        *time.Time          `json:"last_seen_at,omitempty"`
}

type ModelRequestDTO struct {
	ID                string              `json:"id,omitempty"`
	ProviderID        string              `json:"provider_id"`
	ProviderType      domain.ProviderType `json:"provider_type,omitempty"`
	Name              string              `json:"name"`
	DisplayName       string              `json:"display_name"`
	Kind              domain.ModelKind    `json:"kind"`
	ContextWindow     int                 `json:"context_window"`
	MaxOutputTokens   int                 `json:"max_output_tokens"`
	Dimension         int                 `json:"dimension,omitempty"`
	SupportsTools     bool                `json:"supports_tools"`
	SupportsStreaming bool                `json:"supports_streaming"`
	DefaultForKind    bool                `json:"default_for_kind"`
	Enabled           bool                `json:"enabled"`
	CostInputPerMTok  float64             `json:"cost_input_per_mtok,omitempty"`
	CostOutputPerMTok float64             `json:"cost_output_per_mtok,omitempty"`
	RoutingWeight     int                 `json:"routing_weight"`
	AllowedAgentRoles []domain.AgentRole  `json:"allowed_agent_roles,omitempty"`
	Metadata          map[string]string   `json:"metadata,omitempty"`
}

type ModelRoutingDTO struct {
	Routes map[string]string `json:"routes"`
}
