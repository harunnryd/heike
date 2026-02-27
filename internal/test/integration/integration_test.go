package integration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harunnryd/heike/internal/concurrency"
	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/egress"
	"github.com/harunnryd/heike/internal/ingress"
	"github.com/harunnryd/heike/internal/orchestrator"
	"github.com/harunnryd/heike/internal/policy"
	"github.com/harunnryd/heike/internal/skill"
	"github.com/harunnryd/heike/internal/store"
	"github.com/harunnryd/heike/internal/tool"
	"github.com/harunnryd/heike/internal/worker"
)

func TestEndToEndFlow(t *testing.T) {
	if os.Getenv("HEIKE_RUN_E2E") != "1" {
		t.Skip("Skipping external E2E test (set HEIKE_RUN_E2E=1 to run)")
	}

	// 1. Setup Environment
	tmpDir, err := os.MkdirTemp("", "heike_e2e")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	os.Setenv("HOME", tmpDir)

	wsID := "default"

	// 2. Initialize Components
	storeWorker, err := store.NewWorker(wsID, "", store.RuntimeConfig{})
	if err != nil {
		t.Fatal(err)
	}
	storeWorker.Start()
	defer storeWorker.Stop()

	policyEngine, _ := policy.NewEngine(config.GovernanceConfig{}, wsID, "")
	registry := tool.NewRegistry()
	toolRunner := tool.NewRunner(registry, policyEngine)
	eg := egress.NewEgress(storeWorker)
	skillsReg := skill.NewRegistry()

	// Config with Mock Model to avoid API calls
	// We need to mock Model Resolver?
	// NewFSM creates Model Resolver internally using cfg.Models.
	// We can't inject mock model easily without changing NewFSM.
	// But we can configure a "local" provider that points to a test server?
	// Or just let it fail on model call?
	// If it fails, it logs error to transcript.
	// That's enough to verify flow!

	cfg := config.Config{
		Models: config.ModelsConfig{
			Default: "gpt-4o-mini", // Use mini for faster test
			Registry: []config.ModelRegistry{
				{
					Name:     "gpt-4o-mini",
					Provider: "openai",
					APIKey:   os.Getenv("OPENAI_API_KEY"), // Use env var for safety/flexibility
				},
				// Fallback for embedding model
				{
					Name:     "text-embedding-3-small",
					Provider: "openai",
					APIKey:   os.Getenv("OPENAI_API_KEY"),
				},
			},
		},
	}

	// Ensure API key is present for real test, or skip if not
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("Skipping integration test: OPENAI_API_KEY not set")
	}

	orch, err := orchestrator.NewKernel(cfg, storeWorker, toolRunner, policyEngine, skillsReg, eg)
	if err != nil {
		t.Fatalf("Failed to create cognitive engine: %v", err)
	}

	if err := orch.Init(context.Background()); err != nil {
		t.Fatalf("Failed to init orchestrator: %v", err)
	}

	if err := orch.Start(context.Background()); err != nil {
		t.Fatalf("Failed to start orchestrator: %v", err)
	}
	defer orch.Stop(context.Background())

	ing := ingress.NewIngress(100, 100, ingress.RuntimeConfig{}, storeWorker)
	locks := concurrency.NewSimpleSessionLockManager()

	w := worker.NewWorker("test", ing.InteractiveQueue(), storeWorker, orch, locks, worker.RuntimeConfig{})
	if _, err := w.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer w.Stop(context.Background())

	// 3. Submit Event
	evt := ingress.NewEvent("cli", ingress.TypeUserMessage, "sess1", "Hello World", nil)
	if err := ing.Submit(context.Background(), &evt); err != nil {
		t.Fatal(err)
	}

	// 4. Verify Transcript
	// Wait for processing (with retry for real API latency)
	transcriptPath := filepath.Join(tmpDir, ".heike", "workspaces", wsID, "sessions", "sess1.jsonl")
	var content []byte

	for i := 0; i < 10; i++ {
		time.Sleep(1 * time.Second)
		content, err = os.ReadFile(transcriptPath)
		if err == nil {
			lines := strings.Split(strings.TrimSpace(string(content)), "\n")
			// We expect at least 2 lines: User message + Assistant response (or error)
			if len(lines) >= 2 {
				break
			}
		}
	}

	if err != nil {
		t.Fatalf("Transcript not found after timeout: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) < 2 {
		t.Fatal("Transcript incomplete (missing response)")
	}

	// First line should be user message
	var entry1 map[string]interface{}
	json.Unmarshal([]byte(lines[0]), &entry1)
	if entry1["content"] != "Hello World" {
		t.Errorf("Expected 'Hello World', got %v", entry1["content"])
	}

	// Second line should be error from model (because fake key)
	if len(lines) > 1 {
		var entry2 map[string]interface{}
		json.Unmarshal([]byte(lines[1]), &entry2)
		// We expect error because API key is fake
		// But at least it reached Orchestrator and persisted result.
		t.Logf("Response: %v", entry2["content"])
	} else {
		t.Error("Expected response from Orchestrator")
	}
}
