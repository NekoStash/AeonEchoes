package mappers

import (
	"strings"

	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/infra/http/v1/dto"
)

func ChapterDTOFromDomain(chapter domain.Chapter) dto.ChapterDTO {
	metadata := CopyStringMapV1(chapter.Metadata)
	return dto.ChapterDTO{ID: chapter.ID, ProjectID: chapter.ProjectID, Number: chapter.Number, Title: chapter.Title, Status: chapter.Status, Summary: metadata["summary"], Metadata: metadata, CreatedAt: chapter.CreatedAt, UpdatedAt: chapter.UpdatedAt}
}

func ChapterDTOsFromDomain(chapters []domain.Chapter) []dto.ChapterDTO {
	items := make([]dto.ChapterDTO, 0, len(chapters))
	for _, chapter := range chapters {
		items = append(items, ChapterDTOFromDomain(chapter))
	}
	return items
}

func EnsureChapterRequestToDomain(projectID string, input dto.EnsureChapterRequestDTO) domain.ChapterEnsureRequest {
	metadata := CopyStringMapV1(input.Metadata)
	if strings.TrimSpace(input.Summary) != "" {
		if metadata == nil {
			metadata = map[string]string{}
		}
		metadata["summary"] = strings.TrimSpace(input.Summary)
	}
	return domain.ChapterEnsureRequest{ProjectID: projectID, ChapterID: input.ChapterID, Number: input.Number, Title: input.Title, Status: input.Status, Metadata: metadata}
}

func ChapterVersionDTOFromDomain(version domain.ChapterVersion) dto.ChapterVersionDTO {
	return dto.ChapterVersionDTO{ID: version.ID, ProjectID: version.ProjectID, ChapterID: version.ChapterID, Version: version.Version, Title: version.Title, Content: version.Content, Summary: version.Summary, AuthorRole: version.AuthorRole, SourceWorkflowID: version.SourceWorkflowID, IndexStatus: version.IndexStatus, Metadata: CopyStringMapV1(version.Metadata), CreatedAt: version.CreatedAt}
}

func ChapterVersionDTOsFromDomain(versions []domain.ChapterVersion) []dto.ChapterVersionDTO {
	items := make([]dto.ChapterVersionDTO, 0, len(versions))
	for _, version := range versions {
		items = append(items, ChapterVersionDTOFromDomain(version))
	}
	return items
}

func ChapterVersionRequestToDomain(projectID, chapterID string, input dto.ChapterVersionRequestDTO) domain.ChapterVersion {
	return domain.ChapterVersion{ID: input.ID, ProjectID: projectID, ChapterID: chapterID, Title: input.Title, Content: input.Content, Summary: input.Summary, AuthorRole: input.AuthorRole, SourceWorkflowID: input.SourceWorkflowID, IndexStatus: input.IndexStatus, Metadata: CopyStringMapV1(input.Metadata)}
}
