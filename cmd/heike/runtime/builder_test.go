package runtime

import (
	"context"
	"testing"

	"github.com/harunnryd/heike/internal/config"
)

func TestNewRuntimeBuilder(t *testing.T) {
	builder := NewRuntimeBuilder()
	if builder == nil {
		t.Error("NewRuntimeBuilder() returned nil")
	}
}

func TestBuilder_WithMethods(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{}
	workspaceID := "test-workspace-" + t.Name()
	adapterOpts := AdapterBuildOptions{IncludeCLI: false, IncludeSystemNull: true}

	builder := NewRuntimeBuilder().
		WithContext(ctx).
		WithConfig(cfg).
		WithWorkspace(workspaceID).
		WithAdapterOptions(adapterOpts)

	impl, ok := builder.(*DefaultRuntimeBuilder)
	if !ok {
		t.Error("Builder is not DefaultRuntimeBuilder")
	}

	if impl.ctx != ctx {
		t.Error("WithContext did not set context")
	}
	if impl.cfg != cfg {
		t.Error("WithConfig did not set config")
	}
	if impl.workspaceID != workspaceID {
		t.Error("WithWorkspace did not set workspaceID")
	}
	if impl.adapterOpts != adapterOpts {
		t.Error("WithAdapterOptions did not set adapter options")
	}
}

func TestNewRuntimeBuilder_DefaultAdapterOptions(t *testing.T) {
	builder := NewRuntimeBuilder()
	impl, ok := builder.(*DefaultRuntimeBuilder)
	if !ok {
		t.Fatal("Builder is not DefaultRuntimeBuilder")
	}

	if !impl.adapterOpts.IncludeCLI {
		t.Error("default adapter options should include CLI adapter")
	}
	if !impl.adapterOpts.IncludeSystemNull {
		t.Error("default adapter options should include system null adapters")
	}
}

func TestBuilder_Build_MissingConfig(t *testing.T) {
	builder := NewRuntimeBuilder().
		WithContext(context.Background())

	_, err := builder.Build()
	if err == nil {
		t.Error("Build() should return error when config is missing")
	}
}

func TestBuilder_Build_DefaultWorkspace(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
		},
	}

	builder := NewRuntimeBuilder().
		WithContext(ctx).
		WithConfig(cfg)

	components, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}

	if components.WorkspaceID != DefaultWorkspaceID {
		t.Errorf("WorkspaceID = %v, want %v", components.WorkspaceID, DefaultWorkspaceID)
	}
}
