package initializers

import (
	"context"
	"testing"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/store"
)

func TestNewStoreInitializer(t *testing.T) {
	init := NewStoreInitializer()
	if init == nil {
		t.Error("NewStoreInitializer() returned nil")
	}
}

func TestStoreInitializer_Name(t *testing.T) {
	init := NewStoreInitializer()
	got := init.Name()
	want := "store"
	if got != want {
		t.Errorf("Name() = %v, want %v", got, want)
	}
}

func TestStoreInitializer_Dependencies(t *testing.T) {
	init := NewStoreInitializer()
	got := init.Dependencies()
	want := []string{}
	if len(got) != len(want) {
		t.Errorf("Dependencies() = %v, want %v", got, want)
	}
}

func TestStoreInitializer_Initialize(t *testing.T) {
	init := NewStoreInitializer()

	ctx := context.Background()
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
		},
	}
	workspaceID := "test-workspace-" + t.Name()

	component, err := init.Initialize(ctx, cfg, workspaceID)
	if err != nil {
		t.Errorf("Initialize() failed: %v", err)
	}
	if component == nil {
		t.Error("Initialize() returned nil component")
	}

	if storeWorker, ok := component.(*store.Worker); ok {
		defer storeWorker.Stop()
	}
}
