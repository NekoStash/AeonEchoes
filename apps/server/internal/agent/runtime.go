package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/provider"
	"aeonechoes/server/internal/repository"
)

// AgentRunRequest is the provider-facing execution request for one configured agent.
type AgentRunRequest struct {
	AgentID          string            `json:"agent_id"`
	ProjectID        string            `json:"project_id,omitempty"`
	TaskType         string            `json:"task_type,omitempty"`
	Input            map[string]any    `json:"input,omitempty"`
	ContextSelection *ContextSelection `json:"context_selection,omitempty"`
	MaxOutputTokens  int               `json:"max_output_tokens,omitempty"`
}

// AgentRunResult captures persisted run state plus provider output and routing details.
type AgentRunResult struct {
	Run             domain.AgentRun        `json:"run"`
	Content         string                 `json:"content"`
	ToolTrace       []string               `json:"tool_trace,omitempty"`
	ModelResolution domain.ModelResolution `json:"model_resolution"`
}

const (
	AgentRunEventStarted       = "run.started"
	AgentRunEventModelResolved = "model.resolved"
	AgentRunEventToolStarted   = "tool.started"
	AgentRunEventToolCompleted = "tool.completed"
	AgentRunEventContentDelta  = "content.delta"
	AgentRunEventContentReset  = "content.reset"
	AgentRunEventCompleted     = "run.completed"
	AgentRunEventFailed        = "run.failed"
)

