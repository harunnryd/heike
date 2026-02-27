package runtimes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/harunnryd/heike/internal/sandbox"
	"github.com/harunnryd/heike/internal/tool"
)

type NodeRuntime struct {
	nodePath string
}

func NewNodeRuntime() (*NodeRuntime, error) {
	path, err := exec.LookPath("node")
	if err != nil {
		return nil, fmt.Errorf("node not found: %w", err)
	}

	return &NodeRuntime{
		nodePath: path,
	}, nil
}

func (nr *NodeRuntime) ExecuteScript(ctx context.Context, scriptPath string, input json.RawMessage, sb *sandbox.Sandbox) (json.RawMessage, error) {
	inputStr := string(input)
	if inputStr == "" {
		inputStr = "{}"
	}

	args := []string{scriptPath, "--input", inputStr}
	cmd := exec.CommandContext(ctx, nr.nodePath, args...)
	cmd.Dir = filepath.Dir(scriptPath)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("node script execution failed: %w, stderr: %s", err, stderr.String())
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

func (nr *NodeRuntime) ValidateDependencies(ct *tool.CustomTool) error {
	pkgFile := filepath.Join(filepath.Dir(ct.ScriptPath), "package.json")
	if _, err := exec.LookPath("npm"); err != nil {
		return fmt.Errorf("npm not found: %w", err)
	}
	if _, err := os.Stat(pkgFile); os.IsNotExist(err) {
		return nil
	}
	return nil
}

func (nr *NodeRuntime) InstallDependencies(ct *tool.CustomTool) error {
	pkgFile := filepath.Join(filepath.Dir(ct.ScriptPath), "package.json")
	if _, err := os.Stat(pkgFile); os.IsNotExist(err) {
		return nil
	}

	cmd := exec.Command("npm", "install")
	cmd.Dir = filepath.Dir(ct.ScriptPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install node dependencies: %w", err)
	}

	return nil
}

func (nr *NodeRuntime) GetVersion() (string, error) {
	cmd := exec.Command(nr.nodePath, "--version")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

func (nr *NodeRuntime) GetType() tool.ToolType {
	return tool.ToolTypeJS
}
