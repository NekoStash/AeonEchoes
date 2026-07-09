package routes

import (
	"fmt"
	"net/http"
	"strings"

	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/infra/http/v1/dto"
	"aeonechoes/server/internal/infra/http/v1/mappers"
	"aeonechoes/server/internal/infra/http/v1/query"
	"aeonechoes/server/internal/infra/http/v1/respond"
	"aeonechoes/server/internal/mcp"
	"aeonechoes/server/internal/repository"
)

func (s *Router) v1ListMCPServers(w http.ResponseWriter, r *http.Request) {
	limit, err := query.OptionalLimit(r)
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	filter := repository.MCPServerConfigFilter{ProjectID: r.URL.Query().Get("project_id"), Limit: limit}
	if enabled, present, err := query.OptionalBool(r, "enabled"); err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	} else if present {
		filter.Enabled = &enabled
	}
	if status := strings.TrimSpace(r.URL.Query().Get("status")); status != "" {
		filter.Status = domain.MCPServerStatus(status)
		if !filter.Status.Valid() {
			respond.Error(w, r, http.StatusBadRequest, "bad_request", fmt.Sprintf("invalid mcp server status %q", status), nil)
			return
		}
	}
	items, err := s.store.ListMCPServerConfigs(filter)
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusInternalServerError, err)
		return
	}
	respond.List(w, r, http.StatusOK, mappers.McpServerDTOsFromDomain(items), len(items), limit)
}

func (s *Router) v1CreateMCPServer(w http.ResponseWriter, r *http.Request) {
	var input dto.McpServerRequestDTO
	if !respond.Decode(w, r, &input) {
		return
	}
	cfg := mappers.McpServerRequestToDomain(input)
	if cfg.Status == "" {
		cfg.Status = domain.MCPServerStatusUnknown
	}
	created, err := s.store.CreateMCPServerConfig(cfg)
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	respond.Data(w, r, http.StatusCreated, mappers.McpServerDTOFromDomain(created))
}

func (s *Router) v1GetMCPServer(w http.ResponseWriter, r *http.Request) {
	item, err := s.store.GetMCPServerConfig(r.PathValue("id"))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusNotFound, err)
		return
	}
	respond.Data(w, r, http.StatusOK, mappers.McpServerDTOFromDomain(item))
}

func (s *Router) v1UpdateMCPServer(w http.ResponseWriter, r *http.Request) {
	var input dto.McpServerRequestDTO
	if !respond.Decode(w, r, &input) {
		return
	}
	updated, err := s.store.UpdateMCPServerConfig(r.PathValue("id"), mappers.McpServerRequestToDomain(input))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	respond.Data(w, r, http.StatusOK, mappers.McpServerDTOFromDomain(updated))
}

func (s *Router) v1DeleteMCPServer(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteMCPServerConfig(r.PathValue("id")); err != nil {
		respond.ErrorFromErr(w, r, http.StatusNotFound, err)
		return
	}
	respond.Data(w, r, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Router) v1SetMCPServerEnabled(w http.ResponseWriter, r *http.Request) {
	var input dto.EnabledRequestDTO
	if !respond.Decode(w, r, &input) {
		return
	}
	server, err := s.store.GetMCPServerConfig(r.PathValue("id"))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusNotFound, err)
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
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	respond.Data(w, r, http.StatusOK, mappers.McpServerDTOFromDomain(updated))
}

func (s *Router) v1TestMCPServer(w http.ResponseWriter, r *http.Request) {
	server, err := s.store.GetMCPServerConfig(r.PathValue("id"))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusNotFound, err)
		return
	}
	client, err := mcp.NewClient(server, s.mcpDefaultTimeout)
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	if err := client.Test(r.Context()); err != nil {
		server.Status = domain.MCPServerStatusFailed
		if _, updateErr := s.store.UpdateMCPServerConfig(server.ID, server); updateErr != nil {
			respond.ErrorFromErr(w, r, http.StatusInternalServerError, updateErr)
			return
		}
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	server.Status = domain.MCPServerStatusOnline
	updated, err := s.store.UpdateMCPServerConfig(server.ID, server)
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusInternalServerError, err)
		return
	}
	respond.Data(w, r, http.StatusOK, dto.McpServerTestDTO{OK: true, Server: mappers.McpServerDTOFromDomain(updated)})
}

