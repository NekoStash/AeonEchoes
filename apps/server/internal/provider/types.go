package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"aeonechoes/server/internal/domain"
)

// Usage captures token accounting returned by provider APIs.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// ToolCall captures tool/function call data emitted by text providers.
type ToolCall struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// StreamEvent is a normalized streaming event.
type StreamEvent struct {
	Type     string         `json:"type"`
	Delta    string         `json:"delta,omitempty"`
	Response *ModelResponse `json:"response,omitempty"`
	ToolCall *ToolCall      `json:"tool_call,omitempty"`
	Usage    *Usage         `json:"usage,omitempty"`
	Done     bool           `json:"done"`
	Error    string         `json:"error,omitempty"`
}

// Message normalizes chat-style provider messages, including assistant tool calls and
// tool result history needed by provider-neutral tool-call loops.
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	Name       string     `json:"name,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

// MessageContent returns a text representation for providers that do not expose
// native tool-result history in the current adapter. Native adapters can still
// inspect ToolCalls and ToolCallID directly without losing normalized history.
func MessageContent(msg Message) string {
	content := strings.TrimSpace(msg.Content)
	role := strings.ToLower(strings.TrimSpace(msg.Role))
	if role == "tool" {
		label := strings.TrimSpace(msg.Name)
		if label == "" {
			label = "tool"
		}
		if strings.TrimSpace(msg.ToolCallID) != "" {
			label += " call_id=" + strings.TrimSpace(msg.ToolCallID)
		}
		if content == "" {
			content = "{}"
		}
		return fmt.Sprintf("Tool result for %s:\n%s", label, content)
	}
	if content != "" {
		return content
	}
	if len(msg.ToolCalls) == 0 {
		return ""
	}
	payload, err := json.Marshal(msg.ToolCalls)
	if err != nil {
		return ""
	}
	return "Assistant requested tool calls: " + string(payload)
}

// ToolSpec describes a callable tool to upstream providers.
type ToolSpec struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

// TextRequest is the normalized text generation request.
type TextRequest struct {
	Model           string            `json:"model"`
	Messages        []Message         `json:"messages,omitempty"`
	SystemPrompt    string            `json:"system_prompt,omitempty"`
	UserPrompt      string            `json:"user_prompt,omitempty"`
	Tools           []ToolSpec        `json:"tools,omitempty"`
	Temperature     float64           `json:"temperature,omitempty"`
	TopP            float64           `json:"top_p,omitempty"`
	MaxOutputTokens int               `json:"max_output_tokens,omitempty"`
	Stream          bool              `json:"stream,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

// ModelResponse is the common response envelope returned by text providers.
type ModelResponse struct {
	ID           string          `json:"id,omitempty"`
	Provider     string          `json:"provider,omitempty"`
	Model        string          `json:"model,omitempty"`
	Content      string          `json:"content,omitempty"`
	FinishReason string          `json:"finish_reason,omitempty"`
	ToolCalls    []ToolCall      `json:"tool_calls,omitempty"`
	Usage        Usage           `json:"usage,omitempty"`
	Raw          json.RawMessage `json:"raw,omitempty"`
}

