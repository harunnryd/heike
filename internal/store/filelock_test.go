package store

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/flock"
)

func shortLockConfig(timeout time.Duration) *FileLockConfig {
	retry := 10 * time.Millisecond
	maxRetry := int(timeout / retry)
	if maxRetry < 1 {
		maxRetry = 1
	}
	return &FileLockConfig{
		LockTimeout:  timeout,
		LockRetry:    retry,
		LockMaxRetry: maxRetry,
	}
}

func TestNewFileLock(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceID := "test-workspace-" + t.Name()

	lock, err := NewFileLock(workspaceID, tmpDir, nil)
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}

	if lock == nil {
		t.Fatal("Expected non-nil lock")
	}

	if !lock.IsLocked() {
		t.Error("Expected lock to be held")
	}

	lock.Unlock()

	if lock.IsLocked() {
		t.Error("Expected lock to be released after Unlock()")
	}
}

func TestFileLockAcquireWithTimeout(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceID := "test-workspace-" + t.Name()

	lock, err := NewFileLock(workspaceID, tmpDir, nil)
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}
	defer lock.Unlock()

	if !lock.IsLocked() {
		t.Error("Expected lock to be held")
	}

	select {
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for lock acquisition (lock should be already acquired)")
	default:
	}
}

func TestFileLockConcurrentAcquire(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceID := "test-workspace-" + t.Name()
	cfg := shortLockConfig(200 * time.Millisecond)

	lock1, err := NewFileLock(workspaceID, tmpDir, cfg)
	if err != nil {
		t.Fatalf("Failed to acquire first lock: %v", err)
	}
	defer lock1.Unlock()

	if !lock1.IsLocked() {
		t.Error("Expected first lock to be held")
	}

	lock2, err := NewFileLock(workspaceID, tmpDir, cfg)
	if err == nil {
		lock2.Unlock()
		t.Error("Expected second lock acquisition to fail")
	}
}

func TestFileLockUnlock(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceID := "test-workspace-" + t.Name()

	lock, err := NewFileLock(workspaceID, tmpDir, nil)
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}

	if !lock.IsLocked() {
		t.Error("Expected lock to be held")
	}

	lock.Unlock()

	if lock.IsLocked() {
		t.Error("Expected lock to be released after Unlock()")
	}
}

func TestFileLockDoubleUnlock(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceID := "test-workspace-" + t.Name()

	lock, err := NewFileLock(workspaceID, tmpDir, nil)
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}

	if !lock.IsLocked() {
		t.Error("Expected lock to be held")
	}

	lock.Unlock()

	lock.Unlock()

	if lock.IsLocked() {
		t.Error("Expected lock to remain released after double unlock")
	}
}

func TestFileLockHeldDuration(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceID := "test-workspace-" + t.Name()

	lock, err := NewFileLock(workspaceID, tmpDir, nil)
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	if lock.HeldDuration() < 50*time.Millisecond {
		t.Errorf("Expected lock held duration >= 50ms, got %v", lock.HeldDuration())
	}

	lock.Unlock()
}

func TestFileLockRetry(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceID := "test-workspace-" + t.Name()
	cfg := shortLockConfig(120 * time.Millisecond)

	lock1, err := NewFileLock(workspaceID, tmpDir, cfg)
	if err != nil {
		t.Fatalf("Failed to acquire first lock: %v", err)
	}

	var err2 error
	var lock2 *FileLock
	done := make(chan struct{})
	start := time.Now()

	go func() {
		lock2, err2 = NewFileLock(workspaceID, tmpDir, cfg)
		close(done)
	}()

	select {
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Expected second lock acquisition to finish within timeout")
	case <-done:
		if err2 == nil {
			lock2.Unlock()
			t.Error("Expected second lock to fail")
		}
		if elapsed := time.Since(start); elapsed < 100*time.Millisecond {
			t.Errorf("Expected retry behavior before failing, got elapsed=%v", elapsed)
		}
	}

	lock1.Unlock()
}

