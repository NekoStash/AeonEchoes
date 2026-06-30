package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/provider"

	"google.golang.org/genai"
)

// Factory creates Gemini adapters backed by the official Google GenAI SDK.
type Factory struct {
	HTTPClient *http.Client
	Timeout    time.Duration
}

func (f Factory) NewTextClient(cfg domain.ProviderConfig) (provider.TextModelClient, error) {
	client, err := newClient(context.Background(), cfg, f.HTTPClient, f.Timeout)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (f Factory) NewEmbeddingClient(cfg domain.ProviderConfig) (provider.EmbeddingModelClient, error) {
	client, err := newClient(context.Background(), cfg, f.HTTPClient, f.Timeout)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (f Factory) NewModelListClient(cfg domain.ProviderConfig) (provider.ModelListClient, error) {
	client, err := newClient(context.Background(), cfg, f.HTTPClient, f.Timeout)
	if err != nil {
		return nil, err
	}
	return client, nil
}

type Client struct {
	cfg domain.ProviderConfig
	sdk *genai.Client
}

func newClient(ctx context.Context, cfg domain.ProviderConfig, httpClient *http.Client, timeout time.Duration) (*Client, error) {
	if cfg.Type != "" && cfg.Type != domain.ProviderGemini {
		return nil, fmt.Errorf("gemini factory received provider type %q", cfg.Type)
	}
	effectiveTimeout := timeoutFromConfig(cfg, timeout)
	if httpClient == nil {
		httpClient = provider.NewHTTPClient(effectiveTimeout)
	}
	httpOptions := genai.HTTPOptions{APIVersion: "v1beta"}
	if baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"); baseURL != "" {
		httpOptions.BaseURL = baseURL
	}
	if effectiveTimeout > 0 {
		httpOptions.Timeout = &effectiveTimeout
	}
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:      provider.AuthHeaderValue(cfg),
		Backend:     genai.BackendGeminiAPI,
		HTTPClient:  httpClient,
		HTTPOptions: httpOptions,
	})
	if err != nil {
		return nil, fmt.Errorf("create gemini SDK client: %w", err)
	}
	return &Client{cfg: cfg, sdk: client}, nil
}

func (c *Client) Generate(ctx context.Context, req provider.TextRequest) (provider.ModelResponse, error) {
	if strings.TrimSpace(req.Model) == "" {
		return provider.ModelResponse{}, fmt.Errorf("gemini text request model must not be empty")
	}
	contents := geminiContents(req)
	if len(contents) == 0 {
		return provider.ModelResponse{}, fmt.Errorf("gemini text request requires at least one message")
	}
	config, err := geminiGenerateConfig(req)
	if err != nil {
		return provider.ModelResponse{}, err
	}
	resp, err := c.sdk.Models.GenerateContent(ctx, req.Model, contents, config)
	if err != nil {
		return provider.ModelResponse{}, fmt.Errorf("gemini generate content via SDK failed: %w", err)
	}
	if resp == nil {
		return provider.ModelResponse{}, fmt.Errorf("gemini generate content via SDK returned nil response")
	}
	return provider.ModelResponse{
		ID:           resp.ResponseID,
		Provider:     string(domain.ProviderGemini),
		Model:        firstNonEmpty(resp.ModelVersion, req.Model),
		Content:      strings.TrimSpace(resp.Text()),
		FinishReason: geminiFinishReason(resp),
		ToolCalls:    geminiFunctionCalls(resp.FunctionCalls()),
		Usage:        geminiUsage(resp.UsageMetadata),
		Raw:          rawJSON(resp),
	}, nil
}

func (c *Client) Stream(ctx context.Context, req provider.TextRequest) (<-chan provider.StreamEvent, error) {
	// 流式后续通过 SDK streaming 深化；当前统一接口仍以一次性 Generate 结果封装为单个 final 事件，避免手写 SSE 协议。
	resp, err := c.Generate(ctx, req)
	return provider.StreamSingleEvent(ctx, resp, err)
}

