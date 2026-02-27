package sandbox

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/harunnryd/heike/internal/policy"

	"github.com/oklog/ulid/v2"
)

type BasicSandboxManager struct {
	mu                   sync.RWMutex
	sandboxes            map[string]*Sandbox
	baseDir              string
	enableTraversalCheck bool
}

func NewBasicSandboxManager(baseDir string, enableTraversalCheck bool) (*BasicSandboxManager, error) {
	if baseDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		baseDir = filepath.Join(home, ".heike", "sandboxes")
	}

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create sandbox base directory: %w", err)
	}

	return &BasicSandboxManager{
		sandboxes:            make(map[string]*Sandbox),
		baseDir:              baseDir,
		enableTraversalCheck: enableTraversalCheck,
	}, nil
}

func (bsm *BasicSandboxManager) Setup(workspaceID string) (*Sandbox, error) {
	bsm.mu.Lock()
	defer bsm.mu.Unlock()

	sandboxID := ulid.Make().String()
	sandboxPath := filepath.Join(bsm.baseDir, workspaceID, sandboxID)

	if err := os.MkdirAll(sandboxPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create sandbox directory: %w", err)
	}

	sb := &Sandbox{
		ID:        sandboxID,
		RootPath:  sandboxPath,
		Level:     policy.SandboxBasic,
		State:     SandboxStateReady,
		CreatedAt: time.Now(),
		Resources: &policy.ResourceLimits{},
	}

	bsm.sandboxes[sandboxID] = sb
	slog.Info("Basic sandbox created", "sandbox_id", sandboxID, "path", sandboxPath)

	return sb, nil
}

func (bsm *BasicSandboxManager) Teardown(workspaceID string) error {
	bsm.mu.Lock()
	defer bsm.mu.Unlock()

	for sandboxID, sb := range bsm.sandboxes {
		if strings.HasPrefix(sb.RootPath, filepath.Join(bsm.baseDir, workspaceID)) {
			if err := os.RemoveAll(sb.RootPath); err != nil {
				slog.Error("Failed to remove sandbox directory", "error", err, "path", sb.RootPath)
				return err
			}
			delete(bsm.sandboxes, sandboxID)
			slog.Info("Basic sandbox removed", "sandbox_id", sandboxID)
		}
	}

	return nil
}

func (bsm *BasicSandboxManager) GetSandbox(workspaceID string) (*Sandbox, error) {
	bsm.mu.RLock()
	defer bsm.mu.RUnlock()

	for _, sb := range bsm.sandboxes {
		if strings.HasPrefix(sb.RootPath, filepath.Join(bsm.baseDir, workspaceID)) {
			return sb, nil
		}
	}

	return nil, fmt.Errorf("sandbox not found for workspace: %s", workspaceID)
}

func (bsm *BasicSandboxManager) ExecuteInSandbox(workspaceID string, cmd string, args []string, envVars map[string]string) (string, error) {
	sb, err := bsm.GetSandbox(workspaceID)
	if err != nil {
		return "", err
	}
	if cmd == "" {
		return "", fmt.Errorf("command is required")
	}

	if bsm.enableTraversalCheck {
		for _, arg := range args {
			if bsm.containsPathTraversal(arg) {
				return "", fmt.Errorf("path traversal detected in argument: %s", arg)
			}
		}
	}

	slog.Debug("Executing command in sandbox", "sandbox_id", sb.ID, "cmd", cmd, "args", args)

	execCmd := exec.Command(cmd, args...)
	execCmd.Dir = sb.RootPath
	execCmd.Env = os.Environ()
	for k, v := range envVars {
		execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	if err := execCmd.Run(); err != nil {
		output := strings.TrimSpace(stdout.String() + stderr.String())
		if output == "" {
			return "", fmt.Errorf("execute command in sandbox: %w", err)
		}
		return output, fmt.Errorf("execute command in sandbox: %w", err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

func (bsm *BasicSandboxManager) containsPathTraversal(path string) bool {
	return strings.Contains(path, "..") || strings.HasPrefix(path, "/") || strings.Contains(path, "~")
}
