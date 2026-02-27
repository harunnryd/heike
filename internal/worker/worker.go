package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/harunnryd/heike/internal/concurrency"
	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/errors"
	"github.com/harunnryd/heike/internal/ingress"
	"github.com/harunnryd/heike/internal/orchestrator"
	"github.com/harunnryd/heike/internal/store"
)

type RuntimeConfig struct {
	ShutdownTimeout time.Duration
}

type Worker struct {
	mu      sync.RWMutex
	started bool
	quit    chan struct{}
	wg      sync.WaitGroup

	lane   string
	events <-chan *ingress.Event
	store  *store.Worker
	orch   orchestrator.Kernel
	locks  *concurrency.SimpleSessionLockManager

	shutdownTimeout time.Duration
}

func NewWorker(lane string, events <-chan *ingress.Event, store *store.Worker, orch orchestrator.Kernel, locks *concurrency.SimpleSessionLockManager, runtimeCfg RuntimeConfig) *Worker {
	if runtimeCfg.ShutdownTimeout <= 0 {
		d, err := config.DurationOrDefault("", config.DefaultWorkerShutdownTimeout)
		if err == nil {
			runtimeCfg.ShutdownTimeout = d
		}
	}

	return &Worker{
		lane:   lane,
		events: events,
		store:  store,
		orch:   orch,
		locks:  locks,

		shutdownTimeout: runtimeCfg.ShutdownTimeout,
	}
}

func (w *Worker) Start(ctx context.Context) (context.Context, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.started {
		return nil, fmt.Errorf("worker already started: %w", errors.InvalidInput("worker already started"))
	}

	w.started = true
	w.quit = make(chan struct{})

	workerCtx, cancel := context.WithCancel(ctx)

	w.wg.Add(1)
	concurrency.SafeGo(func() {
		defer w.wg.Done()
		defer cancel()

		slog.Info("Worker started", "lane", w.lane)
		w.eventLoop(workerCtx)
		slog.Info("Worker stopped", "lane", w.lane)
	}, nil)

	return workerCtx, nil
}

func (w *Worker) eventLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			slog.Info("Worker stopping (context cancelled)", "lane", w.lane)
			return
		case <-w.quit:
			slog.Info("Worker stopping (quit signal)", "lane", w.lane)
			return
		case evt, ok := <-w.events:
			if !ok {
				slog.Info("Worker stopping (channel closed)", "lane", w.lane)
				return
			}
			w.process(ctx, evt)
		}
	}
}

func (w *Worker) process(ctx context.Context, evt *ingress.Event) {
	start := time.Now()

	slog.Info("Processing event",
		"id", evt.ID,
		"lane", w.lane,
		"session_id", evt.SessionID,
		"type", evt.Type)

	if err := w.processEvent(ctx, evt); err != nil {
		slog.Error("Event processing failed",
			"id", evt.ID,
			"lane", w.lane,
			"error", err)
		return
	}

	slog.Debug("Event processed",
		"id", evt.ID,
		"duration", time.Since(start))
}

func (w *Worker) processEvent(ctx context.Context, evt *ingress.Event) error {
	if err := w.validateEvent(evt); err != nil {
		return fmt.Errorf("validate event: %w", err)
	}

	if err := w.acquireSessionLock(ctx, evt.SessionID); err != nil {
		return fmt.Errorf("acquire session lock: %w", err)
	}
	defer w.locks.Unlock(evt.SessionID)

	if err := w.persistUserEvent(ctx, evt); err != nil {
		return fmt.Errorf("persist event: %w", err)
	}

	if err := w.executeOrchestrator(ctx, evt); err != nil {
		return fmt.Errorf("execute orchestrator: %w", err)
	}

	return nil
}

func (w *Worker) validateEvent(evt *ingress.Event) error {
	if evt == nil {
		return fmt.Errorf("event is nil: %w", errors.InvalidInput("event is nil"))
	}

	if evt.ID == "" {
		return fmt.Errorf("event ID is empty: %w", errors.InvalidInput("event ID is empty"))
	}

	if evt.SessionID == "" {
		return fmt.Errorf("session ID is empty: %w", errors.InvalidInput("session ID is empty"))
	}

	if evt.Type == "" {
		return fmt.Errorf("event type is empty: %w", errors.InvalidInput("event type is empty"))
	}

	return nil
}

func (w *Worker) acquireSessionLock(ctx context.Context, sessionID string) error {
	if w.locks == nil {
		slog.Warn("No lock manager configured", "lane", w.lane)
		return nil
	}

	w.locks.Lock(sessionID)
	return nil
}

func (w *Worker) persistUserEvent(ctx context.Context, evt *ingress.Event) error {
	if evt.Type != ingress.TypeUserMessage {
		return nil
	}

	line, err := json.Marshal(map[string]interface{}{
		"id":      evt.ID,
		"type":    "user_event",
		"content": evt.Content,
		"ts":      time.Now(),
	})
	if err != nil {
		return fmt.Errorf("marshal event: %w", errors.Internal("failed to marshal event"))
	}

	if err := w.store.WriteTranscript(evt.SessionID, line); err != nil {
		return fmt.Errorf("write transcript: %w", errors.Transient("storage write failed"))
	}

	return nil
}

func (w *Worker) executeOrchestrator(ctx context.Context, evt *ingress.Event) error {
	if err := w.orch.Execute(ctx, evt); err != nil {
		return fmt.Errorf("orchestrator execution: %w", err)
	}
	return nil
}

func (w *Worker) Stop(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.started {
		slog.Info("Worker not started, skipping stop", "lane", w.lane)
		return nil
	}

	slog.Info("Stopping worker...", "lane", w.lane)

	close(w.quit)

	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("Worker stopped gracefully", "lane", w.lane)
		w.started = false
		return nil
	case <-time.After(w.shutdownTimeout):
		slog.Warn("Worker shutdown timeout, force stopping", "lane", w.lane)
		w.started = false
		return errors.Internal("shutdown timeout")
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *Worker) Health(ctx context.Context) error {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if !w.started {
		return errors.Internal("worker not started")
	}

	if w.events == nil {
		return errors.Internal("event channel not initialized")
	}

	if w.orch == nil {
		return errors.Internal("orchestrator not configured")
	}

	if w.store == nil {
		return errors.Internal("store not configured")
	}

	return nil
}
