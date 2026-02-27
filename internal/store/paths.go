package store

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/harunnryd/heike/internal/pathutil"
)

// ResolveWorkspaceRootPath resolves configured workspace root path.
// If empty, it falls back to ~/.heike/workspaces.
func ResolveWorkspaceRootPath(workspaceRootPath string) (string, error) {
	if trimmed := strings.TrimSpace(workspaceRootPath); trimmed != "" {
		return pathutil.Expand(trimmed)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".heike", "workspaces"), nil
}

// GetWorkspacePath returns the base path for a workspace.
func GetWorkspacePath(workspaceID string, workspaceRootPath string) (string, error) {
	root, err := ResolveWorkspaceRootPath(workspaceRootPath)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, workspaceID), nil
}

// GetSessionsDir returns the sessions directory for a workspace.
func GetSessionsDir(workspaceID string, workspaceRootPath string) (string, error) {
	base, err := GetWorkspacePath(workspaceID, workspaceRootPath)
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "sessions"), nil
}

// GetLockPath returns the lock file path for a workspace.
func GetLockPath(workspaceID string, workspaceRootPath string) (string, error) {
	base, err := GetWorkspacePath(workspaceID, workspaceRootPath)
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "workspace.lock"), nil
}

// GetSchedulerDir returns the scheduler directory for a workspace.
func GetSchedulerDir(workspaceID string, workspaceRootPath string) (string, error) {
	base, err := GetWorkspacePath(workspaceID, workspaceRootPath)
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "scheduler"), nil
}

// GetSkillsDir returns the global skills directory.
func GetSkillsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".heike", "skills"), nil
}

// GetWorkspaceSkillsDir returns the workspace-specific skills directory.
func GetWorkspaceSkillsDir(workspaceID string, workspaceRootPath string) (string, error) {
	base, err := GetWorkspacePath(workspaceID, workspaceRootPath)
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "skills"), nil
}
