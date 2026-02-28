package runtime

import (
	"github.com/harunnryd/heike/internal/config"

	"github.com/spf13/cobra"
)

const DefaultWorkspaceID = config.DefaultWorkspaceID

func ResolveWorkspaceID(cmd *cobra.Command) string {
	if cmd != nil {
		if flag := cmd.Flag("workspace"); flag != nil {
			if workspaceID := flag.Value.String(); workspaceID != "" {
				return workspaceID
			}
		}
		if workspaceID, err := cmd.Flags().GetString("workspace"); err == nil && workspaceID != "" {
			return workspaceID
		}
	}

	return config.DefaultWorkspaceID
}
