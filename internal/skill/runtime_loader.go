package skill

import (
	"github.com/harunnryd/heike/internal/runtime/discovery"
)

type RuntimeLoadOptions struct {
	WorkspaceID       string
	WorkspaceRootPath string
	WorkspacePath     string
	ProjectPath       string
	SourceOrder       []string
}

// LoadRuntimeRegistry loads skills from configured runtime sources.
// Missing directories are ignored; parse/load errors are returned as warnings.
func LoadRuntimeRegistry(registry *Registry, opts RuntimeLoadOptions) []error {
	if registry == nil {
		return []error{}
	}
	order := opts.SourceOrder
	if len(order) == 0 {
		order = []string{"bundled", "global", "workspace", "project"}
	}

	sources, err := discovery.ResolveSkillSources(discovery.ResolveOptions{
		Order:             order,
		WorkspaceID:       opts.WorkspaceID,
		WorkspaceRootPath: opts.WorkspaceRootPath,
		WorkspacePath:     opts.WorkspacePath,
		ProjectPath:       opts.ProjectPath,
	})
	if err != nil {
		return []error{err}
	}

	warnings := make([]error, 0)
	for _, source := range sources {
		if source.Path == "" {
			continue
		}
		if err := registry.Load(source.Path); err != nil {
			warnings = append(warnings, err)
		}
	}

	return warnings
}
