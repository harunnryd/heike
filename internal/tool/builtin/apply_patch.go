package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	toolcore "github.com/harunnryd/heike/internal/tool"
)

const defaultApplyPatchCommand = "apply_patch"

type applyPatchInput struct {
	Patch   string `json:"patch"`
	DryRun  bool   `json:"dry_run"`
	Workdir string `json:"workdir"`
}

type applyPatchRunner func(ctx context.Context, command, workdir, patch string) (string, error)

func init() {
	toolcore.RegisterBuiltin("apply_patch", func(options toolcore.BuiltinOptions) (toolcore.Tool, error) {
		command := strings.TrimSpace(options.ApplyPatchCommand)
		if command == "" {
			command = defaultApplyPatchCommand
		}
		return &ApplyPatchTool{
			Command: command,
			run:     runApplyPatchCommand,
		}, nil
	})
}

// ApplyPatchTool applies patch text to files.
type ApplyPatchTool struct {
	Command string
	run     applyPatchRunner
}

func (t *ApplyPatchTool) Name() string { return "apply_patch" }

func (t *ApplyPatchTool) Description() string {
	return "Apply a patch text payload to files."
}

func (t *ApplyPatchTool) ToolMetadata() toolcore.ToolMetadata {
	return toolcore.ToolMetadata{
		Source: "builtin",
		Capabilities: []string{
			"filesystem.patch",
			"workspace.edit",
		},
		Risk: toolcore.RiskHigh,
	}
}

func (t *ApplyPatchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"patch": map[string]interface{}{
				"type":        "string",
				"description": "Patch text in apply_patch format",
			},
			"workdir": map[string]interface{}{
				"type":        "string",
				"description": "Optional working directory",
			},
			"dry_run": map[string]interface{}{
				"type":        "boolean",
				"description": "Validate only (currently unsupported)",
			},
		},
		"required": []string{"patch"},
	}
}

func (t *ApplyPatchTool) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	args, err := parseApplyPatchInput(input)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(args.Patch) == "" {
		return nil, fmt.Errorf("patch is required")
	}
	if args.DryRun {
		return nil, fmt.Errorf("dry_run is not supported")
	}

	command := strings.TrimSpace(t.Command)
	if command == "" {
		command = defaultApplyPatchCommand
	}

	runner := t.run
	if runner == nil {
		runner = runApplyPatchCommand
	}

	output, err := runner(ctx, command, strings.TrimSpace(args.Workdir), args.Patch)
	if err != nil {
		return nil, err
	}

	return json.Marshal(map[string]interface{}{
		"applied": true,
		"command": command,
		"workdir": strings.TrimSpace(args.Workdir),
		"output":  output,
	})
}

func parseApplyPatchInput(input json.RawMessage) (applyPatchInput, error) {
	var args applyPatchInput
	if len(input) == 0 {
		return args, fmt.Errorf("invalid input: empty payload")
	}

	if err := json.Unmarshal(input, &args); err == nil {
		return args, nil
	}

	var rawPatch string
	if err := json.Unmarshal(input, &rawPatch); err == nil {
		args.Patch = rawPatch
		return args, nil
	}

	return args, fmt.Errorf("invalid input: expected object with patch field")
}

func runApplyPatchCommand(ctx context.Context, command, workdir, patch string) (string, error) {
	if _, err := exec.LookPath(command); err != nil {
		return "", fmt.Errorf("apply_patch command %q not found in PATH", command)
	}

	cmd := exec.CommandContext(ctx, command)
	if strings.TrimSpace(workdir) != "" {
		cmd.Dir = workdir
	}
	cmd.Stdin = strings.NewReader(patch)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stdout.String() + "\n" + stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("apply_patch failed: %s", msg)
	}

	return strings.TrimSpace(stdout.String() + "\n" + stderr.String()), nil
}
