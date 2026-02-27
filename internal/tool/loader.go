package tool

import (
	"encoding/json"
)

type ToolType string

const (
	ToolTypePython ToolType = "python"
	ToolTypeGo     ToolType = "go"
	ToolTypeJS     ToolType = "javascript"
	ToolTypeShell  ToolType = "shell"
	ToolTypeRuby   ToolType = "ruby"
	ToolTypeRust   ToolType = "rust"
)

type SandboxLevel string

const (
	SandboxBasic     SandboxLevel = "basic"
	SandboxMedium    SandboxLevel = "medium"
	SandboxAdvanced  SandboxLevel = "advanced"
	SandboxContainer SandboxLevel = "container"
)

type CustomTool struct {
	Name         string
	Language     ToolType
	ScriptPath   string
	Description  string
	Parameters   json.RawMessage
	Capabilities []string
	Source       string
	Risk         RiskLevel
	SandboxLevel SandboxLevel
}

type ToolLoader interface {
	LoadFromSkill(skillPath string) ([]*CustomTool, error)
	LoadFromSource(skillsPath string) ([]*CustomTool, error)
}
