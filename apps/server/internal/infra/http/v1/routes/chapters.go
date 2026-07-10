package routes

import (
	"fmt"
	"net/http"
	"strings"

	"aeonechoes/server/internal/infra/http/v1/dto"
	"aeonechoes/server/internal/infra/http/v1/mappers"
	"aeonechoes/server/internal/infra/http/v1/respond"
)

func (s *Router) v1ListChapters(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListChapters(r.PathValue("projectID"))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	respond.List(w, r, http.StatusOK, mappers.ChapterDTOsFromDomain(items), len(items), 0)
}

func (s *Router) v1CreateChapter(w http.ResponseWriter, r *http.Request) {
	var input dto.CreateChapterRequestDTO
	if !respond.Decode(w, r, &input) {
		return
	}
	if strings.TrimSpace(input.Title) == "" {
		respond.Error(w, r, http.StatusBadRequest, "bad_request", "chapter title must not be empty", nil)
		return
	}
	chapter, err := s.store.CreateChapter(mappers.CreateChapterRequestToDomain(r.PathValue("projectID"), input))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	respond.Data(w, r, http.StatusCreated, mappers.ChapterDTOFromDomain(chapter))
}

func (s *Router) v1GetChapter(w http.ResponseWriter, r *http.Request) {
	chapter, err := s.store.GetChapter(r.PathValue("chapterID"))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusNotFound, err)
		return
	}
	if chapter.ProjectID != r.PathValue("projectID") {
		respond.Error(w, r, http.StatusNotFound, "not_found", fmt.Sprintf("chapter %q not found in project %q", chapter.ID, r.PathValue("projectID")), nil)
		return
	}
	respond.Data(w, r, http.StatusOK, mappers.ChapterDTOFromDomain(chapter))
}

func (s *Router) v1UpdateChapter(w http.ResponseWriter, r *http.Request) {
	var input dto.UpdateChapterRequestDTO
	if !respond.Decode(w, r, &input) {
		return
	}
	if !input.HasChanges() {
		respond.Error(w, r, http.StatusBadRequest, "bad_request", "chapter update must include at least one field", nil)
		return
	}
	chapter, err := s.store.UpdateChapter(mappers.UpdateChapterRequestToDomain(r.PathValue("projectID"), r.PathValue("chapterID"), input))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	respond.Data(w, r, http.StatusOK, mappers.ChapterDTOFromDomain(chapter))
}

func (s *Router) v1ListChapterVersions(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListChapterVersions(r.PathValue("projectID"), r.PathValue("chapterID"))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusInternalServerError, err)
		return
	}
	respond.List(w, r, http.StatusOK, mappers.ChapterVersionDTOsFromDomain(items), len(items), 0)
}

func (s *Router) v1CreateChapterVersion(w http.ResponseWriter, r *http.Request) {
	var input dto.ChapterVersionRequestDTO
	if !respond.Decode(w, r, &input) {
		return
	}
	if strings.TrimSpace(input.Title) == "" {
		respond.Error(w, r, http.StatusBadRequest, "bad_request", "chapter version title must not be empty", nil)
		return
	}
	if strings.TrimSpace(input.Content) == "" {
		respond.Error(w, r, http.StatusBadRequest, "bad_request", "chapter version content must not be empty", nil)
		return
	}
	if !input.AuthorRole.Valid() {
		respond.Error(w, r, http.StatusBadRequest, "bad_request", fmt.Sprintf("chapter version author_role %q is invalid", input.AuthorRole), nil)
		return
	}
	created, job, err := s.store.SaveChapterVersion(mappers.ChapterVersionRequestToDomain(r.PathValue("projectID"), r.PathValue("chapterID"), input))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	s.notifyIndexWorker()
	respond.Data(w, r, http.StatusCreated, dto.SaveChapterVersionResponseDTO{ChapterVersion: mappers.ChapterVersionDTOFromDomain(created), IndexJob: mappers.IndexJobDTOFromDomain(job)})
}
