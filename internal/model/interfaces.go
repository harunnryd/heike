package model

import (
	"context"

	"github.com/harunnryd/heike/internal/model/contract"
)

type ModelRouter interface {
	Route(ctx context.Context, model string, req contract.CompletionRequest) (*contract.CompletionResponse, error)
	RouteEmbedding(ctx context.Context, model string, text string) ([]float32, error)
	ListModels() []string
	Health(ctx context.Context) error
}

type Provider interface {
	Generate(ctx context.Context, req contract.CompletionRequest) (*contract.CompletionResponse, error)
	Embed(ctx context.Context, text string) ([]float32, error)
	Name() string
	Type() string
	Health(ctx context.Context) error
}

type ProviderConfig interface {
	Name() string
	Type() string
	APIKey() string
	BaseURL() string
	Options() map[string]interface{}
}
