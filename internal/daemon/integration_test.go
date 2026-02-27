package daemon_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/harunnryd/heike/internal/adapter"
	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/daemon"
	"github.com/harunnryd/heike/internal/daemon/components"
)

func setupTestWorkspace(t *testing.T) (string, func()) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	t.Cleanup(func() {
		if oldHome != "" {
			os.Setenv("HOME", oldHome)
		}
	})
	os.Setenv("HOME", tmpDir)
	return tmpDir, func() {}
}

func newTestAdapterManager(t *testing.T, cfg *config.Config) *adapter.RuntimeManager {
	t.Helper()
	eventHandler := func(ctx context.Context, source string, eventType string, sessionID string, content string, metadata map[string]string) error {
		return nil
	}
	mgr, err := adapter.NewRuntimeManager(cfg.Adapters, eventHandler, adapter.RuntimeAdapterOptions{
		IncludeCLI:        false,
		IncludeSystemNull: true,
	})
	if err != nil {
		t.Fatalf("failed to create adapter manager: %v", err)
	}
	return mgr
}

func TestDaemonFullLifecycle(t *testing.T) {
	_, cleanup := setupTestWorkspace(t)
	defer cleanup()

	workspaceID := fmt.Sprintf("test-%d", time.Now().UnixNano())

	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8081,
		},
		Governance: config.GovernanceConfig{},
		Models: config.ModelsConfig{
			Default: "test-model",
			Registry: []config.ModelRegistry{
				{
					Name:     "test-model",
					Provider: "openai",
					APIKey:   "test-key-for-testing",
				},
			},
		},
	}

	d, err := daemon.NewDaemon(workspaceID, cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	storeComp := components.NewStoreWorkerComponent(workspaceID, cfg.Daemon.WorkspacePath, &cfg.Store)
	d.AddComponent(storeComp)

	policyComp := components.NewPolicyEngineComponent(&cfg.Governance, workspaceID, cfg.Daemon.WorkspacePath)
	d.AddComponent(policyComp)

	adapterMgr := newTestAdapterManager(t, cfg)
	orchComp := components.NewOrchestratorComponent(cfg, storeComp, policyComp, adapterMgr)
	d.AddComponent(orchComp)

	ingressComp := components.NewIngressComponent(storeComp, &cfg.Ingress, &cfg.Governance)
	d.AddComponent(ingressComp)

	workersComp := components.NewWorkersComponent(cfg, ingressComp, orchComp, storeComp)
	d.AddComponent(workersComp)

	schedulerComp := components.NewSchedulerComponent(cfg, ingressComp, workspaceID)
	d.AddComponent(schedulerComp)

	d.AddComponent(components.NewHTTPServerComponent(d, &cfg.Server))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	startDone := make(chan error, 1)
	go func() {
		startDone <- d.Start(ctx)
	}()

	time.Sleep(500 * time.Millisecond)

	if d.Health() != daemon.StatusRunning {
		t.Errorf("Expected StatusRunning, got %v", d.Health())
	}

	healths := d.ComponentHealth()
	if len(healths) != 7 {
		t.Errorf("Expected 7 components, got %d", len(healths))
	}

	healthResp, err := http.Get("http://127.0.0.1:8081/health")
	if err != nil {
		t.Fatalf("Failed to get health endpoint: %v", err)
	}
	defer healthResp.Body.Close()

	if healthResp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", healthResp.StatusCode)
	}

	body, err := io.ReadAll(healthResp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if len(body) == 0 {
		t.Error("Health endpoint returned empty body")
	}

	cancel()

	select {
	case err := <-startDone:
		if err == nil {
			t.Error("Daemon.Start() should have returned error when context cancelled")
		} else if !strings.Contains(err.Error(), "context canceled") && !strings.Contains(err.Error(), "shutdown cancelled") {
			t.Errorf("Daemon.Start() returned unexpected error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Error("Daemon did not shut down within timeout")
	}

	// Wait a bit for shutdown to complete
	time.Sleep(100 * time.Millisecond)

	if d.Health() != daemon.StatusStopped {
		t.Errorf("Expected StatusStopped after shutdown, got %v", d.Health())
	}
}

func TestDaemonComponentInitOrder(t *testing.T) {
	_, cleanup := setupTestWorkspace(t)
	defer cleanup()

	workspaceID := fmt.Sprintf("test-%d", time.Now().UnixNano())

	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8082,
		},
		Governance: config.GovernanceConfig{},
	}

	d, err := daemon.NewDaemon(workspaceID, cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	storeComp := components.NewStoreWorkerComponent(workspaceID, cfg.Daemon.WorkspacePath, &cfg.Store)
	policyComp := components.NewPolicyEngineComponent(&cfg.Governance, workspaceID, cfg.Daemon.WorkspacePath)

	d.AddComponent(storeComp)
	d.AddComponent(policyComp)

	adapterMgr := newTestAdapterManager(t, cfg)
	orchComp := components.NewOrchestratorComponent(cfg, storeComp, policyComp, adapterMgr)
	d.AddComponent(orchComp)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	startDone := make(chan error, 1)
	go func() {
		startDone <- d.Start(ctx)
	}()

	time.Sleep(300 * time.Millisecond)

	if d.Health() != daemon.StatusRunning {
		t.Errorf("Expected StatusRunning, got %v", d.Health())
	}

	cancel()

	select {
	case err := <-startDone:
		if err == nil {
			t.Error("Daemon.Start() should have returned error when context cancelled")
		} else if !strings.Contains(err.Error(), "context canceled") && !strings.Contains(err.Error(), "shutdown cancelled") {
			t.Errorf("Daemon.Start() returned unexpected error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Error("Daemon did not shut down within timeout")
	}
}

func TestDaemonHealthEndpoint(t *testing.T) {
	_, cleanup := setupTestWorkspace(t)
	defer cleanup()

	workspaceID := fmt.Sprintf("test-%d", time.Now().UnixNano())

	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8083,
		},
		Governance: config.GovernanceConfig{},
	}

	d, err := daemon.NewDaemon(workspaceID, cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	storeComp := components.NewStoreWorkerComponent(workspaceID, cfg.Daemon.WorkspacePath, &cfg.Store)
	policyComp := components.NewPolicyEngineComponent(&cfg.Governance, workspaceID, cfg.Daemon.WorkspacePath)
	adapterMgr := newTestAdapterManager(t, cfg)
	orchComp := components.NewOrchestratorComponent(cfg, storeComp, policyComp, adapterMgr)
	ingressComp := components.NewIngressComponent(storeComp, &cfg.Ingress, &cfg.Governance)
	workersComp := components.NewWorkersComponent(cfg, ingressComp, orchComp, storeComp)
	schedulerComp := components.NewSchedulerComponent(cfg, ingressComp, workspaceID)

	d.AddComponent(storeComp)
	d.AddComponent(policyComp)
	d.AddComponent(orchComp)
	d.AddComponent(ingressComp)
	d.AddComponent(workersComp)
	d.AddComponent(schedulerComp)
	d.AddComponent(components.NewHTTPServerComponent(d, &cfg.Server))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		_ = d.Start(ctx)
	}()

	time.Sleep(2 * time.Second)

	tests := []struct {
		name           string
		method         string
		expectedStatus int
	}{
		{
			name:           "GET health endpoint",
			method:         "GET",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST health endpoint (should fail)",
			method:         "POST",
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(tt.method, "http://127.0.0.1:8083/health", nil)
			client := &http.Client{Timeout: 2 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %v, got %v", tt.expectedStatus, resp.StatusCode)
			}
		})
	}

	cancel()
	time.Sleep(500 * time.Millisecond)
}

