package dto

import (
	"time"
)

type WorldlineDTO struct {
	ID        string            `json:"id"`
	ProjectID string            `json:"project_id"`
	Name      string            `json:"name"`
	Summary   string            `json:"summary"`
	Canonical bool              `json:"canonical"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type EntityDTO struct {
	ID          string            `json:"id"`
	ProjectID   string            `json:"project_id"`
	WorldlineID string            `json:"worldline_id,omitempty"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Aliases     []string          `json:"aliases,omitempty"`
	Summary     string            `json:"summary"`
	Traits      map[string]string `json:"traits,omitempty"`
	Importance  int               `json:"importance"`
	Status      string            `json:"status"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type FactDTO struct {
	ID               string            `json:"id"`
	ProjectID        string            `json:"project_id"`
	WorldlineID      string            `json:"worldline_id,omitempty"`
	EntityID         string            `json:"entity_id,omitempty"`
	ChapterID        string            `json:"chapter_id,omitempty"`
	ChapterVersionID string            `json:"chapter_version_id,omitempty"`
	Claim            string            `json:"claim"`
	Source           string            `json:"source"`
	Confidence       float64           `json:"confidence"`
	Status           string            `json:"status"`
	EmbeddingRef     string            `json:"embedding_ref,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

type GraphEdgeDTO struct {
	ID              string            `json:"id"`
	ProjectID       string            `json:"project_id"`
	WorldlineID     string            `json:"worldline_id,omitempty"`
	SourceEntityID  string            `json:"source_entity_id"`
	TargetEntityID  string            `json:"target_entity_id"`
	Type            string            `json:"type"`
	Label           string            `json:"label"`
	Weight          float64           `json:"weight"`
	EvidenceFactIDs []string          `json:"evidence_fact_ids,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

type PlotThreadDTO struct {
	ID               string            `json:"id"`
	ProjectID        string            `json:"project_id"`
	WorldlineID      string            `json:"worldline_id,omitempty"`
	Title            string            `json:"title"`
	Summary          string            `json:"summary"`
	Status           string            `json:"status"`
	Priority         int               `json:"priority"`
	RelatedEntityIDs []string          `json:"related_entity_ids,omitempty"`
	OpenedChapterID  string            `json:"opened_chapter_id,omitempty"`
	ClosedChapterID  string            `json:"closed_chapter_id,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

type GraphExpansionDTO struct {
	ProjectID   string         `json:"project_id"`
	Depth       int            `json:"depth"`
	Entities    []EntityDTO    `json:"entities"`
	Edges       []GraphEdgeDTO `json:"edges"`
	Facts       []FactDTO      `json:"facts"`
	GeneratedAt time.Time      `json:"generated_at"`
}

type GraphExpansionRequestDTO struct {
	EntityIDs []string `json:"entity_ids,omitempty"`
	Depth     int      `json:"depth,omitempty"`
}
