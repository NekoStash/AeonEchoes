package mappers

import (
	"aeonechoes/server/internal/agent"
	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/infra/http/v1/dto"
	"aeonechoes/server/internal/skills"
)

func ContextSelectionDTOToAgent(input *dto.ContextSelectionDTO) *agent.ContextSelection {
	if input == nil {
		return nil
	}
	return &agent.ContextSelection{ChapterIDs: CopyStringSliceV1(input.ChapterIDs), CharacterIDs: CopyStringSliceV1(input.CharacterIDs), CharacterNames: CopyStringSliceV1(input.CharacterNames), IncludeWorldRules: input.IncludeWorldRules}
}

func ContextPreviewRequestToAgent(input dto.ContextPreviewRequestDTO) agent.ContextSelectionPreviewRequest {
	return agent.ContextSelectionPreviewRequest{ProjectID: input.ProjectID, ChapterID: input.ChapterID, Title: input.Title, Brief: input.Brief, Prompt: input.Prompt, ContextNodeIDs: CopyStringSliceV1(input.ContextNodeIDs), ContextSelection: ContextSelectionDTOToAgent(input.ContextSelection), ReferenceSelection: ContextSelectionDTOToAgent(input.ReferenceSelection), StyleConstraints: CopyStringSliceV1(input.StyleConstraints), Role: input.Role, TokenBudget: input.TokenBudget}
}

func ContextPreviewResponseDTOFromAgent(result agent.ContextSelectionPreviewResult) dto.ContextPreviewResponseDTO {
	return dto.ContextPreviewResponseDTO{ContextPack: ContextPackDTOFromDomain(result.ContextPack), Summary: result.Summary, EstimatedTokens: result.EstimatedTokens, IndexFreshness: IndexFreshnessDTOFromDomain(result.IndexFreshness), ModelResolution: ModelResolutionDTOFromDomain(result.ModelResolution)}
}

func ChapterIdeaRequestToAgent(input dto.ChapterIdeaRequestDTO) agent.ChapterIdeaRequest {
	return agent.ChapterIdeaRequest{ProjectID: input.ProjectID, ChapterID: input.ChapterID, Title: input.Title, Brief: input.Brief, Prompt: input.Prompt, ContextNodeIDs: CopyStringSliceV1(input.ContextNodeIDs), ContextSelection: ContextSelectionDTOToAgent(input.ContextSelection), ReferenceSelection: ContextSelectionDTOToAgent(input.ReferenceSelection), StyleConstraints: CopyStringSliceV1(input.StyleConstraints), MaxOutputTokens: input.MaxOutputTokens}
}

func ChapterIdeaResponseDTOFromAgent(result agent.ChapterIdeaResult) dto.ChapterIdeaResponseDTO {
	return dto.ChapterIdeaResponseDTO{Workflow: WorkflowDTOFromDomain(result.Workflow), ContextPack: ContextPackDTOFromDomain(result.ContextPack), ChapterIdea: result.ChapterIdea, ModelResolution: ModelResolutionDTOFromDomain(result.ModelResolution), ToolTrace: CopyStringSliceV1(result.ToolTrace)}
}

func CharacterProfilesRequestToAgent(input dto.CharacterProfilesRequestDTO) agent.CharacterProfilesRequest {
	return agent.CharacterProfilesRequest{ProjectID: input.ProjectID, Focus: input.Focus, Count: input.Count, Brief: input.Brief, ChapterID: input.ChapterID, ContextNodeIDs: CopyStringSliceV1(input.ContextNodeIDs), ContextSelection: ContextSelectionDTOToAgent(input.ContextSelection), MaxOutputTokens: input.MaxOutputTokens}
}

func CharacterProfilesResponseDTOFromAgent(result agent.CharacterProfilesResult) dto.CharacterProfilesResponseDTO {
	characters := make([]dto.StoryBibleCharacterDTO, 0, len(result.Characters))
	for index, profile := range result.Characters {
		characters = append(characters, CharacterProfileDTOFromDomain(profile, index))
	}
	mappings := make([]dto.CharacterProfileMappingDTO, 0, len(result.Mappings))
	for _, mapping := range result.Mappings {
		mappings = append(mappings, dto.CharacterProfileMappingDTO{Name: mapping.Name, EntityID: mapping.EntityID, Action: mapping.Action})
	}
	return dto.CharacterProfilesResponseDTO{Workflow: WorkflowDTOFromDomain(result.Workflow), ContextPack: ContextPackDTOFromDomain(result.ContextPack), Characters: characters, Entities: EntityDTOsFromDomain(result.Entities), Mappings: mappings, ModelResolution: ModelResolutionDTOFromDomain(result.ModelResolution), ToolTrace: CopyStringSliceV1(result.ToolTrace)}
}

