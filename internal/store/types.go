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

// --- Task State (governance/tasks.json) ---

type TaskStatus string

const (
	TaskIdle   TaskStatus = "idle"
	TaskLeased TaskStatus = "leased"
)

type TaskState struct {
	Name           string     `json:"name"`
	Spec           string     `json:"spec"` // Cron spec "@daily"
	LastRun        time.Time  `json:"last_run"`
	NextRun        time.Time  `json:"next_run"`
	Status         TaskStatus `json:"status"`
	LeaseExpiresAt time.Time  `json:"lease_expires_at,omitempty"`
	LastRunID      string     `json:"last_run_id,omitempty"`
}

type TaskRegistry struct {
	Tasks map[string]*TaskState `json:"tasks"`
}

// --- Approval Store (governance/approvals.json) ---

type ApprovalStatus string

const (
	StatusPending  ApprovalStatus = "pending"
	StatusApproved ApprovalStatus = "approved"
	StatusDenied   ApprovalStatus = "denied"
)

type Approval struct {
	ID        string         `json:"id"`
	SessionID string         `json:"session_id"`
	ToolName  string         `json:"tool_name"`
	Args      map[string]any `json:"args"`
	Status    ApprovalStatus `json:"status"`
	RequestAt time.Time      `json:"request_at"`
	DecidedAt time.Time      `json:"decided_at,omitempty"`
	DecidedBy string         `json:"decided_by,omitempty"` // User ID
}

type ApprovalStore struct {
	Approvals map[string]*Approval `json:"approvals"`
}
