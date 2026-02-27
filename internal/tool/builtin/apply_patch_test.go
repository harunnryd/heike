package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyPatchToolExecute_ObjectInput(t *testing.T) {
	tool := &ApplyPatchTool{
		Command: "apply_patch",
		run: func(ctx context.Context, command, workdir, patch string) (string, error) {
			assert.Equal(t, "apply_patch", command)
			assert.Equal(t, "/tmp", workdir)
			assert.Contains(t, patch, "*** Begin Patch")
			return "ok", nil
		},
	}

	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"patch":"*** Begin Patch\n*** End Patch\n","workdir":"/tmp"}`))
	require.NoError(t, err)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &resp))
	assert.Equal(t, true, resp["applied"])
	assert.Equal(t, "ok", resp["output"])
}

func TestApplyPatchToolExecute_RawPatchStringInput(t *testing.T) {
	tool := &ApplyPatchTool{
		Command: "apply_patch",
		run: func(ctx context.Context, command, workdir, patch string) (string, error) {
			assert.Equal(t, "*** Begin Patch\n*** End Patch\n", patch)
			return "ok", nil
		},
	}

	input, err := json.Marshal("*** Begin Patch\n*** End Patch\n")
	require.NoError(t, err)

	_, err = tool.Execute(context.Background(), input)
	require.NoError(t, err)
}

func TestApplyPatchToolExecute_DryRunUnsupported(t *testing.T) {
	tool := &ApplyPatchTool{}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"patch":"*** Begin Patch\n*** End Patch\n","dry_run":true}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dry_run")
}
