package mappers

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/infra/http/v1/dto"
	"aeonechoes/server/internal/infra/http/v1/shared"
)

const (
	storyBiblePremiseMetadataKey        = "story_bible_premise"
	storyBibleWorldRulesMetadataKey     = "story_bible_world_rules"
	storyBibleCharactersMetadataKey     = "story_bible_characters"
	storyBibleForeshadowsMetadataKey    = "story_bible_foreshadows"
	storyBibleChapterPlanMetadataKey    = "story_bible_chapter_plan"
	storyBibleLegacyChaptersMetadataKey = "story_bible_chapters"
)

func StoryBibleDTOFromDomain(bible domain.StoryBible) (dto.StoryBibleDTO, error) {
	metadata := bible.SourceSeed.Metadata
	premise := shared.FirstNonEmpty(metadata[storyBiblePremiseMetadataKey], bible.SourceSeed.Premise, bible.Logline, bible.Synopsis)
	worldRules, err := MetadataJSONValue(metadata, storyBibleWorldRulesMetadataKey, WorldRulesFromDomainRules(bible.Rules))
	if err != nil {
		return dto.StoryBibleDTO{}, err
	}
	characters, err := MetadataJSONValue(metadata, storyBibleCharactersMetadataKey, []dto.StoryBibleCharacterDTO{})
	if err != nil {
		return dto.StoryBibleDTO{}, err
	}
	foreshadows, err := MetadataJSONValue(metadata, storyBibleForeshadowsMetadataKey, []dto.StoryBibleForeshadowDTO{})
	if err != nil {
		return dto.StoryBibleDTO{}, err
	}
	chapterPlan, err := MetadataJSONValue(metadata, storyBibleChapterPlanMetadataKey, []dto.StoryBibleChapterPlanDTO{})
	if err != nil {
		return dto.StoryBibleDTO{}, err
	}
	for index, chapter := range chapterPlan {
		if !chapter.Status.Valid() {
			return dto.StoryBibleDTO{}, fmt.Errorf("chapter plan item %d status %q is invalid", index, chapter.Status)
		}
	}
	themes := CopyStringSliceV1(bible.Themes)
	if len(themes) == 0 {
		themes = CopyStringSliceV1(bible.SourceSeed.Themes)
	}
	return dto.StoryBibleDTO{
		ID:                bible.ID,
		ProjectID:         bible.ProjectID,
		Version:           bible.Version,
		Title:             bible.Title,
		Logline:           bible.Logline,
		Synopsis:          bible.Synopsis,
		Genre:             bible.Genre,
		Tone:              bible.Tone,
		Audience:          bible.Audience,
		Language:          bible.Language,
		Rules:             CopyStringMapV1(bible.Rules),
		WorldlineIDs:      CopyStringSliceV1(bible.WorldlineIDs),
		EntityIDs:         CopyStringSliceV1(bible.EntityIDs),
		PlotThreadIDs:     CopyStringSliceV1(bible.PlotThreadIDs),
		GenesisWorkflowID: bible.GenesisWorkflowID,
		Approved:          bible.Approved,
		CreatedAt:         bible.CreatedAt,
		Premise:           premise,
		Themes:            themes,
		WorldRules:        worldRules,
		Characters:        characters,
		Foreshadows:       foreshadows,
		ChapterPlan:       chapterPlan,
		SourceSeed:        ProjectSeedDTOFromDomain(bible.SourceSeed),
	}, nil
}

