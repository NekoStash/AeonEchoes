package tooling

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"aeonechoes/server/internal/agent"
	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/provider"
	"aeonechoes/server/internal/repository"
)

const builtinToolIDPrefix = "builtin:"

// Store is the catalog and invocation persistence surface required by Registry.
type Store interface {
	UpsertToolDefinition(tool domain.ToolDefinition) (domain.ToolDefinition, error)
	GetToolDefinition(id string) (domain.ToolDefinition, error)
	ListToolDefinitions(filter repository.ToolDefinitionFilter) ([]domain.ToolDefinition, error)
	DeleteToolDefinition(id string) error
	SetToolDefinitionEnabled(id string, enabled bool) (domain.ToolDefinition, error)
	CreateToolInvocation(invocation domain.ToolInvocation) (domain.ToolInvocation, error)
	UpdateToolInvocation(id string, invocation domain.ToolInvocation) (domain.ToolInvocation, error)
	ListAgentConfigs(filter repository.AgentConfigFilter) ([]domain.AgentConfig, error)
	UpdateAgentConfig(id string, cfg domain.AgentConfig) (domain.AgentConfig, error)
}

// ToolExecutionContext records runtime identity for persisted tool invocation traces.
type ToolExecutionContext struct {
	AgentRunID string
	AgentID    string
	ProjectID  string
}

// Registry exposes persisted tool catalog entries to model providers and dispatches builtin tools.
type Registry struct {
	store     Store
	toolStore agent.ToolStore
}

func NewRegistry(store Store, toolStore agent.ToolStore) *Registry {
	return &Registry{store: store, toolStore: toolStore}
}

// SeedBuiltinTools stores the narrative builtin tool catalog using stable builtin-prefixed IDs.
// Any builtin catalog row that is not part of the current NarrativeToolSpecs is deleted, and agent
// configs lose those tool_ids so stale entries (for example chapter.ensure) cannot survive upgrades.
func (r *Registry) SeedBuiltinTools(ctx context.Context) error {
	if r == nil || r.store == nil {
		return fmt.Errorf("tooling registry store is not configured")
	}
	current := currentBuiltinToolNames()
	for _, spec := range agent.NarrativeToolSpecs() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		schema, err := parseToolSchema(spec.Parameters, spec.Name)
		if err != nil {
			return err
		}
		_, err = r.store.UpsertToolDefinition(domain.ToolDefinition{
			ID:          builtinToolID(spec.Name),
			Name:        spec.Name,
			DisplayName: spec.Name,
			Description: spec.Description,
			Kind:        domain.ToolDefinitionBuiltin,
			Status:      domain.ToolStatusActive,
			InputSchema: schema,
			Metadata:    map[string]string{"source": "agent.NarrativeToolSpecs"},
		})
		if err != nil {
			return fmt.Errorf("upsert builtin tool %q: %w", spec.Name, err)
		}
	}
	if err := r.deleteObsoleteBuiltinTools(ctx, current); err != nil {
		return err
	}
	if err := r.scrubAgentBuiltinToolIDs(ctx, current); err != nil {
		return err
	}
	return nil
}

func (r *Registry) deleteObsoleteBuiltinTools(ctx context.Context, current map[string]bool) error {
	builtins, err := r.store.ListToolDefinitions(repository.ToolDefinitionFilter{Kind: domain.ToolDefinitionBuiltin})
	if err != nil {
		return fmt.Errorf("list builtin tool definitions for seed cleanup: %w", err)
	}
	for _, tool := range builtins {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		name := builtinToolName(tool)
		if name != "" && current[name] {
			continue
		}
		if err := r.store.DeleteToolDefinition(tool.ID); err != nil {
			return fmt.Errorf("delete obsolete builtin tool %q: %w", tool.ID, err)
		}
	}
	return nil
}

func (r *Registry) scrubAgentBuiltinToolIDs(ctx context.Context, current map[string]bool) error {
	agents, err := r.store.ListAgentConfigs(repository.AgentConfigFilter{})
	if err != nil {
		return fmt.Errorf("list agent configs for tool_ids cleanup: %w", err)
	}
	for _, cfg := range agents {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		cleaned, changed := scrubObsoleteBuiltinToolIDs(cfg.ToolIDs, current)
		if !changed {
			continue
		}
		cfg.ToolIDs = cleaned
		if _, err := r.store.UpdateAgentConfig(cfg.ID, cfg); err != nil {
			return fmt.Errorf("scrub obsolete tool_ids for agent %q: %w", cfg.ID, err)
		}
	}
	return nil
}

