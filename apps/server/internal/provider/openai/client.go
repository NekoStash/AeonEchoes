package openai

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
	"github.com/openai/openai-go/v3/shared"
)

const defaultBaseURL = "https://api.openai.com/v1"

// Factory creates OpenAI-compatible adapters backed by the official OpenAI SDK.
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
	if cfg.Type != "" && cfg.Type != domain.ProviderOpenAI {
		return nil, fmt.Errorf("openai factory received provider type %q", cfg.Type)
	}
	opts := openAIOptions(cfg, httpClient, timeout)
	return &Client{cfg: cfg, sdk: openaisdk.NewClient(opts...)}, nil
}

func (c *Client) Generate(ctx context.Context, req provider.TextRequest) (provider.ModelResponse, error) {
	if strings.TrimSpace(req.Model) == "" {
		return provider.ModelResponse{}, fmt.Errorf("openai text request model must not be empty")
	}
	messages := openAIChatMessages(req)
	if len(messages) == 0 {
		return provider.ModelResponse{}, fmt.Errorf("openai text request requires at least one message")
	}
	body := openaisdk.ChatCompletionNewParams{
		Model:    openaisdk.ChatModel(req.Model),
		Messages: messages,
	}
	if req.MaxOutputTokens > 0 {
		body.MaxCompletionTokens = oaiparam.NewOpt(int64(req.MaxOutputTokens))
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
		tools, err := openAIChatTools(req.Tools)
		if err != nil {
			return provider.ModelResponse{}, err
		}
		body.Tools = tools
	}

	completion, err := c.sdk.Chat.Completions.New(ctx, body)
	if err != nil {
		return provider.ModelResponse{}, fmt.Errorf("openai chat completion via SDK failed: %w", err)
	}
	if completion == nil {
		return provider.ModelResponse{}, fmt.Errorf("openai chat completion via SDK returned nil response")
	}
	if len(completion.Choices) == 0 {
		return provider.ModelResponse{}, fmt.Errorf("openai chat completion returned no choices")
	}
	choice := completion.Choices[0]
	return provider.ModelResponse{
		ID:           completion.ID,
		Provider:     string(domain.ProviderOpenAI),
		Model:        firstNonEmpty(completion.Model, req.Model),
		Content:      choice.Message.Content,
		FinishReason: choice.FinishReason,
		ToolCalls:    parseChatToolCalls(choice.Message),
		Usage: provider.Usage{
			InputTokens:  int(completion.Usage.PromptTokens),
			OutputTokens: int(completion.Usage.CompletionTokens),
			TotalTokens:  int(completion.Usage.TotalTokens),
		},
		Raw: rawJSON(completion.RawJSON(), completion),
	}, nil
}

func (c *Client) Stream(ctx context.Context, req provider.TextRequest) (<-chan provider.StreamEvent, error) {
	// 流式后续通过 SDK streaming 深化；当前统一接口仍以一次性 Generate 结果封装为单个 final 事件，避免手写 SSE 协议。
	resp, err := c.Generate(ctx, req)
	return provider.StreamSingleEvent(ctx, resp, err)
}

