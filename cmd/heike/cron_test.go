package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestCronLsCmd(t *testing.T) {
	t.Run("without tasks", func(t *testing.T) {
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

		args := []string{"ls"}
		if err := cronLsCmd.RunE(cmd, args); err != nil {
			t.Errorf("Cron ls failed: %v", err)
		}
	})

	t.Run("with tasks", func(t *testing.T) {
		tmpDir := t.TempDir()
		home := os.Getenv("HOME")
		defer func() {
			if home != "" {
				os.Setenv("HOME", home)
			}
		}()
		os.Setenv("HOME", tmpDir)

		schedulerDir := filepath.Join(tmpDir, ".heike", "workspaces", "test-workspace-"+t.Name(), "scheduler")
		if err := os.MkdirAll(schedulerDir, 0755); err != nil {
			t.Fatalf("Failed to create scheduler dir: %v", err)
		}

		tasks := map[string]map[string]interface{}{
			"tasks": {
				"task-1": map[string]interface{}{
					"id":          "task-1",
					"schedule":    "0 * * * *",
					"description": "Test task",
					"next_run":    "2024-01-01T00:00:00Z",
				},
			},
		}

		tasksPath := filepath.Join(schedulerDir, "tasks.json")
		tasksData, _ := json.Marshal(tasks)
		if err := os.WriteFile(tasksPath, tasksData, 0644); err != nil {
			t.Fatalf("Failed to create tasks file: %v", err)
		}

		cmd := &cobra.Command{}
		cmd.Flags().StringP("workspace", "w", "", "Target workspace ID")
		_ = cmd.Flags().Set("workspace", "test-workspace-"+t.Name())

		args := []string{"ls"}
		if err := cronLsCmd.RunE(cmd, args); err != nil {
			t.Errorf("Cron ls failed: %v", err)
		}
	})
}