// AgentRunStreamTool describes one tool lifecycle event after its arguments are complete.
// Arguments are included from tool.started; Result is included on tool.completed.
type AgentRunStreamTool struct {
	CallID    string          `json:"call_id"`
	Name      string          `json:"name"`
	Status    string          `json:"status"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
}


// AgentRunStreamEvent is the stable SSE data object shared with API clients.
type AgentRunStreamEvent struct {
	Type            string                  `json:"type"`
	Sequence        int64                   `json:"sequence"`
	RunID           string                  `json:"run_id"`
	Delta           string                  `json:"delta,omitempty"`
	Run             *domain.AgentRun        `json:"run,omitempty"`
	Result          *AgentRunResult         `json:"result,omitempty"`
	ModelResolution *domain.ModelResolution `json:"model_resolution,omitempty"`
	Tool            *AgentRunStreamTool     `json:"tool,omitempty"`
	Error           string                  `json:"error,omitempty"`
}

// Valid reports whether the event satisfies the type-specific public SSE contract.
func (e AgentRunStreamEvent) Valid() bool {
	return e.Validate() == nil
}

// Validate enforces common identity fields, the required payload for each event
// type, and rejects payload fields belonging to another event type.
func (e AgentRunStreamEvent) Validate() error {
	if e.Sequence < 1 {
		return fmt.Errorf("agent run stream event sequence must be positive")
	}
	if strings.TrimSpace(e.RunID) == "" {
		return fmt.Errorf("agent run stream event run_id must not be empty")
	}
	switch e.Type {
	case AgentRunEventStarted:
		if e.Run == nil || e.Run.ID != e.RunID || e.Run.Status != domain.AgentRunStatusRunning {
			return fmt.Errorf("run.started requires the matching running run")
		}
		if e.Delta != "" || e.Result != nil || e.ModelResolution != nil || e.Tool != nil || e.Error != "" {
			return fmt.Errorf("run.started contains fields reserved for another event type")
		}
	case AgentRunEventModelResolved:
		if e.ModelResolution == nil || strings.TrimSpace(e.ModelResolution.ModelID) == "" || strings.TrimSpace(e.ModelResolution.ProviderID) == "" {
			return fmt.Errorf("model.resolved requires model_resolution with model_id and provider_id")
		}
		if e.Delta != "" || e.Run != nil || e.Result != nil || e.Tool != nil || e.Error != "" {
			return fmt.Errorf("model.resolved contains fields reserved for another event type")
		}
	case AgentRunEventToolStarted, AgentRunEventToolCompleted:
		if e.Tool == nil {
			return fmt.Errorf("%s requires tool", e.Type)
		}
		if err := e.Tool.validateForEvent(e.Type); err != nil {
			return err
		}
		if e.Delta != "" || e.Run != nil || e.Result != nil || e.ModelResolution != nil || e.Error != "" {
			return fmt.Errorf("%s contains fields reserved for another event type", e.Type)
		}
	case AgentRunEventContentDelta:
		if e.Delta == "" {
			return fmt.Errorf("content.delta requires a non-empty delta")
		}
		if e.Run != nil || e.Result != nil || e.ModelResolution != nil || e.Tool != nil || e.Error != "" {
			return fmt.Errorf("content.delta contains fields reserved for another event type")
		}
	case AgentRunEventContentReset:
		if e.Delta != "" || e.Run != nil || e.Result != nil || e.ModelResolution != nil || e.Tool != nil || e.Error != "" {
			return fmt.Errorf("content.reset contains fields reserved for another event type")
		}
	case AgentRunEventCompleted:
		if e.Result == nil || e.Result.Run.ID != e.RunID || e.Result.Run.Status != domain.AgentRunStatusCompleted || strings.TrimSpace(e.Result.Content) == "" || strings.TrimSpace(e.Result.ModelResolution.ModelID) == "" {
			return fmt.Errorf("run.completed requires a complete matching AgentRunResult")
		}
		if e.Delta != "" || e.Run != nil || e.ModelResolution != nil || e.Tool != nil || e.Error != "" {
			return fmt.Errorf("run.completed contains fields reserved for another event type")
		}
	case AgentRunEventFailed:
		if strings.TrimSpace(e.Error) == "" {
			return fmt.Errorf("run.failed requires a non-empty error")
		}
		if e.Delta != "" || e.Run != nil || e.Result != nil || e.ModelResolution != nil || e.Tool != nil {
			return fmt.Errorf("run.failed contains fields reserved for another event type")
		}
	default:
		return fmt.Errorf("agent run stream event type %q is invalid", e.Type)
	}
	return nil
}

func (t AgentRunStreamTool) validateForEvent(eventType string) error {
	if strings.TrimSpace(t.CallID) == "" || strings.TrimSpace(t.Name) == "" {
		return fmt.Errorf("%s tool requires call_id and name", eventType)
	}
	expectedStatus := "started"
	if eventType == AgentRunEventToolCompleted {
		expectedStatus = "completed"
	}
	if t.Status != expectedStatus {
		return fmt.Errorf("%s tool status must be %q", eventType, expectedStatus)
	}
	if len(t.Arguments) > 0 && !json.Valid(t.Arguments) {
		return fmt.Errorf("%s tool arguments must be valid JSON", eventType)
	}
	if eventType == AgentRunEventToolCompleted {
		if len(t.Result) == 0 {
			return fmt.Errorf("%s tool requires result", eventType)
		}
		if !json.Valid(t.Result) {
			return fmt.Errorf("%s tool result must be valid JSON", eventType)
		}
	} else if len(t.Result) > 0 {
		return fmt.Errorf("%s tool must not include result", eventType)
	}
	return nil
}


type runtimeExecution struct {
	run       domain.AgentRun
	config    domain.AgentConfig
	request   AgentRunRequest
	projectID string
}

type preparedRuntimeExecution struct {
	runtimeExecution
	client            provider.TextModelClient
	textRequest       provider.TextRequest
	modelResolution   domain.ModelResolution
	supportsStreaming bool
}

// AgentProjectScopeError reports an invalid project scope requested for an agent run.
type AgentProjectScopeError struct {
	AgentID          string
	AgentProjectID   string
	RequestProjectID string
}

func (e *AgentProjectScopeError) Error() string {
	if e == nil {
		return "agent run project scope is invalid"
	}
	if e.RequestProjectID == "" {
		return fmt.Sprintf("project-scoped agent %q requires a non-empty project_id matching %q", e.AgentID, e.AgentProjectID)
	}
	return fmt.Sprintf("project-scoped agent %q belongs to project %q and cannot run for project %q", e.AgentID, e.AgentProjectID, e.RequestProjectID)
}

// RuntimeStore is the repository surface required by AgentRuntime orchestration.
type RuntimeStore interface {
	NewID(prefix string) (string, error)
	GetAgentConfig(id string) (domain.AgentConfig, error)
	ListAgentConfigs(filter repository.AgentConfigFilter) ([]domain.AgentConfig, error)
	CreateAgentRun(run domain.AgentRun) (domain.AgentRun, error)
	UpdateAgentRun(id string, run domain.AgentRun) (domain.AgentRun, error)
	GetAgentRun(id string) (domain.AgentRun, error)
	ListAgentRuns(filter repository.AgentRunFilter) ([]domain.AgentRun, error)
	GetSkill(id string) (domain.Skill, error)
	ListSkills(filter repository.SkillFilter) ([]domain.Skill, error)
	ListModelsByKind(kind domain.ModelKind) ([]domain.ModelConfig, error)
	GetProvider(id string) (domain.ProviderConfig, error)
}

// ToolCatalog avoids an import cycle between agent runtime and concrete tooling registry.
type ToolCatalog interface {
	ListProviderTools(ctx context.Context, cfg domain.AgentConfig) ([]provider.ToolSpec, error)
}

// Runtime coordinates agent execution against configured models, skills and tools.
type Runtime struct {
	store          RuntimeStore
	router         *ModelRouter
	builder        *ContextPackBuilder
	clients        TextClientFactory
	tools          ToolCatalog
	toolExec       ToolStore
	chapterAuditor ChapterAuditor
	rulesAuditor   ContinuityAuditor
}

func NewRuntime(store RuntimeStore, router *ModelRouter, builder *ContextPackBuilder, providers TextClientFactory, tools ToolCatalog) *Runtime {
	var toolExec ToolStore
	if candidate, ok := store.(ToolStore); ok {
		toolExec = candidate
	}
	runtime := &Runtime{store: store, router: router, builder: builder, clients: providers, tools: tools, toolExec: toolExec, rulesAuditor: NewRuleBasedContinuityAuditor()}
	if router != nil && providers != nil {
		runtime.chapterAuditor = NewLLMChapterAuditor(router, providers, builder, runtime.rulesAuditor, toolExec)
	}
	return runtime
}

func (r *Runtime) Run(ctx context.Context, req AgentRunRequest) (AgentRunResult, error) {
	execution, err := r.startExecution(req)
	if err != nil {
		return AgentRunResult{}, err
	}
	prepared, err := r.prepareExecution(ctx, execution)
	if err != nil {
		return r.failExecution(execution.run, AgentRunResult{Run: execution.run}, err)
	}
	result, err := r.executePrepared(ctx, prepared, false, nil)
	if err != nil {
		return r.failExecution(execution.run, result, err)
	}
	return result, nil
}

// Stream starts one persisted run and returns ordered business events. Once the
// channel is returned, every execution error is represented by run.failed.
func (r *Runtime) Stream(ctx context.Context, req AgentRunRequest) (<-chan AgentRunStreamEvent, error) {
	execution, err := r.startExecution(req)
	if err != nil {
		return nil, err
	}
	events := make(chan AgentRunStreamEvent, 8)
	go r.streamExecution(ctx, execution, events)
	return events, nil
}

func (r *Runtime) startExecution(req AgentRunRequest) (runtimeExecution, error) {
	if r == nil || r.store == nil || r.router == nil || r.clients == nil {
		return runtimeExecution{}, fmt.Errorf("agent runtime is not fully configured")
	}
	req.AgentID = strings.TrimSpace(req.AgentID)
	if req.AgentID == "" {
		return runtimeExecution{}, fmt.Errorf("agent run agent_id must not be empty")
	}
	cfg, err := r.store.GetAgentConfig(req.AgentID)
	if err != nil {
		return runtimeExecution{}, err
	}
	req.ProjectID = strings.TrimSpace(req.ProjectID)
	cfg.ProjectID = strings.TrimSpace(cfg.ProjectID)
	if !cfg.Enabled {
		return runtimeExecution{}, fmt.Errorf("agent %q is disabled", cfg.ID)
	}
	projectID, err := validateAgentProjectScope(cfg, req.ProjectID)
	if err != nil {
		return runtimeExecution{}, err
	}
	run, err := r.createRunningRun(req, projectID)
	if err != nil {
		return runtimeExecution{}, err
	}
	return runtimeExecution{run: run, config: cfg, request: req, projectID: projectID}, nil
}

func validateAgentProjectScope(cfg domain.AgentConfig, requestProjectID string) (string, error) {
	requestProjectID = strings.TrimSpace(requestProjectID)
	agentProjectID := strings.TrimSpace(cfg.ProjectID)
	if agentProjectID == "" {
		return requestProjectID, nil
	}
	if requestProjectID == "" || requestProjectID != agentProjectID {
		return "", &AgentProjectScopeError{AgentID: cfg.ID, AgentProjectID: agentProjectID, RequestProjectID: requestProjectID}
	}
	return requestProjectID, nil
}

func (r *Runtime) prepareExecution(ctx context.Context, execution runtimeExecution) (preparedRuntimeExecution, error) {
	skills, err := r.enabledAgentSkills(execution.config, execution.projectID)
	if err != nil {
		return preparedRuntimeExecution{}, err
	}
	role := execution.config.Role
	if role == "" {
		role = domain.AgentRoleWriter
	}
	selection, err := r.selectModel(execution.config, role)
	if err != nil {
		return preparedRuntimeExecution{}, err
	}
	modelResolution := buildModelResolution(selection)
	client, err := r.clients.NewTextClient(selection.Provider)
	if err != nil {
		return preparedRuntimeExecution{runtimeExecution: execution, modelResolution: modelResolution}, err
	}
	contextPack, err := r.buildContextPack(execution.projectID, role, execution.request)
	if err != nil {
		return preparedRuntimeExecution{runtimeExecution: execution, modelResolution: modelResolution}, err
	}
	textReq, err := r.buildTextRequest(execution.config, role, execution.request, selection, skills, contextPack)
	if err != nil {
		return preparedRuntimeExecution{runtimeExecution: execution, modelResolution: modelResolution}, err
	}
	if r.tools != nil {
		tools, err := r.tools.ListProviderTools(ctx, execution.config)
		if err != nil {
			return preparedRuntimeExecution{runtimeExecution: execution, modelResolution: modelResolution}, err
		}
		if len(tools) > 0 && selection.Model.SupportsTools {
			textReq.Tools = tools
		}
	}
	return preparedRuntimeExecution{runtimeExecution: execution, client: client, textRequest: textReq, modelResolution: modelResolution, supportsStreaming: selection.Model.SupportsStreaming}, nil
}

func (r *Runtime) executePrepared(ctx context.Context, prepared preparedRuntimeExecution, streaming bool, hooks *ToolLoopStreamHooks) (AgentRunResult, error) {
	var content string
	var toolTrace []string
	if len(prepared.textRequest.Tools) > 0 {
		if r.toolExec == nil {
			return AgentRunResult{Run: prepared.run, ModelResolution: prepared.modelResolution}, fmt.Errorf("agent runtime tool executor store is not configured")
		}
		executor, err := r.newToolExecutor(prepared.config)
		if err != nil {
			return AgentRunResult{Run: prepared.run, ModelResolution: prepared.modelResolution}, err
		}
		var loopResult ToolLoopResult
		if streaming {
			if hooks == nil {
				return AgentRunResult{Run: prepared.run, ModelResolution: prepared.modelResolution}, fmt.Errorf("agent runtime streaming hooks are not configured")
			}
			loopResult, err = RunToolLoopStream(ctx, prepared.client, prepared.textRequest, executor, defaultToolLoopMaxRounds, *hooks)
		} else {
			loopResult, err = RunToolLoop(ctx, prepared.client, prepared.textRequest, executor, defaultToolLoopMaxRounds)
		}
		if err != nil {
			return AgentRunResult{Run: prepared.run, ModelResolution: prepared.modelResolution}, err
		}
		content = strings.TrimSpace(loopResult.Response.Content)
		toolTrace = loopResult.Trace
	} else if streaming {
		prepared.textRequest.Tools = nil
		prepared.textRequest.Stream = true
		if hooks == nil || hooks.OnContentDelta == nil {
			return AgentRunResult{Run: prepared.run, ModelResolution: prepared.modelResolution}, fmt.Errorf("agent runtime content delta hook is not configured")
		}
		resp, err := consumeTextStream(ctx, prepared.client, prepared.textRequest, contentDeltaSinkFunc(hooks.OnContentDelta))
		if err != nil {
			return AgentRunResult{Run: prepared.run, ModelResolution: prepared.modelResolution}, err
		}
		content = strings.TrimSpace(resp.Content)
	} else {
		prepared.textRequest.Tools = nil
		resp, err := prepared.client.Generate(ctx, prepared.textRequest)
		if err != nil {
			return AgentRunResult{Run: prepared.run, ModelResolution: prepared.modelResolution}, err
		}
		content = strings.TrimSpace(resp.Content)
	}
	if content == "" {
		return AgentRunResult{Run: prepared.run, ModelResolution: prepared.modelResolution}, fmt.Errorf("agent runtime model returned empty content")
	}
	completed, err := r.completeRun(prepared.run, content, prepared.modelResolution, toolTrace)
	if err != nil {
		return AgentRunResult{Run: prepared.run, Content: content, ToolTrace: toolTrace, ModelResolution: prepared.modelResolution}, err
	}
	return AgentRunResult{Run: completed, Content: content, ToolTrace: toolTrace, ModelResolution: prepared.modelResolution}, nil
}

func (r *Runtime) streamExecution(ctx context.Context, execution runtimeExecution, events chan<- AgentRunStreamEvent) {
	defer close(events)
	sequence := int64(0)
	emit := func(event AgentRunStreamEvent) error {
		event.Sequence = sequence + 1
		event.RunID = execution.run.ID
		if err := event.Validate(); err != nil {
			return fmt.Errorf("validate agent run stream event %q: %w", event.Type, err)
		}
		sequence = event.Sequence
		select {
		case <-ctx.Done():
			return ctx.Err()
		case events <- event:
			return nil
		}
	}
	if err := emit(AgentRunStreamEvent{Type: AgentRunEventStarted, Run: &execution.run}); err != nil {
		r.failStreamingExecution(execution.run, AgentRunResult{Run: execution.run}, err, emit)
		return
	}
	prepared, err := r.prepareExecution(ctx, execution)
	if err != nil {
		r.failStreamingExecution(execution.run, AgentRunResult{Run: execution.run}, err, emit)
		return
	}
	resolution := prepared.modelResolution
	if err := emit(AgentRunStreamEvent{Type: AgentRunEventModelResolved, ModelResolution: &resolution}); err != nil {
		r.failStreamingExecution(execution.run, AgentRunResult{Run: execution.run, ModelResolution: resolution}, err, emit)
		return
	}
	if !prepared.supportsStreaming {
		r.failStreamingExecution(execution.run, AgentRunResult{Run: execution.run, ModelResolution: resolution}, fmt.Errorf("selected model %q does not support streaming", resolution.ModelID), emit)
		return
	}
	if err := ctx.Err(); err != nil {
		r.failStreamingExecution(execution.run, AgentRunResult{Run: execution.run, ModelResolution: resolution}, err, emit)
		return
	}
	hooks := ToolLoopStreamHooks{
		OnContentDelta: func(delta string) error {
			return emit(AgentRunStreamEvent{Type: AgentRunEventContentDelta, Delta: delta})
		},
		OnContentReset: func() error {
			return emit(AgentRunStreamEvent{Type: AgentRunEventContentReset})
		},
		OnToolStarted: func(record ToolExecutionRecord) error {
			tool, err := streamTool(record, "started")
			if err != nil {
				return err
			}
			return emit(AgentRunStreamEvent{Type: AgentRunEventToolStarted, Tool: &tool})
		},
		OnToolCompleted: func(record ToolExecutionRecord) error {
			tool, err := streamTool(record, "completed")
			if err != nil {
				return err
			}
			return emit(AgentRunStreamEvent{Type: AgentRunEventToolCompleted, Tool: &tool})
		},
	}
	result, err := r.executePrepared(ctx, prepared, true, &hooks)
	if err != nil {
		r.failStreamingExecution(execution.run, result, err, emit)
		return
	}
	if err := emit(AgentRunStreamEvent{Type: AgentRunEventCompleted, Result: &result}); err != nil && ctx.Err() == nil {
		r.failStreamingExecution(execution.run, result, err, emit)
	}
}

func (r *Runtime) newToolExecutor(cfg domain.AgentConfig) (*ToolExecutor, error) {
	maxRounds, err := ParseAuditMaxRounds(cfg.RuntimeOptions)
	if err != nil {
		return nil, err
	}
	return NewToolExecutor(r.toolExec, ToolExecutorOptions{
		ChapterAuditor: r.chapterAuditor,
		RulesAuditor:   r.rulesAuditor,
		AuditLimiter:   NewAuditCallLimiter(maxRounds),
	}), nil
}

func (r *Runtime) failExecution(run domain.AgentRun, result AgentRunResult, cause error) (AgentRunResult, error) {
	failed, failErr := r.failRun(run, cause)
	if failErr != nil {
		return result, fmt.Errorf("%w; update failed agent run: %v", cause, failErr)
	}
	result.Run = failed
	return result, cause
}

func (r *Runtime) failStreamingExecution(run domain.AgentRun, result AgentRunResult, cause error, emit func(AgentRunStreamEvent) error) {
	failed, failErr := r.failRun(run, cause)
	if failErr != nil {
		cause = fmt.Errorf("%w; update failed agent run: %v", cause, failErr)
	} else {
		result.Run = failed
	}
	_ = emit(AgentRunStreamEvent{Type: AgentRunEventFailed, Error: cause.Error()})
}

func streamTool(record ToolExecutionRecord, status string) (AgentRunStreamTool, error) {
	tool := AgentRunStreamTool{CallID: strings.TrimSpace(record.CallID), Name: strings.TrimSpace(record.Name), Status: status}
	if tool.CallID == "" || tool.Name == "" {
		return AgentRunStreamTool{}, fmt.Errorf("stream tool call_id and name must not be empty")
	}
	if len(record.Arguments) > 0 {
		if !json.Valid(record.Arguments) {
			return AgentRunStreamTool{}, fmt.Errorf("stream tool %q arguments must be valid JSON", tool.Name)
		}
		tool.Arguments = append(json.RawMessage(nil), record.Arguments...)
	}
	if status == "completed" {
		if len(record.Result) == 0 {
			return AgentRunStreamTool{}, fmt.Errorf("stream tool %q completed without result payload", tool.Name)
		}
		if !json.Valid(record.Result) {
			return AgentRunStreamTool{}, fmt.Errorf("stream tool %q result must be valid JSON", tool.Name)
		}
		tool.Result = append(json.RawMessage(nil), record.Result...)
	}
	return tool, nil
}


func (r *Runtime) createRunningRun(req AgentRunRequest, projectID string) (domain.AgentRun, error) {
	input := copyAnyMapRuntime(req.Input)
	if input == nil {
		input = map[string]any{}
	}
	if strings.TrimSpace(req.TaskType) != "" {
		input["task_type"] = strings.TrimSpace(req.TaskType)
	}
	if req.ContextSelection != nil {
		selectionBytes, err := json.Marshal(req.ContextSelection)
		if err != nil {
			return domain.AgentRun{}, fmt.Errorf("marshal context selection: %w", err)
		}
		var selection map[string]any
		if err := json.Unmarshal(selectionBytes, &selection); err != nil {
			return domain.AgentRun{}, fmt.Errorf("encode context selection: %w", err)
		}
		input["context_selection"] = selection
	}
	n := time.Now().UTC()
	return r.store.CreateAgentRun(domain.AgentRun{AgentID: req.AgentID, ProjectID: projectID, Status: domain.AgentRunStatusRunning, Input: input, StartedAt: &n})
}

func (r *Runtime) failRun(run domain.AgentRun, cause error) (domain.AgentRun, error) {
	if cause == nil {
		cause = fmt.Errorf("agent run failed without cause")
	}
	run.Status = domain.AgentRunStatusFailed
	run.Error = cause.Error()
	return r.store.UpdateAgentRun(run.ID, run)
}

func (r *Runtime) completeRun(run domain.AgentRun, content string, resolution domain.ModelResolution, toolTrace []string) (domain.AgentRun, error) {
	run.Status = domain.AgentRunStatusCompleted
	run.Error = ""
	run.Output = map[string]any{
		"content":          content,
		"model":            resolution.ModelName,
		"model_resolution": resolutionOutput(resolution),
	}
	if len(toolTrace) > 0 {
		run.Output["tool_trace"] = append([]string{}, toolTrace...)
	}
	return r.store.UpdateAgentRun(run.ID, run)
}

func (r *Runtime) enabledAgentSkills(cfg domain.AgentConfig, projectID string) ([]domain.Skill, error) {
	allowed := stringSetRuntime(cfg.SkillIDs)
	if len(allowed) == 0 {
		return nil, nil
	}
	listed, err := r.store.ListSkills(repository.SkillFilter{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("list skills for agent %q: %w", cfg.ID, err)
	}
	byID := make(map[string]domain.Skill, len(listed))
	for _, skill := range listed {
		byID[skill.ID] = skill
	}
	result := make([]domain.Skill, 0, len(cfg.SkillIDs))
	for _, skillID := range cfg.SkillIDs {
		if strings.TrimSpace(skillID) == "" {
			continue
		}
		skill, ok := byID[skillID]
		if !ok {
			loaded, err := r.store.GetSkill(skillID)
			if err != nil {
				return nil, err
			}
			skill = loaded
		}
		if !skill.Enabled {
			continue
		}
		result = append(result, skill)
	}
	return result, nil
}

func (r *Runtime) selectModel(cfg domain.AgentConfig, role domain.AgentRole) (ModelSelection, error) {
	if strings.TrimSpace(cfg.ModelID) != "" {
		return r.selectExplicitTextModel(strings.TrimSpace(cfg.ModelID), role)
	}
	return r.router.SelectTextModel(role)
}

func (r *Runtime) selectExplicitTextModel(modelRef string, role domain.AgentRole) (ModelSelection, error) {
	models, err := r.store.ListModelsByKind(domain.ModelKindText)
	if err != nil {
		return ModelSelection{}, fmt.Errorf("list text models for explicit model %q: %w", modelRef, err)
	}
	for _, model := range models {
		if model.ID != modelRef && model.Name != modelRef {
			continue
		}
		providerCfg, err := r.store.GetProvider(model.ProviderID)
		if err != nil {
			return ModelSelection{}, err
		}
		if !providerCfg.Enabled {
			return ModelSelection{}, fmt.Errorf("explicit model %q belongs to disabled provider %q", modelRef, providerCfg.ID)
		}
		return ModelSelection{Model: model, Provider: providerCfg, RouteKey: string(role), ResolutionSource: "agent_config_model"}, nil
	}
	return ModelSelection{}, fmt.Errorf("agent explicit model %q was not found among enabled text models by id or name", modelRef)
}

func (r *Runtime) buildContextPack(projectID string, role domain.AgentRole, req AgentRunRequest) (domain.ContextPack, error) {
	if strings.TrimSpace(projectID) == "" || r.builder == nil {
		return domain.ContextPack{}, nil
	}
	brief := inputBrief(req)
	if brief == "" {
		brief = strings.TrimSpace(req.TaskType)
	}
	chapterID := inputString(req.Input, "chapter_id")
	pack, err := r.builder.BuildWithSelection(projectID, chapterID, role, brief, 4000, req.ContextSelection, nil)
	if err != nil {
		return domain.ContextPack{}, err
	}
	return pack, nil
}

func (r *Runtime) buildTextRequest(cfg domain.AgentConfig, role domain.AgentRole, req AgentRunRequest, selection ModelSelection, skills []domain.Skill, pack domain.ContextPack) (provider.TextRequest, error) {
	systemPrompt := strings.TrimSpace(cfg.SystemPrompt)
	if systemPrompt == "" {
		systemPrompt = defaultRuntimeSystemPrompt(role)
	}
	if skillPrompt := skillsSystemPrompt(skills); skillPrompt != "" {
		systemPrompt = joinNonEmpty([]string{systemPrompt, skillPrompt})
	}
	userPrompt, err := runtimeUserPrompt(req, pack)
	if err != nil {
		return provider.TextRequest{}, err
	}
	return provider.TextRequest{
		Model:           selection.Model.Name,
		SystemPrompt:    systemPrompt,
		UserPrompt:      userPrompt,
		MaxOutputTokens: firstPositive(req.MaxOutputTokens, selection.Model.MaxOutputTokens, 1200),
		Temperature:     0.7,
		Metadata: map[string]string{
			"agent_id":  cfg.ID,
			"task_type": strings.TrimSpace(req.TaskType),
			"role":      string(role),
		},
	}, nil
}

func runtimeUserPrompt(req AgentRunRequest, pack domain.ContextPack) (string, error) {
	payload := map[string]any{}
	if strings.TrimSpace(req.TaskType) != "" {
		payload["task_type"] = strings.TrimSpace(req.TaskType)
	}
	if len(req.Input) > 0 {
		payload["input"] = req.Input
	}
	if pack.ID != "" {
		payload["context_pack"] = pack
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal agent runtime prompt payload: %w", err)
	}
	return "请根据以下 AgentRun JSON 输入完成任务，只输出最终结果内容，不要解释运行过程。\n" + string(payloadBytes), nil
}

func inputBrief(req AgentRunRequest) string {
	return joinNonEmpty([]string{
		inputString(req.Input, "brief"),
		inputString(req.Input, "prompt"),
		inputString(req.Input, "instruction"),
		inputString(req.Input, "query"),
	})
}

func inputString(input map[string]any, key string) string {
	if len(input) == 0 {
		return ""
	}
	value, ok := input[key]
	if !ok || value == nil {
		return ""
	}
	text, ok := value.(string)
	if ok {
		return strings.TrimSpace(text)
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func defaultRuntimeSystemPrompt(role domain.AgentRole) string {
	return fmt.Sprintf("你是 AeonEchoes 后端 AgentRuntime 中的 %s Agent。必须遵循用户输入和提供的 ContextPack；如果上下文不足，应明确保持克制，不编造破坏连续性的事实。", role)
}

func skillsSystemPrompt(skills []domain.Skill) string {
	if len(skills) == 0 {
		return ""
	}
	parts := make([]string, 0, len(skills)+1)
	parts = append(parts, "启用技能：")
	for _, skill := range skills {
		content := strings.TrimSpace(skill.Content)
		if content == "" {
			continue
		}
		label := strings.TrimSpace(skill.Name)
		if label == "" {
			label = skill.ID
		}
		parts = append(parts, fmt.Sprintf("[%s]\n%s", label, content))
	}
	if len(parts) == 1 {
		return ""
	}
	return strings.Join(parts, "\n\n")
}

func resolutionOutput(resolution domain.ModelResolution) map[string]any {
	return map[string]any{
		"route_key":         resolution.RouteKey,
		"resolution_source": resolution.ResolutionSource,
		"provider_id":       resolution.ProviderID,
		"provider_name":     resolution.ProviderName,
		"provider_type":     string(resolution.ProviderType),
		"model_id":          resolution.ModelID,
		"model_name":        resolution.ModelName,
		"model_kind":        string(resolution.ModelKind),
	}
}

func copyAnyMapRuntime(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	copied := make(map[string]any, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return copied
}

func stringSetRuntime(values []string) map[string]bool {
	if len(values) == 0 {
		return nil
	}
	set := make(map[string]bool, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			set[trimmed] = true
		}
	}
	return set
}
