package egress

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/harunnryd/heike/internal/adapter"
	"github.com/harunnryd/heike/internal/errors"
	"github.com/harunnryd/heike/internal/store"
)

type Egress interface {
	// Register registers an output adapter
	Register(adapter adapter.OutputAdapter) error

	// Unregister removes an output adapter
	Unregister(name string) error

	// Send sends content to appropriate adapter based on session metadata
	Send(ctx context.Context, sessionID string, content string) error

	// Health checks egress health and all registered adapters
	Health(ctx context.Context) error

	// ListAdapters returns all registered adapters
	ListAdapters() []adapter.OutputAdapter
}

type DefaultEgress struct {
	mu       sync.RWMutex
	adapters map[string]adapter.OutputAdapter
	store    *store.Worker
}

func NewEgress(store *store.Worker) Egress {
	return &DefaultEgress{
		adapters: make(map[string]adapter.OutputAdapter),
		store:    store,
	}
}

func (e *DefaultEgress) Register(adapter adapter.OutputAdapter) error {
	if adapter == nil {
		return errors.InvalidInput("adapter cannot be nil")
	}

	name := adapter.Name()
	if name == "" {
		return errors.InvalidInput("adapter name cannot be empty")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.adapters[name]; exists {
		return errors.ErrConflict
	}

	e.adapters[name] = adapter
	slog.Info("Egress adapter registered", "name", name)
	return nil
}

func (e *DefaultEgress) Unregister(name string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.adapters[name]; !exists {
		return errors.NotFound("adapter not found: " + name)
	}

	delete(e.adapters, name)
	slog.Info("Egress adapter unregistered", "name", name)
	return nil
}

func (e *DefaultEgress) Send(ctx context.Context, sessionID string, content string) error {
	// Resolve Session
	sess, err := e.store.GetSession(sessionID)
	if err != nil {
		return errors.Wrap(err, "failed to get session")
	}
	if sess == nil {
		return errors.NotFound("session not found: " + sessionID)
	}

	// Identify Source from Metadata
	source, ok := sess.Metadata["source"]
	if !ok || source == "" {
		slog.Warn("Session has no source metadata, cannot route response", "session", sessionID)
		return errors.InvalidInput("session source metadata missing")
	}

	// Select Adapter
	adapter, err := e.getAdapter(source)
	if err != nil {
		return err
	}

	// Send
	if err := adapter.Send(ctx, sessionID, content); err != nil {
		return errors.Wrap(err, "failed to send response")
	}

	slog.Debug("Response sent", "session", sessionID, "source", source, "content_length", len(content))
	return nil
}

func (e *DefaultEgress) getAdapter(name string) (adapter.OutputAdapter, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	adapter, ok := e.adapters[name]
	if !ok {
		return nil, errors.NotFound("no adapter found for source: " + name)
	}

	return adapter, nil
}

func (e *DefaultEgress) Health(ctx context.Context) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if len(e.adapters) == 0 {
		return errors.Internal("no adapters registered")
	}

	var unhealthy []string
	for name, adapter := range e.adapters {
		if err := adapter.Health(ctx); err != nil {
			unhealthy = append(unhealthy, name)
			slog.Warn("Adapter unhealthy", "name", name, "error", err)
		}
	}

	if len(unhealthy) > 0 {
		return errors.Transient(fmt.Sprintf("%d adapter(s) unhealthy: %v", len(unhealthy), unhealthy))
	}

	return nil
}

func (e *DefaultEgress) ListAdapters() []adapter.OutputAdapter {
	e.mu.RLock()
	defer e.mu.RUnlock()

	adapters := make([]adapter.OutputAdapter, 0, len(e.adapters))
	for _, adapter := range e.adapters {
		adapters = append(adapters, adapter)
	}
	return adapters
}
