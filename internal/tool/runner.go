package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	heikeErrors "github.com/harunnryd/heike/internal/errors"
	"github.com/harunnryd/heike/internal/logger"
	"github.com/harunnryd/heike/internal/policy"
)

type Runner struct {
	registry *Registry
	policy   *policy.Engine
}

func (r *Runner) GetDescriptors() []ToolDescriptor {
	if r == nil || r.registry == nil {
		return nil
	}
	return r.registry.GetDescriptors()
}

func NewRunner(registry *Registry, policy *policy.Engine) *Runner {
	return &Runner{
		registry: registry,
		policy:   policy,
	}
}

// Execute handles the full lifecycle: Check Policy -> Run Tool -> Return Result
// It accepts an optional approvalID for retrying previously denied requests.
func (r *Runner) Execute(ctx context.Context, toolName string, input json.RawMessage, approvalID string) (json.RawMessage, error) {
	// Find Tool
	t, ok := r.registry.Get(toolName)
	if !ok {
		return nil, heikeErrors.NotFound("tool not found")
	}
	resolvedToolName := NormalizeToolName(t.Name())

	// Input Validation
	if err := ValidateInput(t.Parameters(), input); err != nil {
		slog.Warn("Tool input validation failed", "tool", resolvedToolName, "requested_name", NormalizeToolName(toolName), "error", err)
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Policy Check
	if approvalID != "" {
		// If ID provided, verify it is GRANTED
		if !r.policy.IsGranted(approvalID) {
			return nil, heikeErrors.PermissionDenied("approval not granted")
		}
		// Proceed
	} else {
		// New check
		allowed, id, err := r.policy.Check(resolvedToolName, input)
		if !allowed {
			if id != "" {
				// Return specific error wrapping as ID so caller can parse it
				return nil, fmt.Errorf("%w: %s", heikeErrors.ErrApprovalRequired, id)
			}
			return nil, err // Denied
		}
	}

	// Execution
	start := time.Now()
	traceID := logger.GetTraceID(ctx)
	slog.Info("Executing tool", "tool", resolvedToolName, "requested_name", NormalizeToolName(toolName), "trace_id", traceID)

	result, err := t.Execute(ctx, input)

	duration := time.Since(start)
	if err != nil {
		slog.Error("Tool execution failed", "tool", resolvedToolName, "requested_name", NormalizeToolName(toolName), "error", err, "duration", duration, "trace_id", traceID)
		return nil, fmt.Errorf("tool execution: %w", heikeErrors.ErrTransient)
	}

	slog.Info("Tool execution success", "tool", resolvedToolName, "requested_name", NormalizeToolName(toolName), "duration", duration, "trace_id", traceID)
	return result, nil
}
