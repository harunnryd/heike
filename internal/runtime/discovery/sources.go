package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/harunnryd/heike/internal/store"
)

type SourceKind string

const (
	SourceBundled  SourceKind = "bundled"
	SourceGlobal   SourceKind = "global"
	SourceWorkspace SourceKind = "workspace"
	SourceProject  SourceKind = "project"
)

type SourceDescriptor struct {
	Kind SourceKind
	Path string
}

type ResolveOptions struct {
	Order             []string
	WorkspaceID       string
	WorkspaceRootPath string
	WorkspacePath     string
	ProjectPath       string
}

func ResolveSkillSources(opts ResolveOptions) ([]SourceDescriptor, error) {
	return resolveSources(opts)
}

func ResolveToolSources(opts ResolveOptions) ([]SourceDescriptor, error) {
	return resolveSources(opts)
}

func resolveSources(opts ResolveOptions) ([]SourceDescriptor, error) {
	if len(opts.Order) == 0 {
		return nil, fmt.Errorf("runtime discovery order cannot be empty")
	}

	projectRoot := resolveProjectRoot(opts.ProjectPath, opts.WorkspacePath)

	out := make([]SourceDescriptor, 0, len(opts.Order))
	seen := make(map[string]struct{}, len(opts.Order))
	for idx, entry := range opts.Order {
		kind := SourceKind(strings.ToLower(strings.TrimSpace(entry)))
		if kind == "" {
			return nil, fmt.Errorf("runtime discovery order[%d] is empty", idx)
		}

		path, err := sourcePath(kind, opts.WorkspaceID, opts.WorkspaceRootPath, projectRoot)
		if err != nil {
			return nil, err
		}
		if path == "" {
			continue
		}

		clean := filepath.Clean(path)
		if _, exists := seen[clean]; exists {
			continue
		}
		seen[clean] = struct{}{}
		out = append(out, SourceDescriptor{Kind: kind, Path: clean})
	}

	return out, nil
}

func resolveProjectRoot(projectPath, workspacePath string) string {
	projectRoot := strings.TrimSpace(projectPath)
	if projectRoot == "" {
		projectRoot = strings.TrimSpace(workspacePath)
	}
	if projectRoot == "" {
		if wd, err := os.Getwd(); err == nil {
			projectRoot = wd
		}
	}
	if projectRoot == "" {
		return ""
	}
	return filepath.Clean(projectRoot)
}

func sourcePath(kind SourceKind, workspaceID, workspaceRootPath, projectRoot string) (string, error) {
	switch kind {
	case SourceBundled:
		if projectRoot == "" {
			return "", nil
		}
		return filepath.Join(projectRoot, "skills"), nil
	case SourceGlobal:
		return store.GetSkillsDir()
	case SourceWorkspace:
		if strings.TrimSpace(workspaceID) == "" {
			return "", nil
		}
		return store.GetWorkspaceSkillsDir(workspaceID, workspaceRootPath)
	case SourceProject:
		if projectRoot == "" {
			return "", nil
		}
		return filepath.Join(projectRoot, ".heike", "skills"), nil
	default:
		return "", fmt.Errorf("unknown runtime source kind %q (allowed: bundled, global, workspace, project)", kind)
	}
}
