package vector

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestQdrantUpsertTextVectorPayload(t *testing.T) {
	var capturedPath string
	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.String()
		if r.Header.Get("api-key") != "secret" {
			t.Fatalf("missing api-key header")
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":true}`))
	}))
	defer server.Close()

	client, err := NewQdrantClient(server.URL, "secret", "chapters", server.Client())
	if err != nil {
		t.Fatalf("NewQdrantClient() error: %v", err)
	}
	err = client.UpsertTextVector(context.Background(), "cv_1", []float64{0.1, 0.2}, PointPayload{ProjectID: "p1", ChapterID: "c1", ChapterVersionID: "cv_1", ContentType: "chapter_version", SourceID: "cv_1", CanonStatus: "pending"})
	if err != nil {
		t.Fatalf("UpsertTextVector() error: %v", err)
	}
	if capturedPath != "/collections/chapters/points?wait=true" {
		t.Fatalf("unexpected path %q", capturedPath)
	}
	points, ok := captured["points"].([]any)
	if !ok || len(points) != 1 {
		t.Fatalf("unexpected points payload: %+v", captured)
	}
	point, ok := points[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected point payload: %+v", points[0])
	}
	pointID, ok := point["id"].(string)
	if !ok || !isUUID(pointID) {
		t.Fatalf("expected UUID point id, got: %+v", point)
	}
	payload, ok := point["payload"].(map[string]any)
	if !ok || payload["project_id"] != "p1" || payload["source_id"] != "cv_1" {
		t.Fatalf("unexpected payload: %+v", point["payload"])
	}
}

