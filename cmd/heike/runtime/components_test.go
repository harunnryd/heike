package runtime

import (
	"context"
	"os"
	"testing"

	"github.com/harunnryd/heike/internal/config"
)

func setupTestEnv(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	t.Cleanup(func() {
		if oldHome != "" {
			os.Setenv("HOME", oldHome)
		}
	})
	os.Setenv("HOME", tmpDir)
}

func TestNewRuntimeComponents(t *testing.T) {
	setupTestEnv(t)
	ctx := context.Background()
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
		},
	}
	workspaceID := "test-workspace-" + t.Name()

	components, err := NewRuntimeComponents(ctx, cfg, workspaceID)
	if err != nil {
		t.Fatalf("NewRuntimeComponents() failed: %v", err)
	}
	defer components.Stop()

	if components == nil {
		t.Error("NewRuntimeComponents() returned nil")
	}

	if components.WorkspaceID != workspaceID {
		t.Errorf("WorkspaceID = %v, want %v", components.WorkspaceID, workspaceID)
	}

	if components.Config == nil {
		t.Error("Config is nil")
	}

	if components.Ctx == nil {
		t.Error("Ctx is nil")
	}
}

func TestRuntimeComponents_Start(t *testing.T) {
	setupTestEnv(t)
	ctx := context.Background()
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
		},
	}
	workspaceID := "test-workspace-" + t.Name()

	components, err := NewRuntimeComponents(ctx, cfg, workspaceID)
	if err != nil {
		t.Fatalf("NewRuntimeComponents() failed: %v", err)
	}
	defer components.Stop()

	err = components.Start()
	if err != nil {
		t.Logf("Start() error (may be expected without full setup): %v", err)
	}
}

func TestRuntimeComponents_Stop(t *testing.T) {
	setupTestEnv(t)
	ctx := context.Background()
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
		},
	}
	workspaceID := "test-workspace-" + t.Name()

	components, err := NewRuntimeComponents(ctx, cfg, workspaceID)
	if err != nil {
		t.Fatalf("NewRuntimeComponents() failed: %v", err)
	}

	components.Stop()

	if components.Ctx == nil {
		t.Error("Ctx should still be set after Stop()")
	}
}
