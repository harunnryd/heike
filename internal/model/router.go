package model

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"

	"github.com/harunnryd/heike/internal/config"
	heikeErrors "github.com/harunnryd/heike/internal/errors"
	"github.com/harunnryd/heike/internal/logger"
	"github.com/harunnryd/heike/internal/model/contract"
	anthropicProvider "github.com/harunnryd/heike/internal/model/providers/anthropic"
	codexProvider "github.com/harunnryd/heike/internal/model/providers/codex"
	geminiProvider "github.com/harunnryd/heike/internal/model/providers/gemini"
	openaiProvider "github.com/harunnryd/heike/internal/model/providers/openai"
	zaiProvider "github.com/harunnryd/heike/internal/model/providers/zai"
)

// DefaultModelRouter implements ModelRouter interface
type DefaultModelRouter struct {
	cfg       config.ModelsConfig
	providers map[string]Provider
	mu        sync.RWMutex
}

// NewModelRouter creates a new model router
func NewModelRouter(cfg config.ModelsConfig) (*DefaultModelRouter, error) {
	router := &DefaultModelRouter{
		cfg:       cfg,
		providers: make(map[string]Provider),
	}

	if err := router.initProviders(); err != nil {
		return nil, err
	}

	return router, nil
}

// Route routes a completion request to the appropriate provider
func (r *DefaultModelRouter) Route(ctx context.Context, model string, req contract.CompletionRequest) (*contract.CompletionResponse, error) {
	traceID := logger.GetTraceID(ctx)

	slog.Info("Routing completion request", "model", model, "trace_id", traceID)

	provider, err := r.resolveProvider(ctx, model)
	if err != nil {
		return nil, err
	}

	resp, err := r.executeWithFallback(ctx, model, provider, req, traceID)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// RouteEmbedding routes an embedding request to the appropriate provider
func (r *DefaultModelRouter) RouteEmbedding(ctx context.Context, model string, text string) ([]float32, error) {
	traceID := logger.GetTraceID(ctx)

	slog.Info("Routing embedding request", "model", model, "trace_id", traceID)

	tryModels := r.embeddingTryOrder(model)
	var lastErr error

	for _, tryModel := range tryModels {
		select {
		case <-ctx.Done():
			return nil, heikeErrors.Wrap(ctx.Err(), "embedding request cancelled")
		default:
		}

		r.mu.RLock()
		provider, exists := r.providers[tryModel]
		r.mu.RUnlock()
		if !exists {
			continue
		}

		embeddings, err := provider.Embed(ctx, text)
		if err == nil {
			slog.Info("Embedding completed", "model", tryModel, "trace_id", traceID)
			return embeddings, nil
		}

		if isEmbeddingUnsupported(err) {
			slog.Warn("Embedding unsupported by provider, trying next model", "model", tryModel, "error", err, "trace_id", traceID)
			continue
		}

		lastErr = err
		slog.Warn("Embedding failed for model, trying next model", "model", tryModel, "error", err, "trace_id", traceID)
	}

	if lastErr != nil {
		return nil, heikeErrors.WrapWithCategory(lastErr, "embedding failed", heikeErrors.ErrInternal)
	}

	return nil, heikeErrors.NotFound("no embedding-capable model configured")
}

func (r *DefaultModelRouter) embeddingTryOrder(requestedModel string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	seen := make(map[string]struct{}, len(r.providers)+2)
	order := make([]string, 0, len(r.providers)+2)

	appendUnique := func(name string) {
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		order = append(order, name)
	}

	appendUnique(requestedModel)
	appendUnique(r.cfg.Fallback)

	registered := make([]string, 0, len(r.providers))
	for name := range r.providers {
		registered = append(registered, name)
	}
	sort.Strings(registered)

	for _, name := range registered {
		appendUnique(name)
	}

	return order
}

func isEmbeddingUnsupported(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "embedding not supported") ||
		strings.Contains(msg, "embeddings not implemented") ||
		strings.Contains(msg, "not support embeddings")
}

// ListModels returns all registered model names
func (r *DefaultModelRouter) ListModels() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	models := make([]string, 0, len(r.providers))
	for name := range r.providers {
		models = append(models, name)
	}

	return models
}

// Health checks the health of the router and its providers
func (r *DefaultModelRouter) Health(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for name, provider := range r.providers {
		if err := provider.Health(ctx); err != nil {
			slog.Warn("Provider unhealthy", "provider", name, "error", err)
			return heikeErrors.Transient(fmt.Sprintf("provider %s unhealthy", name))
		}
	}

	return nil
}

// initProviders initializes all providers from configuration
func (r *DefaultModelRouter) initProviders() error {
	for _, entry := range r.cfg.Registry {
		provider, err := r.createProvider(entry)
		if err != nil {
			slog.Warn("Failed to create provider", "provider", entry.Provider, "model", entry.Name, "error", err)
			continue
		}

		r.providers[entry.Name] = provider
		slog.Info("Provider initialized", "name", entry.Name, "type", entry.Provider)
	}

	if len(r.providers) == 0 && len(r.cfg.Registry) > 0 {
		return heikeErrors.Internal("no providers initialized")
	}

	return nil
}

// resolveProvider resolves a provider by model name with fallback
func (r *DefaultModelRouter) resolveProvider(ctx context.Context, model string) (Provider, error) {
	select {
	case <-ctx.Done():
		return nil, heikeErrors.Wrap(ctx.Err(), "provider resolution cancelled")
	default:
	}

	r.mu.RLock()
	provider, exists := r.providers[model]
	r.mu.RUnlock()

	if !exists {
		slog.Warn("Model not found", "model", model)

		if r.cfg.Fallback != "" && model != r.cfg.Fallback {
			slog.Info("Trying fallback model", "model", model, "fallback", r.cfg.Fallback)

			fallbackProvider, fallbackExists := r.providers[r.cfg.Fallback]
			if !fallbackExists {
				return nil, heikeErrors.NotFound(fmt.Sprintf("model %s not found", model))
			}

			return fallbackProvider, nil
		}

		return nil, heikeErrors.NotFound(fmt.Sprintf("model %s not found", model))
	}

	return provider, nil
}

