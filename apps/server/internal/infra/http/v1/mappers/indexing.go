package mappers

import (
	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/indexing"
	"aeonechoes/server/internal/infra/http/v1/dto"
)

func IndexJobDTOFromDomain(item domain.IndexJob) dto.IndexJobDTO {
	return dto.IndexJobDTO{ID: item.ID, ProjectID: item.ProjectID, ChapterID: item.ChapterID, ChapterVersionID: item.ChapterVersionID, Kind: item.Kind, Status: item.Status, Attempts: item.Attempts, Error: item.Error, Payload: CopyStringMapV1(item.Payload), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt, ScheduledAt: item.ScheduledAt, StartedAt: item.StartedAt, CompletedAt: item.CompletedAt}
}

func IndexJobDTOsFromDomain(items []domain.IndexJob) []dto.IndexJobDTO {
	jobs := make([]dto.IndexJobDTO, 0, len(items))
	for _, item := range items {
		jobs = append(jobs, IndexJobDTOFromDomain(item))
	}
	return jobs
}

func RunPendingIndexDTOFromDomain(result indexing.RunResult) dto.RunPendingIndexDTO {
	return dto.RunPendingIndexDTO{Processed: IndexJobDTOsFromDomain(result.Processed), Count: result.Count, Error: result.Error}
}

func RebuildVectorsDTOFromDomain(result indexing.RebuildVectorsResult) dto.RebuildVectorsDTO {
	return dto.RebuildVectorsDTO{EmbeddingModelID: result.EmbeddingModelID, EmbeddingModelName: result.EmbeddingModelName, EmbeddingDimension: result.EmbeddingDimension, ProjectCount: result.ProjectCount, ChapterVersionCount: result.ChapterVersionCount, JobCount: result.JobCount}
}
