package runtime

import (
	"context"
	"fmt"
	"sync"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/daemon"
)

type DaemonRuntimeComponent struct {
	mu          sync.RWMutex
	cfg         *config.Config
	workspaceID string
	adapterOpts AdapterBuildOptions
	runtime     *RuntimeComponents
	initialized bool
	started     bool
	stopped     bool
}

func NewDaemonRuntimeComponent(workspaceID string, cfg *config.Config, adapterOpts AdapterBuildOptions) *DaemonRuntimeComponent {
	return &DaemonRuntimeComponent{
		cfg:         cfg,
		workspaceID: workspaceID,
		adapterOpts: adapterOpts,
	}
}

func (c *DaemonRuntimeComponent) Name() string {
	return "Runtime"
}

func (c *DaemonRuntimeComponent) Dependencies() []string {
	return []string{}
}

func (c *DaemonRuntimeComponent) Init(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cfg == nil {
		return fmt.Errorf("runtime config not provided")
	}
	if c.workspaceID == "" {
		return fmt.Errorf("workspace id not provided")
	}
	if c.stopped {
		return fmt.Errorf("runtime component already stopped")
	}

	if c.runtime == nil {
		components, err := NewRuntimeBuilder().
			WithContext(ctx).
			WithConfig(c.cfg).
			WithWorkspace(c.workspaceID).
			WithAdapterOptions(c.adapterOpts).
			Build()
		if err != nil {
			return fmt.Errorf("build runtime: %w", err)
		}
		c.runtime = components
	}

	c.initialized = true
	return nil
}

func (c *DaemonRuntimeComponent) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.initialized {
		return fmt.Errorf("runtime component not initialized")
	}
	if c.stopped {
		return fmt.Errorf("runtime component already stopped")
	}
	if c.started {
		return nil
	}

	if err := c.runtime.Start(); err != nil {
		return err
	}

	c.started = true
	return nil
}

func (c *DaemonRuntimeComponent) Stop(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stopped {
		return nil
	}
	if c.runtime == nil {
		c.stopped = true
		c.started = false
		return nil
	}

	c.runtime.Stop()
	c.stopped = true
	c.started = false
	return nil
}

func (c *DaemonRuntimeComponent) Health(ctx context.Context) (*daemon.ComponentHealth, error) {
	c.mu.RLock()
	r := c.runtime
	initialized := c.initialized
	started := c.started
	stopped := c.stopped
	c.mu.RUnlock()

	if r == nil {
		return &daemon.ComponentHealth{Name: c.Name(), Healthy: false, Error: fmt.Errorf("runtime components not configured")}, nil
	}
	if !initialized {
		return &daemon.ComponentHealth{Name: c.Name(), Healthy: false, Error: fmt.Errorf("not initialized")}, nil
	}
	if stopped {
		return &daemon.ComponentHealth{Name: c.Name(), Healthy: false, Error: fmt.Errorf("stopped")}, nil
	}
	if !started {
		return &daemon.ComponentHealth{Name: c.Name(), Healthy: false, Error: fmt.Errorf("not started")}, nil
	}

	if r.StoreWorker == nil {
		return &daemon.ComponentHealth{Name: c.Name(), Healthy: false, Error: fmt.Errorf("store worker not initialized")}, nil
	}
	if !r.StoreWorker.IsLockHeld() {
		return &daemon.ComponentHealth{Name: c.Name(), Healthy: false, Error: fmt.Errorf("store lock not held")}, nil
	}
	if !r.StoreWorker.IsRunning() {
		return &daemon.ComponentHealth{Name: c.Name(), Healthy: false, Error: fmt.Errorf("store worker not running")}, nil
	}
	if r.Orchestrator == nil {
		return &daemon.ComponentHealth{Name: c.Name(), Healthy: false, Error: fmt.Errorf("orchestrator not initialized")}, nil
	}
	if _, err := r.Orchestrator.Health(ctx); err != nil {
		return &daemon.ComponentHealth{Name: c.Name(), Healthy: false, Error: fmt.Errorf("orchestrator unhealthy: %w", err)}, nil
	}
	if r.Ingress == nil {
		return &daemon.ComponentHealth{Name: c.Name(), Healthy: false, Error: fmt.Errorf("ingress not initialized")}, nil
	}
	if err := r.Ingress.Health(ctx); err != nil {
		return &daemon.ComponentHealth{Name: c.Name(), Healthy: false, Error: fmt.Errorf("ingress unhealthy: %w", err)}, nil
	}
	if r.InteractiveWorker == nil || r.BackgroundWorker == nil {
		return &daemon.ComponentHealth{Name: c.Name(), Healthy: false, Error: fmt.Errorf("workers not initialized")}, nil
	}
	if err := r.InteractiveWorker.Health(ctx); err != nil {
		return &daemon.ComponentHealth{Name: c.Name(), Healthy: false, Error: fmt.Errorf("interactive worker unhealthy: %w", err)}, nil
	}
	if err := r.BackgroundWorker.Health(ctx); err != nil {
		return &daemon.ComponentHealth{Name: c.Name(), Healthy: false, Error: fmt.Errorf("background worker unhealthy: %w", err)}, nil
	}
	if r.Scheduler == nil {
		return &daemon.ComponentHealth{Name: c.Name(), Healthy: false, Error: fmt.Errorf("scheduler not initialized")}, nil
	}
	if err := r.Scheduler.Health(ctx); err != nil {
		return &daemon.ComponentHealth{Name: c.Name(), Healthy: false, Error: fmt.Errorf("scheduler unhealthy: %w", err)}, nil
	}
	if r.AdapterMgr == nil {
		return &daemon.ComponentHealth{Name: c.Name(), Healthy: false, Error: fmt.Errorf("adapter manager not initialized")}, nil
	}
	if err := r.AdapterMgr.Health(ctx); err != nil {
		return &daemon.ComponentHealth{Name: c.Name(), Healthy: false, Error: fmt.Errorf("adapter manager unhealthy: %w", err)}, nil
	}

	return &daemon.ComponentHealth{Name: c.Name(), Healthy: true}, nil
}
