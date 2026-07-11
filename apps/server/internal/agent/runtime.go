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
	store    RuntimeStore
	router   *ModelRouter
	builder  *ContextPackBuilder
	clients  TextClientFactory
	tools    ToolCatalog
	toolExec ToolStore
}

func NewRuntime(store RuntimeStore, router *ModelRouter, builder *ContextPackBuilder, providers TextClientFactory, tools ToolCatalog) *Runtime {
	var toolExec ToolStore
	if candidate, ok := store.(ToolStore); ok {
		toolExec = candidate
	}
	return &Runtime{store: store, router: router, builder: builder, clients: providers, tools: tools, toolExec: toolExec}
}

func (r *Runtime) Run(ctx context.Context, req AgentRunRequest) (AgentRunResult, error) {
	if r == nil || r.store == nil || r.router == nil || r.clients == nil {
		return AgentRunResult{}, fmt.Errorf("agent runtime is not fully configured")
	}
	req.AgentID = strings.TrimSpace(req.AgentID)
	if req.AgentID == "" {
		return AgentRunResult{}, fmt.Errorf("agent run agent_id must not be empty")
	}
	cfg, err := r.store.GetAgentConfig(req.AgentID)
	if err != nil {
		return AgentRunResult{}, err
	}
	req.ProjectID = strings.TrimSpace(req.ProjectID)
	cfg.ProjectID = strings.TrimSpace(cfg.ProjectID)
	if !cfg.Enabled {
		return AgentRunResult{}, fmt.Errorf("agent %q is disabled", cfg.ID)
	}
	projectID, err := validateAgentProjectScope(cfg, req.ProjectID)
	if err != nil {
		return AgentRunResult{}, err
	}
	run, err := r.createRunningRun(req, projectID)
	if err != nil {
		return AgentRunResult{}, err
	}
	result, runErr := r.runCreated(ctx, run, cfg, req, projectID)
	if runErr != nil {
		failed, failErr := r.failRun(run, runErr)
		if failErr != nil {
			return AgentRunResult{Run: run}, fmt.Errorf("%w; update failed agent run: %v", runErr, failErr)
		}
		result.Run = failed
		return result, runErr
	}
	return result, nil
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

func (r *Runtime) runCreated(ctx context.Context, run domain.AgentRun, cfg domain.AgentConfig, req AgentRunRequest, projectID string) (AgentRunResult, error) {
	skills, err := r.enabledAgentSkills(cfg, projectID)
	if err != nil {
		return AgentRunResult{Run: run}, err
	}
	role := cfg.Role
	if role == "" {
		role = domain.AgentRoleWriter
	}
	selection, err := r.selectModel(cfg, role)
	if err != nil {
		return AgentRunResult{Run: run}, err
	}
	modelResolution := buildModelResolution(selection)
	client, err := r.clients.NewTextClient(selection.Provider)
	if err != nil {
		return AgentRunResult{Run: run, ModelResolution: modelResolution}, err
	}
	contextPack, err := r.buildContextPack(projectID, role, req)
	if err != nil {
		return AgentRunResult{Run: run, ModelResolution: modelResolution}, err
	}
	textReq, err := r.buildTextRequest(cfg, role, req, selection, skills, contextPack)
	if err != nil {
		return AgentRunResult{Run: run, ModelResolution: modelResolution}, err
	}
	if r.tools != nil {
		tools, err := r.tools.ListProviderTools(ctx, cfg)
		if err != nil {
			return AgentRunResult{Run: run, ModelResolution: modelResolution}, err
		}
		if len(tools) > 0 && selection.Model.SupportsTools {
			textReq.Tools = tools
		}
	}
	var content string
	toolTrace := []string(nil)
	if len(textReq.Tools) > 0 && selection.Model.SupportsTools {
		if r.toolExec == nil {
			return AgentRunResult{Run: run, ModelResolution: modelResolution}, fmt.Errorf("agent runtime tool executor store is not configured")
		}
		loopResult, err := RunToolLoop(ctx, client, textReq, NewToolExecutor(r.toolExec), defaultToolLoopMaxRounds)
		if err != nil {
			return AgentRunResult{Run: run, ModelResolution: modelResolution}, err
		}
		content = strings.TrimSpace(loopResult.Response.Content)
		toolTrace = loopResult.Trace
	} else {
		textReq.Tools = nil
		resp, err := client.Generate(ctx, textReq)
		if err != nil {
			return AgentRunResult{Run: run, ModelResolution: modelResolution}, err
		}
		content = strings.TrimSpace(resp.Content)
	}
	if content == "" {
		return AgentRunResult{Run: run, ModelResolution: modelResolution}, fmt.Errorf("agent runtime model returned empty content")
	}
	completed, err := r.completeRun(run, content, modelResolution, toolTrace)
	if err != nil {
		return AgentRunResult{Run: run, Content: content, ToolTrace: toolTrace, ModelResolution: modelResolution}, err
	}
	return AgentRunResult{Run: completed, Content: content, ToolTrace: toolTrace, ModelResolution: modelResolution}, nil
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