func DraftRequestToAgent(input dto.DraftRequestDTO) agent.DraftRequest {
	return agent.DraftRequest{ProjectID: input.ProjectID, ChapterID: input.ChapterID, Title: input.Title, Brief: input.Brief, Prompt: input.Prompt, ChapterIdea: input.ChapterIdea, ChapterIdeaWorkflowID: input.ChapterIdeaWorkflowID, ContextNodeIDs: CopyStringSliceV1(input.ContextNodeIDs), ContextSelection: ContextSelectionDTOToAgent(input.ContextSelection), ReferenceSelection: ContextSelectionDTOToAgent(input.ReferenceSelection), StyleConstraints: CopyStringSliceV1(input.StyleConstraints), Role: input.Role, MaxOutputTokens: input.MaxOutputTokens}
}

func DraftResponseDTOFromAgent(result agent.DraftResult) dto.DraftResponseDTO {
	return dto.DraftResponseDTO{Workflow: WorkflowDTOFromDomain(result.Workflow), ContextPack: ContextPackDTOFromDomain(result.ContextPack), ChapterVersion: ChapterVersionDTOFromDomain(result.ChapterVersion), IndexJob: IndexJobDTOFromDomain(result.IndexJob), IndexFreshness: IndexFreshnessDTOFromDomain(result.IndexFreshness), ModelResolution: ModelResolutionDTOFromDomain(result.ModelResolution), ContinuityAudit: ContinuityAuditDTOFromDomain(result.ContinuityAudit), ToolTrace: CopyStringSliceV1(result.ToolTrace)}
}

func DraftWithIdeaRequestToAgent(input dto.DraftWithIdeaRequestDTO) agent.DraftWithIdeaRequest {
	return agent.DraftWithIdeaRequest{ProjectID: input.ProjectID, ChapterID: input.ChapterID, Title: input.Title, Brief: input.Brief, Prompt: input.Prompt, ContextNodeIDs: CopyStringSliceV1(input.ContextNodeIDs), ContextSelection: ContextSelectionDTOToAgent(input.ContextSelection), ReferenceSelection: ContextSelectionDTOToAgent(input.ReferenceSelection), StyleConstraints: CopyStringSliceV1(input.StyleConstraints), MaxIdeaOutputTokens: input.MaxIdeaOutputTokens, MaxDraftOutputTokens: input.MaxDraftOutputTokens}
}

func DraftWithIdeaResponseDTOFromAgent(result agent.DraftWithIdeaResult) dto.DraftWithIdeaResponseDTO {
	return dto.DraftWithIdeaResponseDTO{ChapterIdea: ChapterIdeaResponseDTOFromAgent(result.ChapterIdea), Draft: DraftResponseDTOFromAgent(result.Draft), ModelResolution: ModelResolutionDTOFromDomain(result.ModelResolution)}
}

