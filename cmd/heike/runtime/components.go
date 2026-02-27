package runtime

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/harunnryd/heike/cmd/heike/runtime/initializers"

	"github.com/harunnryd/heike/internal/adapter"
	"github.com/harunnryd/heike/internal/concurrency"
	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/egress"
	"github.com/harunnryd/heike/internal/ingress"
	"github.com/harunnryd/heike/internal/orchestrator"
	"github.com/harunnryd/heike/internal/policy"
	"github.com/harunnryd/heike/internal/scheduler"
	"github.com/harunnryd/heike/internal/skill"
	"github.com/harunnryd/heike/internal/store"
	"github.com/harunnryd/heike/internal/tool"
	"github.com/harunnryd/heike/internal/worker"
)

type RuntimeComponents struct {
	Ctx    context.Context
	Cancel context.CancelFunc

	Config      *config.Config
	WorkspaceID string

	StoreWorker       *store.Worker
	PolicyEngine      *policy.Engine
	Orchestrator      orchestrator.Kernel
	Ingress           *ingress.Ingress
	Egress            egress.Egress
	InteractiveWorker *worker.Worker
	BackgroundWorker  *worker.Worker
	Scheduler         *scheduler.Scheduler

	ToolRunner    *tool.Runner
	ToolRegistry  *tool.Registry
	SkillRegistry *skill.Registry
	AdapterMgr    *adapter.RuntimeManager

	Locks *concurrency.SimpleSessionLockManager
}

func NewRuntimeComponents(ctx context.Context, cfg *config.Config, workspaceID string) (*RuntimeComponents, error) {
	cancel := func() {}

	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel = context.WithCancel(ctx)

	components := &RuntimeComponents{
		Ctx:         ctx,
		Cancel:      cancel,
		Config:      cfg,
		WorkspaceID: workspaceID,
	}

	eventHandler := func(evtCtx context.Context, source string, eventType string, sessionID string, content string, metadata map[string]string) error {
		if components.Ingress == nil {
			return fmt.Errorf("ingress not initialized")
		}

		msgType := ingress.TypeUserMessage
		switch eventType {
		case string(ingress.TypeCommand):
			msgType = ingress.TypeCommand
		case string(ingress.TypeCron):
			msgType = ingress.TypeCron
		case string(ingress.TypeSystemEvent):
			msgType = ingress.TypeSystemEvent
		}

		evt := ingress.NewEvent(source, msgType, sessionID, content, metadata)
		return components.Ingress.Submit(evtCtx, &evt)
	}

	adapterMgr, err := adapter.NewRuntimeManager(cfg.Adapters, eventHandler, adapter.RuntimeAdapterOptions{
		IncludeCLI:        true,
		IncludeSystemNull: true,
	})
	if err != nil {
		components.cleanup()
		return nil, fmt.Errorf("init adapters: %w", err)
	}
	components.AdapterMgr = adapterMgr

	storeInitializer := initializers.NewStoreInitializer()
	storeComponent, err := storeInitializer.Initialize(ctx, cfg, workspaceID)
	if err != nil {
		components.cleanup()
		return nil, fmt.Errorf("init store worker: %w", err)
	}
	components.StoreWorker = storeComponent.(*store.Worker)

	policyInitializer := initializers.NewPolicyInitializer()
	policyComponent, err := policyInitializer.Initialize(ctx, cfg, workspaceID)
	if err != nil {
		components.cleanup()
		return nil, fmt.Errorf("init policy engine: %w", err)
	}
	components.PolicyEngine = policyComponent.(*policy.Engine)

	toolsInitializer := initializers.NewToolsInitializer(components.StoreWorker, components.PolicyEngine)
	toolsComponent, err := toolsInitializer.Initialize(ctx, cfg, workspaceID)
	if err != nil {
		components.cleanup()
		return nil, fmt.Errorf("init tools: %w", err)
	}
	toolsStruct := toolsComponent.(struct {
		Registry *tool.Registry
		Runner   *tool.Runner
	})
	components.ToolRegistry = toolsStruct.Registry
	components.ToolRunner = toolsStruct.Runner

	components.SkillRegistry = skill.NewRegistry()
	loadWarnings := skill.LoadRuntimeRegistry(components.SkillRegistry, skill.RuntimeLoadOptions{
		WorkspaceID:       workspaceID,
		WorkspaceRootPath: cfg.Daemon.WorkspacePath,
		WorkspacePath:     "",
		ProjectPath:       cfg.Discovery.ProjectPath,
		SourceOrder:       cfg.Discovery.SkillSources,
	})
	for _, warn := range loadWarnings {
		slog.Warn("Failed to load skill registry source", "error", warn, "workspace", workspaceID)
	}
	if names, err := components.SkillRegistry.List("name"); err == nil {
		slog.Info("Skill registry initialized", "workspace", workspaceID, "count", len(names))
	}

	egressComponent := egress.NewEgress(components.StoreWorker)
	for _, outputAdapter := range components.AdapterMgr.OutputAdapters() {
		if err := egressComponent.Register(outputAdapter); err != nil {
			components.cleanup()
			return nil, fmt.Errorf("register output adapter %s: %w", outputAdapter.Name(), err)
		}
	}
	components.Egress = egressComponent

	orchestratorInitializer := initializers.NewOrchestratorInitializer(components.StoreWorker, components.ToolRunner, components.PolicyEngine, components.SkillRegistry, components.Egress)
	orchComponent, err := orchestratorInitializer.Initialize(ctx, cfg, workspaceID)
	if err != nil {
		components.cleanup()
		return nil, fmt.Errorf("init orchestrator: %w", err)
	}
	components.Orchestrator = orchComponent.(orchestrator.Kernel)

	workersInitializer := initializers.NewWorkersInitializer(nil, components.Orchestrator, components.StoreWorker)
	workersComponent, err := workersInitializer.Initialize(ctx, cfg, workspaceID)
	if err != nil {
		components.cleanup()
		return nil, fmt.Errorf("init workers: %w", err)
	}
	workersStruct := workersComponent.(struct {
		Ingress           *ingress.Ingress
		InteractiveWorker *worker.Worker
		BackgroundWorker  *worker.Worker
		Locks             *concurrency.SimpleSessionLockManager
	})
	components.Ingress = workersStruct.Ingress
	components.InteractiveWorker = workersStruct.InteractiveWorker
	components.BackgroundWorker = workersStruct.BackgroundWorker
	components.Locks = workersStruct.Locks

	schedulerInitializer := initializers.NewSchedulerInitializer(components.Ingress)
	schedComponent, err := schedulerInitializer.Initialize(ctx, cfg, workspaceID)
	if err != nil {
		components.cleanup()
		return nil, fmt.Errorf("init scheduler: %w", err)
	}
	components.Scheduler = schedComponent.(*scheduler.Scheduler)

	slog.Info("Runtime components initialized successfully", "workspace", workspaceID)
	return components, nil
}

