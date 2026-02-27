package initializers

import (
	"context"
	"fmt"

	"github.com/harunnryd/heike/internal/concurrency"
	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/ingress"
	"github.com/harunnryd/heike/internal/orchestrator"
	"github.com/harunnryd/heike/internal/store"
	"github.com/harunnryd/heike/internal/worker"
)

type WorkersInitializer struct {
	ingress      *ingress.Ingress
	orchestrator orchestrator.Kernel
	storeWorker  *store.Worker
}

func NewWorkersInitializer(ingress *ingress.Ingress, orchestrator orchestrator.Kernel, storeWorker *store.Worker) *WorkersInitializer {
	return &WorkersInitializer{
		ingress:      ingress,
		orchestrator: orchestrator,
		storeWorker:  storeWorker,
	}
}

func (wi *WorkersInitializer) Name() string {
	return "workers"
}

func (wi *WorkersInitializer) Dependencies() []string {
	return []string{"store", "orchestrator"}
}

func (wi *WorkersInitializer) Initialize(ctx context.Context, cfg *config.Config, workspaceID string) (interface{}, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	interactiveQueueSize := cfg.Ingress.InteractiveQueueSize
	backgroundQueueSize := cfg.Ingress.BackgroundQueueSize
	if interactiveQueueSize <= 0 {
		interactiveQueueSize = config.DefaultIngressInteractiveQueue
	}
	if backgroundQueueSize <= 0 {
		backgroundQueueSize = config.DefaultIngressBackgroundQueue
	}
	interactiveSubmitTimeout, err := config.DurationOrDefault(cfg.Ingress.InteractiveSubmitTimeout, config.DefaultIngressInteractiveSubmitTimeout)
	if err != nil {
		return nil, fmt.Errorf("parse ingress interactive submit timeout: %w", err)
	}
	drainTimeout, err := config.DurationOrDefault(cfg.Ingress.DrainTimeout, config.DefaultIngressDrainTimeout)
	if err != nil {
		return nil, fmt.Errorf("parse ingress drain timeout: %w", err)
	}
	drainPollInterval, err := config.DurationOrDefault(cfg.Ingress.DrainPollInterval, config.DefaultIngressDrainPollInterval)
	if err != nil {
		return nil, fmt.Errorf("parse ingress drain poll interval: %w", err)
	}
	idempotencyTTL, err := config.DurationOrDefault(cfg.Governance.IdempotencyTTL, config.DefaultGovernanceIdempotencyTTL)
	if err != nil {
		return nil, fmt.Errorf("parse governance idempotency ttl: %w", err)
	}
	workerShutdownTimeout, err := config.DurationOrDefault(cfg.Worker.ShutdownTimeout, config.DefaultWorkerShutdownTimeout)
	if err != nil {
		return nil, fmt.Errorf("parse worker shutdown timeout: %w", err)
	}

	if wi.ingress == nil {
		wi.ingress = ingress.NewIngress(
			interactiveQueueSize,
			backgroundQueueSize,
			ingress.RuntimeConfig{
				InteractiveSubmitTimeout: interactiveSubmitTimeout,
				DrainTimeout:             drainTimeout,
				DrainPollInterval:        drainPollInterval,
				IdempotencyTTL:           idempotencyTTL,
			},
			wi.storeWorker,
		)
	}

	if wi.orchestrator == nil {
		return nil, fmt.Errorf("orchestrator not initialized")
	}

	locks := concurrency.NewSimpleSessionLockManager()

	interactiveWorker := worker.NewWorker(
		"interactive",
		wi.ingress.InteractiveQueue(),
		wi.storeWorker,
		wi.orchestrator,
		locks,
		worker.RuntimeConfig{ShutdownTimeout: workerShutdownTimeout},
	)

	backgroundWorker := worker.NewWorker(
		"background",
		wi.ingress.BackgroundQueue(),
		wi.storeWorker,
		wi.orchestrator,
		locks,
		worker.RuntimeConfig{ShutdownTimeout: workerShutdownTimeout},
	)

	return struct {
		Ingress           *ingress.Ingress
		InteractiveWorker *worker.Worker
		BackgroundWorker  *worker.Worker
		Locks             *concurrency.SimpleSessionLockManager
	}{
		Ingress:           wi.ingress,
		InteractiveWorker: interactiveWorker,
		BackgroundWorker:  backgroundWorker,
		Locks:             locks,
	}, nil
}
