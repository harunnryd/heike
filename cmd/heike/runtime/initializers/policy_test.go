package initializers

import (
	"context"
	"testing"

	"github.com/harunnryd/heike/internal/config"
)

func TestNewPolicyInitializer(t *testing.T) {
	init := NewPolicyInitializer()
	if init == nil {
		t.Error("NewPolicyInitializer() returned nil")
	}
}

func TestPolicyInitializer_Name(t *testing.T) {
	init := NewPolicyInitializer()
	got := init.Name()
	want := "policy"
	if got != want {
		t.Errorf("Name() = %v, want %v", got, want)
	}
}

func TestPolicyInitializer_Dependencies(t *testing.T) {
	init := NewPolicyInitializer()
	got := init.Dependencies()
	want := []string{}
	if len(got) != len(want) {
		t.Errorf("Dependencies() = %v, want %v", got, want)
	}
}

func TestPolicyInitializer_Initialize(t *testing.T) {
	init := NewPolicyInitializer()

	ctx := context.Background()
	cfg := &config.Config{
		Governance: config.GovernanceConfig{},
	}
	workspaceID := "test-workspace-" + t.Name()

	component, err := init.Initialize(ctx, cfg, workspaceID)
	if err != nil {
		t.Errorf("Initialize() failed: %v", err)
	}
	if component == nil {
		t.Error("Initialize() returned nil component")
	}
}
