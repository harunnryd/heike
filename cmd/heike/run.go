package main

import (
	"fmt"

	"github.com/harunnryd/heike/cmd/heike/runtime"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start Heike in interactive mode",
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeWithRuntime(cmd, func(r *runtime.RuntimeComponents) error {
			if err := r.Start(); err != nil {
				return fmt.Errorf("failed to start runtime components: %w", err)
			}

			repl := runtime.NewREPL(r)
			return repl.Start()
		})
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringP("workspace", "w", "", "Target workspace ID")
}
