package initializers

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/policy"
	"github.com/harunnryd/heike/internal/store"
	"github.com/harunnryd/heike/internal/tool"
)

func TestNewToolsInitializer(t *testing.T) {
	storeWorker := &store.Worker{}
	policyEngine := &policy.Engine{}
	init := NewToolsInitializer(storeWorker, policyEngine)
	if init == nil {
		t.Error("NewToolsInitializer() returned nil")
	}
}

func TestToolsInitializer_Name(t *testing.T) {
	storeWorker := &store.Worker{}
	policyEngine := &policy.Engine{}
	init := NewToolsInitializer(storeWorker, policyEngine)
	got := init.Name()
	want := "tools"
	if got != want {
		t.Errorf("Name() = %v, want %v", got, want)
	}
}

func TestToolsInitializer_Dependencies(t *testing.T) {
	storeWorker := &store.Worker{}
	policyEngine := &policy.Engine{}
	init := NewToolsInitializer(storeWorker, policyEngine)
	got := init.Dependencies()
	want := []string{"store", "policy"}
	if len(got) != len(want) {
		t.Errorf("Dependencies() = %v, want %v", got, want)
	}
}

func TestToolsInitializer_Initialize(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
		},
	}
	workspaceID := "test-workspace-" + t.Name()

	storeWorker := &store.Worker{}
	policyEngine := &policy.Engine{}

	init := NewToolsInitializer(storeWorker, policyEngine)

	component, err := init.Initialize(ctx, cfg, workspaceID)
	if err != nil {
		t.Errorf("Initialize() failed: %v", err)
	}
	if component == nil {
		t.Error("Initialize() returned nil component")
	}
}

func TestToolsInitializer_InitializeLoadsExecutableCustomTools(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
		},
	}
	workspaceID := "test-custom-tool-" + t.Name()

	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to chdir temp dir: %v", err)
	}

	skillToolsDir := filepath.Join(tempDir, ".heike", "skills", "sample", "tools")
	if err := os.MkdirAll(skillToolsDir, 0755); err != nil {
		t.Fatalf("failed to create skill tools dir: %v", err)
	}

	manifest := `tools:
  - name: custom.echo
    language: shell
    script: echo.sh
    description: Echo test tool
`
	if err := os.WriteFile(filepath.Join(skillToolsDir, "tools.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatalf("failed to write tools.yaml: %v", err)
	}

	script := "#!/usr/bin/env bash\n" + `echo '{"message":"ok"}'` + "\n"
	if err := os.WriteFile(filepath.Join(skillToolsDir, "echo.sh"), []byte(script), 0755); err != nil {
		t.Fatalf("failed to write custom tool script: %v", err)
	}

	storeWorker := &store.Worker{}
	policyEngine, err := policy.NewEngine(config.GovernanceConfig{}, workspaceID, "")
	if err != nil {
		t.Fatalf("failed to create policy engine: %v", err)
	}

	init := NewToolsInitializer(storeWorker, policyEngine)
	component, err := init.Initialize(ctx, cfg, workspaceID)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	tools := component.(struct {
		Registry *tool.Registry
		Runner   *tool.Runner
	})

	result, err := tools.Runner.Execute(ctx, "custom.echo", json.RawMessage(`{}`), "")
	if err != nil {
		t.Fatalf("Execute custom tool failed: %v", err)
	}

	var payload map[string]string
	if err := json.Unmarshal(result, &payload); err != nil {
		t.Fatalf("failed to parse custom tool output: %v", err)
	}
	if payload["message"] != "ok" {
		t.Fatalf("unexpected custom tool output: %v", payload)
	}
}
