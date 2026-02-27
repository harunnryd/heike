package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	toolcore "github.com/harunnryd/heike/internal/tool"
)

func init() {
	toolcore.RegisterBuiltin("view_image", func(options toolcore.BuiltinOptions) (toolcore.Tool, error) {
		return &ViewImageTool{}, nil
	})
}

// ViewImageTool reads local image metadata from disk.
type ViewImageTool struct{}

func (t *ViewImageTool) Name() string { return "view_image" }

func (t *ViewImageTool) Description() string {
	return "Read local image metadata from an absolute filesystem path."
}

func (t *ViewImageTool) ToolMetadata() toolcore.ToolMetadata {
	return toolcore.ToolMetadata{
		Source: "builtin",
		Capabilities: []string{
			"image.read",
			"filesystem.read",
		},
		Risk: toolcore.RiskLow,
	}
}

func (t *ViewImageTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Absolute local filesystem path to image file",
			},
		},
		"required": []string{"path"},
	}
}

func (t *ViewImageTool) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	_ = ctx

	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	rawPath := strings.TrimSpace(args.Path)
	if rawPath == "" {
		return nil, fmt.Errorf("path is required")
	}

	path := rawPath
	if !filepath.IsAbs(path) {
		absPath, err := filepath.Abs(path)
		if err == nil {
			path = absPath
		}
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	header := make([]byte, 512)
	n, readErr := io.ReadFull(file, header)
	if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
		return nil, readErr
	}
	mimeType := http.DetectContentType(header[:n])
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(mimeType)), "image/") {
		return nil, fmt.Errorf("file is not a supported image: %s", mimeType)
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	cfg, format, err := image.DecodeConfig(file)
	if err != nil {
		return nil, fmt.Errorf("decode image metadata: %w", err)
	}

	return json.Marshal(map[string]interface{}{
		"path":       path,
		"size_bytes": info.Size(),
		"mime_type":  mimeType,
		"format":     format,
		"width":      cfg.Width,
		"height":     cfg.Height,
	})
}
