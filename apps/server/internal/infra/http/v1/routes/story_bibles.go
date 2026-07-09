package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/infra/http/v1/dto"
	"aeonechoes/server/internal/infra/http/v1/mappers"
	"aeonechoes/server/internal/infra/http/v1/respond"
	"aeonechoes/server/internal/infra/http/v1/shared"
	"aeonechoes/server/internal/repository"
)

func (s *Router) v1GetCurrentStoryBible(w http.ResponseWriter, r *http.Request) {
	bible, err := s.store.GetStoryBible(r.PathValue("projectID"))
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusNotFound, err)
		return
	}
	dto, err := mappers.StoryBibleDTOFromDomain(bible)
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusInternalServerError, err)
		return
	}
	respond.Data(w, r, http.StatusOK, dto)
}

func (s *Router) v1UpdateStoryBible(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	storyBibleID := r.PathValue("storyBibleID")
	current, err := s.store.GetStoryBible(projectID)
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusNotFound, err)
		return
	}
	if current.ID != storyBibleID {
		respond.Error(w, r, http.StatusNotFound, "not_found", fmt.Sprintf("story bible %q is not the current story bible for project %q", storyBibleID, projectID), nil)
		return
	}
	var input dto.StoryBibleDTO
	if !respond.Decode(w, r, &input) {
		return
	}
	input.ID = storyBibleID
	input.ProjectID = projectID
	bible, err := mappers.StoryBibleDTOToDomain(input)
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	updated, err := s.store.UpdateStoryBible(projectID, bible)
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	dto, err := mappers.StoryBibleDTOFromDomain(updated)
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusInternalServerError, err)
		return
	}
	respond.Data(w, r, http.StatusOK, dto)
}

func (s *Router) v1SyncCharacters(w http.ResponseWriter, r *http.Request) {
	var input dto.CharacterSyncRequestDTO
	if !respond.Decode(w, r, &input) {
		return
	}
	profiles := make([]domain.CharacterProfile, 0, len(input.Characters))
	for _, character := range input.Characters {
		profiles = append(profiles, mappers.CharacterProfileFromDTO(character))
	}
	result, err := syncStoryBibleCharacters(s.store, characterSyncRequest{ProjectID: r.PathValue("projectID"), StoryBibleID: shared.FirstNonEmpty(r.PathValue("storyBibleID"), input.StoryBibleID), Source: input.Source, Characters: profiles, Metadata: mappers.CopyStringMapV1(input.Metadata)})
	if err != nil {
		respond.ErrorFromErr(w, r, http.StatusBadRequest, err)
		return
	}
	responseDTO := dto.CharacterSyncResponseDTO{ProjectID: result.ProjectID, StoryBibleID: result.StoryBibleID, Characters: mappers.EntityDTOsFromDomain(result.Characters), Mappings: make([]dto.CharacterProfileMappingDTO, 0, len(result.Mappings))}
	for _, mapping := range result.Mappings {
		responseDTO.Mappings = append(responseDTO.Mappings, dto.CharacterProfileMappingDTO{Name: mapping.Name, EntityID: mapping.EntityID, Action: mapping.Action})
	}
	respond.Data(w, r, http.StatusOK, responseDTO)
}

type characterSyncRequest struct {
	ProjectID    string                    `json:"project_id,omitempty"`
	StoryBibleID string                    `json:"story_bible_id,omitempty"`
	Source       string                    `json:"source,omitempty"`
	Characters   []domain.CharacterProfile `json:"characters"`
	Metadata     map[string]string         `json:"metadata,omitempty"`
}

type characterSyncResult struct {
	ProjectID    string                           `json:"project_id"`
	StoryBibleID string                           `json:"story_bible_id,omitempty"`
	Characters   []domain.Entity                  `json:"characters"`
	Mappings     []domain.CharacterProfileMapping `json:"mappings"`
}

func syncStoryBibleCharacters(store repository.AppStore, input characterSyncRequest) (characterSyncResult, error) {
	if store == nil {
		return characterSyncResult{}, fmt.Errorf("character sync store is not configured")
	}
	projectID := strings.TrimSpace(input.ProjectID)
	if projectID == "" {
		return characterSyncResult{}, fmt.Errorf("character sync project_id must not be empty")
	}
	if _, err := store.GetProject(projectID); err != nil {
		return characterSyncResult{}, err
	}
	if len(input.Characters) == 0 {
		return characterSyncResult{}, fmt.Errorf("character sync characters must not be empty")
	}
	characters, err := normalizeCharacterSyncProfiles(input.Characters)
	if err != nil {
		return characterSyncResult{}, err
	}
	requestedNames := characterNameSet(characters)
	existing, err := store.ListEntities(projectID)
	if err != nil {
		return characterSyncResult{}, err
	}
	byName := make(map[string]domain.Entity, len(existing))
	for _, entity := range existing {
		nameKey := normalizedCharacterName(entity.Name)
		if nameKey == "" {
			continue
		}
		if entity.Type != "" && entity.Type != "character" {
			if _, requested := requestedNames[nameKey]; requested {
				return characterSyncResult{}, fmt.Errorf("character sync name %q conflicts with existing non-character entity %q of type %q", entity.Name, entity.ID, entity.Type)
			}
			continue
		}
		if previous, ok := byName[nameKey]; ok {
			return characterSyncResult{}, fmt.Errorf("character sync found duplicate existing character name %q for entities %q and %q", entity.Name, previous.ID, entity.ID)
		}
		byName[nameKey] = entity
	}
	result := characterSyncResult{ProjectID: projectID, StoryBibleID: strings.TrimSpace(input.StoryBibleID), Characters: make([]domain.Entity, 0, len(characters)), Mappings: make([]domain.CharacterProfileMapping, 0, len(characters))}
	for _, profile := range characters {
		nameKey := normalizedCharacterName(profile.Name)
		existingEntity, exists := byName[nameKey]
		entity := characterProfileEntity(projectID, profile, existingEntity, input)
		saved, err := store.SaveEntity(entity)
		if err != nil {
			return characterSyncResult{}, err
		}
		action := "created"
		if exists {
			action = "updated"
		}
		byName[nameKey] = saved
		result.Characters = append(result.Characters, saved)
		result.Mappings = append(result.Mappings, domain.CharacterProfileMapping{Name: saved.Name, EntityID: saved.ID, Action: action})
	}
	return result, nil
}

