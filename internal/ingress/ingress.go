package ingress

import (
	"context"
	"log/slog"
	"time"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/errors"
	"github.com/harunnryd/heike/internal/store"
)

type RuntimeConfig struct {
	InteractiveSubmitTimeout time.Duration
	DrainTimeout             time.Duration
	DrainPollInterval        time.Duration
	IdempotencyTTL           time.Duration
}

type Ingress struct {
	interactiveQueue         chan *Event
	backgroundQueue          chan *Event
	store                    *store.Worker
	router                   Router
	resolver                 Resolver
	interactiveSubmitTimeout time.Duration
	drainTimeout             time.Duration
	drainPollInterval        time.Duration
	idempotencyTTL           time.Duration
}

func NewIngress(interactiveSize, backgroundSize int, runtimeCfg RuntimeConfig, store *store.Worker) *Ingress {
	if interactiveSize <= 0 {
		interactiveSize = config.DefaultIngressInteractiveQueue
	}
	if backgroundSize <= 0 {
		backgroundSize = config.DefaultIngressBackgroundQueue
	}

	if runtimeCfg.InteractiveSubmitTimeout <= 0 {
		d, err := config.DurationOrDefault("", config.DefaultIngressInteractiveSubmitTimeout)
		if err == nil {
			runtimeCfg.InteractiveSubmitTimeout = d
		}
	}
	if runtimeCfg.DrainTimeout <= 0 {
		d, err := config.DurationOrDefault("", config.DefaultIngressDrainTimeout)
		if err == nil {
			runtimeCfg.DrainTimeout = d
		}
	}
	if runtimeCfg.DrainPollInterval <= 0 {
		d, err := config.DurationOrDefault("", config.DefaultIngressDrainPollInterval)
		if err == nil {
			runtimeCfg.DrainPollInterval = d
		}
	}
	if runtimeCfg.IdempotencyTTL <= 0 {
		d, err := config.DurationOrDefault("", config.DefaultGovernanceIdempotencyTTL)
		if err == nil {
			runtimeCfg.IdempotencyTTL = d
		}
	}

	return &Ingress{
		interactiveQueue:         make(chan *Event, interactiveSize),
		backgroundQueue:          make(chan *Event, backgroundSize),
		store:                    store,
		router:                   NewStandardRouter(),
		resolver:                 NewStandardResolver(store),
		interactiveSubmitTimeout: runtimeCfg.InteractiveSubmitTimeout,
		drainTimeout:             runtimeCfg.DrainTimeout,
		drainPollInterval:        runtimeCfg.DrainPollInterval,
		idempotencyTTL:           runtimeCfg.IdempotencyTTL,
	}
}

