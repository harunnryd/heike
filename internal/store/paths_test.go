package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveWorkspaceRootPath_ExpandsHomeShortcut(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("user home dir: %v", err)
	}

	got, err := ResolveWorkspaceRootPath("~/.heike/workspaces")
	if err != nil {
		t.Fatalf("resolve workspace root path: %v", err)
	}

	want := filepath.Join(home, ".heike", "workspaces")
	if got != want {
		t.Fatalf("path mismatch: got %q want %q", got, want)
	}
}
