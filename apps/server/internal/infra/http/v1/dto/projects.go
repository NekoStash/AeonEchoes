package dto

import (
	"time"
)

type ProjectSeedDTO struct {
	Title          string            `json:"title"`
	Premise        string            `json:"premise"`
	Genre          string            `json:"genre"`
	Tone           string            `json:"tone"`
	Audience       string            `json:"audience"`
	Language       string            `json:"language"`
	Setting        string            `json:"setting"`
	Themes         []string          `json:"themes,omitempty"`
	MainCharacters []string          `json:"main_characters,omitempty"`
	Constraints    []string          `json:"constraints,omitempty"`
	TargetChapters int               `json:"target_chapters"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

type ProjectDTO struct {
	ID                 string            `json:"id"`
	Title              string            `json:"title"`
	Slug               string            `json:"slug"`
	Status             string            `json:"status"`
	Seed               ProjectSeedDTO    `json:"seed"`
	ActiveStoryBibleID string            `json:"active_story_bible_id,omitempty"`
	DefaultWorldlineID string            `json:"default_worldline_id,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
}

type InitializeProjectDTO struct {
	Project    ProjectDTO    `json:"project"`
	StoryBible StoryBibleDTO `json:"story_bible"`
	Workflow   WorkflowDTO   `json:"workflow"`
}
