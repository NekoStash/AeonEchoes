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

func CreateChapterRequestToDomain(projectID string, input dto.CreateChapterRequestDTO) domain.CreateChapterRequest {
	return domain.CreateChapterRequest{ProjectID: projectID, Number: input.Number, Title: input.Title, Status: input.Status, Metadata: chapterRequestMetadata(input.Summary, input.Metadata)}
}

func UpdateChapterRequestToDomain(projectID, chapterID string, input dto.UpdateChapterRequestDTO) domain.UpdateChapterRequest {
	return domain.UpdateChapterRequest{ProjectID: projectID, ChapterID: chapterID, Number: input.Number, Title: input.Title, Status: input.Status, Metadata: chapterUpdateMetadata(input.Summary, input.Metadata)}
}

func chapterRequestMetadata(summary string, input map[string]string) map[string]string {
	metadata := CopyStringMapV1(input)
	if strings.TrimSpace(summary) != "" {
		if metadata == nil {
			metadata = map[string]string{}
		}
		metadata["summary"] = strings.TrimSpace(summary)
	}
	return metadata
}

func chapterUpdateMetadata(summary *string, input *map[string]string) *map[string]string {
	if summary == nil && input == nil {
		return nil
	}
	metadata := map[string]string{}
	if input != nil {
		metadata = CopyStringMapV1(*input)
		if metadata == nil {
			metadata = map[string]string{}
		}
	}
	if summary != nil {
		metadata["summary"] = strings.TrimSpace(*summary)
	}
	return &metadata
}

func ChapterVersionDTOFromDomain(version domain.ChapterVersion) dto.ChapterVersionDTO {
	return dto.ChapterVersionDTO{ID: version.ID, ProjectID: version.ProjectID, ChapterID: version.ChapterID, ParentVersionID: version.ParentVersionID, Version: version.Version, Title: version.Title, Content: version.Content, Summary: version.Summary, AuthorRole: version.AuthorRole, SourceWorkflowID: version.SourceWorkflowID, IndexStatus: version.IndexStatus, Metadata: CopyStringMapV1(version.Metadata), CreatedAt: version.CreatedAt}
}

func ChapterVersionDTOsFromDomain(versions []domain.ChapterVersion) []dto.ChapterVersionDTO {
	items := make([]dto.ChapterVersionDTO, 0, len(versions))
	for _, version := range versions {
		items = append(items, ChapterVersionDTOFromDomain(version))
	}
	return items
}

func ChapterVersionRequestToDomain(projectID, chapterID string, input dto.ChapterVersionRequestDTO) domain.ChapterVersion {
	metadata := CopyStringMapV1(input.Metadata)
	if metadata != nil {
		delete(metadata, "parent_version_id")
	}
	if changeNote := strings.TrimSpace(input.ChangeNote); changeNote != "" {
		if metadata == nil {
			metadata = map[string]string{}
		}
		metadata["change_note"] = changeNote
	}
	return domain.ChapterVersion{ProjectID: projectID, ChapterID: chapterID, ParentVersionID: strings.TrimSpace(input.ParentVersionID), Title: input.Title, Content: input.Content, Summary: input.Summary, AuthorRole: input.AuthorRole, Metadata: metadata}
}
