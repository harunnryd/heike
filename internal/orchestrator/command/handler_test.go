package command

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/harunnryd/heike/internal/cognitive"
	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/policy"
	"github.com/harunnryd/heike/internal/store"
)

type stubSessionManager struct {
	lastRole    string
	lastContent string
}

type stubCommandOutput struct {
	lastSessionID string
	lastContent   string
	sendCalls     int
}

func (s *stubCommandOutput) Send(ctx context.Context, sessionID string, content string) error {
	s.lastSessionID = sessionID
	s.lastContent = content
	s.sendCalls++
	return nil
}

func (s *stubSessionManager) GetContext(ctx context.Context, sessionID string) (*cognitive.CognitiveContext, error) {
	return &cognitive.CognitiveContext{SessionID: sessionID}, nil
}

func (s *stubSessionManager) AppendInteraction(ctx context.Context, sessionID string, role, content string) error {
	s.lastRole = role
	s.lastContent = content
	return nil
}

func (s *stubSessionManager) PersistTool(ctx context.Context, sessionID, toolCallID, content string) error {
	return nil
}

func setupWorker(t *testing.T) *store.Worker {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	worker, err := store.NewWorker("test", "", store.RuntimeConfig{})
	if err != nil {
		t.Fatalf("create store worker: %v", err)
	}
	worker.Start()
	return worker
}

func TestHandler_HelpCommand(t *testing.T) {
	worker := setupWorker(t)
	defer worker.Stop()

	session := &stubSessionManager{}
	output := &stubCommandOutput{}
	handler := NewHandler(nil, session, worker, output)

	if err := handler.Execute(context.Background(), "session-1", "/help"); err != nil {
		t.Fatalf("execute help: %v", err)
	}
	if session.lastRole != "system" {
		t.Fatalf("expected system role, got %s", session.lastRole)
	}
	if session.lastContent == "" {
		t.Fatal("expected help content")
	}
	if output.sendCalls != 1 {
		t.Fatalf("expected output send to be called once, got %d", output.sendCalls)
	}
	if output.lastSessionID != "session-1" {
		t.Fatalf("unexpected output session id: %s", output.lastSessionID)
	}
	if !strings.HasPrefix(output.lastContent, commandOutputPrefix) {
		t.Fatalf("expected command output prefix, got %q", output.lastContent)
	}
	if strings.TrimPrefix(output.lastContent, commandOutputPrefix) != session.lastContent {
		t.Fatalf("output content mismatch: got %q want %q", output.lastContent, session.lastContent)
	}
}

func TestHandler_ModelCommand(t *testing.T) {
	worker := setupWorker(t)
	defer worker.Stop()

	session := &stubSessionManager{}
	handler := NewHandler(nil, session, worker, &stubCommandOutput{})

	sessionID := "session-model"
	if err := worker.SaveSession(&store.SessionMeta{ID: sessionID, Title: "test", Status: "active"}); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	if err := handler.Execute(context.Background(), sessionID, "/model gpt-4o-mini"); err != nil {
		t.Fatalf("execute model: %v", err)
	}

	meta, err := worker.GetSession(sessionID)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if meta == nil || meta.Metadata["model"] != "gpt-4o-mini" {
		t.Fatalf("expected model metadata to be set, got %#v", meta)
	}
	if meta.Metadata["source"] != "cli" {
		t.Fatalf("expected source metadata to remain cli, got %q", meta.Metadata["source"])
	}
}

func TestHandler_ClearCommand(t *testing.T) {
	worker := setupWorker(t)
	defer worker.Stop()

	session := &stubSessionManager{}
	handler := NewHandler(nil, session, worker, &stubCommandOutput{})

	sessionID := "session-clear"
	if err := worker.SaveSession(&store.SessionMeta{ID: sessionID, Title: "test", Status: "active"}); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	if err := worker.WriteTranscript(sessionID, []byte(`{"id":"1","type":"user_event"}`)); err != nil {
		t.Fatalf("seed transcript: %v", err)
	}

	if err := handler.Execute(context.Background(), sessionID, "/clear"); err != nil {
		t.Fatalf("execute clear: %v", err)
	}

	meta, err := worker.GetSession(sessionID)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if meta == nil {
		t.Fatal("expected session to be recreated")
	}
	if meta.Metadata["source"] != "cli" {
		t.Fatalf("expected source metadata to remain cli, got %q", meta.Metadata["source"])
	}
}

func TestHandler_ModelCommand_CreatesSessionWithCLISource(t *testing.T) {
	worker := setupWorker(t)
	defer worker.Stop()

	session := &stubSessionManager{}
	handler := NewHandler(nil, session, worker, &stubCommandOutput{})

	sessionID := "session-model-new"
	if err := handler.Execute(context.Background(), sessionID, "/model gpt-5.2-codex"); err != nil {
		t.Fatalf("execute model: %v", err)
	}

	meta, err := worker.GetSession(sessionID)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if meta == nil {
		t.Fatal("expected session to exist")
	}
	if meta.Metadata["source"] != "cli" {
		t.Fatalf("expected new session source metadata cli, got %q", meta.Metadata["source"])
	}
}

func TestHandler_ApproveCommand(t *testing.T) {
	worker := setupWorker(t)
	defer worker.Stop()

	pol, err := policy.NewEngine(config.GovernanceConfig{
		RequireApproval: []string{"exec_command"},
	}, "test-workspace-approve", "")
	if err != nil {
		t.Fatalf("create policy engine: %v", err)
	}

	_, approvalID, err := pol.Check("exec_command", json.RawMessage(`{"command":"echo test"}`))
	if err == nil {
		t.Fatal("expected approval required error")
	}
	if approvalID == "" {
		t.Fatal("expected approval id")
	}

	session := &stubSessionManager{}
	handler := NewHandler(pol, session, worker, &stubCommandOutput{})

	if err := handler.Execute(context.Background(), "session-approve", "/approve "+approvalID); err != nil {
		t.Fatalf("execute approve: %v", err)
	}
	if !pol.IsGranted(approvalID) {
		t.Fatalf("expected approval %s to be granted", approvalID)
	}
}

func TestHandler_DenyCommand(t *testing.T) {
	worker := setupWorker(t)
	defer worker.Stop()

	pol, err := policy.NewEngine(config.GovernanceConfig{
		RequireApproval: []string{"exec_command"},
	}, "test-workspace-deny", "")
	if err != nil {
		t.Fatalf("create policy engine: %v", err)
	}

	_, approvalID, err := pol.Check("exec_command", json.RawMessage(`{"command":"echo test"}`))
	if err == nil {
		t.Fatal("expected approval required error")
	}
	if approvalID == "" {
		t.Fatal("expected approval id")
	}

	session := &stubSessionManager{}
	handler := NewHandler(pol, session, worker, &stubCommandOutput{})

	if err := handler.Execute(context.Background(), "session-deny", "/deny "+approvalID); err != nil {
		t.Fatalf("execute deny: %v", err)
	}
	if pol.IsGranted(approvalID) {
		t.Fatalf("expected approval %s to remain ungranted", approvalID)
	}
}

func TestFormatCommandOutput_Idempotent(t *testing.T) {
	raw := "Available commands: /help"
	formatted := formatCommandOutput(raw)
	if formatted != commandOutputPrefix+raw {
		t.Fatalf("unexpected formatted output: %q", formatted)
	}

	reformatted := formatCommandOutput(formatted)
	if reformatted != formatted {
		t.Fatalf("expected idempotent formatting, got %q", reformatted)
	}
}
