package components

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/harunnryd/heike/internal/adapter"
	"github.com/harunnryd/heike/internal/daemon"
)

type AdaptersComponent struct {
	manager     *adapter.RuntimeManager
	initialized bool
	started     bool
}

func NewAdaptersComponent(manager *adapter.RuntimeManager) *AdaptersComponent {
	return &AdaptersComponent{manager: manager}
}

func (a *AdaptersComponent) Name() string {
	return "Adapters"
}

func (a *AdaptersComponent) Dependencies() []string {
	return []string{"Ingress", "Workers", "Orchestrator"}
}

func (a *AdaptersComponent) Init(ctx context.Context) error {
	if a.manager == nil {
		return fmt.Errorf("adapter manager not configured")
	}
	a.initialized = true
	return nil
}

func (a *AdaptersComponent) Start(ctx context.Context) error {
	if !a.initialized {
		return fmt.Errorf("adapters component not initialized")
	}
	a.manager.Start(ctx)
	a.started = true
	slog.Info("Adapters started", "component", a.Name())
	return nil
}

func (a *AdaptersComponent) Stop(ctx context.Context) error {
	if !a.started {
		return nil
	}
	err := a.manager.Stop(ctx)
	a.started = false
	if err != nil {
		return err
	}
	slog.Info("Adapters stopped", "component", a.Name())
	return nil
}

func (a *AdaptersComponent) Health(ctx context.Context) (*daemon.ComponentHealth, error) {
	if !a.initialized {
		return &daemon.ComponentHealth{Name: a.Name(), Healthy: false, Error: fmt.Errorf("not initialized")}, nil
	}
	if !a.started {
		return &daemon.ComponentHealth{Name: a.Name(), Healthy: false, Error: fmt.Errorf("not started")}, nil
	}
	if err := a.manager.Health(ctx); err != nil {
		return &daemon.ComponentHealth{Name: a.Name(), Healthy: false, Error: err}, nil
	}
	return &daemon.ComponentHealth{Name: a.Name(), Healthy: true}, nil
}