func AgentConfigDTOFromDomain(item domain.AgentConfig) dto.AgentConfigDTO {
	return dto.AgentConfigDTO{ID: item.ID, ProjectID: item.ProjectID, Name: item.Name, Description: item.Description, Role: item.Role, ModelID: item.ModelID, Enabled: item.Enabled, SystemPrompt: item.SystemPrompt, SkillIDs: CopyStringSliceV1(item.SkillIDs), ToolIDs: CopyStringSliceV1(item.ToolIDs), MCPServerIDs: CopyStringSliceV1(item.MCPServerIDs), MemoryPolicy: CopyAnyMapV1(item.MemoryPolicy), RuntimeOptions: CopyAnyMapV1(item.RuntimeOptions), Metadata: CopyStringMapV1(item.Metadata), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func AgentConfigDTOsFromDomain(items []domain.AgentConfig) []dto.AgentConfigDTO {
	configs := make([]dto.AgentConfigDTO, 0, len(items))
	for _, item := range items {
		configs = append(configs, AgentConfigDTOFromDomain(item))
	}
	return configs
}

func AgentConfigDTOToDomain(input dto.AgentConfigDTO) domain.AgentConfig {
	return domain.AgentConfig{ID: input.ID, ProjectID: input.ProjectID, Name: input.Name, Description: input.Description, Role: input.Role, ModelID: input.ModelID, Enabled: input.Enabled, SystemPrompt: input.SystemPrompt, SkillIDs: CopyStringSliceV1(input.SkillIDs), ToolIDs: CopyStringSliceV1(input.ToolIDs), MCPServerIDs: CopyStringSliceV1(input.MCPServerIDs), MemoryPolicy: CopyAnyMapV1(input.MemoryPolicy), RuntimeOptions: CopyAnyMapV1(input.RuntimeOptions), Metadata: CopyStringMapV1(input.Metadata), CreatedAt: input.CreatedAt, UpdatedAt: input.UpdatedAt}
}

func AgentRunDTOFromDomain(item domain.AgentRun) dto.AgentRunDTO {
	return dto.AgentRunDTO{ID: item.ID, AgentID: item.AgentID, ProjectID: item.ProjectID, Status: item.Status, Input: CopyAnyMapV1(item.Input), Output: CopyAnyMapV1(item.Output), Error: item.Error, ToolInvocationIDs: CopyStringSliceV1(item.ToolInvocationIDs), StartedAt: item.StartedAt, CompletedAt: item.CompletedAt, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func AgentRunDTOsFromDomain(items []domain.AgentRun) []dto.AgentRunDTO {
	runs := make([]dto.AgentRunDTO, 0, len(items))
	for _, item := range items {
		runs = append(runs, AgentRunDTOFromDomain(item))
	}
	return runs
}

func AgentRunRequestToAgent(agentID string, input dto.AgentRunRequestDTO) agent.AgentRunRequest {
	return agent.AgentRunRequest{AgentID: agentID, ProjectID: input.ProjectID, TaskType: input.TaskType, Input: CopyAnyMapV1(input.Input), ContextSelection: ContextSelectionDTOToAgent(input.ContextSelection), MaxOutputTokens: input.MaxOutputTokens}
}

func AgentRunResultDTOFromAgent(result agent.AgentRunResult) dto.AgentRunResultDTO {
	return dto.AgentRunResultDTO{Run: AgentRunDTOFromDomain(result.Run), Content: result.Content, ToolTrace: CopyStringSliceV1(result.ToolTrace), ModelResolution: ModelResolutionDTOFromDomain(result.ModelResolution)}
}

func SkillSourceDTOFromDomain(item domain.SkillSource) dto.SkillSourceDTO {
	return dto.SkillSourceDTO{ID: item.ID, ProjectID: item.ProjectID, Name: item.Name, Type: item.Type, Path: item.Path, InlineText: item.InlineText, Enabled: item.Enabled, Metadata: CopyStringMapV1(item.Metadata), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func SkillSourceDTOsFromDomain(items []domain.SkillSource) []dto.SkillSourceDTO {
	sources := make([]dto.SkillSourceDTO, 0, len(items))
	for _, item := range items {
		sources = append(sources, SkillSourceDTOFromDomain(item))
	}
	return sources
}

func SkillDTOFromDomain(item domain.Skill) dto.SkillDTO {
	return dto.SkillDTO{ID: item.ID, ProjectID: item.ProjectID, SourceID: item.SourceID, Name: item.Name, Description: item.Description, Content: item.Content, Path: item.Path, Enabled: item.Enabled, Metadata: CopyStringMapV1(item.Metadata), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func SkillDTOsFromDomain(items []domain.Skill) []dto.SkillDTO {
	skillItems := make([]dto.SkillDTO, 0, len(items))
	for _, item := range items {
		skillItems = append(skillItems, SkillDTOFromDomain(item))
	}
	return skillItems
}

func SkillDTOToDomain(input dto.SkillDTO) domain.Skill {
	return domain.Skill{ID: input.ID, ProjectID: input.ProjectID, SourceID: input.SourceID, Name: input.Name, Description: input.Description, Content: input.Content, Path: input.Path, Enabled: input.Enabled, Metadata: CopyStringMapV1(input.Metadata), CreatedAt: input.CreatedAt, UpdatedAt: input.UpdatedAt}
}

func SkillScanResultDTOFromDomain(result skills.ScanResult) dto.SkillScanResultDTO {
	return dto.SkillScanResultDTO{SourceID: result.SourceID, Path: result.Path, Created: result.Created, Updated: result.Updated, Deleted: result.Deleted, Unchanged: result.Unchanged, Errors: CopyStringSliceV1(result.Errors), ScannedAt: result.ScannedAt}
}
