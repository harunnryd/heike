package builtin

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecCommandTool_Execute_OneShotCmd(t *testing.T) {
	tool := &ExecCommandTool{}

	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"cmd":"printf 'hello'"}`))
	require.NoError(t, err)

	resp := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(raw, &resp))
	assert.Equal(t, "hello", resp["output"])
	assert.Equal(t, float64(0), resp["exit_code"])
}

func TestExecCommandTool_Execute_Workdir(t *testing.T) {
	tool := &ExecCommandTool{}
	workdir := t.TempDir()

	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"cmd":"pwd","workdir":"`+workdir+`"}`))
	require.NoError(t, err)

	resp := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(raw, &resp))
	output, ok := resp["output"].(string)
	require.True(t, ok)
	assert.Contains(t, filepath.Clean(strings.TrimSpace(output)), filepath.Clean(workdir))
	assert.Equal(t, float64(0), resp["exit_code"])
}

func TestExecCommandAndWriteStdin_InteractiveSession(t *testing.T) {
	execTool := &ExecCommandTool{}
	writeTool := &WriteStdinTool{}

	raw, err := execTool.Execute(context.Background(), json.RawMessage(`{"cmd":"cat","tty":true,"yield_time_ms":10}`))
	require.NoError(t, err)

	start := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(raw, &start))
	sessionIDValue, ok := start["session_id"].(float64)
	require.True(t, ok)
	sessionID := int64(sessionIDValue)
	require.Greater(t, sessionID, int64(0))

	rawWrite, err := writeTool.Execute(context.Background(), json.RawMessage(`{"session_id":`+strconv.FormatInt(sessionID, 10)+`,"chars":"ping\\n","yield_time_ms":50}`))
	require.NoError(t, err)

	wResp := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(rawWrite, &wResp))
	assert.Equal(t, float64(sessionID), wResp["session_id"])

	// Explicit cleanup: terminate the spawned process to keep tests deterministic.
	session, ok := getExecSession(sessionID)
	require.True(t, ok)
	require.NoError(t, session.cmd.Process.Kill())

	stopped := false
	for i := 0; i < 20; i++ {
		rawPoll, pollErr := writeTool.Execute(context.Background(), json.RawMessage(`{"session_id":`+strconv.FormatInt(sessionID, 10)+`,"yield_time_ms":50}`))
		require.NoError(t, pollErr)

		pResp := map[string]interface{}{}
		require.NoError(t, json.Unmarshal(rawPoll, &pResp))
		running, ok := pResp["running"].(bool)
		require.True(t, ok)
		if !running {
			stopped = true
			break
		}
	}

	assert.True(t, stopped)
}

func TestWriteStdinTool_UnknownSession(t *testing.T) {
	writeTool := &WriteStdinTool{}
	_, err := writeTool.Execute(context.Background(), json.RawMessage(`{"session_id":999999}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
}
