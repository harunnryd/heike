package daemon

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/harunnryd/heike/internal/config"
)

type mockComponent struct {
	name         string
	dependencies []string
	initCalled   bool
	startCalled  bool
	stopCalled   bool
	healthCalled bool
	initError    error
	startError   error
	stopError    error
	healthError  error
	healthResult *ComponentHealth
}

func newMockComponent(name string, dependencies []string) *mockComponent {
	return &mockComponent{
		name:         name,
		dependencies: dependencies,
		healthResult: &ComponentHealth{
			Name:    name,
			Healthy: true,
		},
	}
}

func (m *mockComponent) Name() string {
	return m.name
}

func (m *mockComponent) Dependencies() []string {
	return m.dependencies
}

func (m *mockComponent) Init(ctx context.Context) error {
	m.initCalled = true
	return m.initError
}

func (m *mockComponent) Start(ctx context.Context) error {
	m.startCalled = true
	return m.startError
}

func (m *mockComponent) Stop(ctx context.Context) error {
	m.stopCalled = true
	return m.stopError
}

func (m *mockComponent) Health(ctx context.Context) (*ComponentHealth, error) {
	m.healthCalled = true
	return m.healthResult, m.healthError
}

func TestNewDaemon(t *testing.T) {
	tests := []struct {
		name        string
		workspaceID string
		cfg         *config.Config
		wantErr     bool
	}{
		{
			name:        "valid daemon",
			workspaceID: "test-workspace-" + t.Name(),
			cfg:         &config.Config{},
			wantErr:     false,
		},
		{
			name:        "empty workspace ID",
			workspaceID: "",
			cfg:         &config.Config{},
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := NewDaemon(tt.workspaceID, tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDaemon() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if d.workspaceID != tt.workspaceID {
					t.Errorf("workspaceID = %v, want %v", d.workspaceID, tt.workspaceID)
				}
				if len(d.components) != 0 {
					t.Errorf("components = %v, want 0", len(d.components))
				}
			}
		})
	}
}

func TestValidateConfig_ResolvesDefaultWorkspaceRoot(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	workspaceID := fmt.Sprintf("test-%d", time.Now().UnixNano())
	cfg := &config.Config{
		Server: config.ServerConfig{Port: 8080},
	}

	d, err := NewDaemon(workspaceID, cfg)
	if err != nil {
		t.Fatalf("NewDaemon() failed: %v", err)
	}

	if err := d.validateConfig(); err != nil {
		t.Fatalf("validateConfig() failed: %v", err)
	}

	expected := filepath.Join(tmpHome, ".heike", "workspaces", workspaceID)
	if _, err := os.Stat(expected); err != nil {
		t.Fatalf("expected workspace path to exist at %s: %v", expected, err)
	}

	if _, err := os.Stat(workspaceID); err == nil {
		t.Fatalf("unexpected relative workspace path created in current dir: %s", workspaceID)
	}
}

func TestAddComponent(t *testing.T) {
	cfg := &config.Config{}
	d, _ := NewDaemon("test", cfg)

	comp1 := newMockComponent("Comp1", []string{})
	comp2 := newMockComponent("Comp2", []string{"Comp1"})

	d.AddComponent(comp1)
	d.AddComponent(comp2)

	if len(d.components) != 2 {
		t.Errorf("components = %v, want 2", len(d.components))
	}

	if len(d.shutdownOrder) != 2 {
		t.Errorf("shutdownOrder = %v, want 2", len(d.shutdownOrder))
	}

	if d.shutdownOrder[0] != "Comp2" {
		t.Errorf("shutdownOrder[0] = %v, want Comp2", d.shutdownOrder[0])
	}
}

func TestInitializeComponents(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{Port: 8080},
	}
	d, _ := NewDaemon("test", cfg)

	comp1 := newMockComponent("Comp1", []string{})
	comp2 := newMockComponent("Comp2", []string{"Comp1"})

	d.AddComponent(comp1)
	d.AddComponent(comp2)

	ctx := context.Background()
	err := d.initializeComponents(ctx)

	if err != nil {
		t.Errorf("initializeComponents() error = %v", err)
	}

	if !comp1.initCalled {
		t.Error("Comp1.Init() was not called")
	}

	if !comp2.initCalled {
		t.Error("Comp2.Init() was not called")
	}
}

