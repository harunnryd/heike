package integration_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/harunnryd/heike/internal/store"
)

func TestIdempotency(t *testing.T) {
	_ = setupTestEnv(t)
	workspaceID := fmt.Sprintf("idempotency-test-%d", time.Now().UnixNano())

	sw, err := store.NewWorker(workspaceID, "", store.RuntimeConfig{})
	if err != nil {
		t.Fatalf("Failed to create StoreWorker: %v", err)
	}
	sw.Start()
	defer sw.Stop()

	testKey := "test:event-123"

	t.Run("First call should not be duplicate", func(t *testing.T) {
		isDuplicate := sw.CheckAndMarkKey(testKey, 0)
		if isDuplicate {
			t.Error("First call should not be detected as duplicate")
		}
	})

	t.Run("Second call with same key should be duplicate", func(t *testing.T) {
		isDuplicate := sw.CheckAndMarkKey(testKey, 0)
		if !isDuplicate {
			t.Error("Second call with same key should be detected as duplicate")
		}
	})

	t.Run("Different key should not be duplicate", func(t *testing.T) {
		differentKey := "test:event-456"
		isDuplicate := sw.CheckAndMarkKey(differentKey, 0)
		if isDuplicate {
			t.Error("Different key should not be detected as duplicate")
		}
	})
}

func TestIdempotencyPersistence(t *testing.T) {
	_ = setupTestEnv(t)
	workspaceID := fmt.Sprintf("idempotency-persistence-test-%d", time.Now().UnixNano())

	sw, err := store.NewWorker(workspaceID, "", store.RuntimeConfig{})
	if err != nil {
		t.Fatalf("Failed to create StoreWorker: %v", err)
	}
	sw.Start()

	testKey := "persist:event-789"

	sw.CheckAndMarkKey(testKey, 0)
	sw.SaveIdempotencySync()
	sw.Stop()

	sw2, err := store.NewWorker(workspaceID, "", store.RuntimeConfig{})
	if err != nil {
		t.Fatalf("Failed to recreate StoreWorker: %v", err)
	}
	sw2.Start()
	defer sw2.Stop()

	isDuplicate := sw2.CheckAndMarkKey(testKey, 0)
	if !isDuplicate {
		t.Error("Persisted key should be detected as duplicate after restart")
	}
}

func TestIdempotencyExpiry(t *testing.T) {
	t.Skip("Skipping long-running expiry test in CI")
}

func TestMultipleWorkspacesIdempotency(t *testing.T) {
	_ = setupTestEnv(t)

	ws1, err := store.NewWorker(fmt.Sprintf("workspace-1-%d", time.Now().UnixNano()), "", store.RuntimeConfig{})
	if err != nil {
		t.Fatalf("Failed to create StoreWorker for workspace-1: %v", err)
	}
	ws1.Start()
	defer ws1.Stop()

	ws2, err := store.NewWorker(fmt.Sprintf("workspace-2-%d", time.Now().UnixNano()), "", store.RuntimeConfig{})
	if err != nil {
		t.Fatalf("Failed to create StoreWorker for workspace-2: %v", err)
	}
	ws2.Start()
	defer ws2.Stop()

	testKey := "shared:event-999"

	isDuplicate1 := ws1.CheckAndMarkKey(testKey, 0)
	if isDuplicate1 {
		t.Error("First workspace should not see key as duplicate")
	}

	isDuplicate2 := ws2.CheckAndMarkKey(testKey, 0)
	if isDuplicate2 {
		t.Error("Second workspace should not see key as duplicate (different workspace)")
	}
}
