package routes

import (
	"net/http"

	"aeonechoes/server/internal/infra/http/v1/mappers"
	"aeonechoes/server/internal/infra/http/v1/respond"
)

func (s *Router) v1ListProjectWorkflows(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListWorkflows(r.PathValue("projectID"))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusInternalServerError, err)
		return
	}
	respond.List(w, r, http.StatusOK, mappers.WorkflowDTOsFromDomain(items), len(items), 0)
}

func (s *Router) v1GetWorkflow(w http.ResponseWriter, r *http.Request) {
	item, err := s.store.GetWorkflow(r.PathValue("id"))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusNotFound, err)
		return
	}
	respond.Data(w, r, http.StatusOK, mappers.WorkflowDTOFromDomain(item))
}