func TestDaemonGracefulShutdown(t *testing.T) {
	_, cleanup := setupTestWorkspace(t)
	defer cleanup()

	workspaceID := fmt.Sprintf("test-%d", time.Now().UnixNano())

	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8084,
		},
		Governance: config.GovernanceConfig{},
	}

	d, err := daemon.NewDaemon(workspaceID, cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	storeComp := components.NewStoreWorkerComponent(workspaceID, cfg.Daemon.WorkspacePath, &cfg.Store)
	policyComp := components.NewPolicyEngineComponent(&cfg.Governance, workspaceID, cfg.Daemon.WorkspacePath)
	adapterMgr := newTestAdapterManager(t, cfg)
	orchComp := components.NewOrchestratorComponent(cfg, storeComp, policyComp, adapterMgr)
	ingressComp := components.NewIngressComponent(storeComp, &cfg.Ingress, &cfg.Governance)
	workersComp := components.NewWorkersComponent(cfg, ingressComp, orchComp, storeComp)
	schedulerComp := components.NewSchedulerComponent(cfg, ingressComp, workspaceID)

	d.AddComponent(storeComp)
	d.AddComponent(policyComp)
	d.AddComponent(orchComp)
	d.AddComponent(ingressComp)
	d.AddComponent(workersComp)
	d.AddComponent(schedulerComp)
	d.AddComponent(components.NewHTTPServerComponent(d, &cfg.Server))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		_ = d.Start(ctx)
	}()

	time.Sleep(500 * time.Millisecond)

	if d.Health() != daemon.StatusRunning {
		t.Errorf("Expected StatusRunning, got %v", d.Health())
	}

	healths := d.ComponentHealth()
	for name, health := range healths {
		if !health.Healthy {
			t.Logf("Component %s is unhealthy: %v", name, health.Error)
		}
	}

	cancel()

	shutdownStart := time.Now()
	timeout := time.After(10 * time.Second)

	select {
	case <-timeout:
		t.Error("Daemon did not shut down within 10 seconds")
	case <-time.After(100 * time.Millisecond):
	}

	shutdownDuration := time.Since(shutdownStart)
	t.Logf("Graceful shutdown took %v", shutdownDuration)

	if d.Health() != daemon.StatusStopped {
		t.Errorf("Expected StatusStopped after shutdown, got %v", d.Health())
	}
}