func TestQdrantEnsureCollectionTreatsConflictWithMatchingDimensionAsSuccess(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.String())
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/collections/chapters":
			http.Error(w, `{"status":{"error":"Collection already exists"}}`, http.StatusConflict)
		case r.Method == http.MethodGet && r.URL.Path == "/collections/chapters":
			_, _ = w.Write([]byte(`{"result":{"config":{"params":{"vectors":{"size":1536,"distance":"Cosine"}}}}}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client, err := NewQdrantClient(server.URL, "", "chapters", server.Client())
	if err != nil {
		t.Fatalf("NewQdrantClient() error: %v", err)
	}
	if err := client.EnsureCollection(context.Background(), 1536); err != nil {
		t.Fatalf("EnsureCollection() error: %v", err)
	}
	assertRequests(t, requests, []string{"PUT /collections/chapters", "GET /collections/chapters"})
}

func TestQdrantEnsureCollectionReturnsErrorForConflictWithMismatchedDimension(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.String())
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/collections/chapters":
			http.Error(w, `{"status":{"error":"Collection already exists"}}`, http.StatusConflict)
		case r.Method == http.MethodGet && r.URL.Path == "/collections/chapters":
			_, _ = w.Write([]byte(`{"result":{"config":{"params":{"vectors":{"size":768,"distance":"Cosine"}}}}}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client, err := NewQdrantClient(server.URL, "", "chapters", server.Client())
	if err != nil {
		t.Fatalf("NewQdrantClient() error: %v", err)
	}
	err = client.EnsureCollection(context.Background(), 1536)
	if err == nil {
		t.Fatalf("EnsureCollection() expected error")
	}
	if !strings.Contains(err.Error(), "vector dimension mismatch") || !strings.Contains(err.Error(), "existing collection has dimension 768, requested 1536") {
		t.Fatalf("EnsureCollection() error should describe dimension mismatch, got: %v", err)
	}
	assertRequests(t, requests, []string{"PUT /collections/chapters", "GET /collections/chapters"})
}

func TestQdrantRecreateCollectionDeletesThenEnsures(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.String())
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodDelete && r.URL.Path == "/collections/chapters":
			_, _ = w.Write([]byte(`{"result":true}`))
		case r.Method == http.MethodPut && r.URL.Path == "/collections/chapters":
			_, _ = w.Write([]byte(`{"result":true}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client, err := NewQdrantClient(server.URL, "", "chapters", server.Client())
	if err != nil {
		t.Fatalf("NewQdrantClient() error: %v", err)
	}
	if err := client.RecreateCollection(context.Background(), 1536); err != nil {
		t.Fatalf("RecreateCollection() error: %v", err)
	}
	assertRequests(t, requests, []string{"DELETE /collections/chapters", "PUT /collections/chapters"})
}

func TestQdrantRecreateCollectionTreatsDeleteNotFoundAsSuccess(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.String())
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodDelete && r.URL.Path == "/collections/chapters":
			http.Error(w, `{"status":{"error":"Not found"}}`, http.StatusNotFound)
		case r.Method == http.MethodPut && r.URL.Path == "/collections/chapters":
			_, _ = w.Write([]byte(`{"result":true}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client, err := NewQdrantClient(server.URL, "", "chapters", server.Client())
	if err != nil {
		t.Fatalf("NewQdrantClient() error: %v", err)
	}
	if err := client.RecreateCollection(context.Background(), 1536); err != nil {
		t.Fatalf("RecreateCollection() error: %v", err)
	}
	assertRequests(t, requests, []string{"DELETE /collections/chapters", "PUT /collections/chapters"})
}

func TestDeterministicQdrantPointID(t *testing.T) {
	first := deterministicQdrantPointID("cv_1")
	second := deterministicQdrantPointID("cv_1")
	other := deterministicQdrantPointID("cv_2")
	if first != second {
		t.Fatalf("expected stable mapped point ID, got %q and %q", first, second)
	}
	if !isUUID(first) {
		t.Fatalf("expected UUID point ID, got %q", first)
	}
	if first == other {
		t.Fatalf("expected distinct mapped point IDs for different business IDs, got %q", first)
	}
	if !isUUID(other) {
		t.Fatalf("expected UUID point ID for other business ID, got %q", other)
	}
}

func TestQdrantSearchBuildsProjectFilterAndReturnsItems(t *testing.T) {
	var capturedPath string
	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.String()
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method %s", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":[{"id":"fallback-id","score":0.91,"payload":{"project_id":"p1","source_id":"cv_1","chapter_id":"c1"}}]}`))
	}))
	defer server.Close()

	client, err := NewQdrantClient(server.URL, "", "chapters", server.Client())
	if err != nil {
		t.Fatalf("NewQdrantClient() error: %v", err)
	}
	items, err := client.Search(context.Background(), []float64{0.9, 0.1}, "p1", 7)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}
	if capturedPath != "/collections/chapters/points/search" {
		t.Fatalf("unexpected path %q", capturedPath)
	}
	if captured["limit"].(float64) != 7 {
		t.Fatalf("unexpected limit payload: %+v", captured)
	}
	if captured["with_payload"] != true {
		t.Fatalf("expected with_payload=true, got %+v", captured)
	}
	filter, ok := captured["filter"].(map[string]any)
	if !ok {
		t.Fatalf("missing filter: %+v", captured)
	}
	must, ok := filter["must"].([]any)
	if !ok || len(must) != 1 {
		t.Fatalf("unexpected filter.must: %+v", filter)
	}
	clause, ok := must[0].(map[string]any)
	if !ok || clause["key"] != "project_id" {
		t.Fatalf("unexpected filter clause: %+v", must[0])
	}
	match, ok := clause["match"].(map[string]any)
	if !ok || match["value"] != "p1" {
		t.Fatalf("unexpected match clause: %+v", clause)
	}
	if len(items) != 1 || items[0].SourceID != "cv_1" || items[0].Score != 0.91 {
		t.Fatalf("unexpected items: %+v", items)
	}
}

func assertRequests(t *testing.T, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("unexpected requests: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected requests: got %v, want %v", got, want)
		}
	}
}

func isUUID(value string) bool {
	return regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`).MatchString(value)
}
