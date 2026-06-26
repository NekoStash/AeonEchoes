package agent

import (
	"testing"

	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/memory"
)

func TestModelRouterSelectTextModelUsesExplicitSettingBeforeRoleFallback(t *testing.T) {
	store := memory.NewStore()
	providerCfg, err := store.CreateProvider(domain.ProviderConfig{ID: "provider_writer", Name: "Writer Provider", Type: domain.ProviderOpenAI, Enabled: true})
	if err != nil {
		t.Fatalf("CreateProvider() error: %v", err)
	}
	otherProvider, err := store.CreateProvider(domain.ProviderConfig{ID: "provider_editor", Name: "Editor Provider", Type: domain.ProviderAnthropic, Enabled: true})
	if err != nil {
		t.Fatalf("CreateProvider() error: %v", err)
	}
	_, err = store.CreateModel(domain.ModelConfig{ID: "provider_writer:fallback-model", ProviderID: providerCfg.ID, Name: "fallback-model", Kind: domain.ModelKindText, Enabled: true, DefaultForKind: true, RoutingWeight: 1000, AllowedAgentRoles: []domain.AgentRole{domain.AgentRoleWriter}})
	if err != nil {
		t.Fatalf("CreateModel(fallback) error: %v", err)
	}
	_, err = store.CreateModel(domain.ModelConfig{ID: "provider_editor:explicit-model", ProviderID: otherProvider.ID, Name: "explicit-model", Kind: domain.ModelKindText, Enabled: true, RoutingWeight: 1, AllowedAgentRoles: []domain.AgentRole{domain.AgentRoleEditor}})
	if err != nil {
		t.Fatalf("CreateModel(explicit) error: %v", err)
	}
	_, err = store.UpsertSetting(domain.AppSetting{Scope: ModelRoutingSettingScope, Key: string(domain.AgentRoleWriter), Value: map[string]any{ModelRoutingSettingValueKey: "provider_editor:explicit-model"}})
	if err != nil {
		t.Fatalf("UpsertSetting() error: %v", err)
	}

	selection, err := NewModelRouter(store, NewAgentRoleRegistry()).SelectTextModel(domain.AgentRoleWriter)
	if err != nil {
		t.Fatalf("SelectTextModel() error: %v", err)
	}
	if selection.Model.Name != "explicit-model" || selection.Provider.ID != "provider_editor" {
		t.Fatalf("SelectTextModel() = model %q provider %q, want explicit model/provider", selection.Model.Name, selection.Provider.ID)
	}
}

func TestModelRouterSelectTextModelFallsBackWhenNoExplicitSetting(t *testing.T) {
	store := memory.NewStore()
	providerCfg, err := store.CreateProvider(domain.ProviderConfig{ID: "provider_writer", Name: "Writer Provider", Type: domain.ProviderOpenAI, Enabled: true})
	if err != nil {
		t.Fatalf("CreateProvider() error: %v", err)
	}
	_, err = store.CreateModel(domain.ModelConfig{ID: "provider_writer:writer-model", ProviderID: providerCfg.ID, Name: "writer-model", Kind: domain.ModelKindText, Enabled: true, RoutingWeight: 100, AllowedAgentRoles: []domain.AgentRole{domain.AgentRoleWriter}})
	if err != nil {
		t.Fatalf("CreateModel(writer) error: %v", err)
	}
	_, err = store.CreateModel(domain.ModelConfig{ID: "provider_writer:editor-model", ProviderID: providerCfg.ID, Name: "editor-model", Kind: domain.ModelKindText, Enabled: true, RoutingWeight: 1000, AllowedAgentRoles: []domain.AgentRole{domain.AgentRoleEditor}})
	if err != nil {
		t.Fatalf("CreateModel(editor) error: %v", err)
	}

	selection, err := NewModelRouter(store, NewAgentRoleRegistry()).SelectTextModel(domain.AgentRoleWriter)
	if err != nil {
		t.Fatalf("SelectTextModel() error: %v", err)
	}
	if selection.Model.Name != "writer-model" {
		t.Fatalf("SelectTextModel() = %q, want role fallback writer-model", selection.Model.Name)
	}
}

func TestModelRouterSelectEmbeddingModelUsesExplicitSetting(t *testing.T) {
	store := memory.NewStore()
	providerCfg, err := store.CreateProvider(domain.ProviderConfig{ID: "provider_embed", Name: "Embed Provider", Type: domain.ProviderOpenAI, Enabled: true})
	if err != nil {
		t.Fatalf("CreateProvider() error: %v", err)
	}
	_, err = store.CreateModel(domain.ModelConfig{ID: "provider_embed:fallback-embedding", ProviderID: providerCfg.ID, Name: "fallback-embedding", Kind: domain.ModelKindEmbedding, Enabled: true, DefaultForKind: true, RoutingWeight: 1000})
	if err != nil {
		t.Fatalf("CreateModel(fallback) error: %v", err)
	}
	_, err = store.CreateModel(domain.ModelConfig{ID: "provider_embed:explicit-embedding", ProviderID: providerCfg.ID, Name: "explicit-embedding", Kind: domain.ModelKindEmbedding, Enabled: true, RoutingWeight: 1})
	if err != nil {
		t.Fatalf("CreateModel(explicit) error: %v", err)
	}
	_, err = store.UpsertSetting(domain.AppSetting{Scope: ModelRoutingSettingScope, Key: ModelRoutingEmbeddingKey, Value: map[string]any{ModelRoutingSettingValueKey: "provider_embed:explicit-embedding"}})
	if err != nil {
		t.Fatalf("UpsertSetting() error: %v", err)
	}

	selection, err := NewModelRouter(store, NewAgentRoleRegistry()).SelectEmbeddingModel()
	if err != nil {
		t.Fatalf("SelectEmbeddingModel() error: %v", err)
	}
	if selection.Model.Name != "explicit-embedding" {
		t.Fatalf("SelectEmbeddingModel() = %q, want explicit-embedding", selection.Model.Name)
	}
}

func TestModelRouterRejectsInvalidExplicitSetting(t *testing.T) {
	store := memory.NewStore()
	providerCfg, err := store.CreateProvider(domain.ProviderConfig{ID: "provider_writer", Name: "Writer Provider", Type: domain.ProviderOpenAI, Enabled: true})
	if err != nil {
		t.Fatalf("CreateProvider() error: %v", err)
	}
	_, err = store.CreateModel(domain.ModelConfig{ID: "provider_writer:writer-model", ProviderID: providerCfg.ID, Name: "writer-model", Kind: domain.ModelKindText, Enabled: true})
	if err != nil {
		t.Fatalf("CreateModel() error: %v", err)
	}
	_, err = store.UpsertSetting(domain.AppSetting{Scope: ModelRoutingSettingScope, Key: string(domain.AgentRoleWriter), Value: map[string]any{ModelRoutingSettingValueKey: "missing-separator"}})
	if err != nil {
		t.Fatalf("UpsertSetting() error: %v", err)
	}

	_, err = NewModelRouter(store, NewAgentRoleRegistry()).SelectTextModel(domain.AgentRoleWriter)
	if err == nil {
		t.Fatalf("SelectTextModel() error = nil, want invalid setting error")
	}
}
