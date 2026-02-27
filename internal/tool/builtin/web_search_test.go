package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebSearchTool_Execute(t *testing.T) {
	var observedQuery string
	var observedQFT string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observedQuery = r.URL.Query().Get("q")
		observedQFT = r.URL.Query().Get("qft")
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, `
<li class="b_algo"><h2><a href="https://example.com/a">Alpha <b>Result</b></a></h2></li>
<li class="b_algo"><h2><a href="https://example.com/b">Beta Result</a></h2></li>
`)
	}))
	defer server.Close()

	tool := &WebSearchTool{
		Client:     server.Client(),
		BaseURL:    server.URL,
		MaxResults: 5,
	}

	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"q":"golang tools","domains":["example.com"],"recency":7,"max_results":2}`))
	require.NoError(t, err)
	assert.Equal(t, "golang tools site:example.com", observedQuery)
	assert.Equal(t, "+filterui:age-lt10080", observedQFT)

	resp := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(raw, &resp))
	assert.Equal(t, "golang tools", resp["query"])
	assert.Equal(t, "golang tools site:example.com", resp["effective_query"])

	results, ok := resp["results"].([]interface{})
	require.True(t, ok)
	require.Len(t, results, 2)

	first, ok := results[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Alpha Result", first["title"])
	assert.Equal(t, "https://example.com/a", first["url"])
}

func TestWebSearchTool_Execute_RequiresQueryOrQ(t *testing.T) {
	tool := &WebSearchTool{}

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"max_results":1}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "query or q is required")
}

func TestWebSearchTool_Execute_RespectsHardCap(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b strings.Builder
		for i := 1; i <= 20; i++ {
			fmt.Fprintf(&b, `<li class="b_algo"><h2><a href="https://example.com/%d">Result %d</a></h2></li>`, i, i)
		}
		_, _ = io.WriteString(w, b.String())
	}))
	defer server.Close()

	tool := &WebSearchTool{
		Client:     server.Client(),
		BaseURL:    server.URL,
		MaxResults: 5,
	}

	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"x","max_results":999}`))
	require.NoError(t, err)

	resp := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(raw, &resp))
	results, ok := resp["results"].([]interface{})
	require.True(t, ok)
	require.Len(t, results, 10)
}

func TestWebSearchTool_Execute_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer server.Close()

	tool := &WebSearchTool{
		Client:     server.Client(),
		BaseURL:    server.URL,
		MaxResults: 5,
	}

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"test"}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "search request failed")
}

func TestParseBingSearchResults_DecodesEntities(t *testing.T) {
	doc := `<li class="b_algo"><h2><a href="https://example.com/?a=1&amp;b=2">Go &amp; Rust</a></h2></li>`

	results := parseBingSearchResults(doc, 1)
	require.Len(t, results, 1)
	assert.Equal(t, "https://example.com/?a=1&b=2", results[0]["url"])
	assert.Equal(t, "Go & Rust", results[0]["title"])
}

func TestWebSearchTool_Execute_BatchMode(t *testing.T) {
	var qftValues []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		qftValues = append(qftValues, r.URL.Query().Get("qft"))
		_, _ = io.WriteString(w, `<li class="b_algo"><h2><a href="https://example.com/x">X</a></h2></li>`)
	}))
	defer server.Close()

	tool := &WebSearchTool{
		Client:     server.Client(),
		BaseURL:    server.URL,
		MaxResults: 5,
	}

	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"recency":3,"search_query":[{"q":"first"},{"q":"second","recency":1}]}`))
	require.NoError(t, err)

	resp := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(raw, &resp))
	results, ok := resp["results"].([]interface{})
	require.True(t, ok)
	require.Len(t, results, 2)
	require.Len(t, qftValues, 2)
	assert.Equal(t, "+filterui:age-lt4320", qftValues[0])
	assert.Equal(t, "+filterui:age-lt1440", qftValues[1])
}
