package components

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/daemon"
	"github.com/harunnryd/heike/internal/scheduler"
	"github.com/harunnryd/heike/internal/store"
)

type SchedulerComponent struct {
	sched       *scheduler.Scheduler
	cfg         *config.Config
	ingressComp *IngressComponent
	workspaceID string
}

func NewSchedulerComponent(cfg *config.Config, ingComp *IngressComponent, workspaceID string) *SchedulerComponent {
	return &SchedulerComponent{
		cfg:         cfg,
		ingressComp: ingComp,
		workspaceID: workspaceID,
	}
}

func (s *SchedulerComponent) Name() string {
	return "Scheduler"
}

func (s *SchedulerComponent) Dependencies() []string {
	return []string{"Ingress"}
}

func (s *SchedulerComponent) Init(ctx context.Context) error {
	if s.ingressComp == nil {
		return fmt.Errorf("ingressComp not provided")
	}

	ing := s.ingressComp.GetIngress()
	if ing == nil {
		return fmt.Errorf("ingress not initialized")
	}

	schedulerDir, err := store.GetSchedulerDir(s.workspaceID, s.cfg.Daemon.WorkspacePath)
	if err != nil {
		return fmt.Errorf("failed to resolve scheduler directory: %w", err)
	}
	schedulerStorePath := filepath.Join(schedulerDir, "tasks.json")
	store, err := scheduler.NewStore(schedulerStorePath)
	if err != nil {
		return fmt.Errorf("failed to create scheduler store: %w", err)
	}
	sched, err := scheduler.NewScheduler(store, ing, s.cfg.Scheduler)
	if err != nil {
		return fmt.Errorf("failed to create scheduler: %w", err)
	}
	s.sched = sched

	if err := s.sched.Init(ctx); err != nil {
		return fmt.Errorf("failed to initialize scheduler: %w", err)
	}

	slog.Info("Scheduler initialized", "component", s.Name())
	return nil
}

func (s *SchedulerComponent) Start(ctx context.Context) error {
	if s.sched == nil {
		return fmt.Errorf("scheduler not initialized")
	}

	if err := s.sched.Start(ctx); err != nil {
		return fmt.Errorf("failed to start scheduler: %w", err)
	}

	slog.Info("Scheduler started", "component", s.Name())
	return nil
}

func (s *SchedulerComponent) Stop(ctx context.Context) error {
	if s.sched == nil {
		slog.Info("Scheduler not initialized, skipping stop", "component", s.Name())
		return nil
	}

	if err := s.sched.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop scheduler: %w", err)
	}

	slog.Info("Scheduler stopped", "component", s.Name())
	return nil
}

func (s *SchedulerComponent) Health(ctx context.Context) (*daemon.ComponentHealth, error) {
	if s.sched == nil {
		return &daemon.ComponentHealth{
			Name:    s.Name(),
			Healthy: false,
			Error:   fmt.Errorf("not initialized"),
		}, nil
	}

	err := s.sched.Health(ctx)

	if err != nil {
		return &daemon.ComponentHealth{
			Name:    s.Name(),
			Healthy: false,
			Error:   err,
		}, nil
	}

	return &daemon.ComponentHealth{
		Name:    s.Name(),
		Healthy: true,
		Error:   nil,
	}, nil
}

func (s *SchedulerComponent) GetScheduler() *scheduler.Scheduler {
	return s.sched
}
