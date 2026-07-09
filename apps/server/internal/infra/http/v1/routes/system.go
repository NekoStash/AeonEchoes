package routes

import (
	"net/http"
	"time"

	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/infra/http/v1/dto"
	"aeonechoes/server/internal/infra/http/v1/mappers"
	"aeonechoes/server/internal/infra/http/v1/respond"
)

func (s *Router) v1Health(w http.ResponseWriter, r *http.Request) {
	respond.Data(w, r, http.StatusOK, dto.HealthDTO{Status: "ok", Time: time.Now().UTC(), QdrantConfigured: s.cfg.QdrantURL != "", PostgresConfigured: s.cfg.PostgresDSN != ""})
}

func (s *Router) v1SystemStatus(w http.ResponseWriter, r *http.Request) {
	providers, err := s.store.ListProviders()
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusInternalServerError, err)
		return
	}
	models, err := s.store.ListModels()
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusInternalServerError, err)
		return
	}
	jobs, err := s.store.ListPendingIndexJobs("", 0)
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusInternalServerError, err)
		return
	}
	status := domain.SystemStatus{Status: "ok", PostgresConfigured: s.cfg.PostgresDSN != "", QdrantConfigured: s.cfg.QdrantURL != "", ProviderCount: len(providers), ModelCount: len(models), PendingJobsCount: len(jobs), CheckedAt: time.Now().UTC()}
	respond.Data(w, r, http.StatusOK, mappers.SystemStatusDTOFromDomain(status))
}
