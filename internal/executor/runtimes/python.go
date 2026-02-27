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

type PythonRuntime struct {
	pythonPath string
}

func NewPythonRuntime() (*PythonRuntime, error) {
	path, err := exec.LookPath("python3")
	if err != nil {
		path, err = exec.LookPath("python")
		if err != nil {
			return nil, fmt.Errorf("python not found: %w", err)
		}
	}

	return &PythonRuntime{
		pythonPath: path,
	}, nil
}

func (pr *PythonRuntime) ExecuteScript(ctx context.Context, scriptPath string, input json.RawMessage, sb *sandbox.Sandbox) (json.RawMessage, error) {
	inputStr := string(input)
	if inputStr == "" {
		inputStr = "{}"
	}

	var args []string
	if sb != nil {
		args = []string{scriptPath, "--input", inputStr}
	} else {
		args = []string{scriptPath, "--input", inputStr}
	}

	cmd := exec.CommandContext(ctx, pr.pythonPath, args...)
	cmd.Dir = filepath.Dir(scriptPath)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("python script execution failed: %w, stderr: %s", err, stderr.String())
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

func (pr *PythonRuntime) ValidateDependencies(ct *tool.CustomTool) error {
	reqFile := filepath.Join(filepath.Dir(ct.ScriptPath), "requirements.txt")
	if _, err := os.Stat(reqFile); os.IsNotExist(err) {
		return nil
	}

	if _, err := exec.LookPath("pip"); err != nil {
		return fmt.Errorf("pip not found: %w", err)
	}
	return nil
}

func (pr *PythonRuntime) InstallDependencies(ct *tool.CustomTool) error {
	reqFile := filepath.Join(filepath.Dir(ct.ScriptPath), "requirements.txt")
	if _, err := os.Stat(reqFile); os.IsNotExist(err) {
		return nil
	}

	cmd := exec.Command("pip", "install", "-r", reqFile)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install python dependencies: %w", err)
	}

	return nil
}

func (pr *PythonRuntime) GetVersion() (string, error) {
	cmd := exec.Command(pr.pythonPath, "--version")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

func (pr *PythonRuntime) GetType() tool.ToolType {
	return tool.ToolTypePython
}
