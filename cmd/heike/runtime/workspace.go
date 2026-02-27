package runtime

import (
	"github.com/harunnryd/heike/internal/config"

	"github.com/spf13/cobra"
)

const DefaultWorkspaceID = config.DefaultWorkspaceID

func ResolveWorkspaceID(cmd *cobra.Command) string {
	if workspaceID, _ := cmd.Flags().GetString("workspace"); workspaceID != "" {
		return workspaceID
	}

	return config.DefaultWorkspaceID
}
