package runtimes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/harunnryd/heike/internal/sandbox"
	"github.com/harunnryd/heike/internal/tool"
)

type ShellRuntime struct {
	shellPath string
}

func NewShellRuntime() (*ShellRuntime, error) {
	path, err := exec.LookPath("bash")
	if err != nil {
		path, err = exec.LookPath("sh")
		if err != nil {
			return nil, fmt.Errorf("shell not found: %w", err)
		}
	}

	return &ShellRuntime{
		shellPath: path,
	}, nil
}

func (sr *ShellRuntime) ExecuteScript(ctx context.Context, scriptPath string, input json.RawMessage, sb *sandbox.Sandbox) (json.RawMessage, error) {
	inputStr := string(input)
	if inputStr == "" {
		inputStr = "{}"
	}

	args := []string{scriptPath}

	cmd := exec.CommandContext(ctx, sr.shellPath, args...)
	cmd.Dir = filepath.Dir(scriptPath)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("shell script execution failed: %w, stderr: %s", err, stderr.String())
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return json.RawMessage("{}"), nil
	}

	var result json.RawMessage
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		return json.RawMessage(fmt.Sprintf(`{"output": %q}`, output)), nil
	}

	return result, nil
}

func (sr *ShellRuntime) ValidateDependencies(ct *tool.CustomTool) error {
	return nil
}

func (sr *ShellRuntime) InstallDependencies(ct *tool.CustomTool) error {
	return nil
}

func (sr *ShellRuntime) GetVersion() (string, error) {
	cmd := exec.Command(sr.shellPath, "--version")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

func (sr *ShellRuntime) GetType() tool.ToolType {
	return tool.ToolTypeShell
}
