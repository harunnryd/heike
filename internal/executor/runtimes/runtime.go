package runtimes

import (
	"context"
	"encoding/json"

	"github.com/harunnryd/heike/internal/sandbox"
	"github.com/harunnryd/heike/internal/tool"
)

type LanguageRuntime interface {
	ExecuteScript(ctx context.Context, scriptPath string, input json.RawMessage, sb *sandbox.Sandbox) (json.RawMessage, error)
	ValidateDependencies(ct *tool.CustomTool) error
	InstallDependencies(ct *tool.CustomTool) error
	GetVersion() (string, error)
	GetType() tool.ToolType
}
