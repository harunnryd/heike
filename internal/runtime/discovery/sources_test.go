package discovery

import (
	"path/filepath"
	"testing"
)

func TestResolveSkillSources_DefaultOrder(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	t.Setenv("HOME", home)

	got, err := ResolveSkillSources(ResolveOptions{
		Order:             []string{"bundled", "global", "workspace", "project"},
		WorkspaceID:       "default",
		WorkspaceRootPath: "",
		WorkspacePath:     project,
	})
	if err != nil {
		t.Fatalf("resolve skill sources: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("source count = %d, want 4", len(got))
	}

	want := []string{
		filepath.Join(project, "skills"),
		filepath.Join(home, ".heike", "skills"),
		filepath.Join(home, ".heike", "workspaces", "default", "skills"),
		filepath.Join(project, ".heike", "skills"),
	}
	for idx := range want {
		if got[idx].Path != want[idx] {
			t.Fatalf("source[%d] path = %q, want %q", idx, got[idx].Path, want[idx])
		}
	}
}

func TestResolveToolSources_UnknownKindFails(t *testing.T) {
	_, err := ResolveToolSources(ResolveOptions{
		Order:         []string{"bundled", "invalid_kind"},
		WorkspacePath: t.TempDir(),
	})
	if err == nil {
		t.Fatal("expected unknown source kind error")
	}
}

func TestResolveSkillSources_EmptyOrderFails(t *testing.T) {
	_, err := ResolveSkillSources(ResolveOptions{
		Order: []string{},
	})
	if err == nil {
		t.Fatal("expected empty order error")
	}
}
