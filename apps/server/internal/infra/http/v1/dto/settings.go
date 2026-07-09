package dto

import (
	"time"
)

type AppSettingDTO struct {
	Scope     string         `json:"scope"`
	Key       string         `json:"key"`
	Value     map[string]any `json:"value"`
	UpdatedAt time.Time      `json:"updated_at"`
}
