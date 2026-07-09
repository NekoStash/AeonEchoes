package routes

import (
	"net/http"

	"aeonechoes/server/internal/infra/http/v1/dto"
	"aeonechoes/server/internal/infra/http/v1/mappers"
	"aeonechoes/server/internal/infra/http/v1/respond"
	"aeonechoes/server/internal/infra/http/v1/shared"
)

func (s *Router) v1ListSettings(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListSettings(r.URL.Query().Get("scope"))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusInternalServerError, err)
		return
	}
	respond.List(w, r, http.StatusOK, mappers.AppSettingDTOsFromDomain(items), len(items), 0)
}

func (s *Router) v1UpsertSetting(w http.ResponseWriter, r *http.Request) {
	var input dto.AppSettingDTO
	if !respond.Decode(w, r, &input) {
		return
	}
	input.Scope = shared.FirstNonEmpty(r.PathValue("scope"), input.Scope)
	input.Key = shared.FirstNonEmpty(r.PathValue("key"), input.Key)
	updated, err := s.store.UpsertSetting(mappers.AppSettingDTOToDomain(input))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	respond.Data(w, r, http.StatusOK, mappers.AppSettingDTOFromDomain(updated))
}
