package skill

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRuntimeRegistry_LoadsFromConfiguredSources(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	project := t.TempDir()
	bundledSkills := filepath.Join(project, "skills")
	projectSkills := filepath.Join(project, ".heike", "skills")
	workspaceSkills := filepath.Join(home, ".heike", "workspaces", "default", "skills")
	globalSkills := filepath.Join(home, ".heike", "skills")

	createSkillFile(t, bundledSkills, "bundled_skill", `---
name: bundled_skill
description: Bundled skill
tags: [bundled]
tools: [bundled_tool]
---
Use bundled guidance.`)
	createSkillFile(t, projectSkills, "project_skill", `---
name: project_skill
description: Project skill
tags: [project]
tools: [project_tool]
---
Use project guidance.`)
	createSkillFile(t, workspaceSkills, "workspace_skill", `---
name: workspace_skill
description: Workspace skill
tags: [workspace]
tools: [workspace_tool]
---
Use workspace guidance.`)
	createSkillFile(t, globalSkills, "global_skill", `---
name: global_skill
description: Global skill
tags: [global]
tools: [global_tool]
---
Use global guidance.`)

	registry := NewRegistry()
	warnings := LoadRuntimeRegistry(registry, RuntimeLoadOptions{
		WorkspaceID:       "default",
		WorkspaceRootPath: "",
		WorkspacePath:     project,
		ProjectPath:       "",
		SourceOrder:       []string{"bundled", "global", "workspace", "project"},
	})
	if len(warnings) > 0 {
		t.Fatalf("unexpected load warnings: %v", warnings)
	}

	names, err := registry.List("name")
	if err != nil {
		t.Fatalf("list skills: %v", err)
	}
	if len(names) != 4 {
		t.Fatalf("loaded skills = %d, want 4 (%v)", len(names), names)
	}
}

func TestLoadRuntimeRegistry_Precedence_ProjectOverridesEarlierSources(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	project := t.TempDir()
	bundledSkills := filepath.Join(project, "skills")
	projectSkills := filepath.Join(project, ".heike", "skills")
	workspaceSkills := filepath.Join(home, ".heike", "workspaces", "default", "skills")
	globalSkills := filepath.Join(home, ".heike", "skills")

	createSkillFile(t, bundledSkills, "shared", `---
name: shared
description: bundled
tags: [shared]
tools: [bundled_tool]
---
Bundled guidance.`)
	createSkillFile(t, globalSkills, "shared", `---
name: shared
description: global
tags: [shared]
tools: [global_tool]
---
Global guidance.`)
	createSkillFile(t, workspaceSkills, "shared", `---
name: shared
description: workspace
tags: [shared]
tools: [workspace_tool]
---
Workspace guidance.`)
	createSkillFile(t, projectSkills, "shared", `---
name: shared
description: project-local
tags: [shared]
tools: [project_tool]
---
Project guidance.`)

	registry := NewRegistry()
	warnings := LoadRuntimeRegistry(registry, RuntimeLoadOptions{
		WorkspaceID:       "default",
		WorkspaceRootPath: "",
		WorkspacePath:     project,
		ProjectPath:       "",
		SourceOrder:       []string{"bundled", "global", "workspace", "project"},
	})
	if len(warnings) > 0 {
		t.Fatalf("unexpected load warnings: %v", warnings)
	}

	loaded, err := registry.Get("shared")
	if err != nil {
		t.Fatalf("get shared skill: %v", err)
	}
	if loaded.Description != "project-local" {
		t.Fatalf("description = %q, want %q", loaded.Description, "project-local")
	}
}

func TestLoadRuntimeRegistry_CollectsWarnings(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	project := t.TempDir()
	projectSkills := filepath.Join(project, ".heike", "skills")
	if err := os.MkdirAll(filepath.Join(projectSkills, "broken"), 0755); err != nil {
		t.Fatalf("create broken skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectSkills, "broken", "SKILL.md"), []byte("invalid"), 0644); err != nil {
		t.Fatalf("write broken skill file: %v", err)
	}

	registry := NewRegistry()
	warnings := LoadRuntimeRegistry(registry, RuntimeLoadOptions{
		WorkspaceID:       "default",
		WorkspaceRootPath: "",
		WorkspacePath:     project,
		ProjectPath:       "",
		SourceOrder:       []string{"project"},
	})
	if len(warnings) == 0 {
		t.Fatal("expected warnings for invalid skill")
	}
}
