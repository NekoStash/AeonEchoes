package dto

import (
	"aeonechoes/server/internal/domain"
	"time"
)

type ChapterDTO struct {
	ID        string               `json:"id"`
	ProjectID string               `json:"project_id"`
	Number    int                  `json:"number"`
	Title     string               `json:"title"`
	Status    domain.ChapterStatus `json:"status"`
	Summary   string               `json:"summary"`
	Metadata  map[string]string    `json:"metadata,omitempty"`
	CreatedAt time.Time            `json:"created_at"`
	UpdatedAt time.Time            `json:"updated_at"`
}

type CreateChapterRequestDTO struct {
	Number   int                  `json:"number,omitempty"`
	Title    string               `json:"title"`
	Status   domain.ChapterStatus `json:"status,omitempty"`
	Summary  string               `json:"summary,omitempty"`
	Metadata map[string]string    `json:"metadata,omitempty"`
}

type UpdateChapterRequestDTO struct {
	Number   *int                  `json:"number,omitempty"`
	Title    *string               `json:"title,omitempty"`
	Status   *domain.ChapterStatus `json:"status,omitempty"`
	Summary  *string               `json:"summary,omitempty"`
	Metadata *map[string]string    `json:"metadata,omitempty"`
}

func (r UpdateChapterRequestDTO) HasChanges() bool {
	return r.Number != nil || r.Title != nil || r.Status != nil || r.Summary != nil || r.Metadata != nil
}

type ChapterVersionDTO struct {
	ID               string            `json:"id"`
	ProjectID        string            `json:"project_id"`
	ChapterID        string            `json:"chapter_id"`
	ParentVersionID  string            `json:"parent_version_id,omitempty"`
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
	Title           string            `json:"title"`
	Content         string            `json:"content"`
	AuthorRole      domain.AgentRole  `json:"author_role"`
	Summary         string            `json:"summary,omitempty"`
	ChangeNote      string            `json:"change_note,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	ParentVersionID string            `json:"parent_version_id,omitempty"`
}

type SaveChapterVersionResponseDTO struct {
	ChapterVersion ChapterVersionDTO `json:"chapter_version"`
	IndexJob       IndexJobDTO       `json:"index_job"`
}