func (c *Client) Embed(ctx context.Context, req provider.EmbeddingRequest) (provider.EmbeddingResponse, error) {
	if strings.TrimSpace(req.Model) == "" {
		return provider.EmbeddingResponse{}, fmt.Errorf("gemini embedding request model must not be empty")
	}
	if len(req.Inputs) == 0 {
		return provider.EmbeddingResponse{}, fmt.Errorf("gemini embedding request inputs must not be empty")
	}
	contents := make([]*genai.Content, 0, len(req.Inputs))
	for idx, input := range req.Inputs {
		if strings.TrimSpace(input) == "" {
			return provider.EmbeddingResponse{}, fmt.Errorf("gemini embedding input at index %d must not be empty", idx)
		}
		contents = append(contents, genai.NewContentFromText(input, genai.RoleUser))
	}
	resp, err := c.sdk.Models.EmbedContent(ctx, req.Model, contents, nil)
	if err != nil {
		return provider.EmbeddingResponse{}, fmt.Errorf("gemini embed content via SDK failed: %w", err)
	}
	if resp == nil {
		return provider.EmbeddingResponse{}, fmt.Errorf("gemini embed content via SDK returned nil response")
	}
	vectors := make([][]float64, 0, len(resp.Embeddings))
	for idx, embedding := range resp.Embeddings {
		if embedding == nil || len(embedding.Values) == 0 {
			return provider.EmbeddingResponse{}, fmt.Errorf("gemini embedding response contained empty vector at index %d", idx)
		}
		vectors = append(vectors, float32ToFloat64(embedding.Values))
	}
	if len(vectors) == 0 {
		return provider.EmbeddingResponse{}, fmt.Errorf("gemini embedding response contained no vectors")
	}
	usage := provider.Usage{InputTokens: estimatedInputTokens(req.Inputs)}
	usage.TotalTokens = usage.InputTokens
	return provider.EmbeddingResponse{
		Provider: string(domain.ProviderGemini),
		Model:    req.Model,
		Vectors:  vectors,
		Usage:    usage,
		Raw:      rawJSON(resp),
	}, nil
}

func (c *Client) ListModels(ctx context.Context) ([]provider.ModelInfo, error) {
	page, err := c.sdk.Models.List(ctx, &genai.ListModelsConfig{QueryBase: genai.Ptr(true)})
	if err != nil {
		return nil, fmt.Errorf("gemini model list via SDK failed: %w", err)
	}
	models := make([]provider.ModelInfo, 0)
	for {
		for _, item := range page.Items {
			if item == nil {
				continue
			}
			kind := provider.InferModelKindFromMethods(item.SupportedActions)
			if kind != domain.ModelKindEmbedding {
				kind = provider.InferModelKind(item.Name)
			}
			id := strings.TrimPrefix(item.Name, "models/")
			models = append(models, provider.ModelInfo{
				ID:                  id,
				Name:                item.Name,
				DisplayName:         firstNonEmpty(item.DisplayName, item.Name),
				Provider:            domain.ProviderGemini,
				Kind:                kind,
				ContextWindow:       int(item.InputTokenLimit),
				MaxOutputTokens:     int(item.OutputTokenLimit),
				SupportsTools:       kind == domain.ModelKindText,
				SupportsToolsKnown:  false,
				SupportsStream:      kind == domain.ModelKindText,
				SupportsStreamKnown: false,
				Raw:                 rawJSON(item),
			})
		}
		next, err := page.Next(ctx)
		if errors.Is(err, genai.ErrPageDone) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("gemini model list next page via SDK failed: %w", err)
		}
		page = next
	}
	return models, nil
}

func timeoutFromConfig(cfg domain.ProviderConfig, fallback time.Duration) time.Duration {
	if cfg.DefaultRequestTimeoutSec > 0 {
		return time.Duration(cfg.DefaultRequestTimeoutSec) * time.Second
	}
	return fallback
}

