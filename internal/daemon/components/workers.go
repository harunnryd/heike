package components

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/harunnryd/heike/internal/concurrency"
	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/daemon"
	"github.com/harunnryd/heike/internal/worker"
)

type WorkersComponent struct {
	interactiveWorker *worker.Worker
	backgroundWorker  *worker.Worker
	ingressComp       *IngressComponent
	orchestratorComp  *OrchestratorComponent
	storeWorkerComp   *StoreWorkerComponent
	cfg               *config.Config
	locks             *concurrency.SimpleSessionLockManager
	initialized       bool
	started           bool
	mu                sync.RWMutex
	startTime         time.Time
}

func NewWorkersComponent(cfg *config.Config, ingComp *IngressComponent, orchComp *OrchestratorComponent, storeComp *StoreWorkerComponent) *WorkersComponent {
	locks := concurrency.NewSimpleSessionLockManager()
	return &WorkersComponent{
		ingressComp:      ingComp,
		orchestratorComp: orchComp,
		storeWorkerComp:  storeComp,
		cfg:              cfg,
		locks:            locks,
		initialized:      false,
		started:          false,
	}
}

func (w *WorkersComponent) Name() string {
	return "Workers"
}

func (w *WorkersComponent) Dependencies() []string {
	return []string{"Ingress", "Orchestrator"}
}

func (w *WorkersComponent) Init(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.ingressComp == nil || w.orchestratorComp == nil || w.storeWorkerComp == nil {
		return fmt.Errorf("required component dependencies not provided")
	}
	if w.cfg == nil {
		return fmt.Errorf("config not provided")
	}

	ing := w.ingressComp.GetIngress()
	orch := w.orchestratorComp.GetKernel()
	storeW := w.storeWorkerComp.GetWorker()
	if ing == nil || orch == nil || storeW == nil {
		return fmt.Errorf("required dependencies not initialized")
	}

	workerShutdownTimeout, err := config.DurationOrDefault(w.cfg.Worker.ShutdownTimeout, config.DefaultWorkerShutdownTimeout)
	if err != nil {
		return fmt.Errorf("parse worker shutdown timeout: %w", err)
	}

	w.interactiveWorker = worker.NewWorker("interactive", ing.InteractiveQueue(), storeW, orch, w.locks, worker.RuntimeConfig{ShutdownTimeout: workerShutdownTimeout})
	w.backgroundWorker = worker.NewWorker("background", ing.BackgroundQueue(), storeW, orch, w.locks, worker.RuntimeConfig{ShutdownTimeout: workerShutdownTimeout})

	w.initialized = true
	slog.Info("Workers initialized", "component", w.Name())
	return nil
}

func (w *WorkersComponent) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.initialized {
		return fmt.Errorf("Workers not initialized")
	}

	if _, err := w.interactiveWorker.Start(ctx); err != nil {
		return fmt.Errorf("start interactive worker: %w", err)
	}
	if _, err := w.backgroundWorker.Start(ctx); err != nil {
		return fmt.Errorf("start background worker: %w", err)
	}

	w.started = true
	w.startTime = time.Now()
	slog.Info("Workers started", "component", w.Name())
	return nil
}

func (w *WorkersComponent) Stop(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.started {
		slog.Info("Workers not started, skipping stop", "component", w.Name())
		return nil
	}

	slog.Info("Stopping Workers...", "component", w.Name())
	w.interactiveWorker.Stop(ctx)
	w.backgroundWorker.Stop(ctx)
	w.started = false
	slog.Info("Workers stopped", "component", w.Name())
	return nil
}

func (w *WorkersComponent) Health(ctx context.Context) (*daemon.ComponentHealth, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if !w.initialized {
		return &daemon.ComponentHealth{
			Name:    w.Name(),
			Healthy: false,
			Error:   fmt.Errorf("not initialized"),
		}, nil
	}

	if !w.started {
		return &daemon.ComponentHealth{
			Name:    w.Name(),
			Healthy: false,
			Error:   fmt.Errorf("not started"),
		}, nil
	}

	return &daemon.ComponentHealth{
		Name:    w.Name(),
		Healthy: true,
		Error:   nil,
	}, nil
}
