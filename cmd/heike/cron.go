package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/harunnryd/heike/cmd/heike/runtime"

	"github.com/harunnryd/heike/internal/scheduler"
	"github.com/harunnryd/heike/internal/store"

	"github.com/spf13/cobra"
)

var cronCmd = &cobra.Command{
	Use:   "cron",
	Short: "Manage scheduled tasks",
	Long:  `List and manage scheduled tasks in the scheduler.`,
}

var cronLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List scheduled tasks",
	Long:  `Display all scheduled tasks with their ID, schedule, description, and next run time.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		workspaceID := runtime.ResolveWorkspaceID(cmd)
		workspaceRootPath := ""
		if cfg != nil {
			workspaceRootPath = cfg.Daemon.WorkspacePath
		}

		schedulerDir, err := store.GetSchedulerDir(workspaceID, workspaceRootPath)
		if err != nil {
			return fmt.Errorf("failed to get scheduler directory: %w", err)
		}

		tasksPath := filepath.Join(schedulerDir, "tasks.json")
		data, err := os.ReadFile(tasksPath)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("No tasks found (tasks file does not exist).")
				fmt.Println("\nTo schedule tasks, use the scheduler component in daemon mode.")
				return nil
			}
			return fmt.Errorf("failed to read tasks file: %w", err)
		}

		var taskList scheduler.TaskList
		if err := json.Unmarshal(data, &taskList); err != nil {
			return fmt.Errorf("failed to parse tasks: %w", err)
		}

		if len(taskList.Tasks) == 0 {
			fmt.Println("No tasks scheduled.")
			fmt.Println("\nTo schedule tasks, use the scheduler component in daemon mode.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tSCHEDULE\tDESCRIPTION\tNEXT RUN")
		for _, t := range taskList.Tasks {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				t.ID,
				t.Schedule,
				t.Description,
				t.NextRun.Format("2006-01-02 15:04:05"))
		}
		if err := w.Flush(); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}

		fmt.Printf("\nTotal: %d scheduled task(s)\n", len(taskList.Tasks))
		return nil
	},
}

func init() {
	cronCmd.AddCommand(cronLsCmd)
	cronCmd.PersistentFlags().StringP("workspace", "w", "", "Target workspace ID")
	rootCmd.AddCommand(cronCmd)
}
