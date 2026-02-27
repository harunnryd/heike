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

func TestImageQueryToolExecute_SingleWithDomainFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "query", r.URL.Query().Get("action"))
		assert.Equal(t, "6", r.URL.Query().Get("gsrnamespace"))
		assert.Equal(t, "cats", r.URL.Query().Get("gsrsearch"))
		_, _ = io.WriteString(w, `{
			"query":{
				"pages":{
					"1":{"title":"File:Cat1.jpg","imageinfo":[{"url":"https://upload.wikimedia.org/cat1.jpg","thumburl":"https://upload.wikimedia.org/thumb/cat1.jpg","width":1200,"height":800}]},
					"2":{"title":"File:Cat2.jpg","imageinfo":[{"url":"https://example.org/cat2.jpg","thumburl":"https://example.org/thumb/cat2.jpg","width":900,"height":600}]}
				}
			}
		}`)
	}))
	defer server.Close()

	tool := &ImageQueryTool{
		Client:  server.Client(),
		BaseURL: server.URL,
	}

	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"q":"cats","domains":["upload.wikimedia.org"]}`))
	require.NoError(t, err)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &resp))

	results, ok := resp["results"].([]interface{})
	require.True(t, ok)
	require.Len(t, results, 1)

	first, ok := results[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "upload.wikimedia.org", first["source_domain"])
}

func TestImageQueryToolExecute_Batch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{
			"query":{
				"pages":{
					"1":{"title":"File:X.jpg","imageinfo":[{"url":"https://upload.wikimedia.org/x.jpg","thumburl":"https://upload.wikimedia.org/thumb/x.jpg","width":640,"height":480}]}
				}
			}
		}`)
	}))
	defer server.Close()

	tool := &ImageQueryTool{
		Client:  server.Client(),
		BaseURL: server.URL,
	}

	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"image_query":[{"q":"dogs"},{"query":"birds"}]}`))
	require.NoError(t, err)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &resp))

	results, ok := resp["results"].([]interface{})
	require.True(t, ok)
	require.Len(t, results, 2)
}

func TestImageQueryToolExecute_TooManyBatchItems(t *testing.T) {
	tool := &ImageQueryTool{}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"image_query":[{"q":"a"},{"q":"b"},{"q":"c"}]}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at most 2")
}
