package daemon

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/store"
)

type Daemon struct {
	cfg             *config.Config
	workspaceID     string
	components      []Component
	shutdownOrder   []string
	health          HealthStatus
	uptimeStart     time.Time
	mu              sync.RWMutex
	healthCheckDone chan struct{}
	panicChan       chan interface{}
	forceCleanup    bool
}

func NewDaemon(workspaceID string, cfg *config.Config) (*Daemon, error) {
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace ID cannot be empty")
	}

	return &Daemon{
		workspaceID:     workspaceID,
		cfg:             cfg,
		components:      make([]Component, 0),
		shutdownOrder:   make([]string, 0),
		health:          StatusStarting,
		uptimeStart:     time.Now(),
		healthCheckDone: make(chan struct{}),
		panicChan:       make(chan interface{}),
		forceCleanup:    false,
	}, nil
}

func (d *Daemon) AddComponent(comp Component) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.components = append(d.components, comp)
	d.shutdownOrder = append([]string{comp.Name()}, d.shutdownOrder...)
	slog.Info("Component registered", "component", comp.Name(), "total_components", len(d.components))
}

func (d *Daemon) Start(ctx context.Context) error {
	slog.Info("Heike Daemon starting...", "workspace", d.workspaceID)

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	go d.monitorPanic()
	defer close(d.panicChan)

	if err := d.validateConfig(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	if err := d.preInitChecks(ctx, d.forceCleanup); err != nil {
		return fmt.Errorf("pre-init checks failed: %w", err)
	}

	if err := d.initializeComponents(ctx); err != nil {
		d.rollback(ctx)
		return fmt.Errorf("component initialization failed: %w", err)
	}

	if err := d.startComponents(ctx); err != nil {
		startupShutdownTimeout, timeoutErr := config.DurationOrDefault(d.cfg.Daemon.StartupShutdownTimeout, config.DefaultDaemonStartupShutdownTimeout)
		if timeoutErr != nil {
			return fmt.Errorf("parse daemon startup shutdown timeout: %w", timeoutErr)
		}
		d.gracefulShutdown(ctx, startupShutdownTimeout)
		return fmt.Errorf("component startup failed: %w", err)
	}

	d.setHealth(StatusRunning)
	slog.Info("Heike Daemon is running", "workspace", d.workspaceID, "components", len(d.components))

	go d.startHealthMonitor(ctx)

	<-ctx.Done()

	slog.Info("Context cancelled, initiating graceful shutdown", "workspace", d.workspaceID, "reason", ctx.Err())
	d.setHealth(StatusStopping)
	close(d.healthCheckDone)
	shutdownTimeout, err := config.DurationOrDefault(d.cfg.Daemon.ShutdownTimeout, config.DefaultDaemonShutdownTimeout)
	if err != nil {
		return fmt.Errorf("parse daemon shutdown timeout: %w", err)
	}
	shutdownErr := d.gracefulShutdown(context.Background(), shutdownTimeout)
	if shutdownErr != nil {
		return shutdownErr
	}

	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return ctx.Err()
	}
	return nil
}

func (d *Daemon) Health() HealthStatus {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.health
}

func (d *Daemon) SetForceCleanup(force bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.forceCleanup = force
}

func (d *Daemon) ComponentHealth() map[string]*ComponentHealth {
	d.mu.RLock()
	components := make([]Component, len(d.components))
	copy(components, d.components)
	d.mu.RUnlock()

	result := make(map[string]*ComponentHealth)
	for _, comp := range components {
		health, err := comp.Health(context.Background())
		result[comp.Name()] = health
		if err != nil {
			result[comp.Name()].Error = err
		}
	}
	return result
}

func (d *Daemon) setHealth(status HealthStatus) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.health = status
}

func (d *Daemon) validateConfig() error {
	slog.Info("Validating configuration...")

	if d.cfg.Server.Port < 1 || d.cfg.Server.Port > 65535 {
		return fmt.Errorf("invalid port: %d (must be 1-65535)", d.cfg.Server.Port)
	}

	workspacePath, err := store.GetWorkspacePath(d.workspaceID, d.cfg.Daemon.WorkspacePath)
	if err != nil {
		return fmt.Errorf("resolve workspace path: %w", err)
	}

	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		return fmt.Errorf("failed to create workspace directory: %w", err)
	}

	slog.Info("Configuration validated", "workspace", d.workspaceID, "port", d.cfg.Server.Port)
	return nil
}

