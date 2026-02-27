package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/harunnryd/heike/internal/store"
)

func TestCrashRecovery(t *testing.T) {
	workspaceID := fmt.Sprintf("crash-recovery-test-%d", time.Now().UnixNano())

	sw, err := store.NewWorker(workspaceID, "", store.RuntimeConfig{})
	if err != nil {
		t.Fatalf("Failed to create StoreWorker: %v", err)
	}
	sw.Start()

	sessionID := "test-session-crash"
	sw.SaveSession(&store.SessionMeta{
		ID:        sessionID,
		Title:     "Test Session",
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  map[string]string{"test": "crash-recovery"},
	})

	sw.WriteTranscript(sessionID, []byte(`{"role":"user","content":"Hello before crash","ts":"2024-01-01T00:00:00Z"}`))

	sw.Stop()

	time.Sleep(1 * time.Second)

	sw2, err := store.NewWorker(workspaceID, "", store.RuntimeConfig{})
	if err != nil {
		t.Fatalf("Failed to recreate StoreWorker: %v", err)
	}
	sw2.Start()
	defer sw2.Stop()

	sessionMeta, err := sw2.GetSession(sessionID)
	if err != nil {
		t.Fatalf("Failed to recover session metadata: %v", err)
	}

	if sessionMeta.ID != sessionID {
		t.Errorf("Expected session ID %s, got %s", sessionID, sessionMeta.ID)
	}

	if sessionMeta.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", sessionMeta.Status)
	}

	if sessionMeta.Metadata["test"] != "crash-recovery" {
		t.Errorf("Expected metadata test=crash-recovery, got test=%s", sessionMeta.Metadata["test"])
	}

	lines, err := sw2.ReadTranscript(sessionID, 0)
	if err != nil {
		t.Fatalf("Failed to read transcript after recovery: %v", err)
	}

	if len(lines) != 1 {
		t.Errorf("Expected 1 transcript entry, got %d", len(lines))
	}

	expectedContent := `{"role":"user","content":"Hello before crash","ts":"2024-01-01T00:00:00Z"}`
	if lines[0] != expectedContent {
		t.Errorf("Expected content '%s', got '%s'", expectedContent, lines[0])
	}
}

func TestVectorStorePersistence(t *testing.T) {
	workspaceID := fmt.Sprintf("vector-persistence-test-%d", time.Now().UnixNano())

	sw, err := store.NewWorker(workspaceID, "", store.RuntimeConfig{})
	if err != nil {
		t.Fatalf("Failed to create StoreWorker: %v", err)
	}
	sw.Start()

	vector := []float32{0.1, 0.2, 0.3, 0.4}
	metadata := map[string]string{"test": "vector"}
	content := "test content"

	err = sw.UpsertVector("test-collection", "doc1", vector, metadata, content)
	if err != nil {
		t.Fatalf("Failed to upsert vector: %v", err)
	}

	sw.Stop()

	sw2, err := store.NewWorker(workspaceID, "", store.RuntimeConfig{})
	if err != nil {
		t.Fatalf("Failed to recreate StoreWorker: %v", err)
	}
	sw2.Start()
	defer sw2.Stop()

	results, err := sw2.SearchVectors("test-collection", vector, 1)
	if err != nil {
		t.Fatalf("Failed to search vectors after recovery: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 vector result, got %d", len(results))
	}

	if results[0].ID != "doc1" {
		t.Errorf("Expected ID 'doc1', got '%s'", results[0].ID)
	}

	if results[0].Content != "test content" {
		t.Errorf("Expected content 'test content', got '%s'", results[0].Content)
	}
}

func TestSessionRotation(t *testing.T) {
	workspaceID := "rotation-test"

	sw, err := store.NewWorker(workspaceID, "", store.RuntimeConfig{})
	if err != nil {
		t.Fatalf("Failed to create StoreWorker: %v", err)
	}
	sw.Start()
	defer sw.Stop()

	sessionID := "test-session-rotation"

	homeDir, _ := os.UserHomeDir()
	transcriptPath := filepath.Join(homeDir, ".heike", "workspaces", workspaceID, "sessions", sessionID+".jsonl")

	largeContent := make([]byte, 11*1024*1024)
	for i := 0; i < len(largeContent); i++ {
		largeContent[i] = 'a'
	}

	sw.WriteTranscript(sessionID, []byte(`{"role":"user","content":"First line","ts":"2024-01-01T00:00:00Z"}`))
	sw.WriteTranscript(sessionID, largeContent)

	_, err = os.Stat(transcriptPath)
	if err != nil {
		t.Fatalf("Transcript file should exist after rotation: %v", err)
	}

	lines, err := sw.ReadTranscript(sessionID, 0)
	if err != nil {
		t.Fatalf("Failed to read transcript after rotation: %v", err)
	}

	if len(lines) != 2 {
		t.Errorf("Expected 2 transcript entries after rotation, got %d", len(lines))
	}

	if lines[0] != `{"role":"user","content":"First line","ts":"2024-01-01T00:00:00Z"}` {
		t.Errorf("First line should be preserved after rotation")
	}
}
