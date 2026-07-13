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
	body, err := anthropicMessageParams(req)
	if err != nil {
		return provider.ModelResponse{}, err
	}
	msg, err := c.sdk.Messages.New(ctx, body)
	if err != nil {
		return provider.ModelResponse{}, fmt.Errorf("anthropic messages via SDK failed: %w", err)
	}
	if msg == nil {
		return provider.ModelResponse{}, fmt.Errorf("anthropic messages via SDK returned nil response")
	}
	return anthropicModelResponse(msg, req.Model), nil
}

func (c *Client) Stream(ctx context.Context, req provider.TextRequest) (<-chan provider.StreamEvent, error) {
	body, err := anthropicMessageParams(req)
	if err != nil {
		return nil, err
	}
	stream := c.sdk.Messages.NewStreaming(ctx, body)
	events := make(chan provider.StreamEvent, 8)
	go func() {
		defer close(events)
		defer stream.Close()
		var message anthropicsdk.Message
		for stream.Next() {
			event := stream.Current()
			if err := message.Accumulate(event); err != nil {
				provider.SendStreamEvent(ctx, events, provider.StreamEvent{Type: "error", Done: true, Error: fmt.Sprintf("accumulate anthropic stream event: %v", err)})
				return
			}
			if event.Type == "content_block_delta" && event.Delta.Type == "text_delta" && event.Delta.Text != "" {
				if !provider.SendStreamEvent(ctx, events, provider.StreamEvent{Type: "content.delta", Delta: event.Delta.Text}) {
					return
				}
			}
		}
		if err := stream.Err(); err != nil {
			provider.SendStreamEvent(ctx, events, provider.StreamEvent{Type: "error", Done: true, Error: fmt.Sprintf("anthropic messages streaming via SDK failed: %v", err)})
			return
		}
		if strings.TrimSpace(message.ID) == "" {
			provider.SendStreamEvent(ctx, events, provider.StreamEvent{Type: "error", Done: true, Error: "anthropic messages streaming ended without a complete message"})
			return
		}
		response := anthropicModelResponse(&message, req.Model)
		provider.SendStreamEvent(ctx, events, provider.StreamEvent{Type: "final", Response: &response, Usage: &response.Usage, Done: true})
	}()
	return events, nil
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

func anthropicMessageParams(req provider.TextRequest) (anthropicsdk.MessageNewParams, error) {
	if strings.TrimSpace(req.Model) == "" {
		return anthropicsdk.MessageNewParams{}, fmt.Errorf("anthropic text request model must not be empty")
	}
	messages, err := anthropicMessages(req)
	if err != nil {
		return anthropicsdk.MessageNewParams{}, err
	}
	if len(messages) == 0 {
		return anthropicsdk.MessageNewParams{}, fmt.Errorf("anthropic text request requires at least one user or assistant message")
	}
	body := anthropicsdk.MessageNewParams{Model: anthropicsdk.Model(req.Model), Messages: messages, MaxTokens: maxInt64(1, int64(req.MaxOutputTokens))}
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
			return anthropicsdk.MessageNewParams{}, err
		}
		body.Tools = tools
	}
	return body, nil
}

