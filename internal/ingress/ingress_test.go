package ingress

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/harunnryd/heike/internal/store"
)

func setupWorker(t *testing.T) *store.Worker {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	worker, err := store.NewWorker("test", "", store.RuntimeConfig{})
	if err != nil {
		t.Fatalf("Failed to create store worker: %v", err)
	}
	worker.Start()
	return worker
}

func TestIngress_New(t *testing.T) {
	worker := setupWorker(t)
	defer worker.Stop()

	ingress := NewIngress(100, 1000, RuntimeConfig{}, worker)
	if ingress == nil {
		t.Fatal("NewIngress returned nil")
	}

	if ingress.interactiveQueue == nil {
		t.Error("Interactive queue not initialized")
	}

	if ingress.backgroundQueue == nil {
		t.Error("Background queue not initialized")
	}

	if cap(ingress.interactiveQueue) != 100 {
		t.Errorf("Interactive queue capacity: got %d, want 100", cap(ingress.interactiveQueue))
	}

	if cap(ingress.backgroundQueue) != 1000 {
		t.Errorf("Background queue capacity: got %d, want 1000", cap(ingress.backgroundQueue))
	}
}

func TestIngress_Metrics(t *testing.T) {
	worker := setupWorker(t)
	defer worker.Stop()

	ingress := NewIngress(100, 1000, RuntimeConfig{}, worker)

	evt := NewEvent("test", TypeUserMessage, "session1", "hello", nil)

	if err := ingress.Submit(context.Background(), &evt); err != nil {
		t.Fatalf("Submit failed: %v", err)
	}
}

func TestIngress_DuplicateDetection(t *testing.T) {
	worker := setupWorker(t)
	defer worker.Stop()

	ingress := NewIngress(100, 1000, RuntimeConfig{}, worker)

	evt := NewEvent("test", TypeUserMessage, "session1", "hello", nil)

	if err := ingress.Submit(context.Background(), &evt); err != nil {
		t.Fatalf("First submit failed: %v", err)
	}

	if err := ingress.Submit(context.Background(), &evt); err == nil {
		t.Error("Second submit should fail with duplicate error")
	}
}

