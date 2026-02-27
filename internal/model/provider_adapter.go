package model

import (
	"context"
	"fmt"

	"github.com/harunnryd/heike/internal/model/contract"
	anthropicProvider "github.com/harunnryd/heike/internal/model/providers/anthropic"
	codexProvider "github.com/harunnryd/heike/internal/model/providers/codex"
	geminiProvider "github.com/harunnryd/heike/internal/model/providers/gemini"
	openaiProvider "github.com/harunnryd/heike/internal/model/providers/openai"
	zaiProvider "github.com/harunnryd/heike/internal/model/providers/zai"
)

// ProviderAdapter wraps provider-specific implementations to satisfy model.Provider.
type ProviderAdapter struct {
	provider     interface{}
	name         string
	providerType string
}

func (a *ProviderAdapter) Generate(ctx context.Context, req contract.CompletionRequest) (*contract.CompletionResponse, error) {
	switch p := a.provider.(type) {
	case *openaiProvider.Provider:
		return p.Generate(ctx, req)
	case *anthropicProvider.Provider:
		return p.Generate(ctx, req)
	case *geminiProvider.Provider:
		return p.Generate(ctx, req)
	case *zaiProvider.Provider:
		return p.Generate(ctx, req)
	case *codexProvider.Provider:
		return p.Generate(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported provider type: %T", a.provider)
	}
}

func (a *ProviderAdapter) Embed(ctx context.Context, text string) ([]float32, error) {
	switch p := a.provider.(type) {
	case *openaiProvider.Provider:
		return p.Embed(ctx, text)
	case *anthropicProvider.Provider:
		return p.Embed(ctx, text)
	case *geminiProvider.Provider:
		return p.Embed(ctx, text)
	case *zaiProvider.Provider:
		return p.Embed(ctx, text)
	case *codexProvider.Provider:
		return p.Embed(ctx, text)
	default:
		return nil, fmt.Errorf("unsupported provider type: %T", a.provider)
	}
}

func (a *ProviderAdapter) Name() string {
	return a.name
}

func (a *ProviderAdapter) Type() string {
	return a.providerType
}

func (a *ProviderAdapter) Health(ctx context.Context) error {
	return nil
}