// executeWithFallback executes a request with fallback logic
func (r *DefaultModelRouter) executeWithFallback(ctx context.Context, model string, provider Provider, req contract.CompletionRequest, traceID string) (*contract.CompletionResponse, error) {
	maxAttempts := r.cfg.MaxFallbackAttempts
	if maxAttempts <= 0 {
		maxAttempts = config.DefaultModelMaxFallbackAttempts
	}
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	currentModel := model
	currentProvider := provider

	for attempt := 0; attempt < maxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return nil, heikeErrors.Wrap(ctx.Err(), "request execution cancelled")
		default:
		}

		resp, err := currentProvider.Generate(ctx, req)
		if err == nil {
			slog.Info("Request completed", "model", currentModel, "attempt", attempt+1, "trace_id", traceID)
			return resp, nil
		}

		slog.Error("Provider request failed", "model", currentModel, "attempt", attempt+1, "error", err)

		if attempt == 0 && currentModel == r.cfg.Fallback {
			return nil, heikeErrors.WrapWithCategory(err, "provider request failed", heikeErrors.ErrInternal)
		}

		if r.cfg.Fallback == "" || currentModel == r.cfg.Fallback {
			return nil, heikeErrors.WrapWithCategory(err, "provider request failed", heikeErrors.ErrInternal)
		}

		slog.Info("Attempting fallback", "from", currentModel, "to", r.cfg.Fallback)

		fallbackProvider, exists := r.providers[r.cfg.Fallback]
		if !exists {
			return nil, heikeErrors.NotFound(fmt.Sprintf("fallback model %s not found", r.cfg.Fallback))
		}

		currentModel = r.cfg.Fallback
		currentProvider = fallbackProvider
	}

	return nil, heikeErrors.Internal("fallback exhausted")
}

// createProvider creates a provider instance based on registry entry
func (r *DefaultModelRouter) createProvider(entry config.ModelRegistry) (Provider, error) {
	switch entry.Provider {
	case "openai":
		baseURL := entry.BaseURL
		if baseURL == "" {
			baseURL = config.DefaultOpenAIBaseURL
		}

		if entry.APIKey == "" {
			return nil, heikeErrors.InvalidInput("API key required for OpenAI provider")
		}

		return &ProviderAdapter{
			provider:     openaiProvider.New(entry.APIKey, baseURL, entry.Name),
			name:         entry.Name,
			providerType: "openai",
		}, nil

	case "ollama":
		baseURL := entry.BaseURL
		if baseURL == "" {
			baseURL = config.DefaultOllamaBaseURL
		}

		apiKey := entry.APIKey
		if apiKey == "" {
			apiKey = config.DefaultOllamaAPIKey
		}

		return &ProviderAdapter{
			provider:     openaiProvider.New(apiKey, baseURL, entry.Name),
			name:         entry.Name,
			providerType: "ollama",
		}, nil

	case "anthropic":
		if entry.APIKey == "" {
			return nil, heikeErrors.InvalidInput("API key required for Anthropic provider")
		}

		return &ProviderAdapter{
			provider:     anthropicProvider.New(entry.APIKey),
			name:         entry.Name,
			providerType: "anthropic",
		}, nil

	case "gemini":
		if entry.APIKey == "" {
			return nil, heikeErrors.InvalidInput("API key required for Gemini provider")
		}

		provider, err := geminiProvider.New(entry.APIKey)
		if err != nil {
			return nil, heikeErrors.WrapWithCategory(err, "failed to create Gemini provider", heikeErrors.ErrInternal)
		}

		return &ProviderAdapter{
			provider:     provider,
			name:         entry.Name,
			providerType: "gemini",
		}, nil

	case "zai":
		if entry.APIKey == "" {
			return nil, heikeErrors.InvalidInput("API key required for Zai provider")
		}

		provider, err := zaiProvider.New(entry.APIKey, entry.Name)
		if err != nil {
			return nil, heikeErrors.WrapWithCategory(err, "failed to create Zai provider", heikeErrors.ErrInternal)
		}

		return &ProviderAdapter{
			provider:     provider,
			name:         entry.Name,
			providerType: "zai",
		}, nil

	case "openai-codex":
		requestTimeout, err := config.DurationOrDefault(entry.RequestTimeout, config.DefaultCodexRequestTimeout)
		if err != nil {
			return nil, heikeErrors.InvalidInput(fmt.Sprintf("invalid request_timeout for openai-codex model %s: %v", entry.Name, err))
		}

		embeddingInputMaxChars := entry.EmbeddingInputMaxChars
		if embeddingInputMaxChars <= 0 {
			embeddingInputMaxChars = config.DefaultCodexEmbeddingInputMaxChars
		}

		// API Key is optional here because we might use OAuth token from file
		return &ProviderAdapter{
			provider: codexProvider.New(entry.APIKey, entry.BaseURL, entry.AuthFile, codexProvider.RuntimeConfig{
				RequestTimeout:         requestTimeout,
				EmbeddingInputMaxChars: embeddingInputMaxChars,
			}),
			name:         entry.Name,
			providerType: "openai-codex",
		}, nil

	default:
		return nil, heikeErrors.InvalidInput(fmt.Sprintf("unknown provider type: %s", entry.Provider))
	}
}
