package postgres

import (
	"errors"
	"strings"
	"testing"
)

func TestNewIDIncludesPrefixAndRandomSuffix(t *testing.T) {
	id, err := newID("provider")
	if err != nil {
		t.Fatalf("newID() error: %v", err)
	}
	if !strings.HasPrefix(id, "provider_") {
		t.Fatalf("expected provider prefix, got %q", id)
	}
	other, err := newID("provider")
	if err != nil {
		t.Fatalf("newID() second error: %v", err)
	}
	if id == other {
		t.Fatalf("expected unique ids")
	}
}

func TestNewIDReturnsRandomReaderError(t *testing.T) {
	wantErr := errors.New("random source unavailable")
	previousReader := randomReader
	randomReader = errorReader{err: wantErr}
	t.Cleanup(func() { randomReader = previousReader })

	id, err := newID("provider")
	if err == nil {
		t.Fatalf("newID() error = nil, want error")
	}
	if id != "" {
		t.Fatalf("newID() id = %q, want empty", id)
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("newID() error = %v, want wrapped %v", err, wantErr)
	}
	if !strings.Contains(err.Error(), "generate random id bytes") {
		t.Fatalf("newID() error lacks context: %v", err)
	}
}

type errorReader struct {
	err error
}

func (r errorReader) Read(_ []byte) (int, error) {
	return 0, r.err
}

func TestJSONBHelpersRoundTripMap(t *testing.T) {
	data, err := jsonbOrEmptyObject(map[string]string{"mode": "test"})
	if err != nil {
		t.Fatalf("jsonbOrEmptyObject() error: %v", err)
	}
	value, err := unmarshalJSONB[map[string]string](data)
	if err != nil {
		t.Fatalf("unmarshalJSONB() error: %v", err)
	}
	if value["mode"] != "test" {
		t.Fatalf("unexpected round-trip value: %+v", value)
	}
}

func TestJSONBHelpersDefaultArray(t *testing.T) {
	data, err := jsonbOrEmptyArray(nil)
	if err != nil {
		t.Fatalf("jsonbOrEmptyArray() error: %v", err)
	}
	value, err := unmarshalJSONB[[]string](data)
	if err != nil {
		t.Fatalf("unmarshalJSONB() error: %v", err)
	}
	if len(value) != 0 {
		t.Fatalf("expected empty array, got %+v", value)
	}
}
