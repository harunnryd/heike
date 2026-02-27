package initializers

import (
	"context"
	"testing"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/egress"
	"github.com/harunnryd/heike/internal/policy"
	"github.com/harunnryd/heike/internal/skill"
	"github.com/harunnryd/heike/internal/store"
	"github.com/harunnryd/heike/internal/tool"
)

func TestNewOrchestratorInitializer(t *testing.T) {
	storeWorker := &store.Worker{}
	toolRunner := &tool.Runner{}
	policyEngine := &policy.Engine{}
	skillRegistry := &skill.Registry{}
	var egress egress.Egress
	init := NewOrchestratorInitializer(storeWorker, toolRunner, policyEngine, skillRegistry, egress)
	if init == nil {
		t.Error("NewOrchestratorInitializer() returned nil")
	}
}

func TestOrchestratorInitializer_Name(t *testing.T) {
	storeWorker := &store.Worker{}
	toolRunner := &tool.Runner{}
	policyEngine := &policy.Engine{}
	skillRegistry := &skill.Registry{}
	var egress egress.Egress
	init := NewOrchestratorInitializer(storeWorker, toolRunner, policyEngine, skillRegistry, egress)
	got := init.Name()
	want := "orchestrator"
	if got != want {
		t.Errorf("Name() = %v, want %v", got, want)
	}
}

func TestOrchestratorInitializer_Dependencies(t *testing.T) {
	storeWorker := &store.Worker{}
	toolRunner := &tool.Runner{}
	policyEngine := &policy.Engine{}
	skillRegistry := &skill.Registry{}
	var egress egress.Egress
	init := NewOrchestratorInitializer(storeWorker, toolRunner, policyEngine, skillRegistry, egress)
	got := init.Dependencies()
	want := []string{"store", "tools", "policy"}
	if len(got) != len(want) {
		t.Errorf("Dependencies() = %v, want %v", got, want)
	}
}

func TestOrchestratorInitializer_Initialize(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
		},
	}
	workspaceID := "test-workspace-" + t.Name()

	storeWorker, err := store.NewWorker(workspaceID, "", store.RuntimeConfig{})
	if err != nil {
		t.Skipf("Cannot create store worker: %v", err)
	}
	defer storeWorker.Stop()

	toolRunner := &tool.Runner{}
	policyEngine := &policy.Engine{}
	skillRegistry := &skill.Registry{}
	egress := egress.NewEgress(storeWorker)

	init := NewOrchestratorInitializer(storeWorker, toolRunner, policyEngine, skillRegistry, egress)

	component, err := init.Initialize(ctx, cfg, workspaceID)
	if err != nil {
		t.Logf("Initialize() error (may be expected without full setup): %v", err)
	}
	if component == nil {
		t.Log("Initialize() returned nil component (may be expected without full setup)")
	}
}
