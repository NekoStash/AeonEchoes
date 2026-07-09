package routes

import (
	"net/http"
	"strings"

	"aeonechoes/server/internal/infra/http/v1/dto"
	"aeonechoes/server/internal/infra/http/v1/mappers"
	"aeonechoes/server/internal/infra/http/v1/respond"
	"aeonechoes/server/internal/infra/http/v1/shared"
)

func (s *Router) v1ListProjects(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListProjects()
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusInternalServerError, err)
		return
	}
	respond.List(w, r, http.StatusOK, mappers.ProjectDTOsFromDomain(items), len(items), 0)
}

func (s *Router) v1CreateProject(w http.ResponseWriter, r *http.Request) {
	if s.workflow == nil {
		respond.Error(w, r, http.StatusServiceUnavailable, "service_unavailable", "workflow runner is not configured", nil)
		return
	}
	var input dto.ProjectSeedDTO
	if !respond.Decode(w, r, &input) {
		return
	}
	result, err := s.workflow.InitializeProject(r.Context(), mappers.ProjectSeedDTOToDomain(input))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	dto, err := mappers.InitializeProjectDTOFromDomain(result)
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusInternalServerError, err)
		return
	}
	respond.Data(w, r, http.StatusCreated, dto)
}

func (s *Router) v1GetProject(w http.ResponseWriter, r *http.Request) {
	item, err := s.store.GetProject(r.PathValue("projectID"))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusNotFound, err)
		return
	}
	respond.Data(w, r, http.StatusOK, mappers.ProjectDTOFromDomain(item))
}

func (s *Router) v1OptimizeProjectSeed(w http.ResponseWriter, r *http.Request) {
	var input dto.ProjectSeedDTO
	if !respond.Decode(w, r, &input) {
		return
	}
	seed := mappers.ProjectSeedDTOToDomain(input)
	if strings.TrimSpace(seed.Title) == "" {
		respond.Error(w, r, http.StatusBadRequest, "bad_request", "project seed title must not be empty", nil)
		return
	}
	if strings.TrimSpace(seed.Premise) == "" {
		respond.Error(w, r, http.StatusBadRequest, "bad_request", "project seed premise must not be empty", nil)
		return
	}
	if seed.Language == "" {
		seed.Language = "zh-CN"
	}
	if seed.TargetChapters <= 0 {
		seed.TargetChapters = 12
	}
	if seed.Metadata == nil {
		seed.Metadata = map[string]string{}
	}
	seed.Metadata["optimized_prompt"] = shared.BuildOptimizedPrompt(seed)
	respond.Data(w, r, http.StatusOK, mappers.ProjectSeedDTOFromDomain(seed))
}
