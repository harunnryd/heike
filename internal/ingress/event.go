package ingress

import (
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"
)

type EventType string

const (
	TypeUserMessage EventType = "user_message"
	TypeSystemEvent EventType = "system_event"
	TypeCommand     EventType = "command" // Slash command
	TypeCron        EventType = "cron"    // Cron job execution
)

// Event is the normalized data structure for all inputs.
type Event struct {
	// Identity
	ID     string `json:"id"`     // ULID or External ID
	Source string `json:"source"` // "slack", "telegram", "cli", "cron"

	// Routing
	WorkspaceID string `json:"workspace_id"`
	SessionID   string `json:"session_id"`

	// Classification
	Type EventType `json:"type"`

	// Payload
	Content string `json:"content"` // Text message or JSON payload

	// Context
	Metadata  map[string]string `json:"metadata"` // e.g. "user_id": "U123"
	CreatedAt time.Time         `json:"created_at"`
}

// NewEvent creates a normalized event with a fresh ULID.
func NewEvent(source string, eventType EventType, sessionID, content string, metadata map[string]string) Event {
	return Event{
		ID:        ulid.Make().String(),
		Source:    source,
		Type:      eventType,
		SessionID: sessionID,
		Content:   content,
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}
}

// GenerateIdempotencyKey creates a deterministic key for the event.
func GenerateIdempotencyKey(source, externalID string) string {
	return fmt.Sprintf("%s:%s", source, externalID)
}

// HashKey returns a SHA256 hash of the idempotency key for storage efficiency/safety.
func HashKey(key string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
}
