package components

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/daemon"
	"github.com/harunnryd/heike/internal/policy"
)

type PolicyEngineComponent struct {
	cfg               *config.GovernanceConfig
	workspaceID       string
	workspaceRootPath string
	engine            *policy.Engine
	initialized       bool
	started           bool
	mu                sync.RWMutex
	startTime         time.Time
}

func NewPolicyEngineComponent(cfg *config.GovernanceConfig, workspaceID string, workspaceRootPath string) *PolicyEngineComponent {
	return &PolicyEngineComponent{
		cfg:               cfg,
		workspaceID:       workspaceID,
		workspaceRootPath: workspaceRootPath,
		initialized:       false,
		started:           false,
	}
}

func (p *PolicyEngineComponent) Name() string {
	return "PolicyEngine"
}

func (p *PolicyEngineComponent) Dependencies() []string {
	return []string{"StoreWorker"}
}

func (p *PolicyEngineComponent) Init(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return fmt.Errorf("PolicyEngine init cancelled: %w", ctx.Err())
	default:
	}

	engine, err := policy.NewEngine(*p.cfg, p.workspaceID, p.workspaceRootPath)
	if err != nil {
		return err
	}

	p.engine = engine
	p.initialized = true
	slog.Info("PolicyEngine initialized", "component", p.Name(), "workspace", p.workspaceID)
	return nil
}

func (p *PolicyEngineComponent) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.initialized {
		return fmt.Errorf("PolicyEngine not initialized")
	}

	p.started = true
	p.startTime = time.Now()
	slog.Info("PolicyEngine started", "component", p.Name())
	return nil
}

func (p *PolicyEngineComponent) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.started {
		slog.Info("PolicyEngine not started, skipping stop", "component", p.Name())
		return nil
	}

	slog.Info("Stopping PolicyEngine...", "component", p.Name())
	p.started = false
	slog.Info("PolicyEngine stopped", "component", p.Name())
	return nil
}

func (p *PolicyEngineComponent) Health(ctx context.Context) (*daemon.ComponentHealth, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		return &daemon.ComponentHealth{
			Name:    p.Name(),
			Healthy: false,
			Error:   fmt.Errorf("not initialized"),
		}, nil
	}

	if !p.started {
		return &daemon.ComponentHealth{
			Name:    p.Name(),
			Healthy: false,
			Error:   fmt.Errorf("not started"),
		}, nil
	}

	return &daemon.ComponentHealth{
		Name:    p.Name(),
		Healthy: true,
		Error:   nil,
	}, nil
}

func (p *PolicyEngineComponent) GetEngine() *policy.Engine {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.engine
}
