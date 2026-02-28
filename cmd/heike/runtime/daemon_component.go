package runtime

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/daemon"
	"github.com/harunnryd/heike/internal/ingress"
	"github.com/harunnryd/heike/internal/policy"
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

func (c *DaemonRuntimeComponent) runtimeForAPI() (*RuntimeComponents, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.runtime == nil {
		return nil, fmt.Errorf("runtime components not configured")
	}
	if !c.initialized {
		return nil, fmt.Errorf("runtime component not initialized")
	}
	if c.stopped {
		return nil, fmt.Errorf("runtime component already stopped")
	}
	if !c.started {
		return nil, fmt.Errorf("runtime component not started")
	}
	return c.runtime, nil
}

func (c *DaemonRuntimeComponent) SubmitEvent(ctx context.Context, evt daemon.RuntimeEvent) (string, error) {
	r, err := c.runtimeForAPI()
	if err != nil {
		return "", err
	}
	if r.Ingress == nil {
		return "", fmt.Errorf("ingress not initialized")
	}
	if evt.Source == "" {
		return "", fmt.Errorf("event source is required")
	}
	if evt.Content == "" {
		return "", fmt.Errorf("event content is required")
	}

	msgType := ingress.TypeUserMessage
	switch evt.Type {
	case "", string(ingress.TypeUserMessage):
		msgType = ingress.TypeUserMessage
	case string(ingress.TypeCommand):
		msgType = ingress.TypeCommand
	case string(ingress.TypeCron):
		msgType = ingress.TypeCron
	case string(ingress.TypeSystemEvent):
		msgType = ingress.TypeSystemEvent
	default:
		return "", fmt.Errorf("unsupported event type: %s", evt.Type)
	}

	normalized := ingress.NewEvent(evt.Source, msgType, evt.SessionID, evt.Content, evt.Metadata)
	if err := r.Ingress.Submit(ctx, &normalized); err != nil {
		return "", err
	}
	if r.Zanshin != nil {
		r.Zanshin.NotifyInteraction()
	}
	return normalized.ID, nil
}

func (c *DaemonRuntimeComponent) ListSessions(ctx context.Context) ([]daemon.RuntimeSession, error) {
	r, err := c.runtimeForAPI()
	if err != nil {
		return nil, err
	}
	if r.StoreWorker == nil {
		return nil, fmt.Errorf("store worker not initialized")
	}

	ids, err := r.StoreWorker.ListSessions()
	if err != nil {
		return nil, err
	}
	sort.Strings(ids)

	result := make([]daemon.RuntimeSession, 0, len(ids))
	for _, id := range ids {
		meta, err := r.StoreWorker.GetSession(id)
		if err != nil {
			return nil, err
		}
		if meta == nil {
			result = append(result, daemon.RuntimeSession{ID: id})
			continue
		}

		metadataCopy := map[string]string(nil)
		if len(meta.Metadata) > 0 {
			metadataCopy = make(map[string]string, len(meta.Metadata))
			for k, v := range meta.Metadata {
				metadataCopy[k] = v
			}
		}

		result = append(result, daemon.RuntimeSession{
			ID:        meta.ID,
			Title:     meta.Title,
			Status:    meta.Status,
			CreatedAt: meta.CreatedAt,
			UpdatedAt: meta.UpdatedAt,
			Metadata:  metadataCopy,
		})
	}
	return result, nil
}

func (c *DaemonRuntimeComponent) ReadTranscript(ctx context.Context, sessionID string, limit int) ([]string, error) {
	r, err := c.runtimeForAPI()
	if err != nil {
		return nil, err
	}
	if r.StoreWorker == nil {
		return nil, fmt.Errorf("store worker not initialized")
	}
	return r.StoreWorker.ReadTranscript(sessionID, limit)
}

func (c *DaemonRuntimeComponent) ListPendingApprovals(ctx context.Context) ([]daemon.RuntimeApproval, error) {
	r, err := c.runtimeForAPI()
	if err != nil {
		return nil, err
	}
	if r.PolicyEngine == nil {
		return nil, fmt.Errorf("policy engine not initialized")
	}

	approvals := r.PolicyEngine.ListApprovals(policy.StatusPending)
	result := make([]daemon.RuntimeApproval, 0, len(approvals))
	for _, app := range approvals {
		result = append(result, daemon.RuntimeApproval{
			ID:        app.ID,
			Tool:      app.Tool,
			Input:     app.Input,
			Status:    string(app.Status),
			CreatedAt: app.CreatedAt,
		})
	}
	return result, nil
}

func (c *DaemonRuntimeComponent) ResolveApproval(ctx context.Context, approvalID string, approve bool) error {
	r, err := c.runtimeForAPI()
	if err != nil {
		return err
	}
	if r.PolicyEngine == nil {
		return fmt.Errorf("policy engine not initialized")
	}
	return r.PolicyEngine.Resolve(approvalID, approve)
}

func (c *DaemonRuntimeComponent) ZanshinStatus(ctx context.Context) map[string]interface{} {
	r, err := c.runtimeForAPI()
	if err != nil {
		return map[string]interface{}{
			"enabled": false,
			"status":  "unavailable",
			"error":   err.Error(),
		}
	}
	if r.Zanshin == nil {
		return map[string]interface{}{
			"enabled": false,
			"status":  "disabled",
		}
	}
	return r.Zanshin.Status()
}
