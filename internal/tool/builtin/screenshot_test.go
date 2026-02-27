package builtin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScreenshotToolExecute_SingleFromRefID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = io.WriteString(w, "%PDF-1.4 sample")
	}))
	defer server.Close()

	refID := storeWebPage(server.URL+"/doc.pdf", "200 OK", "", nil)

	tool := &ScreenshotTool{
		Client: server.Client(),
		render: func(ctx context.Context, renderer string, pdfBytes []byte, pageNo int) (string, error) {
			assert.Equal(t, 2, pageNo)
			assert.Contains(t, string(pdfBytes), "%PDF")
			dir := t.TempDir()
			path := filepath.Join(dir, "page.png")
			return path, os.WriteFile(path, []byte("png"), 0644)
		},
	}

	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"ref_id":"`+refID+`","pageno":2}`))
	require.NoError(t, err)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &resp))
	assert.Equal(t, float64(2), resp["pageno"])
	assert.Equal(t, "image/png", resp["mime_type"])
	path, ok := resp["file_path"].(string)
	require.True(t, ok)
	_, statErr := os.Stat(path)
	require.NoError(t, statErr)
}

func TestScreenshotToolExecute_RejectsNonPDF(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, "<html>not-pdf</html>")
	}))
	defer server.Close()

	tool := &ScreenshotTool{Client: server.Client()}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"ref_id":"`+server.URL+`/index.html","pageno":0}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PDF")
}

func TestScreenshotToolExecute_Batch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = io.WriteString(w, "%PDF-1.7 batch")
	}))
	defer server.Close()

	tool := &ScreenshotTool{
		Client: server.Client(),
		render: func(ctx context.Context, renderer string, pdfBytes []byte, pageNo int) (string, error) {
			path := filepath.Join(t.TempDir(), "out.png")
			return path, os.WriteFile(path, []byte("png"), 0644)
		},
	}

	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"screenshot":[{"ref_id":"`+server.URL+`/a.pdf","pageno":0},{"ref_id":"`+server.URL+`/b.pdf","pageno":1}]}`))
	require.NoError(t, err)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &resp))
	results, ok := resp["results"].([]interface{})
	require.True(t, ok)
	require.Len(t, results, 2)
}
