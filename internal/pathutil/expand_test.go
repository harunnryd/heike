package pathutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpand_HomeShortcut(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("user home dir: %v", err)
	}

	got, err := Expand("~/.heike/workspaces")
	if err != nil {
		t.Fatalf("expand path: %v", err)
	}

	want := filepath.Join(home, ".heike", "workspaces")
	if got != want {
		t.Fatalf("path mismatch: got %q want %q", got, want)
	}
}

func TestExpand_EnvVar(t *testing.T) {
	t.Setenv("HEIKE_PATH_TEST", "/tmp/heike-path")

	got, err := Expand("$HEIKE_PATH_TEST/workspaces")
	if err != nil {
		t.Fatalf("expand path: %v", err)
	}

	want := filepath.Clean("/tmp/heike-path/workspaces")
	if got != want {
		t.Fatalf("path mismatch: got %q want %q", got, want)
	}
}

func TestExpand_HomeEnvTilde(t *testing.T) {
	t.Setenv("HOME", "~")

	got, err := Expand("~/.heike/workspaces")
	if err != nil {
		t.Fatalf("expand path with HOME=~: %v", err)
	}
	if got == "" {
		t.Fatal("expanded path is empty")
	}
	if got[0] == '~' {
		t.Fatalf("path not expanded: %q", got)
	}
}
