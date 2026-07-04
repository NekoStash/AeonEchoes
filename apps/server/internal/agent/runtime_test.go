package agent

import (
	"context"
	"strings"
	"testing"

	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/memory"
	"aeonechoes/server/internal/provider"
	"aeonechoes/server/internal/repository"
)

func TestRuntimeRunDisabledAgentFails(t *testing.T) {
	store := memory.NewStore()
	agentCfg, err := store.CreateAgentConfig(domain.AgentConfig{Name: "Disabled Writer", Role: domain.AgentRoleWriter, Enabled: false})
	if err != nil {
		t.Fatalf("CreateAgentConfig() error: %v", err)
	}

	runtime := NewRuntime(store, NewModelRouter(store, NewAgentRoleRegistry()), nil, fakeTextClientFactory{client: &fakeTextClient{responses: []provider.ModelResponse{{Content: "unused"}}}}, nil)
	_, err = runtime.Run(context.Background(), AgentRunRequest{AgentID: agentCfg.ID, TaskType: "draft", Input: map[string]any{"brief": "write"}})
	if err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("Run(disabled agent) error = %v, want disabled error", err)
	}
	runs, err := store.ListAgentRuns(repository.AgentRunFilter{AgentID: agentCfg.ID})
	if err != nil {
		t.Fatalf("ListAgentRuns() error: %v", err)
	}
	if len(runs) != 0 {
		t.Fatalf("disabled agent created runs: %+v", runs)
	}
}

func TestRuntimeRunWithoutToolsCreatesCompletedAgentRun(t *testing.T) {
	store := memory.NewStore()
	providerCfg, err := store.CreateProvider(domain.ProviderConfig{ID: "provider_runtime", Name: "Runtime Provider", Type: domain.ProviderOpenAI, Enabled: true})
	if err != nil {
		t.Fatalf("CreateProvider() error: %v", err)
	}
	modelCfg, err := store.CreateModel(domain.ModelConfig{ID: "model_runtime", ProviderID: providerCfg.ID, Name: "runtime-model", Kind: domain.ModelKindText, Enabled: true, DefaultForKind: true, SupportsTools: false, MaxOutputTokens: 512, AllowedAgentRoles: []domain.AgentRole{domain.AgentRoleWriter}})
	if err != nil {
		t.Fatalf("CreateModel() error: %v", err)
	}
	agentCfg, err := store.CreateAgentConfig(domain.AgentConfig{Name: "Runtime Writer", Role: domain.AgentRoleWriter, Enabled: true})
	if err != nil {
		t.Fatalf("CreateAgentConfig() error: %v", err)
	}
	textClient := &fakeTextClient{responses: []provider.ModelResponse{{Content: "生成完成"}}}
	runtime := NewRuntime(store, NewModelRouter(store, NewAgentRoleRegistry()), nil, fakeTextClientFactory{client: textClient}, nil)

	result, err := runtime.Run(context.Background(), AgentRunRequest{AgentID: agentCfg.ID, TaskType: "draft", Input: map[string]any{"brief": "写一段"}})
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if result.Run.Status != domain.AgentRunStatusCompleted {
		t.Fatalf("run status = %q, want completed", result.Run.Status)
	}
	if result.Content != "生成完成" || result.Run.Output["content"] != "生成完成" {
		t.Fatalf("content mismatch: result=%q output=%+v", result.Content, result.Run.Output)
	}
	if result.ModelResolution.ModelID != modelCfg.ID || result.ModelResolution.ProviderID != providerCfg.ID {
		t.Fatalf("model resolution = %+v, want model %q provider %q", result.ModelResolution, modelCfg.ID, providerCfg.ID)
	}
	if len(textClient.requests) != 1 {
		t.Fatalf("Generate request count = %d, want 1", len(textClient.requests))
	}
	if len(textClient.requests[0].Tools) != 0 {
		t.Fatalf("Generate request tools = %+v, want none", textClient.requests[0].Tools)
	}
	loaded, err := store.GetAgentRun(result.Run.ID)
	if err != nil {
		t.Fatalf("GetAgentRun() error: %v", err)
	}
	if loaded.Status != domain.AgentRunStatusCompleted || loaded.CompletedAt == nil {
		t.Fatalf("persisted run mismatch: %+v", loaded)
	}
}
