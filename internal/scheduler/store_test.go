package scheduler

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLeaseLogic(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "heike_sched_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	storePath := filepath.Join(tmpDir, "tasks.json")
	st, err := NewStore(storePath)
	if err != nil {
		t.Fatal(err)
	}

	taskID := "heartbeat"
	if err := st.UpdateTask(&Task{
		ID:          taskID,
		Schedule:    "@every 10s",
		Description: "System Heartbeat",
		NextRun:     time.Now(),
	}); err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Acquire Lease
	runID1 := "run1"
	if err := st.LeaseTask(taskID, runID1, 1*time.Minute); err != nil {
		t.Fatalf("Failed to acquire lease: %v", err)
	}

	// Verify lease
	tasks := st.GetAll()
	if len(tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Lease == nil || tasks[0].Lease.RunID != runID1 {
		t.Error("Lease not persisted correctly")
	}

	// Try to acquire again (Should Fail)
	if err := st.LeaseTask(taskID, "run2", 1*time.Minute); err == nil {
		t.Error("Expected error when leasing already leased task")
	}

	// Expire Lease
	// Manually modify state to simulate time passing
	st.mu.Lock()
	st.data.Tasks[taskID].Lease.ExpiresAt = time.Now().Add(-1 * time.Minute)
	st.mu.Unlock()

	// Try to acquire again (Should Succeed - Recovery)
	runID3 := "run3"
	if err := st.LeaseTask(taskID, runID3, 1*time.Minute); err != nil {
		t.Errorf("Failed to acquire expired lease: %v", err)
	}

	// Verify new lease
	tasks = st.GetAll()
	if tasks[0].Lease.RunID != runID3 {
		t.Error("Lease not updated to new runID")
	}

	// Complete Task
	if err := st.CompleteTask(taskID, time.Now().Add(1*time.Hour)); err != nil {
		t.Errorf("Failed to complete task: %v", err)
	}

	// Verify lease cleared
	tasks = st.GetAll()
	if tasks[0].Lease != nil {
		t.Error("Lease should be cleared after completion")
	}
}
