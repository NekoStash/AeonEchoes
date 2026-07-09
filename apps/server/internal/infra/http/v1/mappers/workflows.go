package mappers

import (
	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/infra/http/v1/dto"
)

func WorkflowStepDTOFromDomain(item domain.WorkflowStep) dto.WorkflowStepDTO {
	return dto.WorkflowStepDTO{Name: item.Name, Status: item.Status, StartedAt: item.StartedAt, EndedAt: item.EndedAt, Error: item.Error, Metadata: CopyStringMapV1(item.Metadata)}
}

func WorkflowStepDTOsFromDomain(items []domain.WorkflowStep) []dto.WorkflowStepDTO {
	steps := make([]dto.WorkflowStepDTO, 0, len(items))
	for _, item := range items {
		steps = append(steps, WorkflowStepDTOFromDomain(item))
	}
	return steps
}

func ModelResolutionDTOFromDomain(item domain.ModelResolution) dto.ModelResolutionDTO {
	return dto.ModelResolutionDTO{RouteKey: item.RouteKey, ResolutionSource: item.ResolutionSource, ProviderID: item.ProviderID, ProviderName: item.ProviderName, ProviderType: item.ProviderType, ModelID: item.ModelID, ModelName: item.ModelName, ModelKind: item.ModelKind}
}

func ModelResolutionDTOFromDomainPtr(item *domain.ModelResolution) *dto.ModelResolutionDTO {
	if item == nil {
		return nil
	}
	dto := ModelResolutionDTOFromDomain(*item)
	return &dto
}

func WorkflowDTOFromDomain(item domain.AIWorkflow) dto.WorkflowDTO {
	return dto.WorkflowDTO{ID: item.ID, ProjectID: item.ProjectID, Kind: item.Kind, Role: item.Role, Status: item.Status, ModelID: item.ModelID, ContextPackID: item.ContextPackID, ModelResolution: ModelResolutionDTOFromDomainPtr(item.ModelResolution), Steps: WorkflowStepDTOsFromDomain(item.Steps), Input: CopyStringMapV1(item.Input), Output: CopyStringMapV1(item.Output), Error: item.Error, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func WorkflowDTOsFromDomain(items []domain.AIWorkflow) []dto.WorkflowDTO {
	workflows := make([]dto.WorkflowDTO, 0, len(items))
	for _, item := range items {
		workflows = append(workflows, WorkflowDTOFromDomain(item))
	}
	return workflows
}

func IndexFreshnessDTOFromDomain(item domain.IndexFreshness) dto.IndexFreshnessDTO {
	return dto.IndexFreshnessDTO{ProjectID: item.ProjectID, ChapterID: item.ChapterID, Status: item.Status, LatestChapterVersionID: item.LatestChapterVersionID, LatestChapterVersionCreatedAt: item.LatestChapterVersionCreatedAt, LatestIndexedChapterVersionID: item.LatestIndexedChapterVersionID, LatestIndexedAt: item.LatestIndexedAt, PendingJobCount: item.PendingJobCount}
}

func ContinuityAuditDTOFromDomain(item domain.ContinuityAudit) dto.ContinuityAuditDTO {
	issues := make([]dto.ContinuityIssueDTO, 0, len(item.Issues))
	for _, issue := range item.Issues {
		evidence := make([]dto.ContinuityEvidenceRefDTO, 0, len(issue.Evidence))
		for _, ref := range issue.Evidence {
			evidence = append(evidence, dto.ContinuityEvidenceRefDTO{SourceType: ref.SourceType, SourceID: ref.SourceID, Label: ref.Label, Excerpt: ref.Excerpt})
		}
		issues = append(issues, dto.ContinuityIssueDTO{Type: issue.Type, Severity: issue.Severity, Message: issue.Message, DraftExcerpt: issue.DraftExcerpt, Suggestion: issue.Suggestion, Evidence: evidence})
	}
	return dto.ContinuityAuditDTO{Status: item.Status, Issues: issues}
}

func ChapterSummaryDTOFromDomain(item domain.ChapterSummary) dto.ChapterSummaryDTO {
	return dto.ChapterSummaryDTO{ChapterID: item.ChapterID, ChapterVersionID: item.ChapterVersionID, Title: item.Title, Summary: item.Summary}
}

func ChapterSummaryDTOsFromDomain(items []domain.ChapterSummary) []dto.ChapterSummaryDTO {
	summaries := make([]dto.ChapterSummaryDTO, 0, len(items))
	for _, item := range items {
		summaries = append(summaries, ChapterSummaryDTOFromDomain(item))
	}
	return summaries
}

func ContextPackDTOFromDomain(item domain.ContextPack) dto.ContextPackDTO {
	return dto.ContextPackDTO{ID: item.ID, ProjectID: item.ProjectID, ChapterID: item.ChapterID, Role: item.Role, TokenBudget: item.TokenBudget, Query: item.Query, StoryBibleID: item.StoryBibleID, WorldRules: CopyStringMapV1(item.WorldRules), Facts: FactDTOsFromDomain(item.Facts), Entities: EntityDTOsFromDomain(item.Entities), Edges: GraphEdgeDTOsFromDomain(item.Edges), PlotThreads: PlotThreadDTOsFromDomain(item.PlotThreads), ChapterSummaries: ChapterSummaryDTOsFromDomain(item.ChapterSummaries), ToolTrace: CopyStringSliceV1(item.ToolTrace), Metadata: CopyStringMapV1(item.Metadata), CreatedAt: item.CreatedAt}
}