// Submit ingests an event and routes it to the appropriate lane.
// It returns an error if the queue is full (backpressure) or if it's a duplicate.
func (i *Ingress) Submit(ctx context.Context, evt *Event) error {
	if evt == nil {
		return errors.InvalidInput("event is nil")
	}
	if i.store == nil {
		return errors.Internal("store not initialized")
	}
	if i.router == nil {
		return errors.Internal("router not initialized")
	}
	if i.resolver == nil {
		return errors.Internal("resolver not initialized")
	}

	slog.Debug("Ingress received event", "id", evt.ID, "type", evt.Type, "source", evt.Source)

	key := GenerateIdempotencyKey(evt.Source, evt.ID)
	if i.store.CheckAndMarkKey(key, i.idempotencyTTL) {
		slog.Warn("Duplicate event detected", "key", key)
		return errors.ErrDuplicateEvent
	}

	dest := i.router.Route(ctx, evt)
	switch dest.Type {
	case DestDrop:
		slog.Info("Event dropped by router", "id", evt.ID)
		return nil
	case DestCommand:
		slog.Info("Handling as command", "id", evt.ID)
		if dest.Handler != nil {
			return dest.Handler(ctx, evt)
		}
		return nil
	case DestPipeline:
		// Continue to Resolvers -> Queue
	default:
		return errors.InvalidInput("unknown destination type")
	}

	ws, err := i.resolver.ResolveWorkspace(ctx, evt)
	if err != nil {
		return errors.Wrap(err, "workspace resolution failed")
	}
	evt.WorkspaceID = ws

	sess, err := i.resolver.ResolveSession(ctx, evt)
	if err != nil {
		return errors.Wrap(err, "session resolution failed")
	}
	evt.SessionID = sess

	if evt.Type == TypeUserMessage || evt.Type == TypeCommand {
		select {
		case i.interactiveQueue <- evt:
			slog.Debug("Event routed", "id", evt.ID, "lane", "interactive", "session", evt.SessionID)
			return nil
		case <-time.After(i.interactiveSubmitTimeout):
			slog.Warn("Interactive queue full, dropping event", "id", evt.ID)
			return errors.ErrTransient
		case <-ctx.Done():
			return ctx.Err()
		}
	} else {
		select {
		case i.backgroundQueue <- evt:
			slog.Debug("Event routed", "id", evt.ID, "lane", "background", "session", evt.SessionID)
			return nil
		default:
			slog.Warn("Background queue full, dropping event", "id", evt.ID)
			return errors.ErrTransient
		}
	}
}

func (i *Ingress) InteractiveQueue() <-chan *Event {
	return i.interactiveQueue
}

func (i *Ingress) BackgroundQueue() <-chan *Event {
	return i.backgroundQueue
}

// Close gracefully shuts down ingress by draining queues and closing them.
func (i *Ingress) Close() error {
	slog.Info("Ingress shutting down, draining queues")

	drainStart := time.Now()

	drainQueue := func(ch chan *Event, name string) {
		remaining := len(ch)
		if remaining == 0 {
			close(ch)
			return
		}

		slog.Info("Draining queue", "name", name, "remaining", remaining)

		stalled := false
		for remaining > 0 && time.Since(drainStart) < i.drainTimeout {
			select {
			case <-ch:
				remaining--
			case <-time.After(i.drainPollInterval):
				if remaining == len(ch) {
					slog.Warn("Queue drain stalled", "name", name, "remaining", remaining)
					stalled = true
					break
				}
				remaining = len(ch)
			}
			if stalled {
				break
			}
		}

		if remaining > 0 {
			slog.Warn("Queue drain incomplete", "name", name, "remaining", remaining)
		}
		close(ch)
		slog.Info("Queue drained", "name", name)
	}

	drainQueue(i.interactiveQueue, "interactive")
	drainQueue(i.backgroundQueue, "background")

	slog.Info("Ingress shutdown complete")
	return nil
}

// Health checks ingress health
func (i *Ingress) Health(ctx context.Context) error {
	if i.interactiveQueue == nil || i.backgroundQueue == nil {
		return errors.Internal("queues not initialized")
	}

	interactiveUsage := float64(len(i.interactiveQueue)) / float64(cap(i.interactiveQueue))
	backgroundUsage := float64(len(i.backgroundQueue)) / float64(cap(i.backgroundQueue))

	slog.Debug("Ingress health metrics",
		"interactive_queue_len", len(i.interactiveQueue),
		"interactive_queue_cap", cap(i.interactiveQueue),
		"interactive_usage", interactiveUsage,
		"background_queue_len", len(i.backgroundQueue),
		"background_queue_cap", cap(i.backgroundQueue),
		"background_usage", backgroundUsage,
	)

	if interactiveUsage > 0.9 {
		return errors.Transient("interactive queue nearly full")
	}

	if backgroundUsage > 0.9 {
		return errors.Transient("background queue nearly full")
	}

	if i.resolver == nil {
		return errors.Internal("resolver not initialized")
	}

	if i.router == nil {
		return errors.Internal("router not initialized")
	}

	return nil
}
