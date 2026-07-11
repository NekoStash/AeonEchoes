package openairesponses

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/provider"

	openaisdk "github.com/openai/openai-go/v3"
	oaioption "github.com/openai/openai-go/v3/option"
	oaiparam "github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
)

const defaultBaseURL = "https://api.openai.com/v1"

// Factory creates OpenAI Responses API adapters backed by the official OpenAI SDK.
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
	sdk openaisdk.Client
}

func newClient(cfg domain.ProviderConfig, httpClient *http.Client, timeout time.Duration) (*Client, error) {
	if cfg.Type != "" && cfg.Type != domain.ProviderOpenAIResponses {
		return nil, fmt.Errorf("openai-responses factory received provider type %q", cfg.Type)
	}
	opts := openAIOptions(cfg, httpClient, timeout)
	return &Client{cfg: cfg, sdk: openaisdk.NewClient(opts...)}, nil
}

func (c *Client) Generate(ctx context.Context, req provider.TextRequest) (provider.ModelResponse, error) {
	body, err := responsesParams(req)
	if err != nil {
		return provider.ModelResponse{}, err
	}
	resp, err := c.sdk.Responses.New(ctx, body)
	if err != nil {
		return provider.ModelResponse{}, fmt.Errorf("openai responses API via SDK failed: %w", err)
	}
	if resp == nil {
		return provider.ModelResponse{}, fmt.Errorf("openai responses API via SDK returned nil response")
	}
	return modelResponse(resp, req.Model), nil
}

func (c *Client) Stream(ctx context.Context, req provider.TextRequest) (<-chan provider.StreamEvent, error) {
	body, err := responsesParams(req)
	if err != nil {
		return nil, err
	}
	stream := c.sdk.Responses.NewStreaming(ctx, body)
	events := make(chan provider.StreamEvent, 8)
	go func() {
		defer close(events)
		defer stream.Close()
		var completed *responses.Response
		for stream.Next() {
			event := stream.Current()
			switch event.Type {
			case "response.output_text.delta":
				if event.Delta != "" && !provider.SendStreamEvent(ctx, events, provider.StreamEvent{Type: "content.delta", Delta: event.Delta}) {
					return
				}
			case "response.completed":
				response := event.Response
				completed = &response
			case "response.failed", "response.incomplete":
				provider.SendStreamEvent(ctx, events, provider.StreamEvent{Type: "error", Done: true, Error: fmt.Sprintf("openai responses streaming ended with %s", event.Type)})
				return
			case "error":
				message := strings.TrimSpace(event.Message)
				if message == "" {
					message = "openai responses streaming returned an error event"
				}
				provider.SendStreamEvent(ctx, events, provider.StreamEvent{Type: "error", Done: true, Error: message})
				return
			}
		}
		if err := stream.Err(); err != nil {
			provider.SendStreamEvent(ctx, events, provider.StreamEvent{Type: "error", Done: true, Error: fmt.Sprintf("openai responses streaming via SDK failed: %v", err)})
			return
		}
		if completed == nil {
			provider.SendStreamEvent(ctx, events, provider.StreamEvent{Type: "error", Done: true, Error: "openai responses streaming ended without response.completed"})
			return
		}
		response := modelResponse(completed, req.Model)
		provider.SendStreamEvent(ctx, events, provider.StreamEvent{Type: "final", Response: &response, Usage: &response.Usage, Done: true})
	}()
	return events, nil
}

func (c *Client) Embed(ctx context.Context, req provider.EmbeddingRequest) (provider.EmbeddingResponse, error) {
	if strings.TrimSpace(req.Model) == "" {
		return provider.EmbeddingResponse{}, fmt.Errorf("openai-responses embedding request model must not be empty")
	}
	if len(req.Inputs) == 0 {
		return provider.EmbeddingResponse{}, fmt.Errorf("openai-responses embedding request inputs must not be empty")
	}
	body := openaisdk.EmbeddingNewParams{
		Model:          openaisdk.EmbeddingModel(req.Model),
		Input:          openaisdk.EmbeddingNewParamsInputUnion{OfArrayOfStrings: req.Inputs},
		EncodingFormat: openaisdk.EmbeddingNewParamsEncodingFormatFloat,
	}
	resp, err := c.sdk.Embeddings.New(ctx, body)
	if err != nil {
		return provider.EmbeddingResponse{}, fmt.Errorf("openai-responses embeddings via SDK failed: %w", err)
	}
	if resp == nil {
		return provider.EmbeddingResponse{}, fmt.Errorf("openai-responses embeddings via SDK returned nil response")
	}
	vectors := make([][]float64, 0, len(resp.Data))
	for _, item := range resp.Data {
		if len(item.Embedding) == 0 {
			return provider.EmbeddingResponse{}, fmt.Errorf("openai-responses embeddings response contained empty vector at index %d", item.Index)
		}
		vectors = append(vectors, item.Embedding)
	}
	if len(vectors) == 0 {
		return provider.EmbeddingResponse{}, fmt.Errorf("openai-responses embeddings response contained no vectors")
	}
	return provider.EmbeddingResponse{
		Provider: string(domain.ProviderOpenAIResponses),
		Model:    firstNonEmpty(resp.Model, req.Model),
		Vectors:  vectors,
		Usage: provider.Usage{
			InputTokens: int(resp.Usage.PromptTokens),
			TotalTokens: int(resp.Usage.TotalTokens),
		},
		Raw: rawJSON(resp.RawJSON(), resp),
	}, nil
}

