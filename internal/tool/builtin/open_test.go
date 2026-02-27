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

func TestOpenTool_Execute_Batch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "<html><body>batch-open</body></html>")
	}))
	defer server.Close()

	tool := &OpenTool{
		Client:           server.Client(),
		maxContentLength: 5000,
	}

	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"open":[{"url":"`+server.URL+`/a"},{"ref_id":"`+server.URL+`/b"}]}`))
	require.NoError(t, err)

	resp := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(raw, &resp))
	results, ok := resp["results"].([]interface{})
	require.True(t, ok)
	require.Len(t, results, 2)
}

func TestOpenTool_CanResolveSearchRefID(t *testing.T) {
	pageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "<html><body>hello from page</body></html>")
	}))
	defer pageServer.Close()

	searchServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `<li class="b_algo"><h2><a href="`+pageServer.URL+`/doc">Doc</a></h2></li>`)
	}))
	defer searchServer.Close()

	searchTool := &WebSearchTool{
		Client:     searchServer.Client(),
		BaseURL:    searchServer.URL,
		MaxResults: 5,
	}

	searchRaw, err := searchTool.Execute(context.Background(), json.RawMessage(`{"q":"docs"}`))
	require.NoError(t, err)

	searchResp := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(searchRaw, &searchResp))
	results, ok := searchResp["results"].([]interface{})
	require.True(t, ok)
	require.NotEmpty(t, results)

	first, ok := results[0].(map[string]interface{})
	require.True(t, ok)
	refID, ok := first["ref_id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, refID)

	openTool := &OpenTool{
		Client:           pageServer.Client(),
		maxContentLength: 5000,
	}
	openRaw, err := openTool.Execute(context.Background(), json.RawMessage(`{"ref_id":"`+refID+`"}`))
	require.NoError(t, err)

	openResp := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(openRaw, &openResp))
	urlValue, ok := openResp["url"].(string)
	require.True(t, ok)
	assert.Contains(t, urlValue, pageServer.URL+"/doc")
	content, _ := openResp["content"].(string)
	assert.Contains(t, content, "hello from page")
}
