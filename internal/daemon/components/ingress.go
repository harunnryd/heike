package components

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/daemon"
	"github.com/harunnryd/heike/internal/ingress"
)

type IngressComponent struct {
	ingress         *ingress.Ingress
	storeWorkerComp *StoreWorkerComponent
	cfg             *config.IngressConfig
	governanceCfg   *config.GovernanceConfig
	initialized     bool
	started         bool
	mu              sync.RWMutex
	startTime       time.Time
}

func NewIngressComponent(storeComp *StoreWorkerComponent, cfg *config.IngressConfig, governanceCfg *config.GovernanceConfig) *IngressComponent {
	return &IngressComponent{
		storeWorkerComp: storeComp,
		cfg:             cfg,
		governanceCfg:   governanceCfg,
		initialized:     false,
		started:         false,
	}
}

func (i *IngressComponent) Name() string {
	return "Ingress"
}

func (i *IngressComponent) Dependencies() []string {
	return []string{"StoreWorker"}
}

func (i *IngressComponent) Init(ctx context.Context) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.storeWorkerComp == nil {
		return fmt.Errorf("storeWorkerComp not provided")
	}

	storeWorker := i.storeWorkerComp.GetWorker()
	if storeWorker == nil {
		return fmt.Errorf("storeWorker not initialized")
	}

	if i.cfg == nil {
		return fmt.Errorf("ingress config not provided")
	}

	interactiveSubmitTimeout, err := config.DurationOrDefault(i.cfg.InteractiveSubmitTimeout, config.DefaultIngressInteractiveSubmitTimeout)
	if err != nil {
		return fmt.Errorf("parse ingress interactive submit timeout: %w", err)
	}
	drainTimeout, err := config.DurationOrDefault(i.cfg.DrainTimeout, config.DefaultIngressDrainTimeout)
	if err != nil {
		return fmt.Errorf("parse ingress drain timeout: %w", err)
	}
	drainPollInterval, err := config.DurationOrDefault(i.cfg.DrainPollInterval, config.DefaultIngressDrainPollInterval)
	if err != nil {
		return fmt.Errorf("parse ingress drain poll interval: %w", err)
	}
	idempotencyTTLValue := ""
	if i.governanceCfg != nil {
		idempotencyTTLValue = i.governanceCfg.IdempotencyTTL
	}
	idempotencyTTL, err := config.DurationOrDefault(idempotencyTTLValue, config.DefaultGovernanceIdempotencyTTL)
	if err != nil {
		return fmt.Errorf("parse governance idempotency ttl: %w", err)
	}

	i.ingress = ingress.NewIngress(
		i.cfg.InteractiveQueueSize,
		i.cfg.BackgroundQueueSize,
		ingress.RuntimeConfig{
			InteractiveSubmitTimeout: interactiveSubmitTimeout,
			DrainTimeout:             drainTimeout,
			DrainPollInterval:        drainPollInterval,
			IdempotencyTTL:           idempotencyTTL,
		},
		storeWorker,
	)
	i.initialized = true
	slog.Info("Ingress initialized", "component", i.Name())
	return nil
}

func (i *IngressComponent) Start(ctx context.Context) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if !i.initialized {
		return fmt.Errorf("Ingress not initialized")
	}

	i.started = true
	i.startTime = time.Now()
	slog.Info("Ingress started", "component", i.Name())
	return nil
}

func (i *IngressComponent) Stop(ctx context.Context) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if !i.started {
		slog.Info("Ingress not started, skipping stop", "component", i.Name())
		return nil
	}

	slog.Info("Stopping Ingress...", "component", i.Name())
	if i.ingress != nil {
		i.ingress.Close()
	}
	i.started = false
	slog.Info("Ingress stopped", "component", i.Name())
	return nil
}

func (i *IngressComponent) Health(ctx context.Context) (*daemon.ComponentHealth, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if !i.started {
		return &daemon.ComponentHealth{
			Name:    i.Name(),
			Healthy: false,
			Error:   fmt.Errorf("not started"),
		}, nil
	}

	return &daemon.ComponentHealth{
		Name:    i.Name(),
		Healthy: true,
		Error:   nil,
	}, nil
}

func (i *IngressComponent) GetIngress() *ingress.Ingress {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.ingress
}
