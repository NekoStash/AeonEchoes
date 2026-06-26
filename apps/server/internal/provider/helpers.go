package provider

import (
	"encoding/json"
	"strings"

	"aeonechoes/server/internal/domain"
)

// InferModelKind infers a model kind from a model identifier when the remote API does not provide an explicit kind.
func InferModelKind(modelID string) domain.ModelKind {
	id := strings.ToLower(strings.TrimSpace(modelID))
	if id == "" {
		return domain.ModelKindText
	}
	if strings.Contains(id, "embed") || strings.Contains(id, "embedding") {
		return domain.ModelKindEmbedding
	}
	return domain.ModelKindText
}

// InferModelKindFromMethods uses capability method names to identify embedding models.
func InferModelKindFromMethods(methods []string) domain.ModelKind {
	for _, method := range methods {
		m := strings.ToLower(strings.TrimSpace(method))
		if strings.Contains(m, "embed") {
			return domain.ModelKindEmbedding
		}
	}
	return domain.ModelKindText
}

// UsageFromMap converts a generic usage object into the normalized usage structure.
func UsageFromMap(raw map[string]any) Usage {
	var usage Usage
	if raw == nil {
		return usage
	}
	usage.InputTokens = int(asFloat(raw["input_tokens"]))
	if usage.InputTokens == 0 {
		usage.InputTokens = int(asFloat(raw["prompt_tokens"]))
	}
	usage.OutputTokens = int(asFloat(raw["output_tokens"]))
	if usage.OutputTokens == 0 {
		usage.OutputTokens = int(asFloat(raw["completion_tokens"]))
	}
	usage.TotalTokens = int(asFloat(raw["total_tokens"]))
	if usage.TotalTokens == 0 {
		usage.TotalTokens = usage.InputTokens + usage.OutputTokens
	}
	return usage
}

func asFloat(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case int32:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	default:
		return 0
	}
}
