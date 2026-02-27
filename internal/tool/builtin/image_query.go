package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"

	toolcore "github.com/harunnryd/heike/internal/tool"
)

const (
	defaultImageQueryBaseURL = "https://commons.wikimedia.org/w/api.php"
	defaultImageQueryLimit   = 8
	maxImageQueryBatchSize   = 2
	maxImageQueryLimit       = 20
)

type imageQueryInput struct {
	Query   string   `json:"query"`
	Q       string   `json:"q"`
	Domains []string `json:"domains"`
	Recency int      `json:"recency"`
	Limit   int      `json:"limit"`
}

type imageQueryRequest struct {
	Query      string            `json:"query"`
	Q          string            `json:"q"`
	Domains    []string          `json:"domains"`
	Recency    int               `json:"recency"`
	Limit      int               `json:"limit"`
	ImageQuery []imageQueryInput `json:"image_query"`
}

type wikimediaImageInfo struct {
	URL      string `json:"url"`
	ThumbURL string `json:"thumburl"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

type wikimediaPage struct {
	Title     string               `json:"title"`
	ImageInfo []wikimediaImageInfo `json:"imageinfo"`
}

type wikimediaResponse struct {
	Query struct {
		Pages map[string]wikimediaPage `json:"pages"`
	} `json:"query"`
}

func init() {
	toolcore.RegisterBuiltin("image_query", func(options toolcore.BuiltinOptions) (toolcore.Tool, error) {
		timeout := options.ImageQueryTimeout
		if timeout <= 0 {
			timeout = options.WebTimeout
		}
		if timeout <= 0 {
			timeout = toolcore.DefaultBuiltinWebTimeout
		}

		baseURL := strings.TrimSpace(options.ImageQueryBaseURL)
		if baseURL == "" {
			baseURL = defaultImageQueryBaseURL
		}

		return &ImageQueryTool{
			Client:  &http.Client{Timeout: timeout},
			BaseURL: baseURL,
		}, nil
	})
}

// ImageQueryTool searches image results for a text query.
type ImageQueryTool struct {
	Client  *http.Client
	BaseURL string
}

func (t *ImageQueryTool) Name() string { return "image_query" }

func (t *ImageQueryTool) Description() string {
	return "Search image results for a query string."
}

func (t *ImageQueryTool) ToolMetadata() toolcore.ToolMetadata {
	return toolcore.ToolMetadata{
		Source: "builtin",
		Capabilities: []string{
			"image.search",
			"http.get",
			"research.web",
		},
		Risk: toolcore.RiskMedium,
	}
}

func (t *ImageQueryTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Image search query",
			},
			"q": map[string]interface{}{
				"type":        "string",
				"description": "Compatibility alias for query",
			},
			"domains": map[string]interface{}{
				"type":        "array",
				"description": "Optional domain filter list",
				"items":       map[string]interface{}{"type": "string"},
			},
			"recency": map[string]interface{}{
				"type":        "integer",
				"description": "Optional recency in days (best effort)",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum image results per query (default 8, max 20)",
			},
			"image_query": map[string]interface{}{
				"type":        "array",
				"description": "Batch mode, max 2 queries",
				"items":       map[string]interface{}{"type": "object"},
			},
		},
	}
}

func (t *ImageQueryTool) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	var args imageQueryRequest
	if err := json.Unmarshal(input, &args); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	if len(args.ImageQuery) > 0 {
		if len(args.ImageQuery) > maxImageQueryBatchSize {
			return nil, fmt.Errorf("image_query supports at most %d queries per call", maxImageQueryBatchSize)
		}
		results := make([]map[string]interface{}, 0, len(args.ImageQuery))
		for _, query := range args.ImageQuery {
			result, err := t.executeOne(ctx, imageQueryInput{
				Query:   normalizeQuery(query.Query, query.Q),
				Domains: inheritDomains(query.Domains, args.Domains),
				Recency: effectiveRecency(query.Recency, args.Recency),
				Limit:   effectiveImageQueryLimit(query.Limit, args.Limit),
			})
			if err != nil {
				return nil, err
			}
			results = append(results, result)
		}
		return json.Marshal(map[string]interface{}{"results": results})
	}

	result, err := t.executeOne(ctx, imageQueryInput{
		Query:   normalizeQuery(args.Query, args.Q),
		Domains: args.Domains,
		Recency: args.Recency,
		Limit:   effectiveImageQueryLimit(args.Limit, 0),
	})
	if err != nil {
		return nil, err
	}
	return json.Marshal(result)
}

func (t *ImageQueryTool) executeOne(ctx context.Context, query imageQueryInput) (map[string]interface{}, error) {
	qText := strings.TrimSpace(query.Query)
	if qText == "" {
		return nil, fmt.Errorf("query or q is required")
	}

	baseURL := strings.TrimSpace(t.BaseURL)
	if baseURL == "" {
		baseURL = defaultImageQueryBaseURL
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid image endpoint: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid image endpoint")
	}

	searchQuery := qText
	if query.Recency > 0 {
		searchQuery += " recent"
	}

	q := parsed.Query()
	q.Set("action", "query")
	q.Set("format", "json")
	q.Set("generator", "search")
	q.Set("gsrsearch", searchQuery)
	q.Set("gsrnamespace", "6")
	q.Set("gsrlimit", fmt.Sprintf("%d", query.Limit))
	q.Set("prop", "imageinfo")
	q.Set("iiprop", "url|size")
	q.Set("origin", "*")
	parsed.RawQuery = q.Encode()

	client := t.Client
	if client == nil {
		client = &http.Client{Timeout: toolcore.DefaultBuiltinWebTimeout}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
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
		return nil, fmt.Errorf("image_query request failed: %s", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}

	var payload wikimediaResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode image_query response: %w", err)
	}

	results := make([]map[string]interface{}, 0, len(payload.Query.Pages))
	keys := make([]string, 0, len(payload.Query.Pages))
	for key := range payload.Query.Pages {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		page := payload.Query.Pages[key]
		if len(page.ImageInfo) == 0 {
			continue
		}
		info := page.ImageInfo[0]
		imageURL := strings.TrimSpace(info.URL)
		if imageURL == "" {
			continue
		}
		if !matchesImageDomains(imageURL, query.Domains) {
			continue
		}

		results = append(results, map[string]interface{}{
			"title":          strings.TrimSpace(page.Title),
			"url":            imageURL,
			"thumbnail_url":  strings.TrimSpace(info.ThumbURL),
			"width":          info.Width,
			"height":         info.Height,
			"source_domain":  imageHost(imageURL),
			"source_dataset": "wikimedia_commons",
		})
	}

	return map[string]interface{}{
		"query":        qText,
		"domains":      query.Domains,
		"recency_days": query.Recency,
		"results":      results,
	}, nil
}

func effectiveImageQueryLimit(primary, fallback int) int {
	limit := primary
	if limit <= 0 {
		limit = fallback
	}
	if limit <= 0 {
		limit = defaultImageQueryLimit
	}
	if limit > maxImageQueryLimit {
		limit = maxImageQueryLimit
	}
	return limit
}

func matchesImageDomains(imageURL string, domains []string) bool {
	if len(domains) == 0 {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(imageHost(imageURL)))
	if host == "" {
		return false
	}

	for _, rawDomain := range domains {
		domain := strings.ToLower(strings.TrimSpace(rawDomain))
		if domain == "" {
			continue
		}
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return true
		}
	}
	return false
}

func imageHost(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return ""
	}
	return parsed.Hostname()
}
