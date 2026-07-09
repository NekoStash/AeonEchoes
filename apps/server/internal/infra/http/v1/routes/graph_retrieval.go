package routes

import (
	"net/http"

	"aeonechoes/server/internal/infra/http/v1/dto"
	"aeonechoes/server/internal/infra/http/v1/mappers"
	"aeonechoes/server/internal/infra/http/v1/respond"
	"aeonechoes/server/internal/infra/http/v1/shared"
)

func (s *Router) v1PreviewContextSelection(w http.ResponseWriter, r *http.Request) {
	if s.workflow == nil {
		respond.Error(w, r, http.StatusServiceUnavailable, "service_unavailable", "workflow runner is not configured", nil)
		return
	}
	var input dto.ContextPreviewRequestDTO
	if !respond.Decode(w, r, &input) {
		return
	}
	input.ProjectID = r.PathValue("projectID")
	result, err := s.workflow.PreviewContextSelection(r.Context(), mappers.ContextPreviewRequestToAgent(input))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	respond.Data(w, r, http.StatusOK, mappers.ContextPreviewResponseDTOFromAgent(result))
}

func (s *Router) v1GenerateChapterIdea(w http.ResponseWriter, r *http.Request) {
	if s.workflow == nil {
		respond.Error(w, r, http.StatusServiceUnavailable, "service_unavailable", "workflow runner is not configured", nil)
		return
	}
	var input dto.ChapterIdeaRequestDTO
	if !respond.Decode(w, r, &input) {
		return
	}
	input.ProjectID = r.PathValue("projectID")
	input.ChapterID = shared.FirstNonEmpty(r.PathValue("chapterID"), input.ChapterID)
	result, err := s.workflow.GenerateChapterIdea(r.Context(), mappers.ChapterIdeaRequestToAgent(input))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	respond.Data(w, r, http.StatusCreated, mappers.ChapterIdeaResponseDTOFromAgent(result))
}

func (s *Router) v1DraftChapter(w http.ResponseWriter, r *http.Request) {
	if s.workflow == nil {
		respond.Error(w, r, http.StatusServiceUnavailable, "service_unavailable", "workflow runner is not configured", nil)
		return
	}
	var input dto.DraftRequestDTO
	if !respond.Decode(w, r, &input) {
		return
	}
	input.ProjectID = r.PathValue("projectID")
	input.ChapterID = shared.FirstNonEmpty(r.PathValue("chapterID"), input.ChapterID)
	result, err := s.workflow.DraftChapter(r.Context(), mappers.DraftRequestToAgent(input))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	s.notifyIndexWorker()
	respond.Data(w, r, http.StatusCreated, mappers.DraftResponseDTOFromAgent(result))
}

func (s *Router) v1GenerateCharacterProfiles(w http.ResponseWriter, r *http.Request) {
	if s.workflow == nil {
		respond.Error(w, r, http.StatusServiceUnavailable, "service_unavailable", "workflow runner is not configured", nil)
		return
	}
	var input dto.CharacterProfilesRequestDTO
	if !respond.Decode(w, r, &input) {
		return
	}
	input.ProjectID = r.PathValue("projectID")
	result, err := s.workflow.GenerateCharacterProfiles(r.Context(), mappers.CharacterProfilesRequestToAgent(input))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	respond.Data(w, r, http.StatusCreated, mappers.CharacterProfilesResponseDTOFromAgent(result))
}

func (s *Router) v1ExpandGraph(w http.ResponseWriter, r *http.Request) {
	var input dto.GraphExpansionRequestDTO
	if !respond.Decode(w, r, &input) {
		return
	}
	depth := input.Depth
	if depth == 0 {
		depth = 1
	}
	expansion, err := s.store.ExpandGraph(r.PathValue("projectID"), mappers.CopyStringSliceV1(input.EntityIDs), depth)
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	respond.Data(w, r, http.StatusOK, mappers.GraphExpansionDTOFromDomain(expansion))
}

func (s *Router) v1SemanticSearch(w http.ResponseWriter, r *http.Request) {
	if s.retrieval == nil {
		respond.Error(w, r, http.StatusServiceUnavailable, "service_unavailable", "semantic retrieval is not configured", nil)
		return
	}
	var input dto.SemanticSearchRequestDTO
	if !respond.Decode(w, r, &input) {
		return
	}
	input.ProjectID = r.PathValue("projectID")
	result, err := s.retrieval.Search(r.Context(), mappers.SemanticSearchRequestToDomain(input))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	respond.Data(w, r, http.StatusOK, mappers.SemanticSearchResultDTOFromDomain(result))
}
