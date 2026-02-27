package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTimeTool_Execute_Batch(t *testing.T) {
	tool := &TimeTool{}
	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"time":[{"utc_offset":"+00:00"},{"utc_offset":"+07:00"}]}`))
	require.NoError(t, err)

	resp := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(raw, &resp))
	results, ok := resp["results"].([]interface{})
	require.True(t, ok)
	require.Len(t, results, 2)
}
