package initializers

import (
	"context"
	"fmt"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/store"
)

type StoreInitializer struct{}

func NewStoreInitializer() *StoreInitializer {
	return &StoreInitializer{}
}

func (si *StoreInitializer) Name() string {
	return "store"
}

func (si *StoreInitializer) Dependencies() []string {
	return []string{}
}

func (si *StoreInitializer) Initialize(ctx context.Context, cfg *config.Config, workspaceID string) (interface{}, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	workspaceRootPath := ""
	workspaceRootPath = cfg.Daemon.WorkspacePath
	lockTimeout, err := config.DurationOrDefault(cfg.Store.LockTimeout, config.DefaultStoreLockTimeout)
	if err != nil {
		return nil, fmt.Errorf("parse store lock timeout: %w", err)
	}
	lockRetry, err := config.DurationOrDefault(cfg.Store.LockRetry, config.DefaultStoreLockRetry)
	if err != nil {
		return nil, fmt.Errorf("parse store lock retry: %w", err)
	}
	lockMaxRetry := cfg.Store.LockMaxRetry
	if lockMaxRetry <= 0 {
		lockMaxRetry = config.DefaultStoreLockMaxRetry
	}
	inboxSize := cfg.Store.InboxSize
	if inboxSize <= 0 {
		inboxSize = config.DefaultStoreInboxSize
	}
	transcriptRotateMaxBytes := cfg.Store.TranscriptRotateMaxBytes
	if transcriptRotateMaxBytes <= 0 {
		transcriptRotateMaxBytes = config.DefaultStoreTranscriptRotateMaxBytes
	}

	worker, err := store.NewWorker(workspaceID, workspaceRootPath, store.RuntimeConfig{
		LockTimeout:              lockTimeout,
		LockRetry:                lockRetry,
		LockMaxRetry:             lockMaxRetry,
		InboxSize:                inboxSize,
		TranscriptRotateMaxBytes: transcriptRotateMaxBytes,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create store worker: %w", err)
	}
	worker.Start()
	return worker, nil
}