func TestIngress_Close(t *testing.T) {
	worker := setupWorker(t)
	defer worker.Stop()

	ingress := NewIngress(100, 1000, RuntimeConfig{}, worker)

	if err := ingress.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestIngress_Health(t *testing.T) {
	worker := setupWorker(t)
	defer worker.Stop()

	ingress := NewIngress(100, 1000, RuntimeConfig{}, worker)

	if err := ingress.Health(context.Background()); err != nil {
		t.Errorf("Health check failed: %v", err)
	}
}

func TestRouter_RegisterCommand(t *testing.T) {
	worker := setupWorker(t)
	defer worker.Stop()

	router := NewStandardRouter()

	called := false
	router.RegisterCommand("/test", func(ctx context.Context, evt *Event) error {
		called = true
		return nil
	})

	evt := NewEvent("cli", TypeUserMessage, "session1", "/test", nil)
	dest := router.Route(context.Background(), &evt)

	if dest.Type != DestCommand {
		t.Errorf("Route type: got %d, want DestCommand", dest.Type)
	}

	if dest.Handler == nil {
		t.Error("Handler should not be nil")
	}

	if dest.Handler != nil {
		dest.Handler(context.Background(), &evt)
		if !called {
			t.Error("Handler was not called")
		}
	}
}

func TestRouter_SlashCommandsRoutedToPipeline(t *testing.T) {
	worker := setupWorker(t)
	defer worker.Stop()

	router := NewStandardRouter()

	testCommands := []string{"/clear", "/model", "/help", "/exit"}

	for _, cmd := range testCommands {
		evt := NewEvent("cli", TypeUserMessage, "session1", cmd, nil)
		dest := router.Route(context.Background(), &evt)

		if dest.Type != DestPipeline {
			t.Errorf("Command %s: got type %d, want DestPipeline", cmd, dest.Type)
		}
		if evt.Type != TypeCommand {
			t.Errorf("Command %s: got event type %s, want %s", cmd, evt.Type, TypeCommand)
		}
	}
}

func TestRouter_UnknownSlash(t *testing.T) {
	worker := setupWorker(t)
	defer worker.Stop()

	router := NewStandardRouter()

	evt := NewEvent("cli", TypeUserMessage, "session1", "/unknown", nil)
	dest := router.Route(context.Background(), &evt)

	if dest.Type != DestPipeline {
		t.Errorf("Unknown slash: got type %d, want DestPipeline", dest.Type)
	}
	if evt.Type != TypeCommand {
		t.Errorf("Unknown slash should be converted to command, got %s", evt.Type)
	}
}

func TestRouter_Text(t *testing.T) {
	worker := setupWorker(t)
	defer worker.Stop()

	router := NewStandardRouter()

	evt := NewEvent("cli", TypeUserMessage, "session1", "hello world", nil)
	dest := router.Route(context.Background(), &evt)

	if dest.Type != DestPipeline {
		t.Errorf("Text message: got type %d, want DestPipeline", dest.Type)
	}
}

func TestIngress_QueueDrain(t *testing.T) {
	worker := setupWorker(t)
	defer worker.Stop()

	ingress := NewIngress(10, 10, RuntimeConfig{}, worker)

	for i := 0; i < 5; i++ {
		evt := NewEvent("test", TypeUserMessage, "session1", "hello", nil)
		if err := ingress.Submit(context.Background(), &evt); err != nil {
			t.Fatalf("Submit failed: %v", err)
		}
	}

	if len(ingress.interactiveQueue) != 5 {
		t.Errorf("Queue length: got %d, want 5", len(ingress.interactiveQueue))
	}

	if err := ingress.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestIngress_BackgroundQueueDrop(t *testing.T) {
	worker := setupWorker(t)
	defer worker.Stop()

	ingress := NewIngress(10, 5, RuntimeConfig{}, worker)

	for i := 0; i < 6; i++ {
		evt := NewEvent("test", TypeSystemEvent, "session1", "system event", nil)
		ingress.Submit(context.Background(), &evt)
	}
}

func TestResolver_SchedulerSessionStable(t *testing.T) {
	worker := setupWorker(t)
	defer worker.Stop()

	resolver := NewStandardResolver(worker)
	evt1 := NewEvent("scheduler", TypeSystemEvent, "", "heartbeat", map[string]string{"workspace_id": "default"})
	evt2 := NewEvent("scheduler", TypeSystemEvent, "", "heartbeat", map[string]string{"workspace_id": "default"})

	session1, err := resolver.ResolveSession(context.Background(), &evt1)
	if err != nil {
		t.Fatalf("ResolveSession evt1 failed: %v", err)
	}
	session2, err := resolver.ResolveSession(context.Background(), &evt2)
	if err != nil {
		t.Fatalf("ResolveSession evt2 failed: %v", err)
	}

	if session1 != "scheduler:default" {
		t.Fatalf("unexpected scheduler session: %s", session1)
	}
	if session2 != session1 {
		t.Fatalf("scheduler session should be stable: %s vs %s", session1, session2)
	}
}

func TestResolver_UnknownSourceGeneratesSession(t *testing.T) {
	worker := setupWorker(t)
	defer worker.Stop()

	resolver := NewStandardResolver(worker)
	evt := NewEvent("webhook", TypeUserMessage, "", "hello", map[string]string{"source": "webhook"})

	sessionID, err := resolver.ResolveSession(context.Background(), &evt)
	if err != nil {
		t.Fatalf("ResolveSession failed: %v", err)
	}
	if !strings.HasPrefix(sessionID, "sess_") {
		t.Fatalf("unexpected session prefix: %s", sessionID)
	}
}

func TestIngress_SubmitNilEvent(t *testing.T) {
	worker := setupWorker(t)
	defer worker.Stop()

	ing := NewIngress(10, 10, RuntimeConfig{}, worker)
	if err := ing.Submit(context.Background(), nil); err == nil {
		t.Fatal("expected error for nil event")
	}
}
