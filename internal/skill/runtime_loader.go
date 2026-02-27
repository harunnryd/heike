package skill

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/harunnryd/heike/internal/store"
)

// LoadRuntimeRegistry loads skills from bundled, global, workspace, and project-local locations.
// Missing directories are ignored; parse/load errors are returned as warnings.
func LoadRuntimeRegistry(registry *Registry, workspaceID string, workspaceRootPath string, workspacePath string) []error {
	if registry == nil {
		return []error{}
	}

	paths := runtimeSkillPaths(workspaceID, workspaceRootPath, workspacePath)
	warnings := make([]error, 0)
	for _, path := range paths {
		if path == "" {
			continue
		}
		if err := registry.Load(path); err != nil {
			warnings = append(warnings, err)
		}
	}

	return warnings
}

func runtimeSkillPaths(workspaceID string, workspaceRootPath string, workspacePath string) []string {
	candidates := make([]string, 0, 4)

	trimmedWorkspace := strings.TrimSpace(workspacePath)
	if trimmedWorkspace == "" {
		if wd, err := os.Getwd(); err == nil {
			trimmedWorkspace = wd
		}
	}

	// Load order sets precedence because later loads overwrite duplicate names:
	// bundled -> global -> workspace -> project-local.
	if trimmedWorkspace != "" {
		candidates = append(candidates, filepath.Join(trimmedWorkspace, "skills"))
	}

	if globalSkillsDir, err := store.GetSkillsDir(); err == nil {
		candidates = append(candidates, globalSkillsDir)
	}

	if workspaceID != "" {
		if workspaceSkillsDir, err := store.GetWorkspaceSkillsDir(workspaceID, workspaceRootPath); err == nil {
			candidates = append(candidates, workspaceSkillsDir)
		}
	}

	if trimmedWorkspace != "" {
		candidates = append(candidates, filepath.Join(trimmedWorkspace, ".heike", "skills"))
	}

	seen := make(map[string]struct{}, len(candidates))
	paths := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		clean := filepath.Clean(strings.TrimSpace(candidate))
		if clean == "" || clean == "." {
			continue
		}
		if _, exists := seen[clean]; exists {
			continue
		}
		seen[clean] = struct{}{}
		paths = append(paths, clean)
	}

	return paths
}
