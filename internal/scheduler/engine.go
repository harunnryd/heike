package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/harunnryd/heike/internal/config"
	heikeErrors "github.com/harunnryd/heike/internal/errors"
	"github.com/harunnryd/heike/internal/ingress"
)

type Component interface {
	Init(ctx context.Context) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Health(ctx context.Context) error
}

type Scheduler struct {
	store         *Store
	ingressSubmit IngressSubmitter

	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	running       bool
	ticker        *time.Ticker
	inFlightTasks uint

	tickInterval         time.Duration
	shutdownTimeout      time.Duration
	leaseDuration        time.Duration
	maxCatchupRuns       int
	inFlightPollInterval time.Duration
	heartbeatWorkspaceID string
}

type IngressSubmitter interface {
	Submit(ctx context.Context, evt *ingress.Event) error
}

func NewScheduler(store *Store, ingressSubmit IngressSubmitter, cfg config.SchedulerConfig) (*Scheduler, error) {
	tickInterval, err := config.DurationOrDefault(cfg.TickInterval, config.DefaultSchedulerTickInterval)
	if err != nil {
		return nil, fmt.Errorf("parse scheduler tick interval: %w", err)
	}

	shutdownTimeout, err := config.DurationOrDefault(cfg.ShutdownTimeout, config.DefaultSchedulerShutdownTimeout)
	if err != nil {
		return nil, fmt.Errorf("parse scheduler shutdown timeout: %w", err)
	}

	leaseDuration, err := config.DurationOrDefault(cfg.LeaseDuration, config.DefaultSchedulerLeaseDuration)
	if err != nil {
		return nil, fmt.Errorf("parse scheduler lease duration: %w", err)
	}

	inFlightPollInterval, err := config.DurationOrDefault(cfg.InFlightPollInterval, config.DefaultSchedulerInFlightPollInterval)
	if err != nil {
		return nil, fmt.Errorf("parse scheduler in-flight poll interval: %w", err)
	}

	maxCatchupRuns := cfg.MaxCatchupRuns
	if maxCatchupRuns <= 0 {
		maxCatchupRuns = config.DefaultSchedulerMaxCatchupRuns
	}

	heartbeatWorkspaceID := strings.TrimSpace(cfg.HeartbeatWorkspaceID)
	if heartbeatWorkspaceID == "" {
		heartbeatWorkspaceID = config.DefaultSchedulerHeartbeatWorkspaceID
	}

	return &Scheduler{
		store:                store,
		ingressSubmit:        ingressSubmit,
		tickInterval:         tickInterval,
		shutdownTimeout:      shutdownTimeout,
		leaseDuration:        leaseDuration,
		maxCatchupRuns:       maxCatchupRuns,
		inFlightPollInterval: inFlightPollInterval,
		heartbeatWorkspaceID: heartbeatWorkspaceID,
	}, nil
}

func (s *Scheduler) Init(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)

	if err := s.store.Init(ctx); err != nil {
		return fmt.Errorf("init store: %w", err)
	}

	slog.Info("Scheduler initialized")
	return nil
}

func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	s.mu.Unlock()

	s.recoverExpiredLeases(ctx)
	s.processCatchUp(ctx)

	s.ticker = time.NewTicker(s.tickInterval)

	go s.run(ctx)

	slog.Info("Scheduler started")
	return nil
}

func (s *Scheduler) Stop(ctx context.Context) error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	s.mu.Unlock()

	if s.ticker != nil {
		s.ticker.Stop()
	}

	s.cancel()

	done := make(chan struct{})
	go func() {
		s.waitForInFlightTasks()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("Scheduler stopped gracefully")
		return nil
	case <-time.After(s.shutdownTimeout):
		slog.Warn("Scheduler shutdown timeout, force stopping")
		return heikeErrors.Internal("shutdown timeout")
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Scheduler) Health(ctx context.Context) error {
	if s.ctx == nil {
		return heikeErrors.Internal("scheduler not initialized")
	}

	if !s.IsRunning() {
		return heikeErrors.Internal("scheduler not running")
	}

	if _, err := s.store.LoadTasks(); err != nil {
		return fmt.Errorf("load tasks: %w", heikeErrors.ErrTransient)
	}

	return nil
}

func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

func (s *Scheduler) run(ctx context.Context) {
	for {
		select {
		case <-s.ticker.C:
			s.onTick(ctx)
		case <-s.ctx.Done():
			slog.Info("Scheduler run loop stopped")
			return
		}
	}
}

func (s *Scheduler) onTick(ctx context.Context) {
	s.processCronJobs(ctx)
	s.processHeartbeat(ctx)
}

