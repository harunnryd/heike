package policy

import (
	"encoding/json"
	"time"
)

type SandboxLevel string

const (
	SandboxBasic     SandboxLevel = "basic"
	SandboxMedium    SandboxLevel = "medium"
	SandboxAdvanced  SandboxLevel = "advanced"
	SandboxContainer SandboxLevel = "container"
)

type Policy struct {
	Version        string
	WorkspaceRules *WorkspacePolicy
	ToolRules      map[string]*ToolPolicy
	SandboxConfig  *SandboxPolicy
	AuditLog       *AuditPolicy
}

type WorkspacePolicy struct {
	AllowedTools   []string
	DeniedTools    []string
	ApprovalRules  map[string]*ApprovalRule
	ResourceLimits *ResourceLimits
}

type ToolPolicy struct {
	Name            string
	AllowedPaths    []string
	DeniedPatterns  []string
	AllowedCommands []string
	AllowedHosts    []string
	RequireApproval bool
	Timeout         time.Duration
}

type ApprovalRule struct {
	ToolPattern   string
	AutoApprove   bool
	RequireReason bool
	Timeout       time.Duration
}

type ResourceLimits struct {
	MaxMemory    int64
	MaxCPU       float64
	MaxDuration  time.Duration
	MaxProcesses int
}

type SandboxPolicy struct {
	DefaultLevel             SandboxLevel
	EnablePathTraversalCheck bool
	EnableResourceLimiting   bool
}

type AuditPolicy struct {
	Enabled        bool
	LogLevel       string
	RedactPatterns []string
}

type AuditEntry struct {
	Timestamp   time.Time
	TraceID     string
	WorkspaceID string
	ToolName    string
	Action      string
	Status      string
	Input       json.RawMessage
	Output      json.RawMessage
	Duration    time.Duration
	Error       string
}

type AuditFilter struct {
	WorkspaceID string
	ToolName    string
	StartTime   time.Time
	EndTime     time.Time
	Status      string
}
