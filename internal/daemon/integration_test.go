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

	"github.com/harunnryd/heike/cmd/heike/runtime"
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

func newTestConfig(port int) *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Port: port,
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
}

func newRuntimeDaemon(t *testing.T, workspaceID string, cfg *config.Config) (*daemon.Daemon, *runtime.DaemonRuntimeComponent) {
	t.Helper()

	d, err := daemon.NewDaemon(workspaceID, cfg)
	if err != nil {
		t.Fatalf("failed to create daemon: %v", err)
	}

	runtimeComp := runtime.NewDaemonRuntimeComponent(workspaceID, cfg, runtime.AdapterBuildOptions{
		IncludeCLI:        false,
		IncludeSystemNull: true,
	})
	d.AddComponent(runtimeComp)
	d.AddComponent(components.NewHTTPServerComponent(d, &cfg.Server))

	return d, runtimeComp
}

func waitForStatus(t *testing.T, d *daemon.Daemon, want daemon.HealthStatus, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if d.Health() == want {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("daemon status = %v, want %v", d.Health(), want)
}

func assertCancellationError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("daemon start should return an error when context is cancelled")
	}
	if !strings.Contains(err.Error(), "context canceled") && !strings.Contains(err.Error(), "shutdown cancelled") {
		t.Fatalf("unexpected daemon shutdown error: %v", err)
	}
}

func TestDaemonFullLifecycle(t *testing.T) {
	_, cleanup := setupTestWorkspace(t)
	defer cleanup()

	workspaceID := fmt.Sprintf("test-%d", time.Now().UnixNano())
	cfg := newTestConfig(18081)

	d, runtimeComp := newRuntimeDaemon(t, workspaceID, cfg)
	defer func() {
		_ = runtimeComp.Stop(context.Background())
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	startDone := make(chan error, 1)
	go func() {
		startDone <- d.Start(ctx)
	}()

	waitForStatus(t, d, daemon.StatusRunning, 2*time.Second)

	healths := d.ComponentHealth()
	if len(healths) != 2 {
		t.Fatalf("expected 2 components, got %d", len(healths))
	}
	if _, ok := healths["Runtime"]; !ok {
		t.Fatal("runtime health is missing")
	}
	if _, ok := healths["HTTPServer"]; !ok {
		t.Fatal("http health is missing")
	}

	healthResp, err := http.Get("http://127.0.0.1:18081/health")
	if err != nil {
		t.Fatalf("failed to call health endpoint: %v", err)
	}
	defer healthResp.Body.Close()

	if healthResp.StatusCode != http.StatusOK {
		t.Fatalf("health endpoint status = %d, want %d", healthResp.StatusCode, http.StatusOK)
	}

	body, err := io.ReadAll(healthResp.Body)
	if err != nil {
		t.Fatalf("failed to read health response body: %v", err)
	}
	if len(body) == 0 {
		t.Fatal("health endpoint returned empty body")
	}

	cancel()

	select {
	case err := <-startDone:
		assertCancellationError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("daemon did not shut down in time")
	}

	waitForStatus(t, d, daemon.StatusStopped, 2*time.Second)
}

func TestDaemonHealthEndpoint(t *testing.T) {
	_, cleanup := setupTestWorkspace(t)
	defer cleanup()

	workspaceID := fmt.Sprintf("test-%d", time.Now().UnixNano())
	cfg := newTestConfig(18082)

	d, runtimeComp := newRuntimeDaemon(t, workspaceID, cfg)
	defer func() {
		_ = runtimeComp.Stop(context.Background())
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	startDone := make(chan error, 1)
	go func() {
		startDone <- d.Start(ctx)
	}()

	waitForStatus(t, d, daemon.StatusRunning, 2*time.Second)

	tests := []struct {
		name           string
		method         string
		url            string
		body           string
		expectedStatus int
	}{
		{
			name:           "GET health endpoint",
			method:         http.MethodGet,
			url:            "http://127.0.0.1:18082/health",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST health endpoint",
			method:         http.MethodPost,
			url:            "http://127.0.0.1:18082/health",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "POST event endpoint",
			method:         http.MethodPost,
			url:            "http://127.0.0.1:18082/api/v1/events",
			body:           `{"source":"cli","type":"user_message","content":"hello integration test"}`,
			expectedStatus: http.StatusAccepted,
		},
		{
			name:           "GET sessions endpoint",
			method:         http.MethodGet,
			url:            "http://127.0.0.1:18082/api/v1/sessions",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "GET approvals endpoint",
			method:         http.MethodGet,
			url:            "http://127.0.0.1:18082/api/v1/approvals",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "GET zanshin endpoint",
			method:         http.MethodGet,
			url:            "http://127.0.0.1:18082/api/v1/zanshin/status",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body io.Reader
			if tt.body != "" {
				body = strings.NewReader(tt.body)
			}
			req, _ := http.NewRequest(tt.method, tt.url, body)
			client := &http.Client{Timeout: 2 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("failed to call health endpoint: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tt.expectedStatus)
			}
		})
	}

	cancel()

	select {
	case err := <-startDone:
		assertCancellationError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("daemon did not shut down in time")
	}
}

func TestDaemonGracefulShutdown(t *testing.T) {
	_, cleanup := setupTestWorkspace(t)
	defer cleanup()

	workspaceID := fmt.Sprintf("test-%d", time.Now().UnixNano())
	cfg := newTestConfig(18083)

	d, runtimeComp := newRuntimeDaemon(t, workspaceID, cfg)
	defer func() {
		_ = runtimeComp.Stop(context.Background())
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	startDone := make(chan error, 1)
	go func() {
		startDone <- d.Start(ctx)
	}()

	waitForStatus(t, d, daemon.StatusRunning, 2*time.Second)

	healths := d.ComponentHealth()
	for name, health := range healths {
		if !health.Healthy {
			t.Fatalf("component %s unhealthy: %v", name, health.Error)
		}
	}

	cancel()

	select {
	case err := <-startDone:
		assertCancellationError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("daemon did not shut down in time")
	}

	waitForStatus(t, d, daemon.StatusStopped, 2*time.Second)
}
