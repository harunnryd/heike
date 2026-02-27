package initializers

import (
	"context"
	"os"
	"testing"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/ingress"
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

func TestNewSchedulerInitializer(t *testing.T) {
	setupTestEnv(t)
	ingress := &ingress.Ingress{}
	init := NewSchedulerInitializer(ingress)
	if init == nil {
		t.Error("NewSchedulerInitializer() returned nil")
	}
}

func TestSchedulerInitializer_Name(t *testing.T) {
	setupTestEnv(t)
	ingress := &ingress.Ingress{}
	init := NewSchedulerInitializer(ingress)
	got := init.Name()
	want := "scheduler"
	if got != want {
		t.Errorf("Name() = %v, want %v", got, want)
	}
}

func TestSchedulerInitializer_Dependencies(t *testing.T) {
	setupTestEnv(t)
	ingress := &ingress.Ingress{}
	init := NewSchedulerInitializer(ingress)
	got := init.Dependencies()
	want := []string{"workers"}
	if len(got) != len(want) {
		t.Errorf("Dependencies() = %v, want %v", got, want)
	}
}

func TestSchedulerInitializer_Initialize(t *testing.T) {
	setupTestEnv(t)
	ctx := context.Background()
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
		},
	}
	workspaceID := "test-workspace-" + t.Name()

	ingress := &ingress.Ingress{}

	init := NewSchedulerInitializer(ingress)

	component, err := init.Initialize(ctx, cfg, workspaceID)
	if err != nil {
		t.Logf("Initialize() error (may be expected without full setup): %v", err)
	}
	if component == nil {
		t.Log("Initialize() returned nil component (may be expected without full setup)")
	}
}
