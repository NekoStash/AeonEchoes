package vector

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"aeonechoes/server/internal/domain"
)

// QdrantClient is a minimal HTTP client for Qdrant collection and point APIs.
type QdrantClient struct {
	baseURL    string
	apiKey     string
	collection string
	httpClient *http.Client
}

type PointPayload struct {
	ProjectID        string `json:"project_id"`
	ChapterID        string `json:"chapter_id"`
	ChapterVersionID string `json:"chapter_version_id"`
	ContentType      string `json:"content_type"`
	SourceID         string `json:"source_id"`
	CanonStatus      string `json:"canon_status"`
}

func NewQdrantClient(baseURL, apiKey, collection string, httpClient *http.Client) (*QdrantClient, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	collection = strings.TrimSpace(collection)
	if baseURL == "" {
		return nil, fmt.Errorf("qdrant base URL must not be empty")
	}
	if collection == "" {
		collection = "aeonechoes_context"
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &QdrantClient{baseURL: baseURL, apiKey: strings.TrimSpace(apiKey), collection: collection, httpClient: httpClient}, nil
}

func (c *QdrantClient) EnsureCollection(ctx context.Context, dimension int) error {
	if err := c.requireConfigured(); err != nil {
		return err
	}
	if dimension <= 0 {
		return fmt.Errorf("qdrant collection vector dimension must be positive")
	}
	payload := map[string]any{"vectors": map[string]any{"size": dimension, "distance": "Cosine"}}
	path := fmt.Sprintf("/collections/%s", c.collection)
	if err := c.doJSON(ctx, http.MethodPut, path, payload, nil); err != nil {
		var statusErr *qdrantStatusError
		if !errors.As(err, &statusErr) || statusErr.statusCode != http.StatusConflict {
			return err
		}
		if verifyErr := c.ensureExistingCollectionDimension(ctx, dimension); verifyErr != nil {
			return fmt.Errorf("qdrant collection %q already exists but is not compatible: %w", c.collection, verifyErr)
		}
	}
	return nil
}

func (c *QdrantClient) RecreateCollection(ctx context.Context, dimension int) error {
	if err := c.requireConfigured(); err != nil {
		return err
	}
	if dimension <= 0 {
		return fmt.Errorf("qdrant collection vector dimension must be positive")
	}
	path := fmt.Sprintf("/collections/%s", c.collection)
	if err := c.doJSON(ctx, http.MethodDelete, path, nil, nil); err != nil {
		var statusErr *qdrantStatusError
		if !errors.As(err, &statusErr) || statusErr.statusCode != http.StatusNotFound {
			return err
		}
	}
	return c.EnsureCollection(ctx, dimension)
}

func (c *QdrantClient) UpsertTextVector(ctx context.Context, pointID string, vector []float64, payload PointPayload) error {
	if err := c.requireConfigured(); err != nil {
		return err
	}
	if strings.TrimSpace(pointID) == "" {
		return fmt.Errorf("qdrant point id must not be empty")
	}
	if len(vector) == 0 {
		return fmt.Errorf("qdrant vector must not be empty")
	}
	if strings.TrimSpace(payload.ProjectID) == "" || strings.TrimSpace(payload.SourceID) == "" {
		return fmt.Errorf("qdrant payload project_id and source_id must not be empty")
	}
	qdrantPointID := deterministicQdrantPointID(pointID)
	body := map[string]any{"points": []map[string]any{{"id": qdrantPointID, "vector": vector, "payload": payload}}}
	return c.doJSON(ctx, http.MethodPut, fmt.Sprintf("/collections/%s/points?wait=true", c.collection), body, nil)
}

func (c *QdrantClient) DeleteBySource(ctx context.Context, sourceID string) error {
	if err := c.requireConfigured(); err != nil {
		return err
	}
	if strings.TrimSpace(sourceID) == "" {
		return fmt.Errorf("qdrant delete source_id must not be empty")
	}
	body := map[string]any{"filter": map[string]any{"must": []map[string]any{{"key": "source_id", "match": map[string]string{"value": sourceID}}}}}
	return c.doJSON(ctx, http.MethodPost, fmt.Sprintf("/collections/%s/points/delete?wait=true", c.collection), body, nil)
}

func (c *QdrantClient) Search(ctx context.Context, vector []float64, projectID string, limit int) ([]domain.SemanticSearchItem, error) {
	if err := c.requireConfigured(); err != nil {
		return nil, err
	}
	if len(vector) == 0 {
		return nil, fmt.Errorf("qdrant search vector must not be empty")
	}
	cleanProjectID := strings.TrimSpace(projectID)
	if cleanProjectID == "" {
		return nil, fmt.Errorf("qdrant search project_id must not be empty")
	}
	if limit <= 0 {
		limit = 10
	}
	body := map[string]any{
		"vector":       vector,
		"limit":        limit,
		"with_payload": true,
		"filter":       map[string]any{"must": []map[string]any{{"key": "project_id", "match": map[string]string{"value": cleanProjectID}}}},
	}
	var response struct {
		Result []struct {
			ID      any            `json:"id"`
			Score   float64        `json:"score"`
			Payload map[string]any `json:"payload"`
		} `json:"result"`
	}
	if err := c.doJSON(ctx, http.MethodPost, fmt.Sprintf("/collections/%s/points/search", c.collection), body, &response); err != nil {
		return nil, err
	}
	items := make([]domain.SemanticSearchItem, 0, len(response.Result))
	for _, hit := range response.Result {
		sourceID := ""
		if payloadSourceID, ok := hit.Payload["source_id"].(string); ok {
			sourceID = strings.TrimSpace(payloadSourceID)
		}
		if sourceID == "" {
			sourceID = qdrantIDString(hit.ID)
		}
		items = append(items, domain.SemanticSearchItem{SourceID: sourceID, Score: hit.Score, Payload: hit.Payload})
	}
	return items, nil
}

func (c *QdrantClient) Health(ctx context.Context) error {
	if err := c.requireConfigured(); err != nil {
		return err
	}
	return c.doJSON(ctx, http.MethodGet, "/healthz", nil, nil)
}

func (c *QdrantClient) CollectionName() string {
	if c == nil {
		return ""
	}
	return c.collection
}

func (c *QdrantClient) ensureExistingCollectionDimension(ctx context.Context, dimension int) error {
	var response qdrantCollectionResponse
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("/collections/%s", c.collection), nil, &response); err != nil {
		return fmt.Errorf("get existing qdrant collection config: %w", err)
	}
	existingDimension, err := response.VectorDimension()
	if err != nil {
		return fmt.Errorf("read existing qdrant collection vector dimension: %w", err)
	}
	if existingDimension != dimension {
		return fmt.Errorf("vector dimension mismatch: existing collection has dimension %d, requested %d", existingDimension, dimension)
	}
	return nil
}