func normalizeCharacterSyncProfiles(input []domain.CharacterProfile) ([]domain.CharacterProfile, error) {
	characters := make([]domain.CharacterProfile, 0, len(input))
	seen := map[string]struct{}{}
	for i, character := range input {
		character.Name = strings.TrimSpace(character.Name)
		character.Role = strings.TrimSpace(character.Role)
		character.Desire = strings.TrimSpace(character.Desire)
		character.Wound = strings.TrimSpace(character.Wound)
		character.Secret = strings.TrimSpace(character.Secret)
		character.Summary = strings.TrimSpace(character.Summary)
		if character.Name == "" {
			return nil, fmt.Errorf("character sync characters[%d].name must not be empty", i)
		}
		if character.Role == "" {
			return nil, fmt.Errorf("character sync characters[%d].role must not be empty", i)
		}
		if character.Desire == "" {
			return nil, fmt.Errorf("character sync characters[%d].desire must not be empty", i)
		}
		if character.Wound == "" {
			return nil, fmt.Errorf("character sync characters[%d].wound must not be empty", i)
		}
		nameKey := normalizedCharacterName(character.Name)
		if _, ok := seen[nameKey]; ok {
			return nil, fmt.Errorf("character sync duplicate character name %q", character.Name)
		}
		seen[nameKey] = struct{}{}
		characters = append(characters, character)
	}
	return characters, nil
}

func characterNameSet(characters []domain.CharacterProfile) map[string]struct{} {
	set := make(map[string]struct{}, len(characters))
	for _, character := range characters {
		set[normalizedCharacterName(character.Name)] = struct{}{}
	}
	return set
}

func characterProfileEntity(projectID string, profile domain.CharacterProfile, existing domain.Entity, input characterSyncRequest) domain.Entity {
	entity := existing
	entity.ProjectID = projectID
	entity.Name = profile.Name
	entity.Type = "character"
	entity.Summary = characterProfileSummary(profile)
	if entity.Status == "" {
		entity.Status = "active"
	}
	if entity.Importance <= 0 {
		entity.Importance = characterImportance(profile.Role)
	}
	traits := map[string]string{}
	for key, value := range entity.Traits {
		traits[key] = value
	}
	traits["role"] = profile.Role
	traits["desire"] = profile.Desire
	traits["wound"] = profile.Wound
	if profile.Secret != "" {
		traits["secret"] = profile.Secret
	} else {
		delete(traits, "secret")
	}
	if profile.Summary != "" {
		traits["summary"] = profile.Summary
	}
	entity.Traits = traits
	metadata := map[string]string{}
	for key, value := range entity.Metadata {
		metadata[key] = value
	}
	for key, value := range input.Metadata {
		metadata["sync_"+key] = value
	}
	if strings.TrimSpace(input.StoryBibleID) != "" {
		metadata["story_bible_id"] = strings.TrimSpace(input.StoryBibleID)
	}
	if strings.TrimSpace(input.Source) != "" {
		metadata["source"] = strings.TrimSpace(input.Source)
	}
	metadata["source_layer"] = "story_bible"
	metadata["character_profile_json"] = mustMarshalCharacterProfile(profile)
	entity.Metadata = metadata
	return entity
}

func characterProfileSummary(profile domain.CharacterProfile) string {
	if strings.TrimSpace(profile.Summary) != "" {
		return strings.TrimSpace(profile.Summary)
	}
	parts := []string{
		fmt.Sprintf("角色定位：%s", profile.Role),
		fmt.Sprintf("欲望：%s", profile.Desire),
		fmt.Sprintf("创伤：%s", profile.Wound),
	}
	if strings.TrimSpace(profile.Secret) != "" {
		parts = append(parts, fmt.Sprintf("秘密：%s", profile.Secret))
	}
	return strings.Join(parts, "；") + "。"
}

func characterImportance(role string) int {
	role = strings.ToLower(strings.TrimSpace(role))
	if strings.Contains(role, "主角") || strings.Contains(role, "protagonist") || strings.Contains(role, "main") {
		return 100
	}
	if strings.Contains(role, "反派") || strings.Contains(role, "antagonist") || strings.Contains(role, "主要") || strings.Contains(role, "配角") {
		return 80
	}
	return 60
}

func mustMarshalCharacterProfile(profile domain.CharacterProfile) string {
	payload, err := json.Marshal(profile)
	if err != nil {
		panic(fmt.Sprintf("marshal character profile: %v", err))
	}
	return string(payload)
}

func normalizedCharacterName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
