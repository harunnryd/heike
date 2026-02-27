package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	toolcore "github.com/harunnryd/heike/internal/tool"
)

func init() {
	toolcore.RegisterBuiltin("open", func(options toolcore.BuiltinOptions) (toolcore.Tool, error) {
		return NewOpenTool(options.WebTimeout, options.WebMaxContentLength), nil
	})
}

// OpenTool fetches content from a URL.
type OpenTool struct {
	Client           *http.Client
	maxContentLength int
}

func NewOpenTool(timeout time.Duration, maxContentLength int) *OpenTool {
	if timeout <= 0 {
		timeout = toolcore.DefaultBuiltinWebTimeout
	}

	if maxContentLength <= 0 {
		maxContentLength = toolcore.DefaultBuiltinWebMaxContentLength
	}

	return &OpenTool{
		Client: &http.Client{
			Timeout: timeout,
		},
		maxContentLength: maxContentLength,
	}
}

func (t *OpenTool) Name() string {
	return "open"
}

func (t *OpenTool) Description() string {
	return "Fetch content from a URL or previously referenced ref_id. Requires Domain Approval."
}

func (t *OpenTool) ToolMetadata() toolcore.ToolMetadata {
	return toolcore.ToolMetadata{
		Source: "builtin",
		Capabilities: []string{
			"web.fetch",
			"http.get",
			"research.web",
		},
		Risk: toolcore.RiskMedium,
	}
}

func (t *OpenTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"url": map[string]interface{}{
				"type":        "string",
				"description": "URL to fetch",
			},
			"ref_id": map[string]interface{}{
				"type":        "string",
				"description": "Reference id from search/open output, or URL string",
			},
			"lineno": map[string]interface{}{
				"type":        "integer",
				"description": "Optional line number for excerpt when opening an existing ref",
			},
			"open": map[string]interface{}{
				"type":        "array",
				"description": "Batch open mode",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"url":    map[string]interface{}{"type": "string"},
						"ref_id": map[string]interface{}{"type": "string"},
						"lineno": map[string]interface{}{"type": "integer"},
					},
				},
			},
		},
	}
}

type openInput struct {
	URL    string `json:"url"`
	RefID  string `json:"ref_id"`
	Lineno int    `json:"lineno"`
}

type openRequest struct {
	URL    string      `json:"url"`
	RefID  string      `json:"ref_id"`
	Lineno int         `json:"lineno"`
	Open   []openInput `json:"open"`
}

func (t *OpenTool) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	var args openRequest
	if err := json.Unmarshal(input, &args); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	if len(args.Open) > 0 {
		results := make([]map[string]interface{}, 0, len(args.Open))
		for _, req := range args.Open {
			result, err := t.executeOne(ctx, req)
			if err != nil {
				return nil, err
			}
			results = append(results, result)
		}
		return json.Marshal(map[string]interface{}{"results": results})
	}

	result, err := t.executeOne(ctx, openInput{
		URL:    args.URL,
		RefID:  args.RefID,
		Lineno: args.Lineno,
	})
	if err != nil {
		return nil, err
	}
	return json.Marshal(result)
}

func (t *OpenTool) executeOne(ctx context.Context, args openInput) (map[string]interface{}, error) {
	if strings.TrimSpace(args.URL) == "" && strings.TrimSpace(args.RefID) != "" {
		refID := strings.TrimSpace(args.RefID)
		if ref, ok := getWebPage(refID); ok {
			resp := map[string]interface{}{
				"ref_id":  ref.RefID,
				"url":     ref.URL,
				"status":  ref.Status,
				"content": ref.Content,
				"links":   ref.Links,
			}
			if args.Lineno > 0 {
				resp["excerpt"] = lineExcerpt(ref.Content, args.Lineno, 2)
			}
			return resp, nil
		}

		if searchRef, ok := getWebSearchRef(refID); ok {
			args.URL = searchRef.URL
		} else if isLikelyURL(refID) {
			args.URL = refID
		} else {
			return nil, fmt.Errorf("ref_id not found")
		}
	}

	rawURL := strings.TrimSpace(args.URL)
	if rawURL == "" {
		return nil, fmt.Errorf("url or ref_id is required")
	}
	parsedURL, err := url.Parse(rawURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, fmt.Errorf("invalid url")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Heike/1.0 (+https://example.invalid)")

	client := t.Client
	if client == nil {
		client = &http.Client{Timeout: toolcore.DefaultBuiltinWebTimeout}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}

	content := string(body)
	if len(content) > t.maxContentLength {
		content = content[:t.maxContentLength] + "...(truncated)"
	}

	links := parseOpenLinks(parsedURL.String(), string(body))
	refID := storeWebPage(parsedURL.String(), resp.Status, content, links)

	result := map[string]interface{}{
		"ref_id":  refID,
		"url":     parsedURL.String(),
		"status":  resp.Status,
		"content": content,
		"links":   links,
	}
	if args.Lineno > 0 {
		result["excerpt"] = lineExcerpt(content, args.Lineno, 2)
	}
	return result, nil
}

func isLikelyURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	return parsed.Scheme != "" && parsed.Host != ""
}

func lineExcerpt(content string, lineNo int, radius int) string {
	if lineNo <= 0 {
		return ""
	}
	lines := strings.Split(content, "\n")
	if lineNo > len(lines) {
		return ""
	}

	start := lineNo - radius
	if start < 1 {
		start = 1
	}
	end := lineNo + radius
	if end > len(lines) {
		end = len(lines)
	}

	var b strings.Builder
	for i := start; i <= end; i++ {
		b.WriteString(strconv.Itoa(i))
		b.WriteString(": ")
		b.WriteString(lines[i-1])
		if i < end {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
