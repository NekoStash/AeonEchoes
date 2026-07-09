package dto

import (
	"time"
)

type HealthDTO struct {
	Status             string    `json:"status"`
	Time               time.Time `json:"time"`
	QdrantConfigured   bool      `json:"qdrant_configured"`
	PostgresConfigured bool      `json:"postgres_configured"`
}

type SystemStatusDTO struct {
	Status             string    `json:"status"`
	PostgresConfigured bool      `json:"postgres_configured"`
	QdrantConfigured   bool      `json:"qdrant_configured"`
	ProviderCount      int       `json:"provider_count"`
	ModelCount         int       `json:"model_count"`
	PendingJobsCount   int       `json:"pending_jobs_count"`
	CheckedAt          time.Time `json:"checked_at"`
}
