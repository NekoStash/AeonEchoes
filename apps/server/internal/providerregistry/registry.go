package providerregistry

import (
	"fmt"
	"net/http"
	"time"

	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/provider"
	"aeonechoes/server/internal/provider/anthropic"
	"aeonechoes/server/internal/provider/gemini"
	"aeonechoes/server/internal/provider/openai"
	"aeonechoes/server/internal/provider/openairesponses"
)

// Registry maps provider types to concrete adapter factories without creating imports from provider back to adapters.
type Registry struct {
	factories map[domain.ProviderType]provider.ProviderFactory
}

// New creates the default registry containing all supported provider adapters.
func New(client *http.Client, timeout time.Duration) *Registry {
	return &Registry{factories: map[domain.ProviderType]provider.ProviderFactory{
		domain.ProviderOpenAIResponses: openairesponses.Factory{HTTPClient: client, Timeout: timeout},
		domain.ProviderOpenAI:          openai.Factory{HTTPClient: client, Timeout: timeout},
		domain.ProviderAnthropic:       anthropic.Factory{HTTPClient: client, Timeout: timeout},
		domain.ProviderGemini:          gemini.Factory{HTTPClient: client, Timeout: timeout},
	}}
}

func (r *Registry) Factory(providerType domain.ProviderType) (provider.ProviderFactory, error) {
	if r == nil {
		return nil, fmt.Errorf("provider registry is nil")
	}
	factory, ok := r.factories[providerType]
	if !ok {
		return nil, fmt.Errorf("unsupported provider type %q", providerType)
	}
	return factory, nil
}

func (r *Registry) NewTextClient(cfg domain.ProviderConfig) (provider.TextModelClient, error) {
	factory, err := r.Factory(cfg.Type)
	if err != nil {
		return nil, err
	}
	return factory.NewTextClient(cfg)
}

func (r *Registry) NewEmbeddingClient(cfg domain.ProviderConfig) (provider.EmbeddingModelClient, error) {
	factory, err := r.Factory(cfg.Type)
	if err != nil {
		return nil, err
	}
	return factory.NewEmbeddingClient(cfg)
}

func (r *Registry) NewModelListClient(cfg domain.ProviderConfig) (provider.ModelListClient, error) {
	factory, err := r.Factory(cfg.Type)
	if err != nil {
		return nil, err
	}
	return factory.NewModelListClient(cfg)
}
