package builtin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClickAndFindTool_Execute_Batch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `<html><body>needle here <a href="/target">Go target</a></body></html>`)
	}))
	defer server.Close()

	openTool := &OpenTool{
		Client:           server.Client(),
		maxContentLength: 5000,
	}
	openRaw, err := openTool.Execute(context.Background(), json.RawMessage(`{"url":"`+server.URL+`/page"}`))
	require.NoError(t, err)

	openResp := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(openRaw, &openResp))
	refID, ok := openResp["ref_id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, refID)

	clickTool := &ClickTool{}
	clickRaw, err := clickTool.Execute(context.Background(), json.RawMessage(`{"click":[{"ref_id":"`+refID+`","id":1}]}`))
	require.NoError(t, err)

	clickResp := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(clickRaw, &clickResp))
	clickResults, ok := clickResp["results"].([]interface{})
	require.True(t, ok)
	require.Len(t, clickResults, 1)
	firstClick, ok := clickResults[0].(map[string]interface{})
	require.True(t, ok)
	clickURL, ok := firstClick["url"].(string)
	require.True(t, ok)
	assert.Contains(t, clickURL, "/target")

	findTool := &FindTool{}
	findRaw, err := findTool.Execute(context.Background(), json.RawMessage(`{"find":[{"ref_id":"`+refID+`","pattern":"needle"}]}`))
	require.NoError(t, err)

	findResp := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(findRaw, &findResp))
	findResults, ok := findResp["results"].([]interface{})
	require.True(t, ok)
	require.Len(t, findResults, 1)
	firstFind, ok := findResults[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, firstFind["found"])
}
