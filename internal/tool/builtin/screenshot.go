package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	toolcore "github.com/harunnryd/heike/internal/tool"
)

const (
	defaultScreenshotRenderer = "pdftoppm"
	maxScreenshotBatchSize    = 4
)

type screenshotInput struct {
	RefID  string `json:"ref_id"`
	PageNo int    `json:"pageno"`
}

type screenshotRequest struct {
	RefID      string            `json:"ref_id"`
	PageNo     int               `json:"pageno"`
	Screenshot []screenshotInput `json:"screenshot"`
}

type screenshotRendererFn func(ctx context.Context, renderer string, pdfBytes []byte, pageNo int) (string, error)

func init() {
	toolcore.RegisterBuiltin("screenshot", func(options toolcore.BuiltinOptions) (toolcore.Tool, error) {
		timeout := options.ScreenshotTimeout
		if timeout <= 0 {
			timeout = options.WebTimeout
		}
		if timeout <= 0 {
			timeout = 20 * time.Second
		}

		return &ScreenshotTool{
			Client:   &http.Client{Timeout: timeout},
			Renderer: strings.TrimSpace(options.ScreenshotRenderer),
			render:   renderPDFPageToPNG,
		}, nil
	})
}

// ScreenshotTool renders a PDF page to PNG.
type ScreenshotTool struct {
	Client   *http.Client
	Renderer string
	render   screenshotRendererFn
}

func (t *ScreenshotTool) Name() string { return "screenshot" }

func (t *ScreenshotTool) Description() string {
	return "Render a PDF page from ref_id/url into a PNG screenshot."
}

func (t *ScreenshotTool) ToolMetadata() toolcore.ToolMetadata {
	return toolcore.ToolMetadata{
		Source: "builtin",
		Capabilities: []string{
			"document.screenshot",
			"pdf.render",
			"http.get",
		},
		Risk: toolcore.RiskMedium,
	}
}

func (t *ScreenshotTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"ref_id": map[string]interface{}{
				"type":        "string",
				"description": "Reference id from open/search output or direct URL",
			},
			"pageno": map[string]interface{}{
				"type":        "integer",
				"description": "0-based PDF page number",
			},
			"screenshot": map[string]interface{}{
				"type":        "array",
				"description": "Batch mode",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"ref_id": map[string]interface{}{"type": "string"},
						"pageno": map[string]interface{}{"type": "integer"},
					},
				},
			},
		},
		"required": []string{"ref_id"},
	}
}

func (t *ScreenshotTool) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	var args screenshotRequest
	if err := json.Unmarshal(input, &args); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	if len(args.Screenshot) > 0 {
		if len(args.Screenshot) > maxScreenshotBatchSize {
			return nil, fmt.Errorf("screenshot supports at most %d requests per call", maxScreenshotBatchSize)
		}
		results := make([]map[string]interface{}, 0, len(args.Screenshot))
		for _, req := range args.Screenshot {
			result, err := t.executeOne(ctx, req)
			if err != nil {
				return nil, err
			}
			results = append(results, result)
		}
		return json.Marshal(map[string]interface{}{"results": results})
	}

	result, err := t.executeOne(ctx, screenshotInput{
		RefID:  args.RefID,
		PageNo: args.PageNo,
	})
	if err != nil {
		return nil, err
	}
	return json.Marshal(result)
}

func (t *ScreenshotTool) executeOne(ctx context.Context, input screenshotInput) (map[string]interface{}, error) {
	urlValue, err := resolveScreenshotURL(input.RefID)
	if err != nil {
		return nil, err
	}
	if input.PageNo < 0 {
		return nil, fmt.Errorf("pageno must be >= 0")
	}

	client := t.Client
	if client == nil {
		client = &http.Client{Timeout: toolcore.DefaultBuiltinWebTimeout}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlValue, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Heike/1.0 (+https://example.invalid)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("screenshot fetch failed: %s", resp.Status)
	}

	pdfBytes, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, err
	}
	if !isLikelyPDF(resp.Header.Get("Content-Type"), pdfBytes, urlValue) {
		return nil, fmt.Errorf("screenshot currently supports PDF sources only")
	}

	renderer := t.render
	if renderer == nil {
		renderer = renderPDFPageToPNG
	}
	filePath, err := renderer(ctx, t.Renderer, pdfBytes, input.PageNo)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"ref_id":    strings.TrimSpace(input.RefID),
		"url":       urlValue,
		"pageno":    input.PageNo,
		"file_path": filePath,
		"mime_type": "image/png",
	}, nil
}

func resolveScreenshotURL(refID string) (string, error) {
	value := strings.TrimSpace(refID)
	if value == "" {
		return "", fmt.Errorf("ref_id is required")
	}

	if page, ok := getWebPage(value); ok {
		return page.URL, nil
	}
	if searchRef, ok := getWebSearchRef(value); ok {
		return searchRef.URL, nil
	}
	if isLikelyURL(value) {
		return value, nil
	}
	return "", fmt.Errorf("ref_id not found")
}

func isLikelyPDF(contentType string, body []byte, sourceURL string) bool {
	if strings.Contains(strings.ToLower(strings.TrimSpace(contentType)), "application/pdf") {
		return true
	}
	if strings.HasSuffix(strings.ToLower(strings.TrimSpace(sourceURL)), ".pdf") {
		return true
	}
	return len(body) >= 4 && bytes.Equal(body[:4], []byte("%PDF"))
}

func renderPDFPageToPNG(ctx context.Context, renderer string, pdfBytes []byte, pageNo int) (string, error) {
	bin := strings.TrimSpace(renderer)
	if bin == "" {
		bin = defaultScreenshotRenderer
	}

	if _, err := exec.LookPath(bin); err != nil {
		return "", fmt.Errorf("screenshot renderer %q not found in PATH", bin)
	}

	dir, err := os.MkdirTemp("", "heike-screenshot-*")
	if err != nil {
		return "", err
	}

	pdfPath := filepath.Join(dir, "input.pdf")
	if err := os.WriteFile(pdfPath, pdfBytes, 0644); err != nil {
		return "", err
	}

	page := pageNo + 1
	if page < 1 {
		page = 1
	}

	outBase := filepath.Join(dir, "page")
	cmd := exec.CommandContext(ctx, bin, "-f", strconv.Itoa(page), "-singlefile", "-png", pdfPath, outBase)
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("render screenshot failed: %s", msg)
	}

	outPath := outBase + ".png"
	if _, err := os.Stat(outPath); err != nil {
		return "", fmt.Errorf("rendered output missing: %w", err)
	}

	return outPath, nil
}
