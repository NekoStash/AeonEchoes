package agent

import (
	"context"
	"errors"
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

type runtimeScopeStore struct {
	*memory.Store
	createRunCalls int
	modelListCalls int
}

func (s *runtimeScopeStore) CreateAgentRun(run domain.AgentRun) (domain.AgentRun, error) {
	s.createRunCalls++
	return s.Store.CreateAgentRun(run)
}

func (s *runtimeScopeStore) ListModelsByKind(kind domain.ModelKind) ([]domain.ModelConfig, error) {
	s.modelListCalls++
	return s.Store.ListModelsByKind(kind)
}

type runtimeScopeClientFactory struct {
	client *fakeTextClient
	calls  int
}

func (f *runtimeScopeClientFactory) NewTextClient(_ domain.ProviderConfig) (provider.TextModelClient, error) {
	f.calls++
	return f.client, nil
}

type runtimeScopeToolCatalog struct {
	calls int
}

func (c *runtimeScopeToolCatalog) ListProviderTools(_ context.Context, _ domain.AgentConfig) ([]provider.ToolSpec, error) {
	c.calls++
	return nil, nil
}

func TestRuntimeRunProjectScope(t *testing.T) {
	store := &runtimeScopeStore{Store: memory.NewStore()}
	providerCfg, err := store.CreateProvider(domain.ProviderConfig{ID: "provider_scope", Name: "Scope Provider", Type: domain.ProviderOpenAI, Enabled: true})
	if err != nil {
		t.Fatalf("CreateProvider() error: %v", err)
	}
	_, err = store.CreateModel(domain.ModelConfig{ID: "model_scope", ProviderID: providerCfg.ID, Name: "scope-model", Kind: domain.ModelKindText, Enabled: true, DefaultForKind: true, SupportsTools: false, AllowedAgentRoles: []domain.AgentRole{domain.AgentRoleWriter}})
	if err != nil {
		t.Fatalf("CreateModel() error: %v", err)
	}
	projectAgent, err := store.CreateAgentConfig(domain.AgentConfig{ID: "agent_project_scope", ProjectID: "  project-a  ", Name: "Project Writer", Role: domain.AgentRoleWriter, Enabled: true})
	if err != nil {
		t.Fatalf("CreateAgentConfig(project) error: %v", err)
	}
	globalAgent, err := store.CreateAgentConfig(domain.AgentConfig{ID: "agent_global_scope", Name: "Global Writer", Role: domain.AgentRoleWriter, Enabled: true})
	if err != nil {
		t.Fatalf("CreateAgentConfig(global) error: %v", err)
	}

	t.Run("project agent rejects another project before side effects", func(t *testing.T) {
		assertRuntimeScopeRejected(t, store, projectAgent.ID, " project-b ", "project-a", "project-b")
	})
	t.Run("project agent rejects missing project before side effects", func(t *testing.T) {
		assertRuntimeScopeRejected(t, store, projectAgent.ID, "   ", "project-a", "")
	})
	t.Run("project agent accepts normalized matching project", func(t *testing.T) {
		client := &fakeTextClient{responses: []provider.ModelResponse{{Content: "same project"}}}
		factory := &runtimeScopeClientFactory{client: client}
		tools := &runtimeScopeToolCatalog{}
		runtime := NewRuntime(store, NewModelRouter(store, NewAgentRoleRegistry()), nil, factory, tools)
		result, err := runtime.Run(context.Background(), AgentRunRequest{AgentID: projectAgent.ID, ProjectID: "  project-a  ", Input: map[string]any{"brief": "write"}})
		if err != nil {
			t.Fatalf("Run() error: %v", err)
		}
		if result.Run.ProjectID != "project-a" || result.Run.Status != domain.AgentRunStatusCompleted {
			t.Fatalf("run = %+v, want normalized project-a completed run", result.Run)
		}
		if len(client.requests) != 1 || tools.calls != 1 {
			t.Fatalf("success side effects: model requests=%d tool catalogs=%d, want 1 each", len(client.requests), tools.calls)
		}
	})
	t.Run("global agent accepts requested project", func(t *testing.T) {
		client := &fakeTextClient{responses: []provider.ModelResponse{{Content: "global project"}}}
		factory := &runtimeScopeClientFactory{client: client}
		runtime := NewRuntime(store, NewModelRouter(store, NewAgentRoleRegistry()), nil, factory, nil)
		result, err := runtime.Run(context.Background(), AgentRunRequest{AgentID: globalAgent.ID, ProjectID: "  project-b  ", Input: map[string]any{"brief": "write"}})
		if err != nil {
			t.Fatalf("Run() error: %v", err)
		}
		if result.Run.ProjectID != "project-b" || result.Run.Status != domain.AgentRunStatusCompleted {
			t.Fatalf("run = %+v, want normalized project-b completed run", result.Run)
		}
		if len(client.requests) != 1 {
			t.Fatalf("model requests = %d, want 1", len(client.requests))
		}
	})
}

func assertRuntimeScopeRejected(t *testing.T, store *runtimeScopeStore, agentID, requestProjectID, wantAgentProjectID, wantRequestProjectID string) {
	t.Helper()
	client := &fakeTextClient{responses: []provider.ModelResponse{{Content: "must not run"}}}
	factory := &runtimeScopeClientFactory{client: client}
	tools := &runtimeScopeToolCatalog{}
	runtime := NewRuntime(store, NewModelRouter(store, NewAgentRoleRegistry()), NewContextPackBuilder(nil, nil, nil), factory, tools)
	beforeCreateRuns := store.createRunCalls
	beforeModelLists := store.modelListCalls

	_, err := runtime.Run(context.Background(), AgentRunRequest{AgentID: agentID, ProjectID: requestProjectID, Input: map[string]any{"brief": "write"}})
	var scopeErr *AgentProjectScopeError
	if !errors.As(err, &scopeErr) {
		t.Fatalf("Run() error = %v, want *AgentProjectScopeError", err)
	}
	if scopeErr.AgentProjectID != wantAgentProjectID || scopeErr.RequestProjectID != wantRequestProjectID {
		t.Fatalf("scope error = %+v, want agent project %q request project %q", scopeErr, wantAgentProjectID, wantRequestProjectID)
	}
	if store.createRunCalls != beforeCreateRuns || store.modelListCalls != beforeModelLists || factory.calls != 0 || tools.calls != 0 || len(client.requests) != 0 {
		t.Fatalf("rejected run caused side effects: create runs %d->%d, model lists %d->%d, factories=%d, tools=%d, requests=%d", beforeCreateRuns, store.createRunCalls, beforeModelLists, store.modelListCalls, factory.calls, tools.calls, len(client.requests))
	}
	runs, listErr := store.ListAgentRuns(repository.AgentRunFilter{AgentID: agentID})
	if listErr != nil {
		t.Fatalf("ListAgentRuns() error: %v", listErr)
	}
	for _, run := range runs {
		if run.ProjectID == strings.TrimSpace(requestProjectID) {
			t.Fatalf("rejected request created run: %+v", run)
		}
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
