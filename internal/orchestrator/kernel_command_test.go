package orchestrator

import (
	"context"
	"testing"

	"github.com/harunnryd/heike/internal/ingress"
)

type kernelCommandStub struct {
	canHandleReturn bool
	executeCalls    int
	lastSessionID   string
	lastInput       string
}

func (s *kernelCommandStub) CanHandle(input string) bool {
	return s.canHandleReturn
}

func (s *kernelCommandStub) Execute(ctx context.Context, sessionID string, input string) error {
	s.executeCalls++
	s.lastSessionID = sessionID
	s.lastInput = input
	return nil
}

type kernelTaskStub struct {
	calls int
}

func (s *kernelTaskStub) HandleRequest(ctx context.Context, sessionID string, goal string) error {
	s.calls++
	return nil
}

func TestKernelExecute_RoutesTypeCommandToCommandHandler(t *testing.T) {
	cmd := &kernelCommandStub{canHandleReturn: false}
	task := &kernelTaskStub{}
	k := &DefaultKernel{
		command: cmd,
		task:    task,
	}

	evt := &ingress.Event{
		ID:        "evt-cmd",
		Type:      ingress.TypeCommand,
		SessionID: "session-1",
		Content:   "/help",
	}

	if err := k.Execute(context.Background(), evt); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if cmd.executeCalls != 1 {
		t.Fatalf("expected command execute called once, got %d", cmd.executeCalls)
	}
	if cmd.lastSessionID != "session-1" {
		t.Fatalf("unexpected session id: %s", cmd.lastSessionID)
	}
	if cmd.lastInput != "/help" {
		t.Fatalf("unexpected command input: %s", cmd.lastInput)
	}
	if task.calls != 0 {
		t.Fatalf("task handler should not be called, got %d", task.calls)
	}
}

func TestKernelExecute_UserSlashStillRoutesToCommandHandler(t *testing.T) {
	cmd := &kernelCommandStub{canHandleReturn: true}
	task := &kernelTaskStub{}
	k := &DefaultKernel{
		command: cmd,
		task:    task,
	}

	evt := &ingress.Event{
		ID:        "evt-user-cmd",
		Type:      ingress.TypeUserMessage,
		SessionID: "session-2",
		Content:   "/model gpt-5.2-codex",
	}

	if err := k.Execute(context.Background(), evt); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if cmd.executeCalls != 1 {
		t.Fatalf("expected command execute called once, got %d", cmd.executeCalls)
	}
	if task.calls != 0 {
		t.Fatalf("task handler should not be called, got %d", task.calls)
	}
}
