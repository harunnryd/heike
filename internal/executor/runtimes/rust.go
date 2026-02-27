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

type RustRuntime struct {
	rustcPath string
	cargoPath string
}

func NewRustRuntime() (*RustRuntime, error) {
	rustcPath, err := exec.LookPath("rustc")
	if err != nil {
		return nil, fmt.Errorf("rustc not found: %w", err)
	}

	cargoPath, err := exec.LookPath("cargo")
	if err != nil {
		return nil, fmt.Errorf("cargo not found: %w", err)
	}

	return &RustRuntime{
		rustcPath: rustcPath,
		cargoPath: cargoPath,
	}, nil
}

func (rr *RustRuntime) ExecuteScript(ctx context.Context, scriptPath string, input json.RawMessage, sb *sandbox.Sandbox) (json.RawMessage, error) {
	inputStr := string(input)
	if inputStr == "" {
		inputStr = "{}"
	}

	dir := filepath.Dir(scriptPath)
	tempFile := filepath.Join(dir, "main.rs")
	if err := rr.compileScript(scriptPath, tempFile); err != nil {
		return nil, err
	}

	args := []string{"run", "--", "--input", inputStr}
	cmd := exec.CommandContext(ctx, rr.cargoPath, args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("rust script execution failed: %w, stderr: %s", err, stderr.String())
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

func (rr *RustRuntime) ValidateDependencies(ct *tool.CustomTool) error {
	tomlFile := filepath.Join(filepath.Dir(ct.ScriptPath), "Cargo.toml")
	if _, err := exec.LookPath("cargo"); err != nil {
		return fmt.Errorf("cargo not found: %w", err)
	}
	if _, err := os.Stat(tomlFile); os.IsNotExist(err) {
		return nil
	}
	return nil
}

func (rr *RustRuntime) InstallDependencies(ct *tool.CustomTool) error {
	tomlFile := filepath.Join(filepath.Dir(ct.ScriptPath), "Cargo.toml")
	if _, err := os.Stat(tomlFile); os.IsNotExist(err) {
		return nil
	}

	cmd := exec.Command("cargo", "fetch")
	cmd.Dir = filepath.Dir(ct.ScriptPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to fetch rust dependencies: %w", err)
	}

	return nil
}

func (rr *RustRuntime) GetVersion() (string, error) {
	cmd := exec.Command(rr.rustcPath, "--version")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

func (rr *RustRuntime) GetType() tool.ToolType {
	return tool.ToolTypeRust
}

func (rr *RustRuntime) compileScript(scriptPath, tempFile string) error {
	return nil
}
