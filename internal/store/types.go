package store

import "time"

// --- Session Index (sessions/index.json) ---

type SessionMeta struct {
	ID        string            `json:"id"`
	Title     string            `json:"title"`
	Status    string            `json:"status"` // "active", "archived"
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Metadata  map[string]string `json:"metadata,omitempty"` // e.g. "slack_channel_id": "C123"
}

type SessionIndex struct {
	Sessions map[string]SessionMeta `json:"sessions"`
}

// --- Transcript (sessions/<id>.jsonl) ---

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
	RoleTool      Role = "tool"
)

type TranscriptEntry struct {
	ID         string         `json:"id"` // ULID
	Timestamp  time.Time      `json:"ts"`
	Role       Role           `json:"role"`
	Content    string         `json:"content"`
	Name       string         `json:"name,omitempty"`         // For tools
	ToolCallID string         `json:"tool_call_id,omitempty"` // Link tool result to call
	Metadata   map[string]any `json:"meta,omitempty"`         // Tokens, latency
}

// --- Idempotency Store (governance/processed_keys.json) ---

type ProcessedKeys struct {
	// Key: "source:event_id" -> Value: Timestamp
	Keys map[string]time.Time `json:"keys"`
}