// ListProviderTools returns enabled active tools as provider-neutral specs.
// Builtin names are de-prefixed so the agent ToolExecutor can dispatch them.
// Builtins that are not part of the current NarrativeToolSpecs are never exposed.
func (r *Registry) ListProviderTools(ctx context.Context, cfg domain.AgentConfig) ([]provider.ToolSpec, error) {
	if r == nil || r.store == nil {
		return nil, fmt.Errorf("tooling registry store is not configured")
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	activeTools, err := r.store.ListToolDefinitions(repository.ToolDefinitionFilter{Status: domain.ToolStatusActive})
	if err != nil {
		return nil, fmt.Errorf("list active tool definitions: %w", err)
	}
	allowed := stringSetFromSlice(cfg.ToolIDs)
	defaultBuiltinOnly := len(allowed) == 0
	currentBuiltins := currentBuiltinToolNames()
	result := make([]provider.ToolSpec, 0, len(activeTools))
	for _, tool := range activeTools {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		if len(allowed) > 0 && !allowed[tool.ID] {
			continue
		}
		if defaultBuiltinOnly && tool.Kind != domain.ToolDefinitionBuiltin {
			continue
		}
		if tool.Kind == domain.ToolDefinitionBuiltin {
			name := builtinToolName(tool)
			if name == "" || !currentBuiltins[name] {
				continue
			}
		}
		spec, ok, err := providerToolSpec(tool)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		result = append(result, spec)
	}
	return result, nil
}

func (r *Registry) SetEnabled(ctx context.Context, id string, enabled bool) (domain.ToolDefinition, error) {
	if r == nil || r.store == nil {
		return domain.ToolDefinition{}, fmt.Errorf("tooling registry store is not configured")
	}
	select {
	case <-ctx.Done():
		return domain.ToolDefinition{}, ctx.Err()
	default:
	}
	return r.store.SetToolDefinitionEnabled(id, enabled)
}

// ExecuteBuiltin executes one builtin provider tool call through the narrative ToolExecutor.
func (r *Registry) ExecuteBuiltin(ctx context.Context, call provider.ToolCall, exec ToolExecutionContext) (any, error) {
	if r == nil || r.toolStore == nil {
		return nil, fmt.Errorf("tooling registry tool store is not configured")
	}
	toolName := strings.TrimSpace(call.Name)
	if toolName == "" {
		return nil, fmt.Errorf("tool call name must not be empty")
	}
	toolID := builtinToolID(toolName)
	invocation, recordErr := r.RecordInvocationStarted(ctx, exec, toolID, toolName, call.Arguments)
	if recordErr != nil {
		return nil, recordErr
	}
	result, err := agent.NewToolExecutor(r.toolStore).Execute(ctx, call)
	if err != nil {
		if invocation.ID != "" {
			if _, finishErr := r.RecordInvocationFailed(ctx, invocation, err); finishErr != nil {
				return nil, fmt.Errorf("%w; record failed tool invocation: %v", err, finishErr)
			}
		}
		return nil, err
	}
	if invocation.ID != "" {
		if _, finishErr := r.RecordInvocationSucceeded(ctx, invocation, result); finishErr != nil {
			return nil, finishErr
		}
	}
	return result, nil
}

func (r *Registry) RecordInvocationStarted(ctx context.Context, exec ToolExecutionContext, toolID, toolName string, arguments json.RawMessage) (domain.ToolInvocation, error) {
	if r == nil || r.store == nil {
		return domain.ToolInvocation{}, fmt.Errorf("tooling registry store is not configured")
	}
	select {
	case <-ctx.Done():
		return domain.ToolInvocation{}, ctx.Err()
	default:
	}
	args, err := rawObject(arguments)
	if err != nil {
		return domain.ToolInvocation{}, fmt.Errorf("decode invocation arguments for %q: %w", toolName, err)
	}
	invocation, err := r.store.CreateToolInvocation(domain.ToolInvocation{
		AgentRunID: exec.AgentRunID,
		AgentID:    exec.AgentID,
		ProjectID:  exec.ProjectID,
		ToolID:     toolID,
		ToolName:   strings.TrimSpace(toolName),
		Status:     domain.ToolInvocationStatusRunning,
		Arguments:  args,
	})
	if err != nil {
		return domain.ToolInvocation{}, fmt.Errorf("create tool invocation for %q: %w", toolName, err)
	}
	return invocation, nil
}

func (r *Registry) RecordInvocationSucceeded(ctx context.Context, invocation domain.ToolInvocation, result any) (domain.ToolInvocation, error) {
	return r.finishInvocation(ctx, invocation, domain.ToolInvocationStatusSucceeded, result, nil)
}

func (r *Registry) RecordInvocationFailed(ctx context.Context, invocation domain.ToolInvocation, cause error) (domain.ToolInvocation, error) {
	return r.finishInvocation(ctx, invocation, domain.ToolInvocationStatusFailed, nil, cause)
}

func (r *Registry) finishInvocation(ctx context.Context, invocation domain.ToolInvocation, status domain.ToolInvocationStatus, result any, cause error) (domain.ToolInvocation, error) {
	if r == nil || r.store == nil {
		return domain.ToolInvocation{}, fmt.Errorf("tooling registry store is not configured")
	}
	select {
	case <-ctx.Done():
		return domain.ToolInvocation{}, ctx.Err()
	default:
	}
	invocation.Status = status
	if status == domain.ToolInvocationStatusSucceeded {
		resultMap, err := resultObject(result)
		if err != nil {
			return domain.ToolInvocation{}, fmt.Errorf("encode invocation result for %q: %w", invocation.ToolName, err)
		}
		invocation.Result = resultMap
		invocation.Error = ""
	} else {
		if cause == nil {
			cause = fmt.Errorf("tool invocation failed without cause")
		}
		invocation.Error = cause.Error()
	}
	updated, err := r.store.UpdateToolInvocation(invocation.ID, invocation)
	if err != nil {
		return domain.ToolInvocation{}, fmt.Errorf("update tool invocation %q: %w", invocation.ID, err)
	}
	return updated, nil
}

func parseToolSchema(raw json.RawMessage, name string) (map[string]any, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		return nil, fmt.Errorf("parse builtin tool %q schema: %w", name, err)
	}
	if schema == nil {
		return nil, fmt.Errorf("builtin tool %q schema must be a JSON object", name)
	}
	return schema, nil
}

