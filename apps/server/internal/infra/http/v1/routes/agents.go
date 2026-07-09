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
	"aeonechoes/server/internal/repository"
)

func (s *Router) v1ListAgents(w http.ResponseWriter, r *http.Request) {
	limit, err := query.OptionalLimit(r)
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	filter := repository.AgentConfigFilter{ProjectID: r.URL.Query().Get("project_id"), Limit: limit}
	if enabled, present, err := query.OptionalBool(r, "enabled"); err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	} else if present {
		filter.Enabled = &enabled
	}
	items, err := s.store.ListAgentConfigs(filter)
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusInternalServerError, err)
		return
	}
	respond.List(w, r, http.StatusOK, mappers.AgentConfigDTOsFromDomain(items), len(items), limit)
}

func (s *Router) v1CreateAgent(w http.ResponseWriter, r *http.Request) {
	var input dto.AgentConfigDTO
	if !respond.Decode(w, r, &input) {
		return
	}
	created, err := s.store.CreateAgentConfig(mappers.AgentConfigDTOToDomain(input))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	respond.Data(w, r, http.StatusCreated, mappers.AgentConfigDTOFromDomain(created))
}

func (s *Router) v1GetAgent(w http.ResponseWriter, r *http.Request) {
	item, err := s.store.GetAgentConfig(r.PathValue("id"))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusNotFound, err)
		return
	}
	respond.Data(w, r, http.StatusOK, mappers.AgentConfigDTOFromDomain(item))
}

func (s *Router) v1UpdateAgent(w http.ResponseWriter, r *http.Request) {
	var input dto.AgentConfigDTO
	if !respond.Decode(w, r, &input) {
		return
	}
	updated, err := s.store.UpdateAgentConfig(r.PathValue("id"), mappers.AgentConfigDTOToDomain(input))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	respond.Data(w, r, http.StatusOK, mappers.AgentConfigDTOFromDomain(updated))
}

func (s *Router) v1DeleteAgent(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteAgentConfig(r.PathValue("id")); err != nil {
		respond.ErrorFromErr(w, r, http.StatusNotFound, err)
		return
	}
	respond.Data(w, r, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Router) v1RunAgent(w http.ResponseWriter, r *http.Request) {
	if s.agentRuntime == nil {
		respond.Error(w, r, http.StatusServiceUnavailable, "service_unavailable", "agent runtime is not configured", nil)
		return
	}
	var input dto.AgentRunRequestDTO
	if !respond.Decode(w, r, &input) {
		return
	}
	result, err := s.agentRuntime.Run(r.Context(), mappers.AgentRunRequestToAgent(r.PathValue("id"), input))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	respond.Data(w, r, http.StatusCreated, mappers.AgentRunResultDTOFromAgent(result))
}

func (s *Router) v1ListAgentRuns(w http.ResponseWriter, r *http.Request) {
	limit, err := query.OptionalLimit(r)
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	filter := repository.AgentRunFilter{AgentID: r.URL.Query().Get("agent_id"), ProjectID: r.URL.Query().Get("project_id"), Limit: limit}
	if status := strings.TrimSpace(r.URL.Query().Get("status")); status != "" {
		filter.Status = domain.AgentRunStatus(status)
		if !filter.Status.Valid() {
			respond.Error(w, r, http.StatusBadRequest, "bad_request", fmt.Sprintf("invalid agent run status %q", status), nil)
			return
		}
	}
	items, err := s.store.ListAgentRuns(filter)
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusInternalServerError, err)
		return
	}
	respond.List(w, r, http.StatusOK, mappers.AgentRunDTOsFromDomain(items), len(items), limit)
}

func (s *Router) v1GetAgentRun(w http.ResponseWriter, r *http.Request) {
	item, err := s.store.GetAgentRun(r.PathValue("id"))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusNotFound, err)
		return
	}
	respond.Data(w, r, http.StatusOK, mappers.AgentRunDTOFromDomain(item))
}
