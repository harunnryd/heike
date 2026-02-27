package cognitive

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
)

// ToolExecutor executes a single tool
type ToolExecutor interface {
	Execute(ctx context.Context, name string, args json.RawMessage, input string) (json.RawMessage, error)
}

type UnifiedActor struct {
	toolExecutor ToolExecutor
}

func NewActor(te ToolExecutor) *UnifiedActor {
	return &UnifiedActor{
		toolExecutor: te,
	}
}

func (a *UnifiedActor) Execute(ctx context.Context, action *Action) (*ExecutionResult, error) {
	if action.Type == ActionTypeAnswer {
		return &ExecutionResult{Success: true, Output: action.Content}, nil
	}

	if action.Type == ActionTypeToolCall {
		var results []string
		var toolOutputs []ToolOutput

		for _, tc := range action.ToolCalls {
			slog.Info("Executing tool", "tool", tc.Name)
			slog.Debug("Tool input", "tool", tc.Name, "input", tc.Input)

			res, err := a.toolExecutor.Execute(ctx, tc.Name, json.RawMessage(tc.Input), "")
			outputStr := ""
			if err != nil {
				slog.Error("Tool execution failed", "tool", tc.Name, "error", err)
				outputStr = fmt.Sprintf("Tool %s failed: %v", tc.Name, err)
			} else {
				outputStr = string(res)
				slog.Debug("Tool output", "tool", tc.Name, "output_len", len(outputStr))
			}

			results = append(results, fmt.Sprintf("Tool %s output: %s", tc.Name, outputStr))
			toolOutputs = append(toolOutputs, ToolOutput{
				CallID: tc.ID,
				Name:   tc.Name,
				Output: outputStr,
			})
		}

		// Join results
		output := ""
		for _, r := range results {
			output += r + "\n"
		}

		return &ExecutionResult{
			Success:     true,
			Output:      output,
			ToolOutputs: toolOutputs,
		}, nil
	}

	return nil, fmt.Errorf("unknown action type: %s", action.Type)
}
