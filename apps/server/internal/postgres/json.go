package postgres

import (
	"encoding/json"
	"fmt"
)

func marshalJSONB(value any) ([]byte, error) {
	if value == nil {
		return []byte("null"), nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal jsonb value: %w", err)
	}
	return data, nil
}

func unmarshalJSONB[T any](data []byte) (T, error) {
	var value T
	if len(data) == 0 {
		return value, nil
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return value, fmt.Errorf("unmarshal jsonb value: %w", err)
	}
	return value, nil
}

func jsonbOrEmptyObject(value any) ([]byte, error) {
	if value == nil {
		return []byte("{}"), nil
	}
	return marshalJSONB(value)
}

func jsonbOrEmptyArray(value any) ([]byte, error) {
	if value == nil {
		return []byte("[]"), nil
	}
	return marshalJSONB(value)
}
