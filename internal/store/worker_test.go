package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTranscriptRotation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "heike_store_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Mock HOME for NewWorker
	os.Setenv("HOME", tmpDir)

	w, err := NewWorker("test-ws", "", RuntimeConfig{})
	if err != nil {
		t.Fatal(err)
	}
	w.Start()
	defer w.Stop()

	sessionID := "rotate-sess"
	path := filepath.Join(w.basePath, "sessions", sessionID+".jsonl")

	// Create a dummy large file (simulating > 10MB, but we'll cheat by changing limit or just trust the logic)
	// Since we can't easily change the hardcoded 10MB limit in the test without exporting it or using a config,
	// we will rely on unit testing the logic if we could inject the limit.
	// But since it's hardcoded, we can't easily trigger it without writing 10MB.
	// Writing 10MB is fast enough on modern SSDs.

	largeData := make([]byte, 1024*1024) // 1MB
	// Write 11 times
	for i := 0; i < 11; i++ {
		if err := w.WriteTranscript(sessionID, largeData); err != nil {
			t.Fatal(err)
		}
	}

	// Check file size
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	// Expect size to be small (rotated) + last write
	// Wait, the check is done BEFORE open.
	// So:
	// Write 1: 1MB. Size 1MB.
	// ...
	// Write 10: Size 10MB.
	// Write 11: Check > 10MB? Yes. Rotate. Create new. Write 1MB.
	// Result: path should be ~1MB.

	if info.Size() > 2*1024*1024 {
		t.Errorf("File should have been rotated. Size: %d", info.Size())
	}

	// Check for backup file
	matches, err := filepath.Glob(path + ".*.bak")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Error("Backup file not found")
	}
}
