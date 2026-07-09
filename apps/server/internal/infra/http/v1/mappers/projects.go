package mappers

import (
	"aeonechoes/server/internal/agent"
	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/infra/http/v1/dto"
)

func ProjectSeedDTOFromDomain(seed domain.ProjectSeed) dto.ProjectSeedDTO {
	return dto.ProjectSeedDTO{
		Title:          seed.Title,
		Premise:        seed.Premise,
		Genre:          seed.Genre,
		Tone:           seed.Tone,
		Audience:       seed.Audience,
		Language:       seed.Language,
		Setting:        seed.Setting,
		Themes:         CopyStringSliceV1(seed.Themes),
		MainCharacters: CopyStringSliceV1(seed.MainCharacters),
		Constraints:    CopyStringSliceV1(seed.Constraints),
		TargetChapters: seed.TargetChapters,
		Metadata:       CopyStringMapV1(seed.Metadata),
	}
}

func ProjectSeedDTOToDomain(seed dto.ProjectSeedDTO) domain.ProjectSeed {
	return domain.ProjectSeed{
		Title:          seed.Title,
		Premise:        seed.Premise,
		Genre:          seed.Genre,
		Tone:           seed.Tone,
		Audience:       seed.Audience,
		Language:       seed.Language,
		Setting:        seed.Setting,
		Themes:         CopyStringSliceV1(seed.Themes),
		MainCharacters: CopyStringSliceV1(seed.MainCharacters),
		Constraints:    CopyStringSliceV1(seed.Constraints),
		TargetChapters: seed.TargetChapters,
		Metadata:       CopyStringMapV1(seed.Metadata),
	}
}

func ProjectDTOFromDomain(project domain.Project) dto.ProjectDTO {
	return dto.ProjectDTO{
		ID:                 project.ID,
		Title:              project.Title,
		Slug:               project.Slug,
		Status:             project.Status,
		Seed:               ProjectSeedDTOFromDomain(project.Seed),
		ActiveStoryBibleID: project.ActiveStoryBibleID,
		DefaultWorldlineID: project.DefaultWorldlineID,
		Metadata:           CopyStringMapV1(project.Metadata),
		CreatedAt:          project.CreatedAt,
		UpdatedAt:          project.UpdatedAt,
	}
}

func ProjectDTOsFromDomain(projects []domain.Project) []dto.ProjectDTO {
	items := make([]dto.ProjectDTO, 0, len(projects))
	for _, project := range projects {
		items = append(items, ProjectDTOFromDomain(project))
	}
	return items
}

func InitializeProjectDTOFromDomain(result agent.InitializeResult) (dto.InitializeProjectDTO, error) {
	bible, err := StoryBibleDTOFromDomain(result.Bible)
	if err != nil {
		return dto.InitializeProjectDTO{}, err
	}
	return dto.InitializeProjectDTO{Project: ProjectDTOFromDomain(result.Project), StoryBible: bible, Workflow: WorkflowDTOFromDomain(result.Workflow)}, nil
}
