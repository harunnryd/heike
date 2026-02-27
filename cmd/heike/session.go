package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/harunnryd/heike/cmd/heike/runtime"

	"github.com/harunnryd/heike/internal/store"

	"github.com/gofrs/flock"
	"github.com/spf13/cobra"
)

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage sessions",
	Long:  `List and reset interactive sessions in the workspace.`,
}

var sessionLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List active sessions",
	Long:  `Display all interactive sessions with their IDs.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		workspaceID := runtime.ResolveWorkspaceID(cmd)
		workspaceRootPath := ""
		if cfg != nil {
			workspaceRootPath = cfg.Daemon.WorkspacePath
		}

		sessionsDir, err := store.GetSessionsDir(workspaceID, workspaceRootPath)
		if err != nil {
			return fmt.Errorf("failed to get sessions directory: %w", err)
		}

		entries, err := os.ReadDir(sessionsDir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("No sessions directory found (workspace not initialized yet).")
				fmt.Println("\nRun 'heike run' to create your first session.")
				return nil
			}
			return fmt.Errorf("failed to read sessions directory: %w", err)
		}

		var sessions []string
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".jsonl") {
				id := strings.TrimSuffix(entry.Name(), ".jsonl")
				sessions = append(sessions, id)
			}
		}

		if len(sessions) == 0 {
			fmt.Println("No active sessions found.")
			fmt.Println("\nRun 'heike run' to create your first session.")
			return nil
		}

		fmt.Println("Active Sessions:")
		for _, id := range sessions {
			fmt.Printf("- %s\n", id)
		}

		fmt.Printf("\nTotal: %d session(s)\n", len(sessions))
		return nil
	},
}

var sessionResetCmd = &cobra.Command{
	Use:   "reset [id]",
	Short: "Reset a session (delete data)",
	Long:  `Delete all data for a specific session transcript.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID := args[0]
		workspaceID := runtime.ResolveWorkspaceID(cmd)
		workspaceRootPath := ""
		if cfg != nil {
			workspaceRootPath = cfg.Daemon.WorkspacePath
		}

		lockPath, err := store.GetLockPath(workspaceID, workspaceRootPath)
		if err != nil {
			return fmt.Errorf("failed to get lock path: %w", err)
		}

		fileLock := flock.New(lockPath)
		locked, err := fileLock.TryLock()
		if err != nil {
			return fmt.Errorf("failed to acquire lock: %w", err)
		}
		if !locked {
			return fmt.Errorf("workspace is locked by another Heike instance")
		}
		defer fileLock.Unlock()

		sessionsDir, err := store.GetSessionsDir(workspaceID, workspaceRootPath)
		if err != nil {
			return fmt.Errorf("failed to get sessions directory: %w", err)
		}

		transcriptPath := filepath.Join(sessionsDir, sessionID+".jsonl")
		if err := os.Remove(transcriptPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to delete transcript: %w", err)
		}

		fmt.Printf("✓ Session '%s' reset successfully.\n", sessionID)
		return nil
	},
}

func init() {
	sessionCmd.AddCommand(sessionLsCmd)
	sessionCmd.AddCommand(sessionResetCmd)
	sessionCmd.PersistentFlags().StringP("workspace", "w", "", "Target workspace ID")
	rootCmd.AddCommand(sessionCmd)
}
