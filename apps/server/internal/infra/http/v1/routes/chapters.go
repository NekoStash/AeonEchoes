package routes

import (
	"fmt"
	"net/http"

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
	var input dto.EnsureChapterRequestDTO
	if !respond.Decode(w, r, &input) {
		return
	}
	chapter, err := s.store.EnsureChapter(mappers.EnsureChapterRequestToDomain(r.PathValue("projectID"), input))
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
	var input dto.EnsureChapterRequestDTO
	if !respond.Decode(w, r, &input) {
		return
	}
	input.ChapterID = r.PathValue("chapterID")
	chapter, err := s.store.EnsureChapter(mappers.EnsureChapterRequestToDomain(r.PathValue("projectID"), input))
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
	created, job, err := s.store.SaveChapterVersion(mappers.ChapterVersionRequestToDomain(r.PathValue("projectID"), r.PathValue("chapterID"), input))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	s.notifyIndexWorker()
	respond.Data(w, r, http.StatusCreated, dto.SaveChapterVersionResponseDTO{ChapterVersion: mappers.ChapterVersionDTOFromDomain(created), IndexJob: mappers.IndexJobDTOFromDomain(job)})
}
