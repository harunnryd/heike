package components

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/harunnryd/heike/internal/adapter"
	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/daemon"
	"github.com/harunnryd/heike/internal/egress"
	"github.com/harunnryd/heike/internal/orchestrator"
	"github.com/harunnryd/heike/internal/skill"
	"github.com/harunnryd/heike/internal/tooling"
)

type OrchestratorComponent struct {
	kernel           orchestrator.Kernel
	cfg              *config.Config
	storeWorkerComp  *StoreWorkerComponent
	policyEngineComp *PolicyEngineComponent
}

func NewOrchestratorComponent(cfg *config.Config, storeComp *StoreWorkerComponent, policyComp *PolicyEngineComponent) *OrchestratorComponent {
	return &OrchestratorComponent{
		cfg:              cfg,
		storeWorkerComp:  storeComp,
		policyEngineComp: policyComp,
	}
}

func (o *OrchestratorComponent) Name() string {
	return "Orchestrator"
}

func (o *OrchestratorComponent) Dependencies() []string {
	return []string{"StoreWorker", "PolicyEngine"}
}

func (o *OrchestratorComponent) Init(ctx context.Context) error {
	if o.storeWorkerComp == nil || o.policyEngineComp == nil {
		return fmt.Errorf("required component dependencies not provided")
	}

	storeWorker := o.storeWorkerComp.GetWorker()
	policyEngine := o.policyEngineComp.GetEngine()
	if storeWorker == nil || policyEngine == nil {
		return fmt.Errorf("required dependencies not initialized")
	}

	toolingComponents, err := tooling.Build(
		o.storeWorkerComp.workspaceID,
		policyEngine,
		"",
		o.cfg,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize tooling: %w", err)
	}
	skillRegistry := skill.NewRegistry()
	loadWarnings := skill.LoadRuntimeRegistry(skillRegistry, o.storeWorkerComp.workspaceID, o.cfg.Daemon.WorkspacePath, "")
	for _, warn := range loadWarnings {
		slog.Warn("Failed to load skill registry source", "error", warn, "workspace", o.storeWorkerComp.workspaceID)
	}
	if names, err := skillRegistry.List("name"); err == nil {
		slog.Info("Skill registry initialized", "workspace", o.storeWorkerComp.workspaceID, "count", len(names))
	}
	egressMgr := egress.NewEgress(storeWorker)
	if err := egressMgr.Register(adapter.NewCLIAdapter()); err != nil {
		return fmt.Errorf("failed to register default egress adapter: %w", err)
	}
	if err := egressMgr.Register(adapter.NewNullAdapter("scheduler")); err != nil {
		return fmt.Errorf("failed to register scheduler egress adapter: %w", err)
	}
	if err := egressMgr.Register(adapter.NewNullAdapter("system")); err != nil {
		return fmt.Errorf("failed to register system egress adapter: %w", err)
	}

	kernel, err := orchestrator.NewKernel(*o.cfg, storeWorker, toolingComponents.Runner, policyEngine, skillRegistry, egressMgr)
	if err != nil {
		return fmt.Errorf("failed to create kernel: %w", err)
	}
	o.kernel = kernel

	if err := o.kernel.Init(ctx); err != nil {
		return fmt.Errorf("failed to initialize kernel: %w", err)
	}

	slog.Info("Orchestrator kernel initialized", "component", o.Name())
	return nil
}

func (o *OrchestratorComponent) Start(ctx context.Context) error {
	if o.kernel == nil {
		return fmt.Errorf("kernel not initialized")
	}

	if err := o.kernel.Start(ctx); err != nil {
		return fmt.Errorf("failed to start kernel: %w", err)
	}

	slog.Info("Orchestrator started", "component", o.Name())
	return nil
}

func (o *OrchestratorComponent) Stop(ctx context.Context) error {
	if o.kernel == nil {
		slog.Info("Kernel not initialized, skipping stop", "component", o.Name())
		return nil
	}

	if err := o.kernel.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop kernel: %w", err)
	}

	slog.Info("Orchestrator stopped", "component", o.Name())
	return nil
}

func (o *OrchestratorComponent) Health(ctx context.Context) (*daemon.ComponentHealth, error) {
	if o.kernel == nil {
		return &daemon.ComponentHealth{
			Name:    o.Name(),
			Healthy: false,
			Error:   fmt.Errorf("not initialized"),
		}, nil
	}

	health, err := o.kernel.Health(ctx)
	if err != nil {
		return nil, err
	}

	return &daemon.ComponentHealth{
		Name:    health.Name,
		Healthy: health.Healthy,
		Error:   health.Error,
	}, nil
}

func (o *OrchestratorComponent) GetKernel() orchestrator.Kernel {
	return o.kernel
}
