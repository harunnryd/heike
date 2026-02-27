package executor

import (
	"context"
	"encoding/json"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/executor/runtimes"
	"github.com/harunnryd/heike/internal/sandbox"
	"github.com/harunnryd/heike/internal/tool"
)

type ToolExecutor interface {
	Execute(ctx context.Context, ct *tool.CustomTool, input json.RawMessage) (json.RawMessage, error)
	Validate(ct *tool.CustomTool) error
	GetRuntimeType() tool.ToolType
}

type RuntimeBasedExecutor struct {
	runtimes    *runtimes.RuntimeRegistry
	sandbox     sandbox.SandboxManager
	workspaceID string
}

func NewRuntimeBasedExecutor(runtimes *runtimes.RuntimeRegistry) *RuntimeBasedExecutor {
	return &RuntimeBasedExecutor{
		runtimes: runtimes,
	}
}

func (rbe *RuntimeBasedExecutor) SetSandbox(sb sandbox.SandboxManager) {
	rbe.sandbox = sb
}

func (rbe *RuntimeBasedExecutor) SetWorkspaceID(workspaceID string) {
	rbe.workspaceID = workspaceID
}

func (rbe *RuntimeBasedExecutor) Execute(ctx context.Context, ct *tool.CustomTool, input json.RawMessage) (json.RawMessage, error) {
	runtime, err := rbe.runtimes.Get(ct.Language)
	if err != nil {
		return nil, err
	}

	if err := runtime.ValidateDependencies(ct); err != nil {
		return nil, err
	}

	var sb *sandbox.Sandbox
	if rbe.sandbox != nil {
		workspaceID := rbe.workspaceID
		if workspaceID == "" {
			workspaceID = config.DefaultWorkspaceID
		}

		sb, err = rbe.sandbox.Setup(workspaceID)
		if err != nil {
			return nil, err
		}
		defer rbe.sandbox.Teardown(workspaceID)
	}

	return runtime.ExecuteScript(ctx, ct.ScriptPath, input, sb)
}

func (rbe *RuntimeBasedExecutor) Validate(ct *tool.CustomTool) error {
	runtime, err := rbe.runtimes.Get(ct.Language)
	if err != nil {
		return err
	}

	return runtime.ValidateDependencies(ct)
}

func (rbe *RuntimeBasedExecutor) GetRuntimeType() tool.ToolType {
	return tool.ToolType("runtime-based")
}