func (s *Router) v1RefreshMCPTools(w http.ResponseWriter, r *http.Request) {
	server, err := s.store.GetMCPServerConfig(r.PathValue("id"))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusNotFound, err)
		return
	}
	if !server.Enabled {
		respond.Error(w, r, http.StatusBadRequest, "bad_request", fmt.Sprintf("mcp server %q is disabled", server.ID), nil)
		return
	}
	client, err := mcp.NewClient(server, s.mcpDefaultTimeout)
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	tools, err := client.ListTools(r.Context())
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	seen := map[string]bool{}
	createdOrUpdated := make([]domain.ToolDefinition, 0, len(tools))
	for _, item := range tools {
		toolID := "mcp:" + server.ID + ":" + strings.TrimSpace(item.Name)
		if strings.TrimSpace(item.Name) == "" {
			respond.Error(w, r, http.StatusBadRequest, "bad_request", fmt.Sprintf("mcp server %q returned a tool with empty name", server.ID), nil)
			return
		}
		seen[toolID] = true
		schema, err := mappers.RawSchemaObject(item.InputSchema)
		if err != nil {
			respond.ErrorFromErr(w, r, http.StatusBadRequest, fmt.Errorf("tool %q input schema: %w", item.Name, err))
			return
		}
		tool, err := s.store.UpsertToolDefinition(domain.ToolDefinition{ID: toolID, Name: item.Name, DisplayName: item.Name, Description: item.Description, Kind: domain.ToolDefinitionMCP, Status: domain.ToolStatusActive, MCPServerID: server.ID, InputSchema: schema, Metadata: map[string]string{"mcp_server": server.ID}})
		if err != nil {
			respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
			return
		}
		createdOrUpdated = append(createdOrUpdated, tool)
	}
	existing, err := s.store.ListToolDefinitions(repository.ToolDefinitionFilter{Kind: domain.ToolDefinitionMCP, MCPServerID: server.ID})
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusInternalServerError, err)
		return
	}
	unavailable := 0
	for _, tool := range existing {
		if seen[tool.ID] {
			continue
		}
		tool.Status = domain.ToolStatusUnavailable
		if _, err := s.store.UpsertToolDefinition(tool); err != nil {
			respond.ErrorFromErr(w, r, http.StatusInternalServerError, err)
			return
		}
		unavailable++
	}
	respond.Data(w, r, http.StatusOK, dto.McpToolRefreshDTO{Tools: mappers.ToolDefinitionDTOsFromDomain(createdOrUpdated), Count: len(createdOrUpdated), Unavailable: unavailable})
}

func (s *Router) v1ListMCPServerTools(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListToolDefinitions(repository.ToolDefinitionFilter{Kind: domain.ToolDefinitionMCP, MCPServerID: r.PathValue("id")})
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusInternalServerError, err)
		return
	}
	respond.List(w, r, http.StatusOK, mappers.ToolDefinitionDTOsFromDomain(items), len(items), 0)
}

func (s *Router) v1ListToolCatalog(w http.ResponseWriter, r *http.Request) {
	limit, err := query.OptionalLimit(r)
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	filter := repository.ToolDefinitionFilter{ProjectID: r.URL.Query().Get("project_id"), MCPServerID: r.URL.Query().Get("mcp_server_id"), SourceID: r.URL.Query().Get("source_id"), SkillID: r.URL.Query().Get("skill_id"), Limit: limit}
	if kind := strings.TrimSpace(r.URL.Query().Get("kind")); kind != "" {
		filter.Kind = domain.ToolDefinitionKind(kind)
		if !filter.Kind.Valid() {
			respond.Error(w, r, http.StatusBadRequest, "bad_request", fmt.Sprintf("invalid tool kind %q", kind), nil)
			return
		}
	}
	if status := strings.TrimSpace(r.URL.Query().Get("status")); status != "" {
		filter.Status = domain.ToolStatus(status)
		if !filter.Status.Valid() {
			respond.Error(w, r, http.StatusBadRequest, "bad_request", fmt.Sprintf("invalid tool status %q", status), nil)
			return
		}
	}
	items, err := s.store.ListToolDefinitions(filter)
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusInternalServerError, err)
		return
	}
	respond.List(w, r, http.StatusOK, mappers.ToolDefinitionDTOsFromDomain(items), len(items), limit)
}

func (s *Router) v1SetToolEnabled(w http.ResponseWriter, r *http.Request) {
	var input dto.EnabledRequestDTO
	if !respond.Decode(w, r, &input) {
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
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	respond.Data(w, r, http.StatusOK, mappers.ToolDefinitionDTOFromDomain(updated))
}

func (s *Router) v1ListToolInvocations(w http.ResponseWriter, r *http.Request) {
	limit, err := query.OptionalLimit(r)
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	filter := repository.ToolInvocationFilter{AgentRunID: r.URL.Query().Get("agent_run_id"), AgentID: r.URL.Query().Get("agent_id"), ProjectID: r.URL.Query().Get("project_id"), ToolID: r.URL.Query().Get("tool_id"), Limit: limit}
	if status := strings.TrimSpace(r.URL.Query().Get("status")); status != "" {
		filter.Status = domain.ToolInvocationStatus(status)
		if !filter.Status.Valid() {
			respond.Error(w, r, http.StatusBadRequest, "bad_request", fmt.Sprintf("invalid tool invocation status %q", status), nil)
			return
		}
	}
	items, err := s.store.ListToolInvocations(filter)
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusInternalServerError, err)
		return
	}
	respond.List(w, r, http.StatusOK, mappers.ToolInvocationDTOsFromDomain(items), len(items), limit)
}