func geminiGenerateConfig(req provider.TextRequest) (*genai.GenerateContentConfig, error) {
	config := &genai.GenerateContentConfig{}
	if strings.TrimSpace(req.SystemPrompt) != "" {
		config.SystemInstruction = genai.NewContentFromText(req.SystemPrompt, genai.RoleUser)
	}
	if req.MaxOutputTokens > 0 {
		config.MaxOutputTokens = int32(req.MaxOutputTokens)
	}
	if req.Temperature > 0 {
		config.Temperature = genai.Ptr(float32(req.Temperature))
	}
	if req.TopP > 0 {
		config.TopP = genai.Ptr(float32(req.TopP))
	}
	if len(req.Metadata) > 0 {
		config.Labels = req.Metadata
	}
	if len(req.Tools) > 0 {
		tools, err := geminiTools(req.Tools)
		if err != nil {
			return nil, err
		}
		config.Tools = tools
	}
	return config, nil
}

func geminiContents(req provider.TextRequest) []*genai.Content {
	messages := make([]*genai.Content, 0, len(req.Messages)+1)
	for _, msg := range req.Messages {
		content := provider.MessageContent(msg)
		if strings.TrimSpace(content) == "" || strings.EqualFold(msg.Role, "system") || strings.EqualFold(msg.Role, "developer") {
			continue
		}
		role := genai.Role(genai.RoleUser)
		if strings.EqualFold(msg.Role, "assistant") || strings.EqualFold(msg.Role, "model") {
			role = genai.Role(genai.RoleModel)
		}
		messages = append(messages, genai.NewContentFromText(content, role))
	}
	if strings.TrimSpace(req.UserPrompt) != "" {
		messages = append(messages, genai.NewContentFromText(req.UserPrompt, genai.RoleUser))
	}
	return messages
}

func geminiTools(tools []provider.ToolSpec) ([]*genai.Tool, error) {
	declarations := make([]*genai.FunctionDeclaration, 0, len(tools))
	for _, tool := range tools {
		if strings.TrimSpace(tool.Name) == "" {
			return nil, fmt.Errorf("gemini tool name must not be empty")
		}
		declaration := &genai.FunctionDeclaration{Name: tool.Name, Description: tool.Description}
		if len(tool.Parameters) > 0 {
			var params map[string]any
			if err := json.Unmarshal(tool.Parameters, &params); err != nil {
				return nil, fmt.Errorf("gemini tool %q parameters must be a JSON object: %w", tool.Name, err)
			}
			if params == nil {
				return nil, fmt.Errorf("gemini tool %q parameters must be a JSON object", tool.Name)
			}
			declaration.ParametersJsonSchema = params
		}
		declarations = append(declarations, declaration)
	}
	return []*genai.Tool{{FunctionDeclarations: declarations}}, nil
}

func geminiFunctionCalls(calls []*genai.FunctionCall) []provider.ToolCall {
	result := make([]provider.ToolCall, 0, len(calls))
	for _, call := range calls {
		if call == nil || strings.TrimSpace(call.Name) == "" {
			continue
		}
		raw, _ := json.Marshal(call.Args)
		result = append(result, provider.ToolCall{ID: call.ID, Type: "function_call", Name: call.Name, Arguments: raw})
	}
	return result
}

func geminiUsage(raw *genai.GenerateContentResponseUsageMetadata) provider.Usage {
	if raw == nil {
		return provider.Usage{}
	}
	return provider.Usage{
		InputTokens:  int(raw.PromptTokenCount),
		OutputTokens: int(raw.CandidatesTokenCount),
		TotalTokens:  int(raw.TotalTokenCount),
	}
}

func geminiFinishReason(resp *genai.GenerateContentResponse) string {
	if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0] == nil {
		return ""
	}
	return string(resp.Candidates[0].FinishReason)
}

func float32ToFloat64(values []float32) []float64 {
	result := make([]float64, len(values))
	for i, value := range values {
		result[i] = float64(value)
	}
	return result
}

func estimatedInputTokens(inputs []string) int {
	total := 0
	for _, input := range inputs {
		total += len(strings.Fields(input))
	}
	return total
}

func rawJSON(value any) json.RawMessage {
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