func anthropicModelResponse(msg *anthropicsdk.Message, requestModel string) provider.ModelResponse {
	inputTokens := msg.Usage.InputTokens + msg.Usage.CacheCreationInputTokens + msg.Usage.CacheReadInputTokens
	return provider.ModelResponse{
		ID:           msg.ID,
		Provider:     string(domain.ProviderAnthropic),
		Model:        firstNonEmpty(string(msg.Model), requestModel),
		Content:      collectContent(msg.Content),
		FinishReason: string(msg.StopReason),
		ToolCalls:    collectToolCalls(msg.Content),
		Usage: provider.Usage{
			InputTokens:  int(inputTokens),
			OutputTokens: int(msg.Usage.OutputTokens),
			TotalTokens:  int(inputTokens + msg.Usage.OutputTokens),
		},
		Raw: rawJSON(msg.RawJSON(), msg),
	}
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

func anthropicMessages(req provider.TextRequest) ([]anthropicsdk.MessageParam, error) {
	messages := make([]anthropicsdk.MessageParam, 0, len(req.Messages)+1)
	// Anthropic requires tool_result blocks to live in a user message that
	// immediately follows the assistant tool_use message. Consecutive tool
	// results from the same tool round are therefore merged into one user turn.
	var pendingToolResults []anthropicsdk.ContentBlockParamUnion
	flushToolResults := func() {
		if len(pendingToolResults) == 0 {
			return
		}
		messages = append(messages, anthropicsdk.NewUserMessage(pendingToolResults...))
		pendingToolResults = nil
	}
	for index, msg := range req.Messages {
		role := strings.ToLower(strings.TrimSpace(msg.Role))
		switch role {
		case "system", "developer":
			continue
		case "assistant":
			flushToolResults()
			blocks, err := anthropicAssistantBlocks(msg, index)
			if err != nil {
				return nil, err
			}
			messages = append(messages, anthropicsdk.NewAssistantMessage(blocks...))
		case "tool":
			block, err := anthropicToolResultBlock(msg, index)
			if err != nil {
				return nil, err
			}
			pendingToolResults = append(pendingToolResults, block)
		case "user", "":
			flushToolResults()
			content := strings.TrimSpace(msg.Content)
			if content == "" {
				return nil, fmt.Errorf("anthropic message[%d] user content must not be empty", index)
			}
			messages = append(messages, anthropicsdk.NewUserMessage(anthropicsdk.NewTextBlock(content)))
		default:
			return nil, fmt.Errorf("anthropic message[%d] has unsupported role %q", index, msg.Role)
		}
	}
	flushToolResults()
	if strings.TrimSpace(req.UserPrompt) != "" {
		messages = append(messages, anthropicsdk.NewUserMessage(anthropicsdk.NewTextBlock(req.UserPrompt)))
	}
	return messages, nil
}

func anthropicAssistantBlocks(msg provider.Message, index int) ([]anthropicsdk.ContentBlockParamUnion, error) {
	blocks := make([]anthropicsdk.ContentBlockParamUnion, 0, len(msg.ToolCalls)+1)
	if content := strings.TrimSpace(msg.Content); content != "" {
		blocks = append(blocks, anthropicsdk.NewTextBlock(content))
	}
	for callIndex, call := range msg.ToolCalls {
		name := strings.TrimSpace(call.Name)
		if name == "" {
			return nil, fmt.Errorf("anthropic message[%d] tool_calls[%d] name must not be empty", index, callIndex)
		}
		id := strings.TrimSpace(call.ID)
		if id == "" {
			return nil, fmt.Errorf("anthropic message[%d] tool_calls[%d] id must not be empty", index, callIndex)
		}
		input, err := decodeToolCallInput(call.Arguments, index, callIndex)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, anthropicsdk.NewToolUseBlock(id, input, name))
	}
	if len(blocks) == 0 {
		return nil, fmt.Errorf("anthropic message[%d] assistant content or tool_calls must not be empty", index)
	}
	return blocks, nil
}

func anthropicToolResultBlock(msg provider.Message, index int) (anthropicsdk.ContentBlockParamUnion, error) {
	toolCallID := strings.TrimSpace(msg.ToolCallID)
	if toolCallID == "" {
		return anthropicsdk.ContentBlockParamUnion{}, fmt.Errorf("anthropic message[%d] tool role requires tool_call_id", index)
	}
	content := msg.Content
	if strings.TrimSpace(content) == "" {
		content = "{}"
	}
	return anthropicsdk.NewToolResultBlock(toolCallID, content, false), nil
}

func decodeToolCallInput(raw json.RawMessage, messageIndex, callIndex int) (any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return map[string]any{}, nil
	}
	var input any
	if err := json.Unmarshal([]byte(trimmed), &input); err != nil {
		return nil, fmt.Errorf("anthropic message[%d] tool_calls[%d] arguments must be valid JSON: %w", messageIndex, callIndex, err)
	}
	if input == nil {
		return map[string]any{}, nil
	}
	return input, nil
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
