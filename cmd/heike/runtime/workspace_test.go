package runtime

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestResolveWorkspaceID_WithFlag(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringP("workspace", "w", "", "Target workspace ID")

	testCases := []struct {
		name      string
		flagValue string
		want      string
	}{
		{
			name:      "custom workspace",
			flagValue: "custom-workspace",
			want:      "custom-workspace",
		},
		{
			name:      "empty flag",
			flagValue: "",
			want:      DefaultWorkspaceID,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd.Flags().Set("workspace", tc.flagValue)
			got := ResolveWorkspaceID(cmd)
			if got != tc.want {
				t.Errorf("ResolveWorkspaceID() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResolveWorkspaceID_Default(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringP("workspace", "w", "", "Target workspace ID")

	got := ResolveWorkspaceID(cmd)
	if got != DefaultWorkspaceID {
		t.Errorf("ResolveWorkspaceID() = %v, want %v", got, DefaultWorkspaceID)
	}
}
