package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/provider"

	anthropicsdk "github.com/anthropics/anthropic-sdk-go"
	anthrooption "github.com/anthropics/anthropic-sdk-go/option"
	anthroparam "github.com/anthropics/anthropic-sdk-go/packages/param"
)

const defaultBaseURL = "https://api.anthropic.com"

// Factory creates Anthropic adapters backed by the official Anthropic SDK.
type Factory struct {
	HTTPClient *http.Client
	Timeout    time.Duration
}

func (f Factory) NewTextClient(cfg domain.ProviderConfig) (provider.TextModelClient, error) {
	client, err := newClient(cfg, f.HTTPClient, f.Timeout)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (f Factory) NewEmbeddingClient(cfg domain.ProviderConfig) (provider.EmbeddingModelClient, error) {
	client, err := newClient(cfg, f.HTTPClient, f.Timeout)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (f Factory) NewModelListClient(cfg domain.ProviderConfig) (provider.ModelListClient, error) {
	client, err := newClient(cfg, f.HTTPClient, f.Timeout)
	if err != nil {
		return nil, err
	}
	return client, nil
}

type Client struct {
	cfg domain.ProviderConfig
	sdk anthropicsdk.Client
}

func newClient(cfg domain.ProviderConfig, httpClient *http.Client, timeout time.Duration) (*Client, error) {
	if cfg.Type != "" && cfg.Type != domain.ProviderAnthropic {
		return nil, fmt.Errorf("anthropic factory received provider type %q", cfg.Type)
	}
	opts := anthropicOptions(cfg, httpClient, timeout)
	return &Client{cfg: cfg, sdk: anthropicsdk.NewClient(opts...)}, nil
}

func (c *Client) Generate(ctx context.Context, req provider.TextRequest) (provider.ModelResponse, error) {
	if strings.TrimSpace(req.Model) == "" {
		return provider.ModelResponse{}, fmt.Errorf("anthropic text request model must not be empty")
	}
	messages := anthropicMessages(req)
	if len(messages) == 0 {
		return provider.ModelResponse{}, fmt.Errorf("anthropic text request requires at least one user or assistant message")
	}
	body := anthropicsdk.MessageNewParams{
		Model:     anthropicsdk.Model(req.Model),
		Messages:  messages,
		MaxTokens: maxInt64(1, int64(req.MaxOutputTokens)),
	}
	if strings.TrimSpace(req.SystemPrompt) != "" {
		body.System = []anthropicsdk.TextBlockParam{{Text: req.SystemPrompt}}
	}
	if req.Temperature > 0 {
		body.Temperature = anthroparam.NewOpt(req.Temperature)
	}
	if req.TopP > 0 {
		body.TopP = anthroparam.NewOpt(req.TopP)
	}
	if len(req.Tools) > 0 {
		tools, err := anthropicTools(req.Tools)
		if err != nil {
			return provider.ModelResponse{}, err
		}
		body.Tools = tools
	}

	msg, err := c.sdk.Messages.New(ctx, body)
	if err != nil {
		return provider.ModelResponse{}, fmt.Errorf("anthropic messages via SDK failed: %w", err)
	}
	if msg == nil {
		return provider.ModelResponse{}, fmt.Errorf("anthropic messages via SDK returned nil response")
	}
	inputTokens := msg.Usage.InputTokens + msg.Usage.CacheCreationInputTokens + msg.Usage.CacheReadInputTokens
	return provider.ModelResponse{
		ID:           msg.ID,
		Provider:     string(domain.ProviderAnthropic),
		Model:        firstNonEmpty(string(msg.Model), req.Model),
		Content:      collectContent(msg.Content),
		FinishReason: string(msg.StopReason),
		ToolCalls:    collectToolCalls(msg.Content),
		Usage: provider.Usage{
			InputTokens:  int(inputTokens),
			OutputTokens: int(msg.Usage.OutputTokens),
			TotalTokens:  int(inputTokens + msg.Usage.OutputTokens),
		},
		Raw: rawJSON(msg.RawJSON(), msg),
	}, nil
}

func (c *Client) Stream(ctx context.Context, req provider.TextRequest) (<-chan provider.StreamEvent, error) {
	// 流式后续通过 SDK streaming 深化；当前统一接口仍以一次性 Generate 结果封装为单个 final 事件，避免手写 SSE 协议。
	resp, err := c.Generate(ctx, req)
	return provider.StreamSingleEvent(ctx, resp, err)
}

func (c *Client) Embed(ctx context.Context, req provider.EmbeddingRequest) (provider.EmbeddingResponse, error) {
	return provider.EmbeddingResponse{}, fmt.Errorf("anthropic provider does not expose an embedding API; configure an openai or gemini embedding model")
}

func (c *Client) ListModels(ctx context.Context) ([]provider.ModelInfo, error) {
	pager := c.sdk.Models.ListAutoPaging(ctx, anthropicsdk.ModelListParams{})
	models := make([]provider.ModelInfo, 0)
	for pager.Next() {
		item := pager.Current()
		models = append(models, provider.ModelInfo{
			ID:                  item.ID,
			Name:                item.ID,
			DisplayName:         firstNonEmpty(item.DisplayName, item.ID),
			Provider:            domain.ProviderAnthropic,
			Kind:                domain.ModelKindText,
			ContextWindow:       int(item.MaxInputTokens),
			MaxOutputTokens:     int(item.MaxTokens),
			SupportsTools:       true,
			SupportsToolsKnown:  false,
			SupportsStream:      true,
			SupportsStreamKnown: false,
			Raw:                 rawJSON(item.RawJSON(), item),
		})
	}
	if err := pager.Err(); err != nil {
		return nil, fmt.Errorf("anthropic model list via SDK failed: %w", err)
	}
	return models, nil
}

func anthropicOptions(cfg domain.ProviderConfig, httpClient *http.Client, timeout time.Duration) []anthrooption.RequestOption {
	effectiveTimeout := timeoutFromConfig(cfg, timeout)
	if httpClient == nil {
		httpClient = provider.NewHTTPClient(effectiveTimeout)
	}
	opts := []anthrooption.RequestOption{
		anthrooption.WithoutEnvironmentDefaults(),
		anthrooption.WithHTTPClient(httpClient),
		anthrooption.WithBaseURL(normalizeAnthropicBaseURL(cfg.BaseURL)),
	}
	if key := provider.AuthHeaderValue(cfg); key != "" {
		opts = append(opts, anthrooption.WithAPIKey(key))
	}
	if effectiveTimeout > 0 {
		opts = append(opts, anthrooption.WithRequestTimeout(effectiveTimeout))
	}
	return opts
}

func timeoutFromConfig(cfg domain.ProviderConfig, fallback time.Duration) time.Duration {
	if cfg.DefaultRequestTimeoutSec > 0 {
		return time.Duration(cfg.DefaultRequestTimeoutSec) * time.Second
	}
	return fallback
}

func normalizeAnthropicBaseURL(baseURL string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmed == "" {
		return defaultBaseURL
	}
	return trimmed
}

func anthropicMessages(req provider.TextRequest) []anthropicsdk.MessageParam {
	messages := make([]anthropicsdk.MessageParam, 0, len(req.Messages)+1)
	for _, msg := range req.Messages {
		content := provider.MessageContent(msg)
		if strings.TrimSpace(content) == "" {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(msg.Role)) {
		case "system", "developer":
			continue
		case "assistant":
			messages = append(messages, anthropicsdk.NewAssistantMessage(anthropicsdk.NewTextBlock(content)))
		default:
			messages = append(messages, anthropicsdk.NewUserMessage(anthropicsdk.NewTextBlock(content)))
		}
	}
	if strings.TrimSpace(req.UserPrompt) != "" {
		messages = append(messages, anthropicsdk.NewUserMessage(anthropicsdk.NewTextBlock(req.UserPrompt)))
	}
	return messages
}

func anthropicTools(tools []provider.ToolSpec) ([]anthropicsdk.ToolUnionParam, error) {
	result := make([]anthropicsdk.ToolUnionParam, 0, len(tools))
	for _, tool := range tools {
		if strings.TrimSpace(tool.Name) == "" {
			return nil, fmt.Errorf("anthropic tool name must not be empty")
		}
		schema, err := anthropicToolInputSchema(tool)
		if err != nil {
			return nil, err
		}
		param := anthropicsdk.ToolParam{
			Name:        tool.Name,
			InputSchema: schema,
		}
		if strings.TrimSpace(tool.Description) != "" {
			param.Description = anthroparam.NewOpt(tool.Description)
		}
		result = append(result, anthropicsdk.ToolUnionParam{OfTool: &param})
	}
	return result, nil
}

func anthropicToolInputSchema(tool provider.ToolSpec) (anthropicsdk.ToolInputSchemaParam, error) {
	if len(tool.Parameters) == 0 {
		return anthropicsdk.ToolInputSchemaParam{Properties: map[string]any{}}, nil
	}
	var params map[string]any
	if err := json.Unmarshal(tool.Parameters, &params); err != nil {
		return anthropicsdk.ToolInputSchemaParam{}, fmt.Errorf("anthropic tool %q parameters must be a JSON object: %w", tool.Name, err)
	}
	if params == nil {
		return anthropicsdk.ToolInputSchemaParam{}, fmt.Errorf("anthropic tool %q parameters must be a JSON object", tool.Name)
	}
	schema := anthropicsdk.ToolInputSchemaParam{ExtraFields: make(map[string]any, len(params))}
	if properties, ok := params["properties"]; ok {
		schema.Properties = properties
	} else {
		schema.Properties = map[string]any{}
	}
	if required, ok := stringSlice(params["required"]); ok {
		schema.Required = required
	}
	for key, value := range params {
		if key == "properties" || key == "required" || key == "type" {
			continue
		}
		schema.ExtraFields[key] = value
	}
	return schema, nil
}

func stringSlice(value any) ([]string, bool) {
	items, ok := value.([]any)
	if !ok {
		stringsValue, ok := value.([]string)
		return stringsValue, ok
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := item.(string)
		if !ok {
			return nil, false
		}
		result = append(result, text)
	}
	return result, true
}

func collectContent(blocks []anthropicsdk.ContentBlockUnion) string {
	var b strings.Builder
	for _, block := range blocks {
		if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString(block.Text)
		}
	}
	return b.String()
}

func collectToolCalls(blocks []anthropicsdk.ContentBlockUnion) []provider.ToolCall {
	calls := make([]provider.ToolCall, 0)
	for _, block := range blocks {
		if block.Type != "tool_use" || strings.TrimSpace(block.Name) == "" {
			continue
		}
		calls = append(calls, provider.ToolCall{ID: block.ID, Type: block.Type, Name: block.Name, Arguments: block.Input})
	}
	return calls
}

func rawJSON(raw string, value any) json.RawMessage {
	if strings.TrimSpace(raw) != "" && json.Valid([]byte(raw)) {
		return json.RawMessage(raw)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	return encoded
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
