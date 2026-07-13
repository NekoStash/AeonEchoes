package agent

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

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

func TestAgentRunStreamEventValidateContracts(t *testing.T) {
	runID := "agent_run_contract"
	running := domain.AgentRun{ID: runID, AgentID: "agent_contract", Status: domain.AgentRunStatusRunning}
	resolution := domain.ModelResolution{ProviderID: "provider_contract", ModelID: "model_contract", ModelName: "model-contract"}
	completedRun := running
	completedRun.Status = domain.AgentRunStatusCompleted
	result := AgentRunResult{Run: completedRun, Content: "complete", ModelResolution: resolution}
	valid := []AgentRunStreamEvent{
		{Type: AgentRunEventStarted, Sequence: 1, RunID: runID, Run: &running},
		{Type: AgentRunEventModelResolved, Sequence: 2, RunID: runID, ModelResolution: &resolution},
		{Type: AgentRunEventToolStarted, Sequence: 3, RunID: runID, Tool: &AgentRunStreamTool{CallID: "call-1", Name: "character.search", Status: "started", Arguments: json.RawMessage(`{"project_id":"p1","query":"林"}`)}},
		{Type: AgentRunEventToolCompleted, Sequence: 4, RunID: runID, Tool: &AgentRunStreamTool{CallID: "call-1", Name: "character.search", Status: "completed", Arguments: json.RawMessage(`{"project_id":"p1","query":"林"}`), Result: json.RawMessage(`{"count":1}`)}},
		{Type: AgentRunEventContentDelta, Sequence: 5, RunID: runID, Delta: "delta"},
		{Type: AgentRunEventContentReset, Sequence: 6, RunID: runID},
		{Type: AgentRunEventCompleted, Sequence: 7, RunID: runID, Result: &result},
		{Type: AgentRunEventFailed, Sequence: 8, RunID: runID, Error: "failed"},
	}

	for _, event := range valid {
		if err := event.Validate(); err != nil || !event.Valid() {
			t.Fatalf("valid event %q rejected: %v", event.Type, err)
		}
	}

	invalid := []AgentRunStreamEvent{
		{Type: AgentRunEventStarted, Sequence: 1, RunID: runID},
		{Type: AgentRunEventModelResolved, Sequence: 1, RunID: runID},
		{Type: AgentRunEventToolStarted, Sequence: 1, RunID: runID, Tool: &AgentRunStreamTool{CallID: "", Name: "character.search", Status: "started"}},
		{Type: AgentRunEventToolCompleted, Sequence: 1, RunID: runID, Tool: &AgentRunStreamTool{CallID: "call-1", Name: "character.search", Status: "started"}},
		{Type: AgentRunEventToolCompleted, Sequence: 1, RunID: runID, Tool: &AgentRunStreamTool{CallID: "call-1", Name: "character.search", Status: "completed"}},
		{Type: AgentRunEventToolStarted, Sequence: 1, RunID: runID, Tool: &AgentRunStreamTool{CallID: "call-1", Name: "character.search", Status: "started", Result: json.RawMessage(`{"too":"early"}`)}},
		{Type: AgentRunEventContentDelta, Sequence: 1, RunID: runID},
		{Type: AgentRunEventCompleted, Sequence: 1, RunID: runID},
		{Type: AgentRunEventFailed, Sequence: 1, RunID: runID},
	}

	for _, event := range invalid {
		if err := event.Validate(); err == nil || event.Valid() {
			t.Fatalf("invalid event %q accepted: %+v", event.Type, event)
		}
	}

	mixed := valid[4]
	mixed.Error = "not allowed"
	if err := mixed.Validate(); err == nil {
		t.Fatal("content.delta with run.failed field was accepted")
	}
	if err := (AgentRunStreamEvent{Type: AgentRunEventFailed, Sequence: 0, RunID: runID, Error: "failed"}).Validate(); err == nil {
		t.Fatal("zero sequence was accepted")
	}
}

type runtimeAnyClientFactory struct {
	client provider.TextModelClient
}

func (f runtimeAnyClientFactory) NewTextClient(domain.ProviderConfig) (provider.TextModelClient, error) {
	if f.client == nil {
		return nil, errors.New("runtime test client is nil")
	}
	return f.client, nil
}

type runtimeStreamingClient struct {
	rounds         [][]provider.StreamEvent
	streamRequests []provider.TextRequest
	generateCalls  int
}

func (c *runtimeStreamingClient) Generate(context.Context, provider.TextRequest) (provider.ModelResponse, error) {
	c.generateCalls++
	return provider.ModelResponse{}, errors.New("Generate must not be used by streaming execution")
}

