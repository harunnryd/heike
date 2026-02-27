package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestSessionLsCmd(t *testing.T) {
	t.Run("without sessions", func(t *testing.T) {
		tmpDir := t.TempDir()
		home := os.Getenv("HOME")
		defer func() {
			if home != "" {
				os.Setenv("HOME", home)
			}
		}()
		os.Setenv("HOME", tmpDir)

		cmd := &cobra.Command{}
		cmd.Flags().StringP("workspace", "w", "", "Target workspace ID")
		_ = cmd.Flags().Set("workspace", "test-workspace-"+t.Name())

		args := []string{}
		if err := sessionLsCmd.RunE(cmd, args); err != nil {
			t.Errorf("Session ls failed: %v", err)
		}
	})

	t.Run("with sessions", func(t *testing.T) {
		tmpDir := t.TempDir()
		home := os.Getenv("HOME")
		defer func() {
			if home != "" {
				os.Setenv("HOME", home)
			}
		}()
		os.Setenv("HOME", tmpDir)

		sessionsDir := filepath.Join(tmpDir, ".heike", "workspaces", "test-workspace-"+t.Name(), "sessions")
		if err := os.MkdirAll(sessionsDir, 0755); err != nil {
			t.Fatalf("Failed to create sessions dir: %v", err)
		}

		testSession := filepath.Join(sessionsDir, "test-session.jsonl")
		if err := os.WriteFile(testSession, []byte("test data"), 0644); err != nil {
			t.Fatalf("Failed to create test session: %v", err)
		}

		cmd := &cobra.Command{}
		cmd.Flags().StringP("workspace", "w", "", "Target workspace ID")
		_ = cmd.Flags().Set("workspace", "test-workspace-"+t.Name())

		args := []string{}
		if err := sessionLsCmd.RunE(cmd, args); err != nil {
			t.Errorf("Session ls failed: %v", err)
		}
	})
}

func TestSessionResetCmd(t *testing.T) {
	tmpDir := t.TempDir()
	home := os.Getenv("HOME")
	defer func() {
		if home != "" {
			os.Setenv("HOME", home)
		}
	}()
	os.Setenv("HOME", tmpDir)

	sessionsDir := filepath.Join(tmpDir, ".heike", "workspaces", "test-workspace-"+t.Name(), "sessions")
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		t.Fatalf("Failed to create sessions dir: %v", err)
	}

	testSession := filepath.Join(sessionsDir, "test-session.jsonl")
	if err := os.WriteFile(testSession, []byte("test data"), 0644); err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	cmd := &cobra.Command{}
	cmd.Flags().StringP("workspace", "w", "", "Target workspace ID")
	_ = cmd.Flags().Set("workspace", "test-workspace-"+t.Name())

	args := []string{"test-session"}
	if err := sessionResetCmd.RunE(cmd, args); err != nil {
		t.Errorf("Session reset failed: %v", err)
	}

	if _, err := os.Stat(testSession); !os.IsNotExist(err) {
		t.Error("Session file should be deleted after reset")
	}
}
