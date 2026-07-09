package dto

import (
	"aeonechoes/server/internal/domain"
	"time"
)

type ChapterDTO struct {
	ID        string            `json:"id"`
	ProjectID string            `json:"project_id"`
	Number    int               `json:"number"`
	Title     string            `json:"title"`
	Status    string            `json:"status"`
	Summary   string            `json:"summary"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type EnsureChapterRequestDTO struct {
	ChapterID string            `json:"chapter_id,omitempty"`
	Number    int               `json:"number,omitempty"`
	Title     string            `json:"title,omitempty"`
	Status    string            `json:"status,omitempty"`
	Summary   string            `json:"summary,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type EnsureChapterResponseDTO struct {
	Chapter            ChapterDTO `json:"chapter"`
	Created            bool       `json:"created"`
	RequestedChapterID string     `json:"requested_chapter_id,omitempty"`
}

type ChapterVersionDTO struct {
	ID               string            `json:"id"`
	ProjectID        string            `json:"project_id"`
	ChapterID        string            `json:"chapter_id"`
	Version          int               `json:"version"`
	Title            string            `json:"title"`
	Content          string            `json:"content"`
	Summary          string            `json:"summary"`
	AuthorRole       domain.AgentRole  `json:"author_role"`
	SourceWorkflowID string            `json:"source_workflow_id,omitempty"`
	IndexStatus      string            `json:"index_status"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
}

type ChapterVersionRequestDTO struct {
	ID               string            `json:"id,omitempty"`
	Title            string            `json:"title"`
	Content          string            `json:"content"`
	Summary          string            `json:"summary,omitempty"`
	AuthorRole       domain.AgentRole  `json:"author_role"`
	SourceWorkflowID string            `json:"source_workflow_id,omitempty"`
	IndexStatus      string            `json:"index_status,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

type SaveChapterVersionResponseDTO struct {
	ChapterVersion ChapterVersionDTO `json:"chapter_version"`
	IndexJob       IndexJobDTO       `json:"index_job"`
}
