package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"aeonechoes/server/internal/agent"
	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/mcp"
	"aeonechoes/server/internal/repository"
)

type enabledRequest struct {
	Enabled bool `json:"enabled"`
}

type inlineSkillRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Content     string            `json:"content,omitempty"`
	Enabled     *bool             `json:"enabled,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	SourceID    string            `json:"source_id,omitempty"`
	ProjectID   string            `json:"project_id,omitempty"`
	Path        string            `json:"path,omitempty"`
}

func (s *Server) listAgents(w http.ResponseWriter, r *http.Request) {
	filter := repository.AgentConfigFilter{ProjectID: r.URL.Query().Get("project_id"), Limit: parseOptionalLimit(w, r)}
	if filter.Limit < 0 {
		return
	}
	if enabled, ok := parseOptionalBool(w, r, "enabled"); ok {
		filter.Enabled = &enabled
	} else if r.URL.Query().Has("enabled") {
		return
	}
	items, err := s.store.ListAgentConfigs(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) createAgent(w http.ResponseWriter, r *http.Request) {
	var input domain.AgentConfig
	if !decodeRequest(w, r, &input) {
		return
	}
	created, err := s.store.CreateAgentConfig(input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) getAgent(w http.ResponseWriter, r *http.Request) {
	item, err := s.store.GetAgentConfig(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) updateAgent(w http.ResponseWriter, r *http.Request) {
	var input domain.AgentConfig
	if !decodeRequest(w, r, &input) {
		return
	}
	updated, err := s.store.UpdateAgentConfig(r.PathValue("id"), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) deleteAgent(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteAgentConfig(r.PathValue("id")); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) runAgent(w http.ResponseWriter, r *http.Request) {
	if s.agentRuntime == nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Errorf("agent runtime is not configured"))
		return
	}
	var input agent.AgentRunRequest
	if !decodeRequest(w, r, &input) {
		return
	}
	input.AgentID = r.PathValue("id")
	result, err := s.agentRuntime.Run(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) listAgentRuns(w http.ResponseWriter, r *http.Request) {
	filter := repository.AgentRunFilter{AgentID: r.URL.Query().Get("agent_id"), ProjectID: r.URL.Query().Get("project_id"), Limit: parseOptionalLimit(w, r)}
	if filter.Limit < 0 {
		return
	}
	if status := strings.TrimSpace(r.URL.Query().Get("status")); status != "" {
		filter.Status = domain.AgentRunStatus(status)
		if !filter.Status.Valid() {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid agent run status %q", status))
			return
		}
	}
	items, err := s.store.ListAgentRuns(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) getAgentRun(w http.ResponseWriter, r *http.Request) {
	item, err := s.store.GetAgentRun(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) listSkills(w http.ResponseWriter, r *http.Request) {
	filter := repository.SkillFilter{ProjectID: r.URL.Query().Get("project_id"), SourceID: r.URL.Query().Get("source_id"), Limit: parseOptionalLimit(w, r)}
	if filter.Limit < 0 {
		return
	}
	if enabled, ok := parseOptionalBool(w, r, "enabled"); ok {
		filter.Enabled = &enabled
	} else if r.URL.Query().Has("enabled") {
		return
	}
	items, err := s.store.ListSkills(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) createSkill(w http.ResponseWriter, r *http.Request) {
	var input inlineSkillRequest
	if !decodeRequest(w, r, &input) {
		return
	}
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	if strings.TrimSpace(input.SourceID) == "" {
		if s.skillService == nil {
			writeError(w, http.StatusServiceUnavailable, fmt.Errorf("skill service is not configured"))
			return
		}
		created, err := s.skillService.CreateInline(r.Context(), input.Name, input.Description, input.Content, enabled, input.Metadata)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusCreated, created)
		return
	}
	created, err := s.store.CreateSkill(domain.Skill{ProjectID: input.ProjectID, SourceID: input.SourceID, Name: input.Name, Description: input.Description, Content: input.Content, Path: input.Path, Enabled: enabled, Metadata: input.Metadata})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) getSkill(w http.ResponseWriter, r *http.Request) {
	item, err := s.store.GetSkill(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) updateSkill(w http.ResponseWriter, r *http.Request) {
	var input domain.Skill
	if !decodeRequest(w, r, &input) {
		return
	}
	updated, err := s.store.UpdateSkill(r.PathValue("id"), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) deleteSkill(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteSkill(r.PathValue("id")); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) setSkillEnabled(w http.ResponseWriter, r *http.Request) {
	var input enabledRequest
	if !decodeRequest(w, r, &input) {
		return
	}
	skill, err := s.store.GetSkill(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	skill.Enabled = input.Enabled
	updated, err := s.store.UpdateSkill(skill.ID, skill)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) listSkillSources(w http.ResponseWriter, r *http.Request) {
	filter := repository.SkillSourceFilter{ProjectID: r.URL.Query().Get("project_id"), Limit: parseOptionalLimit(w, r)}
	if filter.Limit < 0 {
		return
	}
	if enabled, ok := parseOptionalBool(w, r, "enabled"); ok {
		filter.Enabled = &enabled
	} else if r.URL.Query().Has("enabled") {
		return
	}
	items, err := s.store.ListSkillSources(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) scanDefaultSkillSource(w http.ResponseWriter, r *http.Request) {
	if s.skillService == nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Errorf("skill service is not configured"))
		return
	}
	result, err := s.skillService.ScanDefault(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) scanSkillSource(w http.ResponseWriter, r *http.Request) {
	if s.skillService == nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Errorf("skill service is not configured"))
		return
	}
	result, err := s.skillService.ScanSource(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) listMCPServers(w http.ResponseWriter, r *http.Request) {
	filter := repository.MCPServerConfigFilter{ProjectID: r.URL.Query().Get("project_id"), Limit: parseOptionalLimit(w, r)}
	if filter.Limit < 0 {
		return
	}
	if enabled, ok := parseOptionalBool(w, r, "enabled"); ok {
		filter.Enabled = &enabled
	} else if r.URL.Query().Has("enabled") {
		return
	}
	if status := strings.TrimSpace(r.URL.Query().Get("status")); status != "" {
		filter.Status = domain.MCPServerStatus(status)
		if !filter.Status.Valid() {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid mcp server status %q", status))
			return
		}
	}
	items, err := s.store.ListMCPServerConfigs(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, mcpServerResponses(items))
}

func (s *Server) createMCPServer(w http.ResponseWriter, r *http.Request) {
	var input domain.MCPServerConfig
	if !decodeRequest(w, r, &input) {
		return
	}
	if input.Status == "" {
		input.Status = domain.MCPServerStatusUnknown
	}
	created, err := s.store.CreateMCPServerConfig(input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, mcpServerResponse(created))
}

func (s *Server) getMCPServer(w http.ResponseWriter, r *http.Request) {
	item, err := s.store.GetMCPServerConfig(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, mcpServerResponse(item))
}

func (s *Server) updateMCPServer(w http.ResponseWriter, r *http.Request) {
	var input domain.MCPServerConfig
	if !decodeRequest(w, r, &input) {
		return
	}
	updated, err := s.store.UpdateMCPServerConfig(r.PathValue("id"), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, mcpServerResponse(updated))
}

func (s *Server) deleteMCPServer(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteMCPServerConfig(r.PathValue("id")); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) setMCPServerEnabled(w http.ResponseWriter, r *http.Request) {
	var input enabledRequest
	if !decodeRequest(w, r, &input) {
		return
	}
	server, err := s.store.GetMCPServerConfig(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	server.Enabled = input.Enabled
	if !input.Enabled {
		server.Status = domain.MCPServerStatusDisabled
	} else if server.Status == domain.MCPServerStatusDisabled {
		server.Status = domain.MCPServerStatusUnknown
	}
	updated, err := s.store.UpdateMCPServerConfig(server.ID, server)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, mcpServerResponse(updated))
}

func (s *Server) testMCPServer(w http.ResponseWriter, r *http.Request) {
	server, err := s.store.GetMCPServerConfig(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	client, err := mcp.NewClient(server, s.mcpDefaultTimeout)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := client.Test(r.Context()); err != nil {
		server.Status = domain.MCPServerStatusFailed
		_, _ = s.store.UpdateMCPServerConfig(server.ID, server)
		writeError(w, http.StatusBadRequest, err)
		return
	}
	server.Status = domain.MCPServerStatusOnline
	updated, err := s.store.UpdateMCPServerConfig(server.ID, server)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "server": mcpServerResponse(updated)})
}

func (s *Server) refreshMCPTools(w http.ResponseWriter, r *http.Request) {
	server, err := s.store.GetMCPServerConfig(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if !server.Enabled {
		writeError(w, http.StatusBadRequest, fmt.Errorf("mcp server %q is disabled", server.ID))
		return
	}
	client, err := mcp.NewClient(server, s.mcpDefaultTimeout)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	tools, err := client.ListTools(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	seen := map[string]bool{}
	createdOrUpdated := make([]domain.ToolDefinition, 0, len(tools))
	for _, item := range tools {
		toolID := "mcp:" + server.ID + ":" + strings.TrimSpace(item.Name)
		if strings.TrimSpace(item.Name) == "" {
			writeError(w, http.StatusBadRequest, fmt.Errorf("mcp server %q returned a tool with empty name", server.ID))
			return
		}
		seen[toolID] = true
		schema, err := rawSchemaObject(item.InputSchema)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("tool %q input schema: %w", item.Name, err))
			return
		}
		tool, err := s.store.UpsertToolDefinition(domain.ToolDefinition{ID: toolID, Name: item.Name, DisplayName: item.Name, Description: item.Description, Kind: domain.ToolDefinitionMCP, Status: domain.ToolStatusActive, MCPServerID: server.ID, InputSchema: schema, Metadata: map[string]string{"mcp_server": server.ID}})
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		createdOrUpdated = append(createdOrUpdated, tool)
	}
	existing, err := s.store.ListToolDefinitions(repository.ToolDefinitionFilter{Kind: domain.ToolDefinitionMCP, MCPServerID: server.ID})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	unavailable := 0
	for _, tool := range existing {
		if seen[tool.ID] {
			continue
		}
		tool.Status = domain.ToolStatusUnavailable
		if _, err := s.store.UpsertToolDefinition(tool); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		unavailable++
	}
	writeJSON(w, http.StatusOK, map[string]any{"tools": createdOrUpdated, "count": len(createdOrUpdated), "unavailable": unavailable})
}

func (s *Server) listMCPServerTools(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListToolDefinitions(repository.ToolDefinitionFilter{Kind: domain.ToolDefinitionMCP, MCPServerID: r.PathValue("id")})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) listToolCatalog(w http.ResponseWriter, r *http.Request) {
	filter := repository.ToolDefinitionFilter{ProjectID: r.URL.Query().Get("project_id"), MCPServerID: r.URL.Query().Get("mcp_server_id"), SourceID: r.URL.Query().Get("source_id"), SkillID: r.URL.Query().Get("skill_id"), Limit: parseOptionalLimit(w, r)}
	if filter.Limit < 0 {
		return
	}
	if kind := strings.TrimSpace(r.URL.Query().Get("kind")); kind != "" {
		filter.Kind = domain.ToolDefinitionKind(kind)
		if !filter.Kind.Valid() {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid tool kind %q", kind))
			return
		}
	}
	if status := strings.TrimSpace(r.URL.Query().Get("status")); status != "" {
		filter.Status = domain.ToolStatus(status)
		if !filter.Status.Valid() {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid tool status %q", status))
			return
		}
	}
	items, err := s.store.ListToolDefinitions(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) setToolEnabled(w http.ResponseWriter, r *http.Request) {
	var input enabledRequest
	if !decodeRequest(w, r, &input) {
		return
	}
	var (
		updated domain.ToolDefinition
		err     error
	)
	if s.toolRegistry != nil {
		updated, err = s.toolRegistry.SetEnabled(r.Context(), r.PathValue("id"), input.Enabled)
	} else {
		updated, err = s.store.SetToolDefinitionEnabled(r.PathValue("id"), input.Enabled)
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) listToolInvocations(w http.ResponseWriter, r *http.Request) {
	filter := repository.ToolInvocationFilter{AgentRunID: r.URL.Query().Get("agent_run_id"), AgentID: r.URL.Query().Get("agent_id"), ProjectID: r.URL.Query().Get("project_id"), ToolID: r.URL.Query().Get("tool_id"), Limit: parseOptionalLimit(w, r)}
	if filter.Limit < 0 {
		return
	}
	if status := strings.TrimSpace(r.URL.Query().Get("status")); status != "" {
		filter.Status = domain.ToolInvocationStatus(status)
		if !filter.Status.Valid() {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid tool invocation status %q", status))
			return
		}
	}
	items, err := s.store.ListToolInvocations(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func parseOptionalLimit(w http.ResponseWriter, r *http.Request) int {
	raw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if raw == "" {
		return 0
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("limit must be a positive integer"))
		return -1
	}
	return parsed
}

func parseOptionalBool(w http.ResponseWriter, r *http.Request, key string) (bool, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return false, false
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("%s must be a boolean", key))
		return false, false
	}
	return parsed, true
}

func mcpServerResponses(items []domain.MCPServerConfig) []map[string]any {
	responses := make([]map[string]any, 0, len(items))
	for _, item := range items {
		responses = append(responses, mcpServerResponse(item))
	}
	return responses
}

func mcpServerResponse(item domain.MCPServerConfig) map[string]any {
	return map[string]any{
		"id":                  item.ID,
		"project_id":          item.ProjectID,
		"name":                item.Name,
		"transport":           item.Transport,
		"status":              item.Status,
		"enabled":             item.Enabled,
		"command":             item.Command,
		"args":                item.Args,
		"url":                 item.URL,
		"headers":             item.Headers,
		"secret_headers_hint": secretKeys(item.SecretHeaders),
		"env":                 item.Env,
		"secret_env_hint":     secretKeys(item.SecretEnv),
		"timeout_sec":         item.TimeoutSec,
		"metadata":            item.Metadata,
		"last_seen_at":        item.LastSeenAt,
		"created_at":          item.CreatedAt,
		"updated_at":          item.UpdatedAt,
	}
}

func secretKeys(values map[string]string) []string {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		if strings.TrimSpace(key) != "" {
			keys = append(keys, key)
		}
	}
	return keys
}

func rawSchemaObject(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		return nil, err
	}
	if schema == nil {
		return nil, fmt.Errorf("schema must be a JSON object")
	}
	return schema, nil
}
