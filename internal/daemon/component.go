package daemon

import (
	"context"
	"time"
)

type HealthStatus string

const (
	StatusStarting HealthStatus = "starting"
	StatusRunning  HealthStatus = "running"
	StatusStopping HealthStatus = "stopping"
	StatusStopped  HealthStatus = "stopped"
)

type ComponentHealth struct {
	Name    string
	Healthy bool
	Error   error
}

type Component interface {
	Name() string
	Dependencies() []string
	Init(ctx context.Context) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Health(ctx context.Context) (*ComponentHealth, error)
}

type RuntimeEvent struct {
	Source    string
	Type      string
	SessionID string
	Content   string
	Metadata  map[string]string
}

type RuntimeSession struct {
	ID        string            `json:"id"`
	Title     string            `json:"title,omitempty"`
	Status    string            `json:"status,omitempty"`
	CreatedAt time.Time         `json:"created_at,omitempty"`
	UpdatedAt time.Time         `json:"updated_at,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type RuntimeApproval struct {
	ID        string    `json:"id"`
	Tool      string    `json:"tool"`
	Input     string    `json:"input"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type RuntimeAPI interface {
	SubmitEvent(ctx context.Context, evt RuntimeEvent) (string, error)
	ListSessions(ctx context.Context) ([]RuntimeSession, error)
	ReadTranscript(ctx context.Context, sessionID string, limit int) ([]string, error)
	ListPendingApprovals(ctx context.Context) ([]RuntimeApproval, error)
	ResolveApproval(ctx context.Context, approvalID string, approve bool) error
	ZanshinStatus(ctx context.Context) map[string]interface{}
}