func (d *Daemon) preInitChecks(ctx context.Context, forceCleanup bool) error {
	slog.Info("Running pre-init checks...", "workspace", d.workspaceID)

	preflightTimeout, err := config.DurationOrDefault(d.cfg.Daemon.PreflightTimeout, config.DefaultDaemonPreflightTimeout)
	if err != nil {
		return fmt.Errorf("parse daemon preflight timeout: %w", err)
	}
	checkCtx, cancel := context.WithTimeout(ctx, preflightTimeout)
	defer cancel()

	workspacePath, err := store.GetWorkspacePath(d.workspaceID, d.cfg.Daemon.WorkspacePath)
	if err != nil {
		return fmt.Errorf("resolve workspace path: %w", err)
	}
	staleLockTTL, err := config.DurationOrDefault(d.cfg.Daemon.StaleLockTTL, config.DefaultDaemonStaleLockTTL)
	if err != nil {
		return fmt.Errorf("parse daemon stale lock ttl: %w", err)
	}

	err = store.CleanupStaleLocks(workspacePath, staleLockTTL, forceCleanup)
	if err != nil {
		slog.Warn("Failed to cleanup stale locks", "workspace", d.workspaceID, "error", err)
	}

	select {
	case <-checkCtx.Done():
		return fmt.Errorf("pre-init checks cancelled: %w", checkCtx.Err())
	default:
		slog.Info("Pre-init checks completed", "workspace", d.workspaceID)
		return nil
	}
}

func (d *Daemon) initializeComponents(ctx context.Context) error {
	slog.Info("Initializing components...", "workspace", d.workspaceID)

	if err := d.validateDependencies(); err != nil {
		return fmt.Errorf("dependency validation failed: %w", err)
	}

	initOrder, err := d.resolveInitOrder()
	if err != nil {
		return fmt.Errorf("failed to resolve init order: %w", err)
	}

	for _, compName := range initOrder {
		comp := d.getComponentByName(compName)
		if comp == nil {
			continue
		}
		slog.Info("Initializing component...", "component", comp.Name())
		if err := comp.Init(ctx); err != nil {
			slog.Error("Component initialization failed", "component", comp.Name(), "error", err)
			return fmt.Errorf("component %s init failed: %w", comp.Name(), err)
		}
		slog.Info("Component initialized", "component", comp.Name())
	}

	slog.Info("All components initialized", "count", len(d.components))
	return nil
}

func (d *Daemon) startComponents(ctx context.Context) error {
	slog.Info("Starting components...", "workspace", d.workspaceID)

	for _, comp := range d.components {
		slog.Info("Starting component...", "component", comp.Name())
		if err := comp.Start(ctx); err != nil {
			slog.Error("Component startup failed", "component", comp.Name(), "error", err)
			return fmt.Errorf("component %s startup failed: %w", comp.Name(), err)
		}
		slog.Info("Component started", "component", comp.Name())
	}

	slog.Info("All components started", "count", len(d.components))
	return nil
}

func (d *Daemon) gracefulShutdown(ctx context.Context, timeout time.Duration) error {
	slog.Info("Graceful shutdown initiated", "workspace", d.workspaceID, "timeout", timeout)

	// Create timeout context but also respect parent context cancellation
	shutdownCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- d.shutdownComponents(shutdownCtx)
	}()

	select {
	case err := <-done:
		if err != nil {
			slog.Error("Shutdown completed with error", "workspace", d.workspaceID, "error", err)
		} else {
			slog.Info("Graceful shutdown completed", "workspace", d.workspaceID)
		}
		return err
	case <-shutdownCtx.Done():
		// Check if it was timeout or parent cancellation
		if ctx.Err() != nil {
			slog.Info("Shutdown cancelled by parent context", "workspace", d.workspaceID, "reason", ctx.Err())
			return fmt.Errorf("shutdown cancelled: %w", ctx.Err())
		}
		slog.Error("Shutdown timeout exceeded", "workspace", d.workspaceID, "timeout", timeout)
		return fmt.Errorf("shutdown timeout after %v", timeout)
	}
}