func TestFileLockStaleLock(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "workspace.lock")
	err := os.WriteFile(lockPath, []byte("stale"), 0644)
	if err != nil {
		t.Fatalf("Failed to create stale lock: %v", err)
	}

	staleTime := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(lockPath, staleTime, staleTime); err != nil {
		t.Fatalf("Failed to age lock file: %v", err)
	}

	if err := CleanupStaleLocks(tmpDir, 5*time.Minute, false); err != nil {
		t.Fatalf("CleanupStaleLocks(force=false) failed: %v", err)
	}
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("Expected stale lock to remain when force=false: %v", err)
	}

	if err := CleanupStaleLocks(tmpDir, 5*time.Minute, true); err != nil {
		t.Fatalf("CleanupStaleLocks(force=true) failed: %v", err)
	}
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("Expected stale lock file to be removed, stat err=%v", err)
	}
}

func TestFileLockAcquiresAfterStaleCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceID := "test-workspace-" + t.Name()
	lockPath := filepath.Join(tmpDir, "workspace.lock")

	if err := os.WriteFile(lockPath, []byte("stale"), 0644); err != nil {
		t.Fatalf("Failed to create stale lock file: %v", err)
	}
	staleTime := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(lockPath, staleTime, staleTime); err != nil {
		t.Fatalf("Failed to age lock file: %v", err)
	}

	if err := CleanupStaleLocks(tmpDir, 5*time.Minute, true); err != nil {
		t.Fatalf("CleanupStaleLocks(force=true) failed: %v", err)
	}

	lock, err := NewFileLock(workspaceID, tmpDir, shortLockConfig(200*time.Millisecond))
	if err != nil {
		t.Fatalf("Expected lock acquisition to succeed after stale cleanup: %v", err)
	}
	lock.Unlock()
}

func TestFileLockContext(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceID := "test-workspace-" + t.Name()

	cfg := DefaultFileLockConfig()
	cfg.LockTimeout = 5 * time.Second

	lock, err := NewFileLock(workspaceID, tmpDir, cfg)
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}
	defer lock.Unlock()

	if !lock.IsLocked() {
		t.Error("Expected lock to be held")
	}
}

func TestFileLockConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceID := "test-workspace-" + t.Name()
	cfg := shortLockConfig(500 * time.Millisecond)

	var wg sync.WaitGroup
	numGoroutines := 10
	wg.Add(numGoroutines)

	acquiredCount := 0
	currentInCritical := 0
	maxConcurrent := 0
	var mu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()

			lock, err := NewFileLock(workspaceID, tmpDir, cfg)
			if err != nil {
				return
			}
			defer lock.Unlock()

			mu.Lock()
			acquiredCount++
			currentInCritical++
			if currentInCritical > maxConcurrent {
				maxConcurrent = currentInCritical
			}
			mu.Unlock()

			time.Sleep(10 * time.Millisecond)

			mu.Lock()
			currentInCritical--
			mu.Unlock()
		}()
	}

	wg.Wait()

	if acquiredCount == 0 {
		t.Error("Expected at least one lock to be acquired")
	}

	if maxConcurrent > 1 {
		t.Errorf("Expected lock exclusivity, max concurrent holders=%d", maxConcurrent)
	}
}

func TestFileLockTryLock(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceID := "test-workspace-" + t.Name()

	lock1, err := NewFileLock(workspaceID, tmpDir, nil)
	if err != nil {
		t.Fatalf("Failed to acquire first lock: %v", err)
	}
	defer lock1.Unlock()

	flockFile := flock.New(filepath.Join(tmpDir, "workspace.lock"))
	locked, err := flockFile.TryLock()
	if err != nil {
		t.Fatalf("flock TryLock failed: %v", err)
	}

	if locked {
		t.Error("Expected flock to fail due to held lock")
		flockFile.Unlock()
	}
}
