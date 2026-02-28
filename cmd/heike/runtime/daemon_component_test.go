package runtime

import (
	"context"
	"os"
	"testing"

	"github.com/harunnryd/heike/internal/config"
)

func setupDaemonComponentTestEnv(t *testing.T) {
	t.Helper()

	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	t.Cleanup(func() {
		if oldHome != "" {
			os.Setenv("HOME", oldHome)
		}
	})
	os.Setenv("HOME", tmpDir)
}

func TestDaemonRuntimeComponent_InitValidation(t *testing.T) {
	comp := NewDaemonRuntimeComponent("", nil, AdapterBuildOptions{})

	if err := comp.Init(context.Background()); err == nil {
		t.Fatal("expected init to fail when config is missing")
	}

	comp = NewDaemonRuntimeComponent("", &config.Config{}, AdapterBuildOptions{})
	if err := comp.Init(context.Background()); err == nil {
		t.Fatal("expected init to fail when workspace id is missing")
	}
}

func TestDaemonRuntimeComponent_Lifecycle(t *testing.T) {
	setupDaemonComponentTestEnv(t)

	comp := NewDaemonRuntimeComponent("test-daemon-runtime", &config.Config{
		Server: config.ServerConfig{
			Port: 18090,
		},
	}, AdapterBuildOptions{
		IncludeCLI:        false,
		IncludeSystemNull: true,
	})

	if err := comp.Init(context.Background()); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	beforeStart, err := comp.Health(context.Background())
	if err != nil {
		t.Fatalf("health before start returned error: %v", err)
	}
	if beforeStart.Healthy {
		t.Fatal("component should not be healthy before start")
	}

	if err := comp.Start(context.Background()); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	health, err := comp.Health(context.Background())
	if err != nil {
		t.Fatalf("health after start returned error: %v", err)
	}
	if !health.Healthy {
		t.Fatalf("expected healthy component, got error: %v", health.Error)
	}

	if err := comp.Stop(context.Background()); err != nil {
		t.Fatalf("stop failed: %v", err)
	}

	afterStop, err := comp.Health(context.Background())
	if err != nil {
		t.Fatalf("health after stop returned error: %v", err)
	}
	if afterStop.Healthy {
		t.Fatal("component should report unhealthy after stop")
	}
}
