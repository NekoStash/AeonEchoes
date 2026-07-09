package mappers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/infra/http/v1/dto"
	"aeonechoes/server/internal/infra/http/v1/shared"
	"aeonechoes/server/internal/provider"
)

func ProviderDTOFromDomain(item domain.ProviderConfig) dto.ProviderDTO {
	metadata := CopyStringMapV1(item.Metadata)
	return dto.ProviderDTO{
		ID:                       item.ID,
		Name:                     item.Name,
		Type:                     item.Type,
		BaseURL:                  item.BaseURL,
		APIKeyHint:               ProviderAPIKeyHint(item),
		Enabled:                  item.Enabled,
		Streaming:                metadata["streaming"] == "true",
		TraceEnabled:             item.TraceEnabled,
		TraceRetentionDays:       item.TraceRetentionDays,
		DefaultRequestTimeoutSec: item.DefaultRequestTimeoutSec,
		DefaultModelID:           metadata["default_model_id"],
		Metadata:                 metadata,
		Status:                   ProviderStatus(item),
		LastCheckedAt:            item.LastModelRefreshAt,
		LastModelRefreshAt:       item.LastModelRefreshAt,
		CreatedAt:                item.CreatedAt,
		UpdatedAt:                item.UpdatedAt,
	}
}

func ProviderDTOsFromDomain(items []domain.ProviderConfig) []dto.ProviderDTO {
	providers := make([]dto.ProviderDTO, 0, len(items))
	for _, item := range items {
		providers = append(providers, ProviderDTOFromDomain(item))
	}
	return providers
}

func ProviderRequestToDomain(input dto.ProviderRequestDTO) domain.ProviderConfig {
	cfg := domain.ProviderConfig{ID: input.ID, Name: input.Name, Type: input.Type, BaseURL: input.BaseURL, Enabled: true, TraceRetentionDays: input.TraceRetentionDays, DefaultRequestTimeoutSec: input.DefaultRequestTimeoutSec, Metadata: NormalizedProviderRequestMetadata(input, nil)}
	if input.Enabled != nil {
		cfg.Enabled = *input.Enabled
	}
	if input.TraceEnabled != nil {
		cfg.TraceEnabled = *input.TraceEnabled
	}
	if input.APIKey != nil {
		cfg.APIKey = *input.APIKey
	}
	return cfg
}

func ApplyProviderRequest(input dto.ProviderRequestDTO, existing domain.ProviderConfig) domain.ProviderConfig {
	if input.Type != "" {
		existing.Type = input.Type
	}
	if strings.TrimSpace(input.Name) != "" {
		existing.Name = input.Name
	}
	if strings.TrimSpace(input.BaseURL) != "" {
		existing.BaseURL = input.BaseURL
	}
	if input.APIKey != nil {
		existing.APIKey = *input.APIKey
	}
	existing.APIKeyEnv = ""
	if input.Enabled != nil {
		existing.Enabled = *input.Enabled
	}
	if input.TraceEnabled != nil {
		existing.TraceEnabled = *input.TraceEnabled
	}
	if input.TraceRetentionDays > 0 {
		existing.TraceRetentionDays = input.TraceRetentionDays
	}
	if input.DefaultRequestTimeoutSec > 0 {
		existing.DefaultRequestTimeoutSec = input.DefaultRequestTimeoutSec
	}
	existing.Metadata = NormalizedProviderRequestMetadata(input, existing.Metadata)
	return existing
}

func NormalizedProviderRequestMetadata(input dto.ProviderRequestDTO, existing map[string]string) map[string]string {
	metadata := map[string]string{}
	if input.Metadata != nil {
		for key, value := range input.Metadata {
			metadata[key] = value
		}
	} else {
		for key, value := range existing {
			metadata[key] = value
		}
	}
	if input.Streaming != nil {
		metadata["streaming"] = strconv.FormatBool(*input.Streaming)
	}
	if input.DefaultModelID != nil {
		if strings.TrimSpace(*input.DefaultModelID) != "" {
			metadata["default_model_id"] = strings.TrimSpace(*input.DefaultModelID)
		} else {
			delete(metadata, "default_model_id")
		}
	}
	return metadata
}