func (r *RuntimeComponents) Start() error {
	if r.Orchestrator == nil {
		return fmt.Errorf("orchestrator not initialized")
	}

	if err := r.Orchestrator.Start(r.Ctx); err != nil {
		r.cleanup()
		return fmt.Errorf("start orchestrator: %w", err)
	}

	if r.Scheduler != nil {
		if err := r.Scheduler.Start(r.Ctx); err != nil {
			r.cleanup()
			return fmt.Errorf("start scheduler: %w", err)
		}
	}

	if r.InteractiveWorker != nil {
		if _, err := r.InteractiveWorker.Start(r.Ctx); err != nil {
			r.cleanup()
			return fmt.Errorf("start interactive worker: %w", err)
		}
	}

	if r.BackgroundWorker != nil {
		if _, err := r.BackgroundWorker.Start(r.Ctx); err != nil {
			r.cleanup()
			return fmt.Errorf("start background worker: %w", err)
		}
	}

	if r.AdapterMgr != nil {
		r.AdapterMgr.Start(r.Ctx)
	}
	return nil
}

func (r *RuntimeComponents) Stop() {
	slog.Info("Stopping runtime components...")

	r.Cancel()

	if r.Scheduler != nil {
		r.Scheduler.Stop(r.Ctx)
	}

	if r.InteractiveWorker != nil {
		r.InteractiveWorker.Stop(r.Ctx)
	}

	if r.BackgroundWorker != nil {
		r.BackgroundWorker.Stop(r.Ctx)
	}

	if r.Orchestrator != nil {
		r.Orchestrator.Stop(r.Ctx)
	}

	if r.AdapterMgr != nil {
		if err := r.AdapterMgr.Stop(r.Ctx); err != nil {
			slog.Warn("Failed to stop adapter manager", "error", err)
		}
	}

	if r.Ingress != nil {
		r.Ingress.Close()
	}

	if r.StoreWorker != nil {
		r.StoreWorker.Stop()
	}

	slog.Info("Runtime components stopped")
}

func (r *RuntimeComponents) cleanup() {
	slog.Debug("Cleaning up runtime components...")
	r.Stop()
}
