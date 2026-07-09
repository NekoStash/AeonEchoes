package routes

import (
	"net/http"
	"strconv"
	"strings"

	"aeonechoes/server/internal/infra/http/v1/mappers"
	"aeonechoes/server/internal/infra/http/v1/query"
	"aeonechoes/server/internal/infra/http/v1/respond"
	"aeonechoes/server/internal/repository"
)

func (s *Router) v1ListIndexJobs(w http.ResponseWriter, r *http.Request) {
	limit, err := query.OptionalLimit(r)
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	query := r.URL.Query()
	items, err := s.store.ListIndexJobs(repository.IndexJobFilter{ProjectID: query.Get("project_id"), Status: query.Get("status"), Limit: limit})
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusInternalServerError, err)
		return
	}
	respond.List(w, r, http.StatusOK, mappers.IndexJobDTOsFromDomain(items), len(items), limit)
}

func (s *Router) v1RunIndexJob(w http.ResponseWriter, r *http.Request) {
	if s.indexing == nil {
		respond.Error(w, r, http.StatusServiceUnavailable, "service_unavailable", "indexing service is not configured", nil)
		return
	}
	job, err := s.indexing.RunJob(r.Context(), r.PathValue("id"))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	respond.Data(w, r, http.StatusOK, mappers.IndexJobDTOFromDomain(job))
}

func (s *Router) v1RunPendingIndexJobs(w http.ResponseWriter, r *http.Request) {
	if s.indexing == nil {
		respond.Error(w, r, http.StatusServiceUnavailable, "service_unavailable", "indexing service is not configured", nil)
		return
	}
	limit := 10
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			respond.Error(w, r, http.StatusBadRequest, "bad_request", "limit must be a positive integer", nil)
			return
		}
		limit = parsed
	}
	result, err := s.indexing.RunPending(r.Context(), r.URL.Query().Get("project_id"), limit)
	if err != nil {
		if result.Count > 0 || len(result.Processed) > 0 {
			respond.Data(w, r, http.StatusOK, mappers.RunPendingIndexDTOFromDomain(result))
			return
		}
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	respond.Data(w, r, http.StatusOK, mappers.RunPendingIndexDTOFromDomain(result))
}

func (s *Router) v1RebuildVectors(w http.ResponseWriter, r *http.Request) {
	if s.indexing == nil {
		respond.Error(w, r, http.StatusServiceUnavailable, "service_unavailable", "indexing service is not configured", nil)
		return
	}
	result, err := s.indexing.RebuildVectors(r.Context())
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	respond.Data(w, r, http.StatusOK, mappers.RebuildVectorsDTOFromDomain(result))
}
