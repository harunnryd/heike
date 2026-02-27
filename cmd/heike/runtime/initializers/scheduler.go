package initializers

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/ingress"
	"github.com/harunnryd/heike/internal/scheduler"
	"github.com/harunnryd/heike/internal/store"
)

type SchedulerInitializer struct {
	ingress *ingress.Ingress
}

func NewSchedulerInitializer(ingress *ingress.Ingress) *SchedulerInitializer {
	return &SchedulerInitializer{
		ingress: ingress,
	}
}

func (si *SchedulerInitializer) Name() string {
	return "scheduler"
}

func (si *SchedulerInitializer) Dependencies() []string {
	return []string{"workers"}
}

func (si *SchedulerInitializer) Initialize(ctx context.Context, cfg *config.Config, workspaceID string) (interface{}, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	schedulerDir, err := store.GetSchedulerDir(workspaceID, cfg.Daemon.WorkspacePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve scheduler directory: %w", err)
	}
	schedulerStorePath := filepath.Join(schedulerDir, "tasks.json")
	schedulerStore, err := scheduler.NewStore(schedulerStorePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create scheduler store: %w", err)
	}

	ingress := si.ingress
	if ingress == nil {
		return nil, fmt.Errorf("ingress not initialized")
	}

	sched, err := scheduler.NewScheduler(schedulerStore, ingress, cfg.Scheduler)
	if err != nil {
		return nil, fmt.Errorf("failed to create scheduler: %w", err)
	}
	if err := sched.Init(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize scheduler: %w", err)
	}
	return sched, nil
}
