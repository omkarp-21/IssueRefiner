package providers

import (
	"context"
	"fmt"
)

type Provider interface {
	Type() string
	SupportsModel(model string) bool
	ValidateCredentials(ctx context.Context, apiKey string, model string) error
	Generate(ctx context.Context, apiKey string, model string, input Input) (Result, error)
}

type ModelDiscoverer interface {
	DiscoverModels(ctx context.Context, apiKey string) ([]string, error)
}

type Registry struct {
	providers map[string]Provider
}

func NewRegistry(openAI Provider, gemini Provider, mock Provider) Registry {
	return Registry{providers: map[string]Provider{
		ProviderOpenAI: openAI,
		ProviderGemini: gemini,
		ProviderMock:   mock,
	}}
}

func DefaultRegistry() Registry {
	return NewRegistry(
		NewOpenAIProvider("https://api.openai.com/v1"),
		NewGeminiProvider("https://generativelanguage.googleapis.com/v1beta"),
		NewMockProvider(),
	)
}

func (registry *Registry) Get(providerType string) (Provider, error) {
	provider, ok := registry.providers[providerType]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownProvider, providerType)
	}
	return provider, nil
}
