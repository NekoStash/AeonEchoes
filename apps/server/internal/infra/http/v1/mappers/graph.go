package mappers

import (
	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/infra/http/v1/dto"
)

func SemanticSearchRequestToDomain(input dto.SemanticSearchRequestDTO) domain.SemanticSearchRequest {
	return domain.SemanticSearchRequest{Query: input.Query, ProjectID: input.ProjectID, Limit: input.Limit, Filters: CopyStringMapV1(input.Filters)}
}

func SemanticSearchResultDTOFromDomain(result domain.SemanticSearchResult) dto.SemanticSearchResultDTO {
	items := make([]dto.SemanticSearchItemDTO, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, dto.SemanticSearchItemDTO{SourceID: item.SourceID, Score: item.Score, Payload: CopyAnyMapV1(item.Payload)})
	}
	return dto.SemanticSearchResultDTO{Query: result.Query, ProjectID: result.ProjectID, Items: items}
}

func WorldlineDTOFromDomain(item domain.Worldline) dto.WorldlineDTO {
	return dto.WorldlineDTO{ID: item.ID, ProjectID: item.ProjectID, Name: item.Name, Summary: item.Summary, Canonical: item.Canonical, Metadata: CopyStringMapV1(item.Metadata), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func EntityDTOFromDomain(item domain.Entity) dto.EntityDTO {
	return dto.EntityDTO{ID: item.ID, ProjectID: item.ProjectID, WorldlineID: item.WorldlineID, Name: item.Name, Type: item.Type, Aliases: CopyStringSliceV1(item.Aliases), Summary: item.Summary, Traits: CopyStringMapV1(item.Traits), Importance: item.Importance, Status: item.Status, Metadata: CopyStringMapV1(item.Metadata), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func EntityDTOsFromDomain(items []domain.Entity) []dto.EntityDTO {
	entities := make([]dto.EntityDTO, 0, len(items))
	for _, item := range items {
		entities = append(entities, EntityDTOFromDomain(item))
	}
	return entities
}

func FactDTOFromDomain(item domain.Fact) dto.FactDTO {
	return dto.FactDTO{ID: item.ID, ProjectID: item.ProjectID, WorldlineID: item.WorldlineID, EntityID: item.EntityID, ChapterID: item.ChapterID, ChapterVersionID: item.ChapterVersionID, Claim: item.Claim, Source: item.Source, Confidence: item.Confidence, Status: item.Status, EmbeddingRef: item.EmbeddingRef, Metadata: CopyStringMapV1(item.Metadata), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func FactDTOsFromDomain(items []domain.Fact) []dto.FactDTO {
	facts := make([]dto.FactDTO, 0, len(items))
	for _, item := range items {
		facts = append(facts, FactDTOFromDomain(item))
	}
	return facts
}

func GraphEdgeDTOFromDomain(item domain.GraphEdge) dto.GraphEdgeDTO {
	return dto.GraphEdgeDTO{ID: item.ID, ProjectID: item.ProjectID, WorldlineID: item.WorldlineID, SourceEntityID: item.SourceEntityID, TargetEntityID: item.TargetEntityID, Type: item.Type, Label: item.Label, Weight: item.Weight, EvidenceFactIDs: CopyStringSliceV1(item.EvidenceFactIDs), Metadata: CopyStringMapV1(item.Metadata), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func GraphEdgeDTOsFromDomain(items []domain.GraphEdge) []dto.GraphEdgeDTO {
	edges := make([]dto.GraphEdgeDTO, 0, len(items))
	for _, item := range items {
		edges = append(edges, GraphEdgeDTOFromDomain(item))
	}
	return edges
}

func PlotThreadDTOFromDomain(item domain.PlotThread) dto.PlotThreadDTO {
	return dto.PlotThreadDTO{ID: item.ID, ProjectID: item.ProjectID, WorldlineID: item.WorldlineID, Title: item.Title, Summary: item.Summary, Status: item.Status, Priority: item.Priority, RelatedEntityIDs: CopyStringSliceV1(item.RelatedEntityIDs), OpenedChapterID: item.OpenedChapterID, ClosedChapterID: item.ClosedChapterID, Metadata: CopyStringMapV1(item.Metadata), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func PlotThreadDTOsFromDomain(items []domain.PlotThread) []dto.PlotThreadDTO {
	threads := make([]dto.PlotThreadDTO, 0, len(items))
	for _, item := range items {
		threads = append(threads, PlotThreadDTOFromDomain(item))
	}
	return threads
}

func GraphExpansionDTOFromDomain(item domain.GraphExpansion) dto.GraphExpansionDTO {
	return dto.GraphExpansionDTO{ProjectID: item.ProjectID, Depth: item.Depth, Entities: EntityDTOsFromDomain(item.Entities), Edges: GraphEdgeDTOsFromDomain(item.Edges), Facts: FactDTOsFromDomain(item.Facts), GeneratedAt: item.GeneratedAt}
}
