package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/harunnryd/heike/cmd/heike/runtime"

	"github.com/harunnryd/heike/internal/daemon"
	"github.com/harunnryd/heike/internal/daemon/components"

	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Start Heike in background daemon mode",
	Long:  `Starts Heike as a long-running service using component lifecycle orchestration. It exposes health endpoint and runs scheduled tasks.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		workspaceID := runtime.ResolveWorkspaceID(cmd)
		forceClean, _ := cmd.Flags().GetBool("force-clean-locks")

		if cfg == nil {
			return fmt.Errorf("config not loaded")
		}

		runtimeComp := runtime.NewDaemonRuntimeComponent(workspaceID, cfg, runtime.AdapterBuildOptions{
			IncludeCLI:        false,
			IncludeSystemNull: true,
		})
		defer func() {
			_ = runtimeComp.Stop(context.Background())
		}()

		daemonMgr, err := daemon.NewDaemon(workspaceID, cfg)
		if err != nil {
			return fmt.Errorf("failed to create daemon manager: %w", err)
		}
		daemonMgr.SetForceCleanup(forceClean)

		httpComp := components.NewHTTPServerComponentWithDependencies(daemonMgr, &cfg.Server, []string{runtimeComp.Name()})

		daemonMgr.AddComponent(runtimeComp)
		daemonMgr.AddComponent(httpComp)

		slog.Info("Heike Daemon starting up...", "port", cfg.Server.Port, "workspace", workspaceID)
		err = daemonMgr.Start(context.Background())
		if err != nil {
			// Cancellation via signal/context is a graceful shutdown case for CLI.
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				slog.Info("Heike Daemon stopped gracefully", "workspace", workspaceID)
				return nil
			}
			return fmt.Errorf("daemon failed: %w", err)
		}

		slog.Info("Heike Daemon stopped gracefully", "workspace", workspaceID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(daemonCmd)
	daemonCmd.Flags().StringP("workspace", "w", "", "Target workspace ID")
	daemonCmd.Flags().Bool("force-clean-locks", false, "Force cleanup of stale lock files (default: warn-only)")
}