// EmbeddingRequest is the normalized embedding request.
type EmbeddingRequest struct {
	Model    string            `json:"model"`
	Inputs   []string          `json:"inputs"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// EmbeddingResponse is the normalized embedding response.
type EmbeddingResponse struct {
	ID       string          `json:"id,omitempty"`
	Provider string          `json:"provider,omitempty"`
	Model    string          `json:"model,omitempty"`
	Vectors  [][]float64     `json:"vectors,omitempty"`
	Usage    Usage           `json:"usage,omitempty"`
	Raw      json.RawMessage `json:"raw,omitempty"`
}

// ModelInfo describes a model discovered from a provider model listing.
type ModelInfo struct {
	ID                  string              `json:"id"`
	Name                string              `json:"name,omitempty"`
	DisplayName         string              `json:"display_name,omitempty"`
	Provider            domain.ProviderType `json:"provider"`
	Kind                domain.ModelKind    `json:"kind"`
	ContextWindow       int                 `json:"context_window,omitempty"`
	MaxOutputTokens     int                 `json:"max_output_tokens,omitempty"`
	Dimension           int                 `json:"dimension,omitempty"`
	SupportsTools       bool                `json:"supports_tools,omitempty"`
	SupportsToolsKnown  bool                `json:"supports_tools_known,omitempty"`
	SupportsStream      bool                `json:"supports_stream,omitempty"`
	SupportsStreamKnown bool                `json:"supports_stream_known,omitempty"`
	Raw                 json.RawMessage     `json:"raw,omitempty"`
}

// TextModelClient is the abstraction used by agents to request text generation.
type TextModelClient interface {
	Generate(ctx context.Context, req TextRequest) (ModelResponse, error)
	Stream(ctx context.Context, req TextRequest) (<-chan StreamEvent, error)
}

// EmbeddingModelClient is the abstraction used by index and retrieval jobs.
type EmbeddingModelClient interface {
	Embed(ctx context.Context, req EmbeddingRequest) (EmbeddingResponse, error)
}

// ModelListClient is the provider-facing model discovery interface.
type ModelListClient interface {
	ListModels(ctx context.Context) ([]ModelInfo, error)
}

// ProviderFactory creates provider-specific client adapters from admin-managed configuration.
type ProviderFactory interface {
	NewTextClient(cfg domain.ProviderConfig) (TextModelClient, error)
	NewEmbeddingClient(cfg domain.ProviderConfig) (EmbeddingModelClient, error)
	NewModelListClient(cfg domain.ProviderConfig) (ModelListClient, error)
}

// APIError normalizes provider-side HTTP failures.
type APIError struct {
	Provider   string
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e == nil {
		return "provider API error"
	}
	if e.Provider == "" {
		return fmt.Sprintf("provider request failed with status %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("%s request failed with status %d: %s", e.Provider, e.StatusCode, e.Message)
}

// NewHTTPClient creates an HTTP client with a bounded timeout.
func NewHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &http.Client{Timeout: timeout}
}

// JoinURL appends a relative path to a base URL string while preserving a single slash boundary.
func JoinURL(baseURL, suffix string) string {
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(suffix, "/")
}

// AuthHeaderValue resolves the outgoing bearer token if one is configured.
func AuthHeaderValue(cfg domain.ProviderConfig) string {
	return strings.TrimSpace(cfg.APIKey)
}

// NewJSONRequest creates a JSON POST/PUT request with the provided payload.
func NewJSONRequest(ctx context.Context, method, url string, payload any) (*http.Request, error) {
	var body io.Reader = http.NoBody
	if payload != nil {
		buf, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		body = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// DecodeJSONResponse validates the HTTP status code and decodes the JSON body.
func DecodeJSONResponse(resp *http.Response, out any, providerName string) error {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read %s response: %w", providerName, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{Provider: providerName, StatusCode: resp.StatusCode, Message: parseErrorMessage(body)}
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode %s response: %w", providerName, err)
	}
	return nil
}

func parseErrorMessage(body []byte) string {
	if len(body) == 0 {
		return "empty response body"
	}
	var envelope map[string]any
	if err := json.Unmarshal(body, &envelope); err == nil {
		if v, ok := envelope["error"]; ok {
			switch e := v.(type) {
			case string:
				if strings.TrimSpace(e) != "" {
					return e
				}
			case map[string]any:
				if msg, ok := e["message"].(string); ok && strings.TrimSpace(msg) != "" {
					return msg
				}
				if msg, ok := e["detail"].(string); ok && strings.TrimSpace(msg) != "" {
					return msg
				}
			}
		}
		if msg, ok := envelope["message"].(string); ok && strings.TrimSpace(msg) != "" {
			return msg
		}
		if msg, ok := envelope["detail"].(string); ok && strings.TrimSpace(msg) != "" {
			return msg
		}
	}
	return strings.TrimSpace(string(body))
}

// StreamSingleEvent is a convenience helper for adapters that expose a non-streaming core implementation.
func StreamSingleEvent(ctx context.Context, resp ModelResponse, err error) (<-chan StreamEvent, error) {
	if err != nil {
		return nil, err
	}
	ch := make(chan StreamEvent, 1)
	go func() {
		defer close(ch)
		select {
		case <-ctx.Done():
			ch <- StreamEvent{Type: "error", Done: true, Error: ctx.Err().Error()}
		case ch <- StreamEvent{Type: "final", Response: &resp, Done: true}:
		}
	}()
	return ch, nil
}
