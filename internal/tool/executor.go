package tool

import (
	"context"
	"encoding/json"
)

type ToolExecutor interface {
	Execute(ctx context.Context, tool CustomTool, input json.RawMessage) (json.RawMessage, error)
	Validate(tool CustomTool) error
	GetRuntimeType() ToolType
}