func (c *QdrantClient) requireConfigured() error {
	if c == nil || strings.TrimSpace(c.baseURL) == "" || strings.TrimSpace(c.collection) == "" || c.httpClient == nil {
		return fmt.Errorf("qdrant client is not configured")
	}
	return nil
}

func deterministicQdrantPointID(pointID string) string {
	sum := sha256.Sum256([]byte(pointID))
	uuidBytes := sum[:16]
	uuidBytes[6] = (uuidBytes[6] & 0x0f) | 0x40
	uuidBytes[8] = (uuidBytes[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", uuidBytes[0:4], uuidBytes[4:6], uuidBytes[6:8], uuidBytes[8:10], uuidBytes[10:16])
}

func qdrantIDString(id any) string {
	switch value := id.(type) {
	case string:
		return value
	case float64:
		return strconv.FormatInt(int64(value), 10)
	case json.Number:
		return value.String()
	default:
		return fmt.Sprintf("%v", value)
	}
}

type qdrantCollectionResponse struct {
	Result struct {
		Config struct {
			Params struct {
				Vectors json.RawMessage `json:"vectors"`
			} `json:"params"`
		} `json:"config"`
	} `json:"result"`
}

func (r qdrantCollectionResponse) VectorDimension() (int, error) {
	vectors := bytes.TrimSpace(r.Result.Config.Params.Vectors)
	if len(vectors) == 0 || bytes.Equal(vectors, []byte("null")) {
		return 0, fmt.Errorf("collection config missing vectors params")
	}

	var unnamed struct {
		Size int `json:"size"`
	}
	if err := json.Unmarshal(vectors, &unnamed); err != nil {
		return 0, fmt.Errorf("decode vectors params: %w", err)
	}
	if unnamed.Size <= 0 {
		return 0, fmt.Errorf("collection config vectors.size is missing or invalid: %d", unnamed.Size)
	}
	return unnamed.Size, nil
}

type qdrantStatusError struct {
	method     string
	path       string
	statusCode int
	body       string
}

func (e *qdrantStatusError) Error() string {
	return fmt.Sprintf("qdrant %s %s returned status %d: %s", e.method, e.path, e.statusCode, e.body)
}

func (c *QdrantClient) doJSON(ctx context.Context, method, path string, payload any, out any) error {
	var body io.Reader = http.NoBody
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal qdrant request: %w", err)
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("create qdrant request: %w", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.apiKey != "" {
		req.Header.Set("api-key", c.apiKey)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("qdrant %s %s failed: %w", method, path, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read qdrant response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &qdrantStatusError{method: method, path: path, statusCode: resp.StatusCode, body: strings.TrimSpace(string(data))}
	}
	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("decode qdrant response: %w", err)
		}
	}
	return nil
}