func (c *Client) ListModels(ctx context.Context) ([]provider.ModelInfo, error) {
	pager := c.sdk.Models.ListAutoPaging(ctx)
	models := make([]provider.ModelInfo, 0)
	for pager.Next() {
		item := pager.Current()
		kind := provider.InferModelKind(item.ID)
		models = append(models, provider.ModelInfo{
			ID:                  item.ID,
			Name:                item.ID,
			DisplayName:         item.ID,
			Provider:            domain.ProviderOpenAIResponses,
			Kind:                kind,
			SupportsTools:       kind == domain.ModelKindText,
			SupportsToolsKnown:  false,
			SupportsStream:      kind == domain.ModelKindText,
			SupportsStreamKnown: false,
			Raw:                 rawJSON(item.RawJSON(), item),
		})
	}
	if err := pager.Err(); err != nil {
		return nil, fmt.Errorf("openai-responses model list via SDK failed: %w", err)
	}
	return models, nil
}

func responsesParams(req provider.TextRequest) (responses.ResponseNewParams, error) {
	if strings.TrimSpace(req.Model) == "" {
		return responses.ResponseNewParams{}, fmt.Errorf("openai-responses text request model must not be empty")
	}
	input, hasInput := responsesInput(req)
	if !hasInput {
		return responses.ResponseNewParams{}, fmt.Errorf("openai-responses text request requires at least one message")
	}
	body := responses.ResponseNewParams{Model: shared.ResponsesModel(req.Model), Input: input}
	if strings.TrimSpace(req.SystemPrompt) != "" {
		body.Instructions = oaiparam.NewOpt(req.SystemPrompt)
	}
	if req.MaxOutputTokens > 0 {
		body.MaxOutputTokens = oaiparam.NewOpt(int64(req.MaxOutputTokens))
	}
	if req.Temperature > 0 {
		body.Temperature = oaiparam.NewOpt(req.Temperature)
	}
	if req.TopP > 0 {
		body.TopP = oaiparam.NewOpt(req.TopP)
	}
	if len(req.Metadata) > 0 {
		body.Metadata = shared.Metadata(req.Metadata)
	}
	if len(req.Tools) > 0 {
		tools, err := responsesTools(req.Tools)
		if err != nil {
			return responses.ResponseNewParams{}, err
		}
		body.Tools = tools
	}
	return body, nil
}

func modelResponse(resp *responses.Response, requestModel string) provider.ModelResponse {
	return provider.ModelResponse{
		ID:           resp.ID,
		Provider:     string(domain.ProviderOpenAIResponses),
		Model:        firstNonEmpty(resp.Model, requestModel),
		Content:      strings.TrimSpace(resp.OutputText()),
		FinishReason: string(resp.Status),
		ToolCalls:    collectResponseToolCalls(resp.Output),
		Usage: provider.Usage{
			InputTokens:  int(resp.Usage.InputTokens),
			OutputTokens: int(resp.Usage.OutputTokens),
			TotalTokens:  int(resp.Usage.TotalTokens),
		},
		Raw: rawJSON(resp.RawJSON(), resp),
	}
}

func openAIOptions(cfg domain.ProviderConfig, httpClient *http.Client, timeout time.Duration) []oaioption.RequestOption {
	effectiveTimeout := timeoutFromConfig(cfg, timeout)
	if httpClient == nil {
		httpClient = provider.NewHTTPClient(effectiveTimeout)
	}
	opts := []oaioption.RequestOption{
		oaioption.WithHTTPClient(httpClient),
		oaioption.WithBaseURL(normalizeOpenAIBaseURL(cfg.BaseURL)),
	}
	if key := provider.AuthHeaderValue(cfg); key != "" {
		opts = append(opts, oaioption.WithAPIKey(key))
	}
	if effectiveTimeout > 0 {
		opts = append(opts, oaioption.WithRequestTimeout(effectiveTimeout))
	}
	return opts
}