func StoryBibleDTOToDomain(input dto.StoryBibleDTO) (domain.StoryBible, error) {
	sourceSeed := ProjectSeedDTOToDomain(input.SourceSeed)
	metadata := CopyStringMapV1(sourceSeed.Metadata)
	if metadata == nil {
		metadata = map[string]string{}
	}
	delete(metadata, storyBibleLegacyChaptersMetadataKey)
	premise := strings.TrimSpace(shared.FirstNonEmpty(input.Premise, input.Logline, input.Synopsis, sourceSeed.Premise))
	if premise != "" {
		sourceSeed.Premise = premise
		metadata[storyBiblePremiseMetadataKey] = premise
	}
	worldRules := CopyStringSliceV1(input.WorldRules)
	if err := SetMetadataJSON(metadata, storyBibleWorldRulesMetadataKey, worldRules); err != nil {
		return domain.StoryBible{}, err
	}
	characters := append([]dto.StoryBibleCharacterDTO{}, input.Characters...)
	if err := SetMetadataJSON(metadata, storyBibleCharactersMetadataKey, characters); err != nil {
		return domain.StoryBible{}, err
	}
	foreshadows := append([]dto.StoryBibleForeshadowDTO{}, input.Foreshadows...)
	if err := SetMetadataJSON(metadata, storyBibleForeshadowsMetadataKey, foreshadows); err != nil {
		return domain.StoryBible{}, err
	}
	chapterPlan := append([]dto.StoryBibleChapterPlanDTO{}, input.ChapterPlan...)
	for index, chapter := range chapterPlan {
		if !chapter.Status.Valid() {
			return domain.StoryBible{}, fmt.Errorf("chapter plan item %d status %q is invalid", index, chapter.Status)
		}
	}
	if err := SetMetadataJSON(metadata, storyBibleChapterPlanMetadataKey, chapterPlan); err != nil {
		return domain.StoryBible{}, err
	}
	sourceSeed.Metadata = metadata
	if sourceSeed.Title == "" {
		sourceSeed.Title = input.Title
	}
	if sourceSeed.Genre == "" {
		sourceSeed.Genre = input.Genre
	}
	if sourceSeed.Tone == "" {
		sourceSeed.Tone = input.Tone
	}
	if sourceSeed.Audience == "" {
		sourceSeed.Audience = input.Audience
	}
	if sourceSeed.Language == "" {
		sourceSeed.Language = input.Language
	}
	if len(sourceSeed.Themes) == 0 {
		sourceSeed.Themes = CopyStringSliceV1(input.Themes)
	}
	if sourceSeed.TargetChapters <= 0 && len(chapterPlan) > 0 {
		sourceSeed.TargetChapters = len(chapterPlan)
	}
	rules := CopyStringMapV1(input.Rules)
	if len(worldRules) > 0 {
		rules = WorldRulesToDomainRules(worldRules)
	}
	return domain.StoryBible{
		ID:                input.ID,
		ProjectID:         input.ProjectID,
		Version:           input.Version,
		Title:             shared.FirstNonEmpty(input.Title, sourceSeed.Title),
		Logline:           shared.FirstNonEmpty(input.Logline, premise),
		Synopsis:          shared.FirstNonEmpty(input.Synopsis, premise),
		Genre:             shared.FirstNonEmpty(input.Genre, sourceSeed.Genre),
		Tone:              shared.FirstNonEmpty(input.Tone, sourceSeed.Tone),
		Audience:          shared.FirstNonEmpty(input.Audience, sourceSeed.Audience),
		Language:          shared.FirstNonEmpty(input.Language, sourceSeed.Language),
		Themes:            CopyStringSliceV1(input.Themes),
		Rules:             rules,
		WorldlineIDs:      CopyStringSliceV1(input.WorldlineIDs),
		EntityIDs:         CopyStringSliceV1(input.EntityIDs),
		PlotThreadIDs:     CopyStringSliceV1(input.PlotThreadIDs),
		SourceSeed:        sourceSeed,
		GenesisWorkflowID: input.GenesisWorkflowID,
		Approved:          input.Approved,
		CreatedAt:         input.CreatedAt,
	}, nil
}

func MetadataJSONValue[T any](metadata map[string]string, key string, fallback T) (T, error) {
	raw := strings.TrimSpace(metadata[key])
	if raw == "" {
		return fallback, nil
	}

	var value T
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return fallback, fmt.Errorf("story bible metadata %q contains invalid JSON: %w", key, err)
	}
	if string(raw) == "null" {
		return fallback, nil
	}
	return value, nil
}

func SetMetadataJSON(metadata map[string]string, key string, value any) error {
	if metadata == nil {
		return fmt.Errorf("metadata map must be initialized before writing %q", key)
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal %q: %w", key, err)
	}
	metadata[key] = string(payload)
	return nil
}

func WorldRulesFromDomainRules(rules map[string]string) []string {
	if len(rules) == 0 {
		return nil
	}
	keys := make([]string, 0, len(rules))
	for key := range rules {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	items := make([]string, 0, len(keys))
	for _, key := range keys {
		if value := strings.TrimSpace(rules[key]); value != "" {
			items = append(items, value)
		}
	}
	return items
}

func WorldRulesToDomainRules(worldRules []string) map[string]string {
	if len(worldRules) == 0 {
		return nil
	}
	rules := map[string]string{}
	for index, rule := range worldRules {
		if trimmed := strings.TrimSpace(rule); trimmed != "" {
			rules[fmt.Sprintf("rule_%d", index+1)] = trimmed
		}
	}
	return rules
}

func CharacterProfileFromDTO(dto dto.StoryBibleCharacterDTO) domain.CharacterProfile {
	return domain.CharacterProfile{Name: dto.Name, Role: dto.Role, Desire: dto.Desire, Wound: dto.Wound, Secret: dto.Secret, Summary: dto.Summary}
}

func CharacterProfileDTOFromDomain(profile domain.CharacterProfile, index int) dto.StoryBibleCharacterDTO {
	id := fmt.Sprintf("character-%d", index+1)
	return dto.StoryBibleCharacterDTO{ID: id, Name: profile.Name, Role: profile.Role, Desire: profile.Desire, Wound: profile.Wound, Secret: profile.Secret, Summary: profile.Summary}
}
