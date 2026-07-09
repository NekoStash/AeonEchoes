package dto

type SemanticSearchRequestDTO struct {
	Query     string            `json:"query"`
	ProjectID string            `json:"project_id"`
	Limit     int               `json:"limit,omitempty"`
	Filters   map[string]string `json:"filters,omitempty"`
}

type SemanticSearchItemDTO struct {
	SourceID string         `json:"source_id"`
	Score    float64        `json:"score"`
	Payload  map[string]any `json:"payload,omitempty"`
}

type SemanticSearchResultDTO struct {
	Query     string                  `json:"query"`
	ProjectID string                  `json:"project_id"`
	Items     []SemanticSearchItemDTO `json:"items"`
}