func (s *Scheduler) processCronJobs(ctx context.Context) {
	tasks, err := s.store.LoadTasks()
	if err != nil {
		slog.Error("Failed to load cron tasks", "error", err)
		return
	}

	for _, task := range tasks {
		if task.Schedule == "" {
			continue
		}

		shouldFire, fireTime, err := s.store.ShouldFire(task.ID, task.Schedule)
		if err != nil {
			slog.Error("Failed to check if task should fire", "task", task.ID, "error", err)
			continue
		}

		if shouldFire {
			s.executeTask(ctx, task, fireTime)
		}
	}
}

func (s *Scheduler) executeTask(ctx context.Context, task Task, fireTime time.Time) {
	s.mu.Lock()
	s.inFlightTasks++
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.inFlightTasks--
		s.mu.Unlock()
	}()

	runID := generateRunID()
	leaseExpiresAt := time.Now().Add(s.leaseDuration)

	if err := s.store.AcquireLease(task.ID, runID, leaseExpiresAt); err != nil {
		slog.Error("Failed to acquire lease", "task", task.ID, "error", err)
		return
	}

	evt := &ingress.Event{
		ID:        generateID(),
		Type:      ingress.TypeCron,
		Source:    "scheduler",
		Content:   task.Content,
		SessionID: "scheduler", // Cron events don't have a session
		Metadata: map[string]string{
			"task_id":          task.ID,
			"run_id":           runID,
			"fire_time":        fireTime.Format(time.RFC3339),
			"lease_expires_at": leaseExpiresAt.Format(time.RFC3339),
		},
	}

	if err := s.ingressSubmit.Submit(ctx, evt); err != nil {
		slog.Error("Failed to submit cron event", "task", task.ID, "error", err)
		return
	}

	if err := s.store.MarkTaskDone(task.ID, runID); err != nil {
		slog.Error("Failed to mark task done", "task", task.ID, "error", err)
	}
}

func (s *Scheduler) processHeartbeat(ctx context.Context) {
	workspaceID := s.heartbeatWorkspaceID
	tickTime := time.Now()

	evt := &ingress.Event{
		ID:      generateID(),
		Type:    ingress.TypeSystemEvent,
		Source:  "scheduler",
		Content: "heartbeat tick",
		Metadata: map[string]string{
			"workspace_id": workspaceID,
			"tick_time":    tickTime.Format(time.RFC3339),
		},
	}

	if err := s.ingressSubmit.Submit(ctx, evt); err != nil {
		slog.Warn("Failed to submit heartbeat event", "error", err)
	}
}

func (s *Scheduler) recoverExpiredLeases(ctx context.Context) {
	tasks, err := s.store.LoadTasks()
	if err != nil {
		slog.Error("Failed to load tasks for lease recovery", "error", err)
		return
	}

	recovered := 0
	for _, task := range tasks {
		if task.Schedule == "" {
			continue
		}

		lease, err := s.store.GetLease(task.ID)
		if err != nil {
			slog.Warn("Failed to get lease", "task", task.ID, "error", err)
			continue
		}

		if lease != nil && time.Now().After(lease.ExpiresAt) {
			slog.Info("Recovering expired lease", "task", task.ID, "run_id", lease.RunID)
			recovered++
		}
	}

	if recovered > 0 {
		slog.Info("Recovered expired leases", "count", recovered)
	}
}

func (s *Scheduler) processCatchUp(ctx context.Context) {
	tasks, err := s.store.LoadTasks()
	if err != nil {
		slog.Error("Failed to load tasks for catch-up", "error", err)
		return
	}

	missed := 0
	now := time.Now()

	for _, task := range tasks {
		if task.Schedule == "" {
			continue
		}

		if !task.NextRun.IsZero() && task.NextRun.Before(now) {
			missed++
		}
	}

	if missed > s.maxCatchupRuns {
		slog.Warn("Too many missed runs", "missed", missed, "max", s.maxCatchupRuns)

		evt := &ingress.Event{
			ID:      generateID(),
			Type:    ingress.TypeSystemEvent,
			Source:  "scheduler",
			Content: fmt.Sprintf("Missed %d scheduled runs", missed),
			Metadata: map[string]string{
				"workspace_id": s.heartbeatWorkspaceID,
				"missed_count": fmt.Sprintf("%d", missed),
			},
		}

		if err := s.ingressSubmit.Submit(ctx, evt); err != nil {
			slog.Warn("Failed to submit missed runs event", "error", err)
		}
	}
}

func (s *Scheduler) waitForInFlightTasks() {
	ticker := time.NewTicker(s.inFlightPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mu.RLock()
			count := s.inFlightTasks
			s.mu.RUnlock()

			if count == 0 {
				return
			}
			slog.Info("Waiting for in-flight tasks", "count", count)
		case <-s.ctx.Done():
			return
		}
	}
}