func (c *Client) Embed(ctx context.Context, req provider.EmbeddingRequest) (provider.EmbeddingResponse, error) {
	if strings.TrimSpace(req.Model) == "" {
		return provider.EmbeddingResponse{}, fmt.Errorf("openai embedding request model must not be empty")
	}
	if len(req.Inputs) == 0 {
		return provider.EmbeddingResponse{}, fmt.Errorf("openai embedding request inputs must not be empty")
	}
	body := openaisdk.EmbeddingNewParams{
		Model:          openaisdk.EmbeddingModel(req.Model),
		Input:          openaisdk.EmbeddingNewParamsInputUnion{OfArrayOfStrings: req.Inputs},
		EncodingFormat: openaisdk.EmbeddingNewParamsEncodingFormatFloat,
	}
	resp, err := c.sdk.Embeddings.New(ctx, body)
	if err != nil {
		return provider.EmbeddingResponse{}, fmt.Errorf("openai embeddings via SDK failed: %w", err)
	}
	if resp == nil {
		return provider.EmbeddingResponse{}, fmt.Errorf("openai embeddings via SDK returned nil response")
	}
	vectors := make([][]float64, 0, len(resp.Data))
	for _, item := range resp.Data {
		if len(item.Embedding) == 0 {
			return provider.EmbeddingResponse{}, fmt.Errorf("openai embeddings response contained empty vector at index %d", item.Index)
		}
		vectors = append(vectors, item.Embedding)
	}
	if len(vectors) == 0 {
		return provider.EmbeddingResponse{}, fmt.Errorf("openai embeddings response contained no vectors")
	}
	return provider.EmbeddingResponse{
		Provider: string(domain.ProviderOpenAI),
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
			Provider:            domain.ProviderOpenAI,
			Kind:                kind,
			SupportsTools:       kind == domain.ModelKindText,
			SupportsToolsKnown:  false,
			SupportsStream:      kind == domain.ModelKindText,
			SupportsStreamKnown: false,
			Raw:                 rawJSON(item.RawJSON(), item),
		})
	}
	if err := pager.Err(); err != nil {
		return nil, fmt.Errorf("openai model list via SDK failed: %w", err)
	}
	return models, nil
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

func openAIChatMessages(req provider.TextRequest) []openaisdk.ChatCompletionMessageParamUnion {
	messages := make([]openaisdk.ChatCompletionMessageParamUnion, 0, len(req.Messages)+2)
	if strings.TrimSpace(req.SystemPrompt) != "" {
		messages = append(messages, openaisdk.SystemMessage(req.SystemPrompt))
	}
	for _, msg := range req.Messages {
		content := provider.MessageContent(msg)
		if strings.TrimSpace(content) == "" {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(msg.Role)) {
		case "system", "developer":
			messages = append(messages, openaisdk.SystemMessage(content))
		case "assistant":
			messages = append(messages, openaisdk.AssistantMessage(content))
		default:
			messages = append(messages, openaisdk.UserMessage(content))
		}
	}
	if strings.TrimSpace(req.UserPrompt) != "" {
		messages = append(messages, openaisdk.UserMessage(req.UserPrompt))
	}
	return messages
}

func openAIChatTools(tools []provider.ToolSpec) ([]openaisdk.ChatCompletionToolUnionParam, error) {
	result := make([]openaisdk.ChatCompletionToolUnionParam, 0, len(tools))
	for _, tool := range tools {
		if strings.TrimSpace(tool.Name) == "" {
			return nil, fmt.Errorf("openai tool name must not be empty")
		}
		definition := shared.FunctionDefinitionParam{Name: tool.Name}
		if strings.TrimSpace(tool.Description) != "" {
			definition.Description = oaiparam.NewOpt(tool.Description)
		}
		params, err := parseToolParameters("openai", tool)
		if err != nil {
			return nil, err
		}
		if params != nil {
			definition.Parameters = shared.FunctionParameters(params)
		}
		result = append(result, openaisdk.ChatCompletionFunctionTool(definition))
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

func parseChatToolCalls(message openaisdk.ChatCompletionMessage) []provider.ToolCall {
	calls := make([]provider.ToolCall, 0, len(message.ToolCalls))
	for _, call := range message.ToolCalls {
		if strings.TrimSpace(call.Function.Name) == "" {
			continue
		}
		calls = append(calls, provider.ToolCall{
			ID:        call.ID,
			Type:      firstNonEmpty(call.Type, "function"),
			Name:      call.Function.Name,
			Arguments: rawJSONString(call.Function.Arguments),
		})
	}
	if strings.TrimSpace(message.FunctionCall.Name) != "" {
		calls = append(calls, provider.ToolCall{
			Type:      "function_call",
			Name:      message.FunctionCall.Name,
			Arguments: rawJSONString(message.FunctionCall.Arguments),
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