func timeoutFromConfig(cfg domain.ProviderConfig, fallback time.Duration) time.Duration {
	if cfg.DefaultRequestTimeoutSec > 0 {
		return time.Duration(cfg.DefaultRequestTimeoutSec) * time.Second
	}
	return fallback
}

func normalizeOpenAIBaseURL(baseURL string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmed == "" {
		return defaultBaseURL
	}
	if strings.HasSuffix(trimmed, "/v1") {
		return trimmed
	}
	return trimmed + "/v1"
}

func responsesInput(req provider.TextRequest) (responses.ResponseNewParamsInputUnion, bool) {
	items := make([]responses.ResponseInputItemUnionParam, 0, len(req.Messages)+1)
	for _, msg := range req.Messages {
		content := provider.MessageContent(msg)
		if strings.TrimSpace(content) == "" {
			continue
		}
		role := responses.EasyInputMessageRoleUser
		switch strings.ToLower(strings.TrimSpace(msg.Role)) {
		case "system":
			role = responses.EasyInputMessageRoleSystem
		case "developer":
			role = responses.EasyInputMessageRoleDeveloper
		case "assistant":
			role = responses.EasyInputMessageRoleAssistant
		}
		items = append(items, easyInputMessage(role, content))
	}
	if strings.TrimSpace(req.UserPrompt) != "" {
		items = append(items, easyInputMessage(responses.EasyInputMessageRoleUser, req.UserPrompt))
	}
	if len(items) == 0 {
		return responses.ResponseNewParamsInputUnion{}, false
	}
	return responses.ResponseNewParamsInputUnion{OfInputItemList: responses.ResponseInputParam(items)}, true
}

func easyInputMessage(role responses.EasyInputMessageRole, content string) responses.ResponseInputItemUnionParam {
	return responses.ResponseInputItemUnionParam{OfMessage: &responses.EasyInputMessageParam{
		Role: role,
		Content: responses.EasyInputMessageContentUnionParam{
			OfString: oaiparam.NewOpt(content),
		},
	}}
}

func responsesTools(tools []provider.ToolSpec) ([]responses.ToolUnionParam, error) {
	result := make([]responses.ToolUnionParam, 0, len(tools))
	for _, tool := range tools {
		if strings.TrimSpace(tool.Name) == "" {
			return nil, fmt.Errorf("openai-responses tool name must not be empty")
		}
		params, err := parseToolParameters("openai-responses", tool)
		if err != nil {
			return nil, err
		}
		if params == nil {
			params = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		fn := &responses.FunctionToolParam{
			Name:       tool.Name,
			Parameters: params,
			Strict:     oaiparam.NewOpt(false),
		}
		if strings.TrimSpace(tool.Description) != "" {
			fn.Description = oaiparam.NewOpt(tool.Description)
		}
		result = append(result, responses.ToolUnionParam{OfFunction: fn})
	}
	return result, nil
}

func parseToolParameters(providerName string, tool provider.ToolSpec) (map[string]any, error) {
	if len(tool.Parameters) == 0 {
		return nil, nil
	}
	var params map[string]any
	if err := json.Unmarshal(tool.Parameters, &params); err != nil {
		return nil, fmt.Errorf("%s tool %q parameters must be a JSON object: %w", providerName, tool.Name, err)
	}
	if params == nil {
		return nil, fmt.Errorf("%s tool %q parameters must be a JSON object", providerName, tool.Name)
	}
	return params, nil
}

func collectResponseToolCalls(output []responses.ResponseOutputItemUnion) []provider.ToolCall {
	calls := make([]provider.ToolCall, 0)
	for _, item := range output {
		if item.Type != "function_call" || strings.TrimSpace(item.Name) == "" {
			continue
		}
		calls = append(calls, provider.ToolCall{
			ID:        firstNonEmpty(item.ID, item.CallID),
			Type:      item.Type,
			Name:      item.Name,
			Arguments: rawJSONString(item.Arguments.OfString),
		})
	}
	return calls
}

func rawJSONString(value string) json.RawMessage {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	if json.Valid([]byte(trimmed)) {
		return json.RawMessage(trimmed)
	}
	encoded, _ := json.Marshal(trimmed)
	return encoded
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
