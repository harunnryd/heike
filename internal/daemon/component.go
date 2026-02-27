package daemon

import (
	"context"
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
