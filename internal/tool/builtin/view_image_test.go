package builtin

import (
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestViewImageToolExecute_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.png")

	img := image.NewRGBA(image.Rect(0, 0, 3, 2))
	img.Set(1, 1, color.RGBA{R: 255, G: 0, B: 0, A: 255})

	file, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, png.Encode(file, img))
	require.NoError(t, file.Close())

	tool := &ViewImageTool{}
	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"path":"`+path+`"}`))
	require.NoError(t, err)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &resp))

	assert.Equal(t, path, resp["path"])
	assert.Equal(t, "png", resp["format"])
	assert.Equal(t, float64(3), resp["width"])
	assert.Equal(t, float64(2), resp["height"])
}

func TestViewImageToolExecute_NotImage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "not-image.txt")
	require.NoError(t, os.WriteFile(path, []byte("hello"), 0644))

	tool := &ViewImageTool{}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"path":"`+path+`"}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a supported image")
}