func ModelDTOFromDomain(item domain.ModelConfig) dto.ModelDTO {
	return dto.ModelDTO{ID: item.ID, ProviderID: item.ProviderID, ProviderType: item.ProviderType, Name: item.Name, DisplayName: item.DisplayName, Kind: item.Kind, ContextWindow: item.ContextWindow, MaxOutputTokens: item.MaxOutputTokens, Dimension: item.Dimension, SupportsTools: item.SupportsTools, SupportsStreaming: item.SupportsStreaming, DefaultForKind: item.DefaultForKind, Enabled: item.Enabled, CostInputPerMTok: item.CostInputPerMTok, CostOutputPerMTok: item.CostOutputPerMTok, RoutingWeight: item.RoutingWeight, AllowedAgentRoles: append([]domain.AgentRole(nil), item.AllowedAgentRoles...), Metadata: CopyStringMapV1(item.Metadata), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt, LastSeenAt: item.LastSeenAt}
}

func ModelDTOsFromDomain(items []domain.ModelConfig) []dto.ModelDTO {
	models := make([]dto.ModelDTO, 0, len(items))
	for _, item := range items {
		models = append(models, ModelDTOFromDomain(item))
	}
	return models
}

func ModelRequestToDomain(input dto.ModelRequestDTO) domain.ModelConfig {
	return domain.ModelConfig{ID: input.ID, ProviderID: input.ProviderID, ProviderType: input.ProviderType, Name: input.Name, DisplayName: input.DisplayName, Kind: input.Kind, ContextWindow: input.ContextWindow, MaxOutputTokens: input.MaxOutputTokens, Dimension: input.Dimension, SupportsTools: input.SupportsTools, SupportsStreaming: input.SupportsStreaming, DefaultForKind: input.DefaultForKind, Enabled: input.Enabled, CostInputPerMTok: input.CostInputPerMTok, CostOutputPerMTok: input.CostOutputPerMTok, RoutingWeight: input.RoutingWeight, AllowedAgentRoles: append([]domain.AgentRole(nil), input.AllowedAgentRoles...), Metadata: CopyStringMapV1(input.Metadata)}
}

func DiscoveredModelConfig(cfg domain.ProviderConfig, info provider.ModelInfo, seenAt time.Time) domain.ModelConfig {
	return domain.ModelConfig{
		ID:                fmt.Sprintf("%s:%s", cfg.ID, info.ID),
		ProviderID:        cfg.ID,
		ProviderType:      cfg.Type,
		Name:              info.ID,
		DisplayName:       shared.FirstNonEmpty(info.DisplayName, info.Name, info.ID),
		Kind:              info.Kind,
		ContextWindow:     info.ContextWindow,
		MaxOutputTokens:   info.MaxOutputTokens,
		Dimension:         info.Dimension,
		SupportsTools:     info.SupportsTools,
		SupportsStreaming: info.SupportsStream,
		Enabled:           true,
		RoutingWeight:     100,
		LastSeenAt:        &seenAt,
	}
}

func MergeDiscoveredModel(existing, discovered domain.ModelConfig, info provider.ModelInfo) domain.ModelConfig {
	existing.ProviderID = discovered.ProviderID
	existing.ProviderType = discovered.ProviderType
	existing.Name = discovered.Name
	existing.DisplayName = discovered.DisplayName
	existing.Kind = discovered.Kind
	if discovered.ContextWindow > 0 {
		existing.ContextWindow = discovered.ContextWindow
	}
	if discovered.MaxOutputTokens > 0 {
		existing.MaxOutputTokens = discovered.MaxOutputTokens
	}
	if discovered.Dimension > 0 {
		existing.Dimension = discovered.Dimension
	}
	if info.SupportsToolsKnown {
		existing.SupportsTools = discovered.SupportsTools
	}
	if info.SupportsStreamKnown {
		existing.SupportsStreaming = discovered.SupportsStreaming
	}
	existing.LastSeenAt = discovered.LastSeenAt
	return existing
}

func ProviderAPIKeyHint(item domain.ProviderConfig) string {
	if strings.TrimSpace(item.APIKey) != "" {
		return "configured"
	}
	return ""
}

func ProviderStatus(item domain.ProviderConfig) string {
	if !item.Enabled {
		return "offline"
	}
	if item.LastModelRefreshAt != nil {
		return "online"
	}
	return "unknown"
}
