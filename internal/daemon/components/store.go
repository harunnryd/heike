package components

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/daemon"
	"github.com/harunnryd/heike/internal/store"
)

type StoreWorkerComponent struct {
	workspaceID       string
	workspaceRootPath string
	storeCfg          *config.StoreConfig
	worker            *store.Worker
	initialized       bool
	started           bool
	mu                sync.RWMutex
	startTime         time.Time
}

func NewStoreWorkerComponent(workspaceID string, workspaceRootPath string, storeCfg *config.StoreConfig) *StoreWorkerComponent {
	return &StoreWorkerComponent{
		workspaceID:       workspaceID,
		workspaceRootPath: workspaceRootPath,
		storeCfg:          storeCfg,
		initialized:       false,
		started:           false,
	}
}

func (s *StoreWorkerComponent) Name() string {
	return "StoreWorker"
}

func (s *StoreWorkerComponent) Dependencies() []string {
	return []string{}
}

func (s *StoreWorkerComponent) Init(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return fmt.Errorf("StoreWorker init cancelled: %w", ctx.Err())
	default:
	}

	lockTimeoutValue := ""
	lockRetryValue := ""
	lockMaxRetry := 0
	inboxSize := 0
	transcriptRotateMaxBytes := int64(0)
	if s.storeCfg != nil {
		lockTimeoutValue = s.storeCfg.LockTimeout
		lockRetryValue = s.storeCfg.LockRetry
		lockMaxRetry = s.storeCfg.LockMaxRetry
		inboxSize = s.storeCfg.InboxSize
		transcriptRotateMaxBytes = s.storeCfg.TranscriptRotateMaxBytes
	}

	lockTimeout, err := config.DurationOrDefault(lockTimeoutValue, config.DefaultStoreLockTimeout)
	if err != nil {
		return fmt.Errorf("parse store lock timeout: %w", err)
	}
	lockRetry, err := config.DurationOrDefault(lockRetryValue, config.DefaultStoreLockRetry)
	if err != nil {
		return fmt.Errorf("parse store lock retry: %w", err)
	}
	if lockMaxRetry <= 0 {
		lockMaxRetry = config.DefaultStoreLockMaxRetry
	}
	if inboxSize <= 0 {
		inboxSize = config.DefaultStoreInboxSize
	}
	if transcriptRotateMaxBytes <= 0 {
		transcriptRotateMaxBytes = config.DefaultStoreTranscriptRotateMaxBytes
	}

	worker, err := store.NewWorker(s.workspaceID, s.workspaceRootPath, store.RuntimeConfig{
		LockTimeout:              lockTimeout,
		LockRetry:                lockRetry,
		LockMaxRetry:             lockMaxRetry,
		InboxSize:                inboxSize,
		TranscriptRotateMaxBytes: transcriptRotateMaxBytes,
	})
	if err != nil {
		if strings.Contains(err.Error(), "is locked by another instance") {
			return fmt.Errorf("workspace %s is locked by another instance: %w", s.workspaceID, err)
		}
		return fmt.Errorf("failed to init store worker: %w", err)
	}

	s.worker = worker
	s.initialized = true
	slog.Info("StoreWorker initialized", "component", s.Name(), "workspace", s.workspaceID)
	return nil
}

func (s *StoreWorkerComponent) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		return fmt.Errorf("StoreWorker not initialized")
	}

	s.worker.Start()
	s.started = true
	s.startTime = time.Now()
	slog.Info("StoreWorker started", "component", s.Name())
	return nil
}

func (s *StoreWorkerComponent) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		slog.Info("StoreWorker not started, skipping stop", "component", s.Name())
		return nil
	}

	slog.Info("Stopping StoreWorker...", "component", s.Name())
	s.worker.Stop()
	s.started = false
	slog.Info("StoreWorker stopped", "component", s.Name())
	return nil
}

func (s *StoreWorkerComponent) Health(ctx context.Context) (*daemon.ComponentHealth, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized {
		return &daemon.ComponentHealth{
			Name:    s.Name(),
			Healthy: false,
			Error:   fmt.Errorf("not initialized"),
		}, nil
	}

	if !s.started {
		return &daemon.ComponentHealth{
			Name:    s.Name(),
			Healthy: false,
			Error:   fmt.Errorf("not started"),
		}, nil
	}

	if !s.worker.IsLockHeld() {
		return &daemon.ComponentHealth{
			Name:    s.Name(),
			Healthy: false,
			Error:   fmt.Errorf("lock not held"),
		}, nil
	}

	if !s.worker.IsRunning() {
		return &daemon.ComponentHealth{
			Name:    s.Name(),
			Healthy: false,
			Error:   fmt.Errorf("loop not running"),
		}, nil
	}

	return &daemon.ComponentHealth{
		Name:    s.Name(),
		Healthy: true,
		Error:   nil,
	}, nil
}

func (s *StoreWorkerComponent) GetWorker() *store.Worker {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.worker
}
