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

type GoRuntime struct {
	goPath string
}

func NewGoRuntime() (*GoRuntime, error) {
	path, err := exec.LookPath("go")
	if err != nil {
		return nil, fmt.Errorf("go not found: %w", err)
	}

	return &GoRuntime{
		goPath: path,
	}, nil
}

func (gr *GoRuntime) ExecuteScript(ctx context.Context, scriptPath string, input json.RawMessage, sb *sandbox.Sandbox) (json.RawMessage, error) {
	inputStr := string(input)
	if inputStr == "" {
		inputStr = "{}"
	}

	args := []string{"run", scriptPath, "--input", inputStr}
	cmd := exec.CommandContext(ctx, gr.goPath, args...)
	cmd.Dir = filepath.Dir(scriptPath)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go script execution failed: %w, stderr: %s", err, stderr.String())
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

func (gr *GoRuntime) ValidateDependencies(ct *tool.CustomTool) error {
	modFile := filepath.Join(filepath.Dir(ct.ScriptPath), "go.mod")
	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("go not found: %w", err)
	}
	if _, err := os.Stat(modFile); os.IsNotExist(err) {
		return nil
	}
	return nil
}

func (gr *GoRuntime) InstallDependencies(ct *tool.CustomTool) error {
	modFile := filepath.Join(filepath.Dir(ct.ScriptPath), "go.mod")
	if _, err := os.Stat(modFile); os.IsNotExist(err) {
		return nil
	}

	cmd := exec.Command("go", "mod", "download")
	cmd.Dir = filepath.Dir(ct.ScriptPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to download go dependencies: %w", err)
	}

	return nil
}

func (gr *GoRuntime) GetVersion() (string, error) {
	cmd := exec.Command(gr.goPath, "version")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

func (gr *GoRuntime) GetType() tool.ToolType {
	return tool.ToolTypeGo
}
