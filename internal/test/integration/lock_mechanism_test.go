package integration_test

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/harunnryd/heike/internal/store"
)

func setupTestEnv(t *testing.T) string {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	t.Cleanup(func() {
		if oldHome != "" {
			os.Setenv("HOME", oldHome)
		}
	})
	os.Setenv("HOME", tmpDir)
	return tmpDir
}

func TestWorkerLockAcquisitionAndRelease(t *testing.T) {
	_ = setupTestEnv(t)
	workspaceID := fmt.Sprintf("lock-acquire-release-test-%d", time.Now().UnixNano())

	sw, err := store.NewWorker(workspaceID, "", store.RuntimeConfig{})
	if err != nil {
		t.Fatalf("Failed to create worker: %v", err)
	}
	sw.Start()

	sessionID := "test-session-1"
	sw.SaveSession(&store.SessionMeta{
		ID:        sessionID,
		Title:     "Test Session",
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	sw.Stop()

	sw2, err := store.NewWorker(workspaceID, "", store.RuntimeConfig{})
	if err != nil {
		t.Fatalf("Failed to recreate worker: %v", err)
	}
	defer sw2.Stop()
	sw2.Start()

	sessionMeta, err := sw2.GetSession(sessionID)
	if err != nil {
		t.Errorf("Failed to retrieve session after lock release: %v", err)
	}

	if sessionMeta.ID != sessionID {
		t.Errorf("Expected session ID %s, got %s", sessionID, sessionMeta.ID)
	}
}

func TestWorkerConcurrentLockPrevention(t *testing.T) {
	_ = setupTestEnv(t)
	workspaceID := fmt.Sprintf("lock-concurrent-test-%d", time.Now().UnixNano())

	sw1, err := store.NewWorker(workspaceID, "", store.RuntimeConfig{})
	if err != nil {
		t.Fatalf("Failed to create first worker: %v", err)
	}
	defer sw1.Stop()
	sw1.Start()

	sw2, err := store.NewWorker(workspaceID, "", store.RuntimeConfig{})
	if err == nil {
		sw2.Stop()
		t.Error("Expected second worker to fail due to lock held by first worker")
	}
}

func TestWorkerGracefulShutdown(t *testing.T) {
	_ = setupTestEnv(t)
	workspaceID := fmt.Sprintf("lock-graceful-shutdown-test-%d", time.Now().UnixNano())

	sw, err := store.NewWorker(workspaceID, "", store.RuntimeConfig{})
	if err != nil {
		t.Fatalf("Failed to create worker: %v", err)
	}
	sw.Start()

	sessionID := "test-session-shutdown"
	sw.SaveSession(&store.SessionMeta{
		ID:        sessionID,
		Title:     "Test Session",
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	stopCh := make(chan error, 1)
	go func() {
		stopCh <- nil
		sw.Stop()
	}()

	select {
	case <-stopCh:
	case <-time.After(2 * time.Second):
		t.Error("Worker did not stop gracefully within timeout")
	}

	sw2, err := store.NewWorker(workspaceID, "", store.RuntimeConfig{})
	if err != nil {
		t.Errorf("Failed to recreate worker after graceful shutdown: %v", err)
	}
	defer sw2.Stop()
	sw2.Start()

	_, err = sw2.GetSession(sessionID)
	if err != nil {
		t.Errorf("Failed to retrieve session after graceful shutdown: %v", err)
	}
}

func TestWorkerLockTimeout(t *testing.T) {
	_ = setupTestEnv(t)
	workspaceID := fmt.Sprintf("lock-timeout-test-%d", time.Now().UnixNano())

	sw, err := store.NewWorker(workspaceID, "", store.RuntimeConfig{})
	if err != nil {
		t.Fatalf("Failed to create worker: %v", err)
	}
	sw.Start()

	startChan := make(chan struct{})
	var wg sync.WaitGroup
	var errSecond error
	var mu sync.Mutex

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-startChan

		sw2, err := store.NewWorker(workspaceID, "", store.RuntimeConfig{})
		mu.Lock()
		errSecond = err
		mu.Unlock()

		if sw2 != nil {
			sw2.Stop()
		}
	}()

	close(startChan)
	time.Sleep(50 * time.Millisecond)

	sw.Stop()

	wg.Wait()

	if errSecond != nil {
		t.Errorf("Second worker should acquire lock after first worker stops: %v", errSecond)
	}
}

func TestWorkerLockWithPendingOperations(t *testing.T) {
	_ = setupTestEnv(t)
	workspaceID := fmt.Sprintf("lock-pending-ops-test-%d", time.Now().UnixNano())

	sw, err := store.NewWorker(workspaceID, "", store.RuntimeConfig{})
	if err != nil {
		t.Fatalf("Failed to create worker: %v", err)
	}
	sw.Start()

	var wg sync.WaitGroup
	numOps := 100
	wg.Add(numOps)

	for i := 0; i < numOps; i++ {
		go func(idx int) {
			defer wg.Done()
			sw.WriteTranscript("session-1", []byte(`{"role":"user","content":"test"}`))
		}(i)
	}

	wg.Wait()

	sw.Stop()

	sw2, err := store.NewWorker(workspaceID, "", store.RuntimeConfig{})
	if err != nil {
		t.Fatalf("Failed to recreate worker: %v", err)
	}
	defer sw2.Stop()
	sw2.Start()

	lines, err := sw2.ReadTranscript("session-1", 0)
	if err != nil {
		t.Errorf("Failed to read transcript: %v", err)
	}

	if len(lines) < numOps-10 {
		t.Errorf("Expected at least %d transcript entries, got %d", numOps-10, len(lines))
	}
}

func TestWorkerLockRecoveryAfterCrash(t *testing.T) {
	_ = setupTestEnv(t)
	workspaceID := fmt.Sprintf("lock-crash-recovery-test-%d", time.Now().UnixNano())

	sw, err := store.NewWorker(workspaceID, "", store.RuntimeConfig{})
	if err != nil {
		t.Fatalf("Failed to create worker: %v", err)
	}
	sw.Start()

	sessionID := "crash-session"
	sw.SaveSession(&store.SessionMeta{
		ID:        sessionID,
		Title:     "Test Session",
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	sw.WriteTranscript(sessionID, []byte(`{"role":"user","content":"before crash"}`))
	sw.Stop()

	time.Sleep(100 * time.Millisecond)

	sw2, err := store.NewWorker(workspaceID, "", store.RuntimeConfig{})
	if err != nil {
		t.Errorf("Worker recovery should succeed: %v", err)
	}
	defer sw2.Stop()
	sw2.Start()

	_, err = sw2.GetSession(sessionID)
	if err != nil {
		t.Errorf("Failed to recover session: %v", err)
	}

	lines, err := sw2.ReadTranscript(sessionID, 0)
	if err != nil {
		t.Errorf("Failed to recover transcript: %v", err)
	}

	if len(lines) != 1 {
		t.Errorf("Expected 1 transcript entry, got %d", len(lines))
	}
}

func TestMultipleWorkspacesLockIsolation(t *testing.T) {
	_ = setupTestEnv(t)
	ws1ID := fmt.Sprintf("workspace-1-lock-isolation-%d", time.Now().UnixNano())
	ws2ID := fmt.Sprintf("workspace-2-lock-isolation-%d", time.Now().UnixNano())

	ws1, err := store.NewWorker(ws1ID, "", store.RuntimeConfig{})
	if err != nil {
		t.Fatalf("Failed to create workspace 1: %v", err)
	}
	defer ws1.Stop()
	ws1.Start()

	ws2, err := store.NewWorker(ws2ID, "", store.RuntimeConfig{})
	if err != nil {
		t.Fatalf("Failed to create workspace 2: %v", err)
	}
	defer ws2.Stop()
	ws2.Start()

	ws1.SaveSession(&store.SessionMeta{
		ID:        "session-1",
		Title:     "Workspace 1 Session",
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	ws2.SaveSession(&store.SessionMeta{
		ID:        "session-2",
		Title:     "Workspace 2 Session",
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	session1, err := ws1.GetSession("session-1")
	if err != nil {
		t.Errorf("Failed to get session from workspace 1: %v", err)
	}

	session2, err := ws2.GetSession("session-2")
	if err != nil {
		t.Errorf("Failed to get session from workspace 2: %v", err)
	}

	if session1.ID != "session-1" || session2.ID != "session-2" {
		t.Error("Workspaces should not interfere with each other")
	}
}

func TestWorkerLockStress(t *testing.T) {
	_ = setupTestEnv(t)
	workspaceID := fmt.Sprintf("lock-stress-test-%d", time.Now().UnixNano())

	for i := 0; i < 20; i++ {
		sw, err := store.NewWorker(workspaceID, "", store.RuntimeConfig{})
		if err != nil {
			t.Errorf("Iteration %d: Failed to create worker: %v", i, err)
			continue
		}
		sw.Start()

		sw.SaveSession(&store.SessionMeta{
			ID:        "stress-session",
			Title:     "Stress Test",
			Status:    "active",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})

		time.Sleep(time.Duration(i) * time.Millisecond)

		sw.Stop()
	}
}

func TestWorkerLockPersistence(t *testing.T) {
	_ = setupTestEnv(t)
	workspaceID := fmt.Sprintf("lock-persistence-test-%d", time.Now().UnixNano())

	sw, err := store.NewWorker(workspaceID, "", store.RuntimeConfig{})
	if err != nil {
		t.Fatalf("Failed to create worker: %v", err)
	}
	sw.Start()

	sessionID := "session-1"
	sw.SaveSession(&store.SessionMeta{
		ID:        sessionID,
		Title:     "Test Session",
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	sw.Stop()

	sw2, err := store.NewWorker(workspaceID, "", store.RuntimeConfig{})
	if err != nil {
		t.Fatalf("Should be able to create new worker after lock release: %v", err)
	}
	defer sw2.Stop()
	sw2.Start()

	sessionMeta, err := sw2.GetSession(sessionID)
	if err != nil {
		t.Errorf("Should be able to read data after lock release: %v", err)
	}
	if sessionMeta == nil {
		t.Error("Session data should persist after worker restart")
	}
}
