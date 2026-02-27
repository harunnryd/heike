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

type RubyRuntime struct {
	rubyPath string
}

func NewRubyRuntime() (*RubyRuntime, error) {
	path, err := exec.LookPath("ruby")
	if err != nil {
		return nil, fmt.Errorf("ruby not found: %w", err)
	}

	return &RubyRuntime{
		rubyPath: path,
	}, nil
}

func (rr *RubyRuntime) ExecuteScript(ctx context.Context, scriptPath string, input json.RawMessage, sb *sandbox.Sandbox) (json.RawMessage, error) {
	inputStr := string(input)
	if inputStr == "" {
		inputStr = "{}"
	}

	args := []string{scriptPath, "--input", inputStr}
	cmd := exec.CommandContext(ctx, rr.rubyPath, args...)
	cmd.Dir = filepath.Dir(scriptPath)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ruby script execution failed: %w, stderr: %s", err, stderr.String())
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

func (rr *RubyRuntime) ValidateDependencies(ct *tool.CustomTool) error {
	gemFile := filepath.Join(filepath.Dir(ct.ScriptPath), "Gemfile")
	if _, err := exec.LookPath("gem"); err != nil {
		return fmt.Errorf("gem not found: %w", err)
	}
	if _, err := os.Stat(gemFile); os.IsNotExist(err) {
		return nil
	}
	return nil
}

func (rr *RubyRuntime) InstallDependencies(ct *tool.CustomTool) error {
	gemFile := filepath.Join(filepath.Dir(ct.ScriptPath), "Gemfile")
	if _, err := os.Stat(gemFile); os.IsNotExist(err) {
		return nil
	}

	cmd := exec.Command("bundle", "install")
	cmd.Dir = filepath.Dir(ct.ScriptPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install ruby dependencies: %w", err)
	}

	return nil
}

func (rr *RubyRuntime) GetVersion() (string, error) {
	cmd := exec.Command(rr.rubyPath, "--version")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

func (rr *RubyRuntime) GetType() tool.ToolType {
	return tool.ToolTypeRuby
}