func (c *runtimeStreamingClient) Stream(ctx context.Context, req provider.TextRequest) (<-chan provider.StreamEvent, error) {
	c.streamRequests = append(c.streamRequests, req)
	if len(c.rounds) == 0 {
		return nil, errors.New("missing streaming round")
	}
	round := c.rounds[0]
	c.rounds = c.rounds[1:]
	events := make(chan provider.StreamEvent, len(round))
	for _, event := range round {
		events <- event
	}
	close(events)
	return events, nil
}

type runtimeControlledStreamClient struct {
	events chan provider.StreamEvent
}

func (c *runtimeControlledStreamClient) Generate(context.Context, provider.TextRequest) (provider.ModelResponse, error) {
	return provider.ModelResponse{}, errors.New("Generate must not be used by streaming execution")
}

func (c *runtimeControlledStreamClient) Stream(context.Context, provider.TextRequest) (<-chan provider.StreamEvent, error) {
	return c.events, nil
}

func TestRuntimeStreamPublishesDeltaBeforeFinalResponse(t *testing.T) {
	store := memory.NewStore()
	providerCfg, err := store.CreateProvider(domain.ProviderConfig{ID: "provider_live_delta", Name: "Live Delta Provider", Type: domain.ProviderOpenAI, Enabled: true})
	if err != nil {
		t.Fatalf("CreateProvider() error: %v", err)
	}
	modelCfg, err := store.CreateModel(domain.ModelConfig{ID: "model_live_delta", ProviderID: providerCfg.ID, Name: "live-delta-model", Kind: domain.ModelKindText, Enabled: true, DefaultForKind: true, SupportsStreaming: true, AllowedAgentRoles: []domain.AgentRole{domain.AgentRoleWriter}})
	if err != nil {
		t.Fatalf("CreateModel() error: %v", err)
	}
	agentCfg, err := store.CreateAgentConfig(domain.AgentConfig{ID: "agent_live_delta", Name: "Live Delta Writer", Role: domain.AgentRoleWriter, Enabled: true})
	if err != nil {
		t.Fatalf("CreateAgentConfig() error: %v", err)
	}
	providerEvents := make(chan provider.StreamEvent)
	client := &runtimeControlledStreamClient{events: providerEvents}
	runtime := NewRuntime(store, NewModelRouter(store, NewAgentRoleRegistry()), nil, runtimeAnyClientFactory{client: client}, nil)
	events, err := runtime.Stream(context.Background(), AgentRunRequest{AgentID: agentCfg.ID, Input: map[string]any{"brief": "write"}})
	if err != nil {
		t.Fatalf("Stream() error: %v", err)
	}
	if event := <-events; event.Type != AgentRunEventStarted {
		t.Fatalf("first event = %+v", event)
	}
	if event := <-events; event.Type != AgentRunEventModelResolved {
		t.Fatalf("second event = %+v", event)
	}

	deltaSent := make(chan struct{})
	go func() {
		providerEvents <- provider.StreamEvent{Type: "content.delta", Delta: "live"}
		close(deltaSent)
	}()
	select {
	case event := <-events:
		if event.Type != AgentRunEventContentDelta || event.Delta != "live" {
			t.Fatalf("live delta event = %+v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("runtime did not publish content.delta before final response")
	}
	<-deltaSent

	final := provider.ModelResponse{ID: "response_live_delta", Model: modelCfg.Name, Content: "live final", FinishReason: "stop"}
	providerEvents <- provider.StreamEvent{Type: "final", Response: &final, Done: true}
	close(providerEvents)
	completed := <-events
	if completed.Type != AgentRunEventCompleted || completed.Result == nil || completed.Result.Content != "live final" {
		t.Fatalf("completed event = %+v", completed)
	}
	if _, ok := <-events; ok {
		t.Fatal("runtime stream remained open after completion")
	}
}

func TestRuntimeStreamCompletesWithOrderedEventsAndPersistence(t *testing.T) {
	store := memory.NewStore()
	providerCfg, err := store.CreateProvider(domain.ProviderConfig{ID: "provider_stream", Name: "Stream Provider", Type: domain.ProviderOpenAI, Enabled: true})
	if err != nil {
		t.Fatalf("CreateProvider() error: %v", err)
	}
	modelCfg, err := store.CreateModel(domain.ModelConfig{ID: "model_stream", ProviderID: providerCfg.ID, Name: "stream-model", Kind: domain.ModelKindText, Enabled: true, DefaultForKind: true, SupportsStreaming: true, AllowedAgentRoles: []domain.AgentRole{domain.AgentRoleWriter}})
	if err != nil {
		t.Fatalf("CreateModel() error: %v", err)
	}
	agentCfg, err := store.CreateAgentConfig(domain.AgentConfig{ID: "agent_stream", Name: "Stream Writer", Role: domain.AgentRoleWriter, Enabled: true})
	if err != nil {
		t.Fatalf("CreateAgentConfig() error: %v", err)
	}
	final := provider.ModelResponse{ID: "response_stream", Model: modelCfg.Name, Content: "生成完成", FinishReason: "stop", Usage: provider.Usage{TotalTokens: 4}}
	client := &runtimeStreamingClient{rounds: [][]provider.StreamEvent{{
		{Type: "content.delta", Delta: "生成"},
		{Type: "content.delta", Delta: "完成"},
		{Type: "final", Response: &final, Done: true},
	}}}
	runtime := NewRuntime(store, NewModelRouter(store, NewAgentRoleRegistry()), nil, runtimeAnyClientFactory{client: client}, nil)

	eventChannel, err := runtime.Stream(context.Background(), AgentRunRequest{AgentID: agentCfg.ID, Input: map[string]any{"brief": "write"}})
	if err != nil {
		t.Fatalf("Stream() error: %v", err)
	}
	events := make([]AgentRunStreamEvent, 0)
	for event := range eventChannel {
		events = append(events, event)
	}
	wantTypes := []string{AgentRunEventStarted, AgentRunEventModelResolved, AgentRunEventContentDelta, AgentRunEventContentDelta, AgentRunEventCompleted}
	if len(events) != len(wantTypes) {
		t.Fatalf("events = %+v", events)
	}
	for i, event := range events {
		if event.Type != wantTypes[i] || event.Sequence != int64(i+1) || event.RunID == "" {
			t.Fatalf("event[%d] = %+v", i, event)
		}
	}
	completed := events[len(events)-1]
	if completed.Result == nil || completed.Result.Content != "生成完成" || completed.Result.Run.Status != domain.AgentRunStatusCompleted {
		t.Fatalf("completed event = %+v", completed)
	}
	loaded, err := store.GetAgentRun(completed.RunID)
	if err != nil {
		t.Fatalf("GetAgentRun() error: %v", err)
	}
	if loaded.Status != domain.AgentRunStatusCompleted || loaded.CompletedAt == nil || loaded.Output["content"] != "生成完成" {
		t.Fatalf("persisted run = %+v", loaded)
	}
	if client.generateCalls != 0 || len(client.streamRequests) != 1 || !client.streamRequests[0].Stream {
		t.Fatalf("client calls: generate=%d streams=%+v", client.generateCalls, client.streamRequests)
	}
}

func TestRuntimeStreamUnsupportedModelFailsWithoutGenerateFallback(t *testing.T) {
	store := memory.NewStore()
	providerCfg, err := store.CreateProvider(domain.ProviderConfig{ID: "provider_no_stream", Name: "No Stream Provider", Type: domain.ProviderOpenAI, Enabled: true})
	if err != nil {
		t.Fatalf("CreateProvider() error: %v", err)
	}
	_, err = store.CreateModel(domain.ModelConfig{ID: "model_no_stream", ProviderID: providerCfg.ID, Name: "no-stream-model", Kind: domain.ModelKindText, Enabled: true, DefaultForKind: true, SupportsStreaming: false, AllowedAgentRoles: []domain.AgentRole{domain.AgentRoleWriter}})
	if err != nil {
		t.Fatalf("CreateModel() error: %v", err)
	}
	agentCfg, err := store.CreateAgentConfig(domain.AgentConfig{ID: "agent_no_stream", Name: "No Stream Writer", Role: domain.AgentRoleWriter, Enabled: true})
	if err != nil {
		t.Fatalf("CreateAgentConfig() error: %v", err)
	}
	client := &runtimeStreamingClient{}
	runtime := NewRuntime(store, NewModelRouter(store, NewAgentRoleRegistry()), nil, runtimeAnyClientFactory{client: client}, nil)
	eventChannel, err := runtime.Stream(context.Background(), AgentRunRequest{AgentID: agentCfg.ID, Input: map[string]any{"brief": "write"}})
	if err != nil {
		t.Fatalf("Stream() error: %v", err)
	}
	var failed AgentRunStreamEvent
	for event := range eventChannel {
		if event.Type == AgentRunEventFailed {
			failed = event
		}
	}
	if failed.Error == "" || !strings.Contains(failed.Error, "does not support streaming") {
		t.Fatalf("failed event = %+v", failed)
	}
	loaded, err := store.GetAgentRun(failed.RunID)
	if err != nil {
		t.Fatalf("GetAgentRun() error: %v", err)
	}
	if loaded.Status != domain.AgentRunStatusFailed || loaded.CompletedAt == nil || loaded.Error == "" {
		t.Fatalf("persisted failed run = %+v", loaded)
	}
	if client.generateCalls != 0 || len(client.streamRequests) != 0 {
		t.Fatalf("unsupported model called client: generate=%d stream=%d", client.generateCalls, len(client.streamRequests))
	}
}

type runtimeBlockingStreamClient struct{}

func (runtimeBlockingStreamClient) Generate(context.Context, provider.TextRequest) (provider.ModelResponse, error) {
	return provider.ModelResponse{}, errors.New("Generate must not be used by streaming execution")
}

func (runtimeBlockingStreamClient) Stream(ctx context.Context, _ provider.TextRequest) (<-chan provider.StreamEvent, error) {
	events := make(chan provider.StreamEvent)
	go func() {
		defer close(events)
		<-ctx.Done()
	}()
	return events, nil
}

func TestRuntimeStreamCancellationPersistsFailedRun(t *testing.T) {
	store := memory.NewStore()
	providerCfg, err := store.CreateProvider(domain.ProviderConfig{ID: "provider_cancel", Name: "Cancel Provider", Type: domain.ProviderOpenAI, Enabled: true})
	if err != nil {
		t.Fatalf("CreateProvider() error: %v", err)
	}
	_, err = store.CreateModel(domain.ModelConfig{ID: "model_cancel", ProviderID: providerCfg.ID, Name: "cancel-model", Kind: domain.ModelKindText, Enabled: true, DefaultForKind: true, SupportsStreaming: true, AllowedAgentRoles: []domain.AgentRole{domain.AgentRoleWriter}})
	if err != nil {
		t.Fatalf("CreateModel() error: %v", err)
	}
	agentCfg, err := store.CreateAgentConfig(domain.AgentConfig{ID: "agent_cancel", Name: "Cancel Writer", Role: domain.AgentRoleWriter, Enabled: true})
	if err != nil {
		t.Fatalf("CreateAgentConfig() error: %v", err)
	}
	runtime := NewRuntime(store, NewModelRouter(store, NewAgentRoleRegistry()), nil, runtimeAnyClientFactory{client: runtimeBlockingStreamClient{}}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	eventChannel, err := runtime.Stream(ctx, AgentRunRequest{AgentID: agentCfg.ID, Input: map[string]any{"brief": "write"}})
	if err != nil {
		t.Fatalf("Stream() error: %v", err)
	}
	runID := ""
	for event := range eventChannel {
		runID = event.RunID
		if event.Type == AgentRunEventModelResolved {
			cancel()
		}
	}
	if runID == "" {
		t.Fatal("stream emitted no run id")
	}
	loaded, err := store.GetAgentRun(runID)
	if err != nil {
		t.Fatalf("GetAgentRun() error: %v", err)
	}
	if loaded.Status != domain.AgentRunStatusFailed || loaded.CompletedAt == nil || !strings.Contains(loaded.Error, "context canceled") {
		t.Fatalf("cancelled run = %+v", loaded)
	}
}

type runtimeFixedToolCatalog struct {
	tools []provider.ToolSpec
}

func (c runtimeFixedToolCatalog) ListProviderTools(context.Context, domain.AgentConfig) ([]provider.ToolSpec, error) {
	return append([]provider.ToolSpec(nil), c.tools...), nil
}

func TestRuntimeStreamToolRoundBufferLimitFailsPersistedRun(t *testing.T) {
	store := memory.NewStore()
	providerCfg, err := store.CreateProvider(domain.ProviderConfig{ID: "provider_tool_buffer", Name: "Tool Buffer Provider", Type: domain.ProviderOpenAI, Enabled: true})
	if err != nil {
		t.Fatalf("CreateProvider() error: %v", err)
	}
	_, err = store.CreateModel(domain.ModelConfig{ID: "model_tool_buffer", ProviderID: providerCfg.ID, Name: "tool-buffer-model", Kind: domain.ModelKindText, Enabled: true, DefaultForKind: true, SupportsStreaming: true, SupportsTools: true, AllowedAgentRoles: []domain.AgentRole{domain.AgentRoleWriter}})
	if err != nil {
		t.Fatalf("CreateModel() error: %v", err)
	}
	agentCfg, err := store.CreateAgentConfig(domain.AgentConfig{ID: "agent_tool_buffer", Name: "Tool Buffer Writer", Role: domain.AgentRoleWriter, Enabled: true})
	if err != nil {
		t.Fatalf("CreateAgentConfig() error: %v", err)
	}
	oversized := strings.Repeat("x", maxToolRoundBufferedDeltaBytes+1)
	client := &runtimeStreamingClient{rounds: [][]provider.StreamEvent{{{Type: "content.delta", Delta: oversized}}}}
	runtime := NewRuntime(store, NewModelRouter(store, NewAgentRoleRegistry()), nil, runtimeAnyClientFactory{client: client}, runtimeFixedToolCatalog{tools: []provider.ToolSpec{NarrativeToolSpecs()[0]}})
	eventChannel, err := runtime.Stream(context.Background(), AgentRunRequest{AgentID: agentCfg.ID, Input: map[string]any{"brief": "write"}})
	if err != nil {
		t.Fatalf("Stream() error: %v", err)
	}
	var failed AgentRunStreamEvent
	for event := range eventChannel {
		if event.Type == AgentRunEventFailed {
			failed = event
		}
	}
	if !strings.Contains(failed.Error, "byte limit") {
		t.Fatalf("failed event = %+v", failed)
	}
	loaded, err := store.GetAgentRun(failed.RunID)
	if err != nil {
		t.Fatalf("GetAgentRun() error: %v", err)
	}
	if loaded.Status != domain.AgentRunStatusFailed || loaded.CompletedAt == nil || !strings.Contains(loaded.Error, "byte limit") {
		t.Fatalf("persisted failed run = %+v", loaded)
	}
}

func TestLiveContentDeltaSinkByteLimit(t *testing.T) {
	sink := &liveContentDeltaSink{emit: func(string) error { return nil }}
	oversized := strings.Repeat("x", maxToolRoundBufferedDeltaBytes+1)
	if err := sink.OnContentDelta(oversized); err == nil || !strings.Contains(err.Error(), "byte limit") {
		t.Fatalf("byte overflow error = %v", err)
	}
}

func TestRunToolLoopStreamForwardsLiveTextAndResetsBeforeToolRound(t *testing.T) {
	store := memory.NewStore()
	first := provider.ModelResponse{Content: "不要泄漏", FinishReason: "tool_calls", ToolCalls: []provider.ToolCall{{ID: "call_1", Name: "character.search", Arguments: json.RawMessage(`{"project_id":"project-stream","query":"林"}`)}}}
	second := provider.ModelResponse{Content: "最终提案", FinishReason: "stop"}
	client := &runtimeStreamingClient{rounds: [][]provider.StreamEvent{
		{{Type: "content.delta", Delta: "不要"}, {Type: "content.delta", Delta: "泄漏"}, {Type: "final", Response: &first, Done: true}},
		{{Type: "content.delta", Delta: "最终"}, {Type: "content.delta", Delta: "提案"}, {Type: "final", Response: &second, Done: true}},
	}}
	var content string
	var lifecycle []string
	result, err := RunToolLoopStream(context.Background(), client, provider.TextRequest{Model: "stream-model", UserPrompt: "write", Tools: NarrativeToolSpecs()}, NewToolExecutor(store), 4, ToolLoopStreamHooks{
		OnContentDelta: func(delta string) error {
			content += delta
			lifecycle = append(lifecycle, "delta:"+delta)
			return nil
		},
		OnContentReset: func() error {
			content = ""
			lifecycle = append(lifecycle, "reset")
			return nil
		},
		OnToolStarted: func(record ToolExecutionRecord) error {
			lifecycle = append(lifecycle, "started:"+record.Name)
			return nil
		},
		OnToolCompleted: func(record ToolExecutionRecord) error {
			if len(record.Result) == 0 {
				return errors.New("completed tool result must not be empty")
			}
			lifecycle = append(lifecycle, "completed:"+record.Name)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("RunToolLoopStream() error: %v", err)
	}
	if content != "最终提案" || result.Response.Content != "最终提案" {
		t.Fatalf("content=%q result=%+v", content, result)
	}
	wantLifecycle := []string{"delta:不要", "delta:泄漏", "reset", "started:character.search", "completed:character.search", "delta:最终", "delta:提案"}
	if len(lifecycle) != len(wantLifecycle) {
		t.Fatalf("lifecycle = %+v", lifecycle)
	}
	for index, item := range wantLifecycle {
		if lifecycle[index] != item {
			t.Fatalf("lifecycle = %+v", lifecycle)
		}
	}
	if client.generateCalls != 0 || len(client.streamRequests) != 2 {
		t.Fatalf("client calls: generate=%d streams=%d", client.generateCalls, len(client.streamRequests))
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
