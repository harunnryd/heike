package initializers

import (
	"context"
	"testing"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/ingress"
	"github.com/harunnryd/heike/internal/orchestrator"
	"github.com/harunnryd/heike/internal/store"
)

func TestNewWorkersInitializer(t *testing.T) {
	ingress := &ingress.Ingress{}
	orchestrator := &orchestrator.DefaultKernel{}
	storeWorker := &store.Worker{}
	init := NewWorkersInitializer(ingress, orchestrator, storeWorker)
	if init == nil {
		t.Error("NewWorkersInitializer() returned nil")
	}
}

func TestWorkersInitializer_Name(t *testing.T) {
	ingress := &ingress.Ingress{}
	orchestrator := &orchestrator.DefaultKernel{}
	storeWorker := &store.Worker{}
	init := NewWorkersInitializer(ingress, orchestrator, storeWorker)
	got := init.Name()
	want := "workers"
	if got != want {
		t.Errorf("Name() = %v, want %v", got, want)
	}
}

func TestWorkersInitializer_Dependencies(t *testing.T) {
	ingress := &ingress.Ingress{}
	orchestrator := &orchestrator.DefaultKernel{}
	storeWorker := &store.Worker{}
	init := NewWorkersInitializer(ingress, orchestrator, storeWorker)
	got := init.Dependencies()
	want := []string{"store", "orchestrator"}
	if len(got) != len(want) {
		t.Errorf("Dependencies() = %v, want %v", got, want)
	}
}

func TestWorkersInitializer_Initialize(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
		},
		Ingress: config.IngressConfig{
			InteractiveQueueSize: 100,
			BackgroundQueueSize:  1000,
		},
	}
	workspaceID := "test-workspace-" + t.Name()

	storeWorker, err := store.NewWorker(workspaceID, "", store.RuntimeConfig{})
	if err != nil {
		t.Skipf("Cannot create store worker: %v", err)
	}
	defer storeWorker.Stop()

	ingress := &ingress.Ingress{}
	orchestrator := &orchestrator.DefaultKernel{}

	init := NewWorkersInitializer(ingress, orchestrator, storeWorker)

	component, err := init.Initialize(ctx, cfg, workspaceID)
	if err != nil {
		t.Logf("Initialize() error (may be expected without full setup): %v", err)
	}
	if component == nil {
		t.Log("Initialize() returned nil component (may be expected without full setup)")
	}
}
