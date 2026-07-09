package dto

import (
	"time"
)

type StoryBibleCharacterDTO struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Role       string            `json:"role"`
	Desire     string            `json:"desire"`
	Wound      string            `json:"wound"`
	Secret     string            `json:"secret,omitempty"`
	Summary    string            `json:"summary,omitempty"`
	EntityID   string            `json:"entity_id,omitempty"`
	SyncStatus string            `json:"sync_status,omitempty"`
	SyncedAt   string            `json:"synced_at,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type StoryBibleForeshadowDTO struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	PlantedIn  string `json:"planted_in"`
	PayoffHint string `json:"payoff_hint"`
	Status     string `json:"status"`
}

type StoryBibleChapterPlanDTO struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Status  string `json:"status"`
	Summary string `json:"summary"`
}

type StoryBibleDTO struct {
	ID                string                     `json:"id"`
	ProjectID         string                     `json:"project_id"`
	Version           int                        `json:"version"`
	Title             string                     `json:"title"`
	Logline           string                     `json:"logline"`
	Synopsis          string                     `json:"synopsis"`
	Genre             string                     `json:"genre"`
	Tone              string                     `json:"tone"`
	Audience          string                     `json:"audience"`
	Language          string                     `json:"language"`
	Rules             map[string]string          `json:"rules,omitempty"`
	WorldlineIDs      []string                   `json:"worldline_ids,omitempty"`
	EntityIDs         []string                   `json:"entity_ids,omitempty"`
	PlotThreadIDs     []string                   `json:"plot_thread_ids,omitempty"`
	GenesisWorkflowID string                     `json:"genesis_workflow_id,omitempty"`
	Approved          bool                       `json:"approved"`
	CreatedAt         time.Time                  `json:"created_at"`
	Premise           string                     `json:"premise"`
	Themes            []string                   `json:"themes"`
	WorldRules        []string                   `json:"world_rules"`
	Characters        []StoryBibleCharacterDTO   `json:"characters"`
	Foreshadows       []StoryBibleForeshadowDTO  `json:"foreshadows"`
	ChapterPlan       []StoryBibleChapterPlanDTO `json:"chapter_plan"`
	Chapters          []StoryBibleChapterPlanDTO `json:"chapters,omitempty"`
	SourceSeed        ProjectSeedDTO             `json:"source_seed"`
}

type CharacterSyncRequestDTO struct {
	StoryBibleID string                   `json:"story_bible_id,omitempty"`
	Source       string                   `json:"source,omitempty"`
	Characters   []StoryBibleCharacterDTO `json:"characters"`
	Metadata     map[string]string        `json:"metadata,omitempty"`
}

type CharacterSyncResponseDTO struct {
	ProjectID    string                       `json:"project_id"`
	StoryBibleID string                       `json:"story_bible_id,omitempty"`
	Characters   []EntityDTO                  `json:"characters"`
	Mappings     []CharacterProfileMappingDTO `json:"mappings"`
}

type CharacterProfileMappingDTO struct {
	Name     string `json:"name"`
	EntityID string `json:"entity_id"`
	Action   string `json:"action"`
}