func TestInitializeComponentsCircularDependency(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{Port: 8080},
	}
	d, _ := NewDaemon("test", cfg)

	comp1 := newMockComponent("Comp1", []string{"Comp2"})
	comp2 := newMockComponent("Comp2", []string{"Comp1"})

	d.AddComponent(comp1)
	d.AddComponent(comp2)

	ctx := context.Background()
	err := d.initializeComponents(ctx)

	if err == nil {
		t.Error("Expected error for circular dependency, got nil")
	}
}

func TestInitializeComponentsMissingDependency(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{Port: 8080},
	}
	d, _ := NewDaemon("test", cfg)

	comp := newMockComponent("Comp", []string{"NonExistent"})

	d.AddComponent(comp)

	ctx := context.Background()
	err := d.initializeComponents(ctx)

	if err == nil {
		t.Error("Expected error for missing dependency, got nil")
	}
}

func TestStartComponents(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{Port: 8080},
	}
	d, _ := NewDaemon("test", cfg)

	comp1 := newMockComponent("Comp1", []string{})
	comp2 := newMockComponent("Comp2", []string{})

	d.AddComponent(comp1)
	d.AddComponent(comp2)

	ctx := context.Background()
	err := d.startComponents(ctx)

	if err != nil {
		t.Errorf("startComponents() error = %v", err)
	}

	if !comp1.startCalled {
		t.Error("Comp1.Start() was not called")
	}

	if !comp2.startCalled {
		t.Error("Comp2.Start() was not called")
	}
}

func TestShutdownComponents(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{Port: 8080},
	}
	d, _ := NewDaemon("test", cfg)

	comp1 := newMockComponent("Comp1", []string{})
	comp2 := newMockComponent("Comp2", []string{})

	d.AddComponent(comp1)
	d.AddComponent(comp2)

	ctx := context.Background()
	err := d.shutdownComponents(ctx)

	if err != nil {
		t.Errorf("shutdownComponents() error = %v", err)
	}

	if !comp1.stopCalled {
		t.Error("Comp1.Stop() was not called")
	}

	if !comp2.stopCalled {
		t.Error("Comp2.Stop() was not called")
	}
}

func TestComponentHealth(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{Port: 8080},
	}
	d, _ := NewDaemon("test", cfg)

	comp1 := newMockComponent("Comp1", []string{})
	comp1.healthResult.Healthy = true

	comp2 := newMockComponent("Comp2", []string{})
	comp2.healthResult.Healthy = false
	comp2.healthResult.Error = fmt.Errorf("mock error")

	d.AddComponent(comp1)
	d.AddComponent(comp2)

	healths := d.ComponentHealth()

	if len(healths) != 2 {
		t.Errorf("ComponentHealth() returned %v healths, want 2", len(healths))
	}

	if healths["Comp1"].Healthy != true {
		t.Error("Comp1 should be healthy")
	}

	if healths["Comp2"].Healthy != false {
		t.Error("Comp2 should be unhealthy")
	}

	if healths["Comp2"].Error == nil {
		t.Error("Comp2.Error should not be nil")
	}
}

func TestRollback(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{Port: 8080},
	}
	d, _ := NewDaemon("test", cfg)

	comp1 := newMockComponent("Comp1", []string{})
	comp2 := newMockComponent("Comp2", []string{})

	d.AddComponent(comp1)
	d.AddComponent(comp2)

	ctx := context.Background()
	d.rollback(ctx)

	if !comp1.stopCalled {
		t.Error("Comp1.Stop() was not called during rollback")
	}

	if !comp2.stopCalled {
		t.Error("Comp2.Stop() was not called during rollback")
	}

	if d.Health() != StatusStopped {
		t.Errorf("Health = %v, want StatusStopped", d.Health())
	}
}

func TestGetComponentByName(t *testing.T) {
	cfg := &config.Config{}
	d, _ := NewDaemon("test", cfg)

	comp1 := newMockComponent("Comp1", []string{})
	comp2 := newMockComponent("Comp2", []string{})

	d.AddComponent(comp1)
	d.AddComponent(comp2)

	tests := []struct {
		name       string
		searchName string
		wantNil    bool
	}{
		{
			name:       "existing component",
			searchName: "Comp1",
			wantNil:    false,
		},
		{
			name:       "non-existing component",
			searchName: "NonExistent",
			wantNil:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comp := d.getComponentByName(tt.searchName)
			if (comp == nil) != tt.wantNil {
				t.Errorf("getComponentByName() = %v, wantNil %v", comp, tt.wantNil)
			}
		})
	}
}
