package store

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/harunnryd/heike/internal/config"

	"github.com/gofrs/flock"
)

type FileLock struct {
	fileLock    *flock.Flock
	lockPath    string
	workspaceID string
	acquiredAt  time.Time
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
}

type FileLockConfig struct {
	LockTimeout  time.Duration
	LockRetry    time.Duration
	LockMaxRetry int
}

func DefaultFileLockConfig() *FileLockConfig {
	lockTimeout, _ := config.DurationOrDefault(config.DefaultStoreLockTimeout, config.DefaultStoreLockTimeout)
	lockRetry, _ := config.DurationOrDefault(config.DefaultStoreLockRetry, config.DefaultStoreLockRetry)

	return &FileLockConfig{
		LockTimeout:  lockTimeout,
		LockRetry:    lockRetry,
		LockMaxRetry: config.DefaultStoreLockMaxRetry,
	}
}

func NewFileLock(workspaceID, basePath string, cfg *FileLockConfig) (*FileLock, error) {
	if cfg == nil {
		cfg = DefaultFileLockConfig()
	}

	lockPath := filepath.Join(basePath, "workspace.lock")
	fileLock := flock.New(lockPath)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.LockTimeout)

	fl := &FileLock{
		fileLock:    fileLock,
		lockPath:    lockPath,
		workspaceID: workspaceID,
		ctx:         ctx,
		cancel:      cancel,
	}

	if err := fl.acquireWithRetry(cfg); err != nil {
		cancel()
		return nil, err
	}

	fl.acquiredAt = time.Now()
	slog.Info("File lock acquired",
		"workspace", workspaceID,
		"path", lockPath,
		"acquired_at", fl.acquiredAt.Format(time.RFC3339Nano),
	)

	return fl, nil
}

func (fl *FileLock) acquireWithRetry(cfg *FileLockConfig) error {
	for i := 0; i < cfg.LockMaxRetry; i++ {
		select {
		case <-fl.ctx.Done():
			return fmt.Errorf("lock acquisition cancelled: %w", fl.ctx.Err())
		default:
			locked, err := fl.fileLock.TryLock()
			if err != nil {
				return fmt.Errorf("failed to attempt lock: %w", err)
			}
			if locked {
				return nil
			}

			if i < cfg.LockMaxRetry-1 {
				time.Sleep(cfg.LockRetry)
			}
		}
	}

	return fmt.Errorf("workspace %s is locked by another instance (timeout after %v)",
		fl.workspaceID, cfg.LockTimeout)
}

func (fl *FileLock) Unlock() {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	if fl.fileLock == nil {
		slog.Warn("FileLock already unlocked", "workspace", fl.workspaceID)
		return
	}

	heldDuration := time.Since(fl.acquiredAt)
	slog.Info("File lock releasing",
		"workspace", fl.workspaceID,
		"path", fl.lockPath,
		"held_duration_ms", heldDuration.Milliseconds(),
	)

	if err := fl.fileLock.Unlock(); err != nil {
		slog.Error("Failed to release file lock",
			"workspace", fl.workspaceID,
			"path", fl.lockPath,
			"error", err,
		)
	} else {
		slog.Info("File lock released successfully",
			"workspace", fl.workspaceID,
			"held_duration_ms", heldDuration.Milliseconds(),
		)
	}

	if fl.cancel != nil {
		fl.cancel()
	}

	fl.fileLock = nil
}

func (fl *FileLock) IsLocked() bool {
	fl.mu.RLock()
	defer fl.mu.RUnlock()
	return fl.fileLock != nil
}

func (fl *FileLock) HeldDuration() time.Duration {
	fl.mu.RLock()
	defer fl.mu.RUnlock()
	if fl.acquiredAt.IsZero() {
		return 0
	}
	return time.Since(fl.acquiredAt)
}

func CleanupStaleLocks(basePath string, maxAge time.Duration, forceCleanup bool) error {
	lockPath := filepath.Join(basePath, "workspace.lock")
	info, err := os.Stat(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	age := time.Since(info.ModTime())
	if age > maxAge {
		slog.Warn("Found stale lock file",
			"path", lockPath,
			"age", age,
			"max_age", maxAge,
		)

		if !forceCleanup {
			slog.Info("Stale lock detected but not cleaning (use --force-clean-locks to remove)",
				"path", lockPath,
			)
			return nil
		}

		if err := os.Remove(lockPath); err != nil {
			slog.Error("Failed to remove stale lock file",
				"path", lockPath,
				"error", err,
			)
			return err
		}

		slog.Info("Stale lock file removed", "path", lockPath)
	}

	return nil
}
