package httpapi

import (
	"testing"
	"time"

	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/infra/http/v1/mappers"
	"aeonechoes/server/internal/provider"
)

func TestMergeDiscoveredModelPreservesManualValuesWhenDiscoveryOmitsFields(t *testing.T) {
	seenAt := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	existing := domain.ModelConfig{
		ID:                "provider_a:model-a",
		ProviderID:        "provider_a",
		ProviderType:      domain.ProviderOpenAI,
		Name:              "model-a",
		DisplayName:       "Manual Model A",
		Kind:              domain.ModelKindText,
		ContextWindow:     128000,
		MaxOutputTokens:   8192,
		Dimension:         3072,
		SupportsTools:     true,
		SupportsStreaming: true,
		Enabled:           true,
		DefaultForKind:    true,
		RoutingWeight:     900,
		AllowedAgentRoles: []domain.AgentRole{domain.AgentRoleWriter},
		Metadata:          map[string]string{"owner": "operator"},
	}
	discovered := domain.ModelConfig{
		ID:                "provider_a:model-a",
		ProviderID:        "provider_a",
		ProviderType:      domain.ProviderOpenAI,
		Name:              "model-a",
		DisplayName:       "SDK Model A",
		Kind:              domain.ModelKindText,
		ContextWindow:     0,
		MaxOutputTokens:   0,
		Dimension:         0,
		SupportsTools:     false,
		SupportsStreaming: false,
		LastSeenAt:        &seenAt,
	}
	merged := mappers.MergeDiscoveredModel(existing, discovered, provider.ModelInfo{})

	if merged.ContextWindow != 128000 || merged.MaxOutputTokens != 8192 || merged.Dimension != 3072 {
		t.Fatalf("manual numeric values were not preserved: %+v", merged)
	}
	if !merged.SupportsTools || !merged.SupportsStreaming {
		t.Fatalf("manual capability values were not preserved: %+v", merged)
	}
	if !merged.DefaultForKind || merged.RoutingWeight != 900 || len(merged.AllowedAgentRoles) != 1 || merged.Metadata["owner"] != "operator" {
		t.Fatalf("manual routing metadata was not preserved: %+v", merged)
	}
	if merged.DisplayName != "SDK Model A" || merged.LastSeenAt == nil || !merged.LastSeenAt.Equal(seenAt) {
		t.Fatalf("discovered identity fields were not refreshed: %+v", merged)
	}
}

func TestMergeDiscoveredModelOverwritesFieldsWhenDiscoveryHasKnownValues(t *testing.T) {
	existing := domain.ModelConfig{
		ID:                "provider_a:model-a",
		ProviderID:        "provider_a",
		ProviderType:      domain.ProviderOpenAI,
		Name:              "model-a",
		DisplayName:       "Manual Model A",
		Kind:              domain.ModelKindText,
		ContextWindow:     128000,
		MaxOutputTokens:   8192,
		Dimension:         1536,
		SupportsTools:     true,
		SupportsStreaming: true,
	}
	discovered := domain.ModelConfig{
		ID:                "provider_a:model-a",
		ProviderID:        "provider_a",
		ProviderType:      domain.ProviderOpenAI,
		Name:              "model-a",
		DisplayName:       "SDK Model A",
		Kind:              domain.ModelKindEmbedding,
		ContextWindow:     64000,
		MaxOutputTokens:   2048,
		Dimension:         3072,
		SupportsTools:     false,
		SupportsStreaming: false,
	}
	merged := mappers.MergeDiscoveredModel(existing, discovered, provider.ModelInfo{SupportsToolsKnown: true, SupportsStreamKnown: true})

	if merged.ContextWindow != 64000 || merged.MaxOutputTokens != 2048 || merged.Dimension != 3072 {
		t.Fatalf("known discovered numeric values were not applied: %+v", merged)
	}
	if merged.SupportsTools || merged.SupportsStreaming {
		t.Fatalf("known discovered capability values were not applied: %+v", merged)
	}
	if merged.Kind != domain.ModelKindEmbedding {
		t.Fatalf("discovered kind was not applied: %+v", merged)
	}
}

func TestDiscoveredModelConfigKeepsInferredCapabilitiesForNewModels(t *testing.T) {
	seenAt := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	cfg := domain.ProviderConfig{ID: "provider_a", Type: domain.ProviderGemini}
	model := mappers.DiscoveredModelConfig(cfg, provider.ModelInfo{ID: "gemini-2.5", Name: "models/gemini-2.5", DisplayName: "Gemini 2.5", Kind: domain.ModelKindText, SupportsTools: true, SupportsStream: true}, seenAt)

	if model.ID != "provider_a:gemini-2.5" || model.ProviderID != "provider_a" || model.Name != "gemini-2.5" {
		t.Fatalf("unexpected discovered model identity: %+v", model)
	}
	if !model.SupportsTools || !model.SupportsStreaming {
		t.Fatalf("new model should keep discovered default capabilities: %+v", model)
	}
	if model.RoutingWeight != 100 || !model.Enabled {
		t.Fatalf("unexpected discovered routing defaults: %+v", model)
	}
}
