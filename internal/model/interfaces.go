package model

import (
	"context"

	"github.com/harunnryd/heike/internal/model/contract"
)

// Core Interfaces

// ModelRouter is the main facade for model routing and request management
type ModelRouter interface {
	Route(ctx context.Context, model string, req contract.CompletionRequest) (*contract.CompletionResponse, error)
	RouteEmbedding(ctx context.Context, model string, text string) ([]float32, error)
	ListModels() []string
	Health(ctx context.Context) error
}

// Provider is abstraction for AI model providers
type Provider interface {
	Generate(ctx context.Context, req contract.CompletionRequest) (*contract.CompletionResponse, error)
	Embed(ctx context.Context, text string) ([]float32, error)
	Name() string
	Type() string
	Health(ctx context.Context) error
}

// ProviderConfig is the configuration interface for creating providers
type ProviderConfig interface {
	Name() string
	Type() string
	APIKey() string
	BaseURL() string
	Options() map[string]interface{}
}

// Factory Interfaces

// ProviderFactory creates Provider instances with proper validation
type ProviderFactory interface {
	CreateProvider(ctx context.Context, config ProviderConfig) (Provider, error)
	SupportedTypes() []string
	ValidateConfig(config ProviderConfig) error
}

// FactoryRegistry manages provider factories
type FactoryRegistry interface {
	RegisterFactory(providerType string, factory ProviderFactory)
	CreateProvider(ctx context.Context, config ProviderConfig) (Provider, error)
	GetFactory(providerType string) (ProviderFactory, error)
	SupportedTypes() []string
}

// Fallback Interfaces

// FallbackStrategy defines fallback behavior
type FallbackStrategy interface {
	ShouldFallback(model string, err error) bool
	GetFallbackModel(original string) string
	ExecuteFallback(ctx context.Context, req contract.CompletionRequest) (*contract.CompletionResponse, error)
	MaxDepth() int
}

// FallbackChain manages multiple fallback strategies
type FallbackChain interface {
	AddStrategy(strategy FallbackStrategy)
	Execute(ctx context.Context, model string, req contract.CompletionRequest) (*contract.CompletionResponse, error)
	Clear()
}

// Error Handling Interfaces

// ErrorMapper maps provider errors to Heike error taxonomy
type ErrorMapper interface {
	MapError(providerType string, err error) error
	IsRetryable(err error) bool
	Category(err error) string
}

// Tracing Interfaces

// Tracer provides distributed tracing capabilities
type Tracer interface {
	StartTrace(ctx context.Context, name string, metadata map[string]interface{}) string
	EndTrace(traceID string)
	RecordError(traceID string, err error)
	RecordSuccess(traceID string)
}

// TracingProvider wraps provider with tracing
type TracingProvider interface {
	Provider
	WithTracer(tracer Tracer) TracingProvider
}

// Component Interfaces

// Component defines the lifecycle interface for model components
type Component interface {
	Init(ctx context.Context) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Health(ctx context.Context) error
}

// Configuration Interfaces

// ConfigValidator validates provider configurations
type ConfigValidator interface {
	Validate(config ProviderConfig) error
}

// RouterConfig provides configuration for ModelRouter
type RouterConfig struct {
	FactoryRegistry  FactoryRegistry
	FallbackChain    FallbackChain
	ErrorMapper      ErrorMapper
	Tracer           Tracer
	MaxFallbackDepth int
}

// HealthChecker provides health check capabilities
type HealthChecker interface {
	Health(ctx context.Context) error
}

// Closer provides cleanup capabilities
type Closer interface {
	Close() error
}

// ClosableProvider combines Provider with cleanup capabilities
type ClosableProvider interface {
	Provider
	Closer
}

// LifecycleComponent combines Component with health checking
type LifecycleComponent interface {
	Component
	HealthChecker
}
