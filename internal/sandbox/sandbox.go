package sandbox

import (
	"time"

	"github.com/harunnryd/heike/internal/policy"
)

type Sandbox struct {
	ID        string
	RootPath  string
	Level     policy.SandboxLevel
	State     SandboxState
	CreatedAt time.Time
	Resources *policy.ResourceLimits
}

type SandboxState string

const (
	SandboxStateSetup    SandboxState = "setup"
	SandboxStateReady    SandboxState = "ready"
	SandboxStateRunning  SandboxState = "running"
	SandboxStateTeardown SandboxState = "teardown"
	SandboxStateError    SandboxState = "error"
)
