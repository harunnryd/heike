package adapter

import (
	"context"
)

// EventHandler is a callback function for handling events from adapters
// This avoids circular dependencies between adapters and ingress
type EventHandler func(ctx context.Context, source string, eventType string, sessionID string, content string, metadata map[string]string) error

// InputAdapter defines the interface for adapters that receive events from external platforms
type InputAdapter interface {
	// Name returns the adapter name (e.g. "slack", "telegram", "cli").
	Name() string

	// Start begins listening for events (e.g. starts a server or long-poll).
	// Must respect context cancellation.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the adapter.
	Stop(ctx context.Context) error

	// Health checks if the adapter is healthy and connected.
	Health(ctx context.Context) error
}

// OutputAdapter defines the interface for adapters that send responses to external platforms
type OutputAdapter interface {
	// Name returns the adapter name.
	Name() string

	// Send sends a response to the platform.
	// sessionID maps to platform-specific identifier (channel ID, chat ID, etc.).
	Send(ctx context.Context, sessionID string, content string) error

	// Health checks if the adapter is healthy and can send messages.
	Health(ctx context.Context) error
}
