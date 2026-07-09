package dto

import (
	"time"
)

type IndexJobDTO struct {
	ID               string            `json:"id"`
	ProjectID        string            `json:"project_id"`
	ChapterID        string            `json:"chapter_id,omitempty"`
	ChapterVersionID string            `json:"chapter_version_id,omitempty"`
	Kind             string            `json:"kind"`
	Status           string            `json:"status"`
	Attempts         int               `json:"attempts"`
	Error            string            `json:"error,omitempty"`
	Payload          map[string]string `json:"payload,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	ScheduledAt      *time.Time        `json:"scheduled_at,omitempty"`
	StartedAt        *time.Time        `json:"started_at,omitempty"`
	CompletedAt      *time.Time        `json:"completed_at,omitempty"`
}

type RunPendingIndexDTO struct {
	Processed []IndexJobDTO `json:"processed"`
	Count     int           `json:"count"`
	Error     string        `json:"error,omitempty"`
}

type RebuildVectorsDTO struct {
	EmbeddingModelID    string `json:"embedding_model_id"`
	EmbeddingModelName  string `json:"embedding_model_name"`
	EmbeddingDimension  int    `json:"embedding_dimension"`
	ProjectCount        int    `json:"project_count"`
	ChapterVersionCount int    `json:"chapter_version_count"`
	JobCount            int    `json:"job_count"`
}