func (d *Daemon) shutdownComponents(ctx context.Context) error {
	for _, name := range d.shutdownOrder {
		comp := d.getComponentByName(name)
		if comp == nil {
			continue
		}

		slog.Info("Stopping component...", "component", name)
		if err := comp.Stop(ctx); err != nil {
			slog.Error("Component stop failed", "component", name, "error", err)
		} else {
			slog.Info("Component stopped", "component", name)
		}
	}

	d.setHealth(StatusStopped)
	return nil
}

func (d *Daemon) rollback(ctx context.Context) {
	slog.Warn("Rolling back initialized components...", "workspace", d.workspaceID)

	for i := len(d.components) - 1; i >= 0; i-- {
		comp := d.components[i]
		slog.Info("Rolling back component...", "component", comp.Name())
		if err := comp.Stop(ctx); err != nil {
			slog.Error("Rollback failed", "component", comp.Name(), "error", err)
		}
	}

	d.setHealth(StatusStopped)
}

func (d *Daemon) getComponentByName(name string) Component {
	for _, comp := range d.components {
		if comp.Name() == name {
			return comp
		}
	}
	return nil
}

func (d *Daemon) Component(name string) Component {
	d.mu.RLock()
	defer d.mu.RUnlock()
	for _, comp := range d.components {
		if comp.Name() == name {
			return comp
		}
	}
	return nil
}

func (d *Daemon) monitorPanic() {
	for panicValue := range d.panicChan {
		slog.Error("Panic detected in daemon", "panic", panicValue)
		d.setHealth(StatusStopped)
	}
}

func (d *Daemon) startHealthMonitor(ctx context.Context) {
	healthCheckInterval, err := config.DurationOrDefault(d.cfg.Daemon.HealthCheckInterval, config.DefaultDaemonHealthCheckInterval)
	if err != nil {
		slog.Error("Failed to parse daemon health check interval", "error", err)
		return
	}

	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-d.healthCheckDone:
			return
		case <-ticker.C:
			d.checkComponentHealth(ctx)
		}
	}
}

func (d *Daemon) checkComponentHealth(ctx context.Context) {
	healths := d.ComponentHealth()
	unhealthyCount := 0

	for name, health := range healths {
		// Check for context cancellation during health checks
		select {
		case <-ctx.Done():
			slog.Info("Component health check cancelled", "reason", ctx.Err())
			return
		default:
		}

		if !health.Healthy {
			unhealthyCount++
			slog.Warn("Component unhealthy", "component", name, "error", health.Error)
		}
	}

	// Final cancellation check before logging
	select {
	case <-ctx.Done():
		slog.Info("Component health check cancelled before logging", "reason", ctx.Err())
		return
	default:
	}

	if unhealthyCount > 0 {
		slog.Warn("Daemon has unhealthy components", "count", unhealthyCount, "total", len(healths))
	} else {
		slog.Debug("All components healthy", "count", len(healths))
	}
}

func (d *Daemon) validateDependencies() error {
	slog.Info("Validating component dependencies...")

	componentMap := make(map[string]Component)
	for _, comp := range d.components {
		componentMap[comp.Name()] = comp
	}

	for _, comp := range d.components {
		for _, depName := range comp.Dependencies() {
			if _, exists := componentMap[depName]; !exists {
				return fmt.Errorf("component %s depends on %s which is not registered", comp.Name(), depName)
			}
		}
	}

	slog.Info("All dependencies validated", "components", len(d.components))
	return nil
}

func (d *Daemon) resolveInitOrder() ([]string, error) {
	slog.Info("Resolving component initialization order...")

	visited := make(map[string]bool)
	tempVisited := make(map[string]bool)
	order := []string{}

	var visit func(name string) error
	visit = func(name string) error {
		if tempVisited[name] {
			return fmt.Errorf("circular dependency detected involving %s", name)
		}
		if visited[name] {
			return nil
		}

		comp := d.getComponentByName(name)
		if comp == nil {
			return fmt.Errorf("component %s not found", name)
		}

		tempVisited[name] = true
		for _, depName := range comp.Dependencies() {
			if err := visit(depName); err != nil {
				return err
			}
		}
		tempVisited[name] = false
		visited[name] = true
		order = append(order, name)
		return nil
	}

	for _, comp := range d.components {
		if err := visit(comp.Name()); err != nil {
			return nil, err
		}
	}

	slog.Info("Initialization order resolved", "order", order)
	return order, nil
}
