package routes

import (
	"net/http"
	"strings"

	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/infra/http/v1/dto"
	"aeonechoes/server/internal/infra/http/v1/mappers"
	"aeonechoes/server/internal/infra/http/v1/query"
	"aeonechoes/server/internal/infra/http/v1/respond"
	"aeonechoes/server/internal/repository"
)

func (s *Router) v1ListSkillSources(w http.ResponseWriter, r *http.Request) {
	limit, err := query.OptionalLimit(r)
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	filter := repository.SkillSourceFilter{ProjectID: r.URL.Query().Get("project_id"), Limit: limit}
	if enabled, present, err := query.OptionalBool(r, "enabled"); err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	} else if present {
		filter.Enabled = &enabled
	}
	items, err := s.store.ListSkillSources(filter)
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusInternalServerError, err)
		return
	}
	respond.List(w, r, http.StatusOK, mappers.SkillSourceDTOsFromDomain(items), len(items), limit)
}

func (s *Router) v1CreateSkillSource(w http.ResponseWriter, r *http.Request) {
	var input dto.SkillSourceDTO
	if !respond.Decode(w, r, &input) {
		return
	}
	created, err := s.store.CreateSkillSource(domain.SkillSource{ID: input.ID, ProjectID: input.ProjectID, Name: input.Name, Type: input.Type, Path: input.Path, InlineText: input.InlineText, Enabled: input.Enabled, Metadata: mappers.CopyStringMapV1(input.Metadata)})
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	respond.Data(w, r, http.StatusCreated, mappers.SkillSourceDTOFromDomain(created))
}

func (s *Router) v1ScanDefaultSkillSource(w http.ResponseWriter, r *http.Request) {
	if s.skillService == nil {
		respond.Error(w, r, http.StatusServiceUnavailable, "service_unavailable", "skill service is not configured", nil)
		return
	}
	result, err := s.skillService.ScanDefault(r.Context())
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	respond.Data(w, r, http.StatusOK, mappers.SkillScanResultDTOFromDomain(result))
}

func (s *Router) v1ScanSkillSource(w http.ResponseWriter, r *http.Request) {
	if s.skillService == nil {
		respond.Error(w, r, http.StatusServiceUnavailable, "service_unavailable", "skill service is not configured", nil)
		return
	}
	result, err := s.skillService.ScanSource(r.Context(), r.PathValue("id"))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	respond.Data(w, r, http.StatusOK, mappers.SkillScanResultDTOFromDomain(result))
}

func (s *Router) v1ListSkills(w http.ResponseWriter, r *http.Request) {
	limit, err := query.OptionalLimit(r)
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	filter := repository.SkillFilter{ProjectID: r.URL.Query().Get("project_id"), SourceID: r.URL.Query().Get("source_id"), Limit: limit}
	if enabled, present, err := query.OptionalBool(r, "enabled"); err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	} else if present {
		filter.Enabled = &enabled
	}
	items, err := s.store.ListSkills(filter)
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusInternalServerError, err)
		return
	}
	respond.List(w, r, http.StatusOK, mappers.SkillDTOsFromDomain(items), len(items), limit)
}

func (s *Router) v1CreateSkill(w http.ResponseWriter, r *http.Request) {
	var input dto.InlineSkillRequestDTO
	if !respond.Decode(w, r, &input) {
		return
	}
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	if strings.TrimSpace(input.SourceID) == "" {
		if s.skillService == nil {
			respond.Error(w, r, http.StatusServiceUnavailable, "service_unavailable", "skill service is not configured", nil)
			return
		}
		created, err := s.skillService.CreateInline(r.Context(), input.Name, input.Description, input.Content, enabled, input.Metadata)
		if err != nil {
			respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
			return
		}
		respond.Data(w, r, http.StatusCreated, mappers.SkillDTOFromDomain(created))
		return
	}
	created, err := s.store.CreateSkill(domain.Skill{ProjectID: input.ProjectID, SourceID: input.SourceID, Name: input.Name, Description: input.Description, Content: input.Content, Path: input.Path, Enabled: enabled, Metadata: mappers.CopyStringMapV1(input.Metadata)})
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	respond.Data(w, r, http.StatusCreated, mappers.SkillDTOFromDomain(created))
}

func (s *Router) v1GetSkill(w http.ResponseWriter, r *http.Request) {
	item, err := s.store.GetSkill(r.PathValue("id"))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusNotFound, err)
		return
	}
	respond.Data(w, r, http.StatusOK, mappers.SkillDTOFromDomain(item))
}

func (s *Router) v1UpdateSkill(w http.ResponseWriter, r *http.Request) {
	var input dto.SkillDTO
	if !respond.Decode(w, r, &input) {
		return
	}
	updated, err := s.store.UpdateSkill(r.PathValue("id"), mappers.SkillDTOToDomain(input))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	respond.Data(w, r, http.StatusOK, mappers.SkillDTOFromDomain(updated))
}

func (s *Router) v1DeleteSkill(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteSkill(r.PathValue("id")); err != nil {
		respond.ErrorFromErr(w, r, http.StatusNotFound, err)
		return
	}
	respond.Data(w, r, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Router) v1SetSkillEnabled(w http.ResponseWriter, r *http.Request) {
	var input dto.EnabledRequestDTO
	if !respond.Decode(w, r, &input) {
		return
	}
	skill, err := s.store.GetSkill(r.PathValue("id"))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusNotFound, err)
		return
	}
	skill.Enabled = input.Enabled
	updated, err := s.store.UpdateSkill(skill.ID, skill)
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	respond.Data(w, r, http.StatusOK, mappers.SkillDTOFromDomain(updated))
}
