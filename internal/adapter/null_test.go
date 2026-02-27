package adapter

import (
	"context"
	"testing"
)

func TestNewNullAdapter_DefaultName(t *testing.T) {
	adapter := NewNullAdapter("")
	if adapter.Name() != "null" {
		t.Fatalf("expected default name 'null', got %q", adapter.Name())
	}
}

func TestNullAdapter_SendAndHealth(t *testing.T) {
	adapter := NewNullAdapter("scheduler")
	if err := adapter.Send(context.Background(), "session-1", "content"); err != nil {
		t.Fatalf("Send() returned error: %v", err)
	}
	if err := adapter.Health(context.Background()); err != nil {
		t.Fatalf("Health() returned error: %v", err)
	}
}
