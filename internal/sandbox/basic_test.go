package sandbox

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestBasicSandboxExecuteInSandbox(t *testing.T) {
	manager, err := NewBasicSandboxManager(t.TempDir(), true)
	if err != nil {
		t.Fatalf("NewBasicSandboxManager failed: %v", err)
	}

	workspaceID := "ws-exec"
	sb, err := manager.Setup(workspaceID)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	t.Cleanup(func() {
		_ = manager.Teardown(workspaceID)
	})

	out, err := manager.ExecuteInSandbox(workspaceID, "sh", []string{"-c", "pwd"}, nil)
	if err != nil {
		t.Fatalf("ExecuteInSandbox failed: %v", err)
	}

	got := strings.TrimSpace(out)
	want := sb.RootPath
	gotResolved, err := filepath.EvalSymlinks(got)
	if err == nil {
		got = gotResolved
	}
	wantResolved, err := filepath.EvalSymlinks(want)
	if err == nil {
		want = wantResolved
	}

	if got != want {
		t.Fatalf("unexpected output, want sandbox root %q, got %q", want, got)
	}
}

func TestBasicSandboxExecuteInSandboxBlocksTraversal(t *testing.T) {
	manager, err := NewBasicSandboxManager(t.TempDir(), true)
	if err != nil {
		t.Fatalf("NewBasicSandboxManager failed: %v", err)
	}

	workspaceID := "ws-traversal"
	if _, err := manager.Setup(workspaceID); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	t.Cleanup(func() {
		_ = manager.Teardown(workspaceID)
	})

	_, err = manager.ExecuteInSandbox(workspaceID, "cat", []string{"../etc/passwd"}, nil)
	if err == nil {
		t.Fatal("expected traversal error, got nil")
	}
	if !strings.Contains(err.Error(), "path traversal detected") {
		t.Fatalf("unexpected error: %v", err)
	}
}