func providerToolSpec(tool domain.ToolDefinition) (provider.ToolSpec, bool, error) {
	if tool.Status != domain.ToolStatusActive {
		return provider.ToolSpec{}, false, nil
	}
	name := strings.TrimSpace(tool.Name)
	if name == "" {
		return provider.ToolSpec{}, false, fmt.Errorf("tool definition %q has empty name", tool.ID)
	}
	switch tool.Kind {
	case domain.ToolDefinitionBuiltin:
		name = strings.TrimPrefix(name, builtinToolIDPrefix)
		name = strings.TrimPrefix(name, "builtin.")
		if strings.HasPrefix(tool.ID, builtinToolIDPrefix) {
			name = strings.TrimPrefix(tool.ID, builtinToolIDPrefix)
		}
	case domain.ToolDefinitionMCP, domain.ToolDefinitionSkill:
		// MCP and skill-backed tools are exposed from the catalog, but runtime execution is wired later.
	default:
		return provider.ToolSpec{}, false, fmt.Errorf("tool definition %q has unsupported kind %q", tool.ID, tool.Kind)
	}
	parameters, err := json.Marshal(tool.InputSchema)
	if err != nil {
		return provider.ToolSpec{}, false, fmt.Errorf("marshal tool definition %q schema: %w", tool.ID, err)
	}
	if len(tool.InputSchema) == 0 {
		parameters = nil
	}
	return provider.ToolSpec{Name: name, Description: tool.Description, Parameters: parameters}, true, nil
}

func rawObject(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("value must be a JSON object")
	}
	return result, nil
}

func resultObject(result any) (map[string]any, error) {
	if result == nil {
		return nil, nil
	}
	if value, ok := result.(map[string]any); ok {
		return value, nil
	}
	payload, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	var object map[string]any
	if err := json.Unmarshal(payload, &object); err != nil {
		return nil, err
	}
	if object == nil {
		return nil, fmt.Errorf("result must encode to a JSON object")
	}
	return object, nil
}

func builtinToolID(name string) string {
	return builtinToolIDPrefix + strings.TrimSpace(name)
}

func currentBuiltinToolNames() map[string]bool {
	specs := agent.NarrativeToolSpecs()
	set := make(map[string]bool, len(specs))
	for _, spec := range specs {
		name := strings.TrimSpace(spec.Name)
		if name != "" {
			set[name] = true
		}
	}
	return set
}

func builtinToolName(tool domain.ToolDefinition) string {
	if strings.HasPrefix(tool.ID, builtinToolIDPrefix) {
		return strings.TrimSpace(strings.TrimPrefix(tool.ID, builtinToolIDPrefix))
	}
	name := strings.TrimSpace(tool.Name)
	name = strings.TrimPrefix(name, builtinToolIDPrefix)
	name = strings.TrimPrefix(name, "builtin.")
	return strings.TrimSpace(name)
}

func scrubObsoleteBuiltinToolIDs(toolIDs []string, current map[string]bool) ([]string, bool) {
	if len(toolIDs) == 0 {
		return nil, false
	}
	cleaned := make([]string, 0, len(toolIDs))
	changed := false
	for _, raw := range toolIDs {
		id := strings.TrimSpace(raw)
		if id == "" {
			changed = true
			continue
		}
		name, isBuiltin := parseBuiltinToolID(id)
		if isBuiltin && (name == "" || !current[name]) {
			changed = true
			continue
		}
		cleaned = append(cleaned, id)
	}
	if !changed {
		return toolIDs, false
	}
	return cleaned, true
}

func parseBuiltinToolID(id string) (name string, isBuiltin bool) {
	id = strings.TrimSpace(id)
	switch {
	case strings.HasPrefix(id, builtinToolIDPrefix):
		return strings.TrimSpace(strings.TrimPrefix(id, builtinToolIDPrefix)), true
	case strings.HasPrefix(id, "builtin."):
		return strings.TrimSpace(strings.TrimPrefix(id, "builtin.")), true
	default:
		return "", false
	}
}

func stringSetFromSlice(values []string) map[string]bool {
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
