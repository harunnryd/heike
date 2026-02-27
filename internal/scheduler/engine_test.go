package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/ingress"
)

type mockIngressSubmitter struct {
	submitted []*ingress.Event
}

func (m *mockIngressSubmitter) Submit(ctx context.Context, evt *ingress.Event) error {
	m.submitted = append(m.submitted, evt)
	return nil
}

func TestScheduler_NewScheduler(t *testing.T) {
	store, err := NewStore(t.TempDir() + "/scheduler.json")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	cfg := config.SchedulerConfig{}
	submitter := &mockIngressSubmitter{}
	sched, err := NewScheduler(store, submitter, cfg)
	if err != nil {
		t.Fatalf("NewScheduler failed: %v", err)
	}

	if sched == nil {
		t.Error("Scheduler should not be nil")
	}

	if sched.store != store {
		t.Error("Store not set correctly")
	}

	if sched.ingressSubmit != submitter {
		t.Error("IngressSubmitter not set correctly")
	}
}

func TestScheduler_ComponentLifecycle(t *testing.T) {
	store, err := NewStore(t.TempDir() + "/scheduler.json")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	cfg := config.SchedulerConfig{}
	submitter := &mockIngressSubmitter{}
	sched, err := NewScheduler(store, submitter, cfg)
	if err != nil {
		t.Fatalf("NewScheduler failed: %v", err)
	}

	ctx := context.Background()

	if err := sched.Init(ctx); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if sched.ctx == nil {
		t.Error("Context should be set after Init")
	}

	if err := sched.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !sched.IsRunning() {
		t.Error("Scheduler should be running after Start")
	}

	if err := sched.Health(ctx); err != nil {
		t.Errorf("Health check failed: %v", err)
	}

	if err := sched.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	if sched.IsRunning() {
		t.Error("Scheduler should not be running after Stop")
	}
}

func TestScheduler_GracefulShutdown(t *testing.T) {
	store, err := NewStore(t.TempDir() + "/scheduler.json")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	cfg := config.SchedulerConfig{}
	submitter := &mockIngressSubmitter{}
	sched, err := NewScheduler(store, submitter, cfg)
	if err != nil {
		t.Fatalf("NewScheduler failed: %v", err)
	}

	ctx := context.Background()
	sched.Init(ctx)
	sched.Start(ctx)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- sched.Stop(shutdownCtx)
	}()

	select {
	case err := <-errChan:
		if err != nil {
			t.Errorf("Stop failed: %v", err)
		}
	case <-shutdownCtx.Done():
		t.Error("Stop timed out")
	}

	if sched.IsRunning() {
		t.Error("Scheduler should not be running after Stop")
	}
}

func TestScheduler_HealthCheck(t *testing.T) {
	store, err := NewStore(t.TempDir() + "/scheduler.json")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	cfg := config.SchedulerConfig{}
	submitter := &mockIngressSubmitter{}
	sched, err := NewScheduler(store, submitter, cfg)
	if err != nil {
		t.Fatalf("NewScheduler failed: %v", err)
	}

	ctx := context.Background()

	err = sched.Health(ctx)
	if err == nil {
		t.Error("Health should fail when not initialized")
	}

	sched.Init(ctx)
	sched.Start(ctx)

	err = sched.Health(ctx)
	if err != nil {
		t.Errorf("Health should pass after Start: %v", err)
	}

	sched.Stop(ctx)

	err = sched.Health(ctx)
	if err == nil {
		t.Error("Health should fail after Stop")
	}
}

func TestScheduler_IsRunning(t *testing.T) {
	store, err := NewStore(t.TempDir() + "/scheduler.json")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	cfg := config.SchedulerConfig{}
	submitter := &mockIngressSubmitter{}
	sched, err := NewScheduler(store, submitter, cfg)
	if err != nil {
		t.Fatalf("NewScheduler failed: %v", err)
	}

	ctx := context.Background()

	if sched.IsRunning() {
		t.Error("Should not be running initially")
	}

	sched.Init(ctx)
	sched.Start(ctx)

	if !sched.IsRunning() {
		t.Error("Should be running after Start")
	}

	sched.Stop(ctx)

	if sched.IsRunning() {
		t.Error("Should not be running after Stop")
	}
}
