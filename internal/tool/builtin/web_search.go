package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	toolcore "github.com/harunnryd/heike/internal/tool"
)

const (
	defaultWebSearchBaseURL    = "https://www.bing.com/search"
	defaultWebSearchMaxResults = 5
	maxWebSearchResultsHardCap = 10
	maxWebSearchBatchSize      = 4
)

var (
	bingResultRe = regexp.MustCompile(`(?is)<li[^>]*class="[^"]*\bb_algo\b[^"]*"[^>]*>.*?<h2[^>]*>\s*<a[^>]*href="([^"]+)"[^>]*>(.*?)</a>`)
	htmlTagRe    = regexp.MustCompile(`(?is)<[^>]+>`)
)

type webSearchQuery struct {
	Query      string   `json:"query"`
	Q          string   `json:"q"`
	Domains    []string `json:"domains"`
	Recency    int      `json:"recency"`
	MaxResults int      `json:"max_results"`
}

type webSearchInput struct {
	Query          string           `json:"query"`
	Q              string           `json:"q"`
	Domains        []string         `json:"domains"`
	Recency        int              `json:"recency"`
	MaxResults     int              `json:"max_results"`
	SearchQuery    []webSearchQuery `json:"search_query"`
	ResponseLength string           `json:"response_length"`
}

type WebSearchTool struct {
	Client     *http.Client
	BaseURL    string
	MaxResults int
}

func init() {
	toolcore.RegisterBuiltin("search_query", func(options toolcore.BuiltinOptions) (toolcore.Tool, error) {
		timeout := options.WebTimeout
		if timeout <= 0 {
			timeout = toolcore.DefaultBuiltinWebTimeout
		}
		baseURL := strings.TrimSpace(options.WebBaseURL)
		if baseURL == "" {
			baseURL = defaultWebSearchBaseURL
		}

		return &WebSearchTool{
			Client:     &http.Client{Timeout: timeout},
			BaseURL:    baseURL,
			MaxResults: defaultWebSearchMaxResults,
		}, nil
	})
}

func (t *WebSearchTool) Name() string {
	return "search_query"
}

func (t *WebSearchTool) Description() string {
	return "Search the web and return top result links."
}

func (t *WebSearchTool) ToolMetadata() toolcore.ToolMetadata {
	return toolcore.ToolMetadata{
		Source: "builtin",
		Capabilities: []string{
			"web.search",
			"research.web",
			"http.get",
		},
		Risk: toolcore.RiskMedium,
	}
}

func (t *WebSearchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Search query text",
			},
			"q": map[string]interface{}{
				"type":        "string",
				"description": "Compatibility alias for query",
			},
			"domains": map[string]interface{}{
				"type":        "array",
				"description": "Optional domain filters (e.g. [\"example.com\"])",
				"items": map[string]interface{}{
					"type": "string",
				},
			},
			"recency": map[string]interface{}{
				"type":        "integer",
				"description": "Optional recency hint in days",
			},
			"max_results": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of links to return (default 5, max 10)",
			},
			"response_length": map[string]interface{}{
				"type":        "string",
				"description": "Optional response length hint (short|medium|long)",
			},
			"search_query": map[string]interface{}{
				"type":        "array",
				"description": "Batch query mode; up to 4 query objects",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{"type": "string"},
						"q":     map[string]interface{}{"type": "string"},
						"domains": map[string]interface{}{
							"type":  "array",
							"items": map[string]interface{}{"type": "string"},
						},
						"recency": map[string]interface{}{"type": "integer"},
						"max_results": map[string]interface{}{
							"type": "integer",
						},
					},
				},
			},
		},
	}
}

func (t *WebSearchTool) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	var args webSearchInput
	if err := json.Unmarshal(input, &args); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	if len(args.SearchQuery) > 0 {
		if len(args.SearchQuery) > maxWebSearchBatchSize {
			return nil, fmt.Errorf("search_query supports at most %d queries per call", maxWebSearchBatchSize)
		}

		batchResults := make([]map[string]interface{}, 0, len(args.SearchQuery))
		for _, query := range args.SearchQuery {
			result, err := t.executeSearchQuery(ctx, normalizeQuery(query.Query, query.Q),
				inheritDomains(query.Domains, args.Domains),
				effectiveRecency(query.Recency, args.Recency),
				effectiveMaxResults(query.MaxResults, args.MaxResults, t.MaxResults),
			)
			if err != nil {
				return nil, err
			}
			batchResults = append(batchResults, result)
		}

		return json.Marshal(map[string]interface{}{
			"results":         batchResults,
			"response_length": strings.TrimSpace(args.ResponseLength),
		})
	}

	result, err := t.executeSearchQuery(ctx, normalizeQuery(args.Query, args.Q), args.Domains, args.Recency,
		effectiveMaxResults(args.MaxResults, 0, t.MaxResults))
	if err != nil {
		return nil, err
	}
	return json.Marshal(result)
}

func (t *WebSearchTool) executeSearchQuery(
	ctx context.Context,
	query string,
	domains []string,
	recency int,
	maxResults int,
) (map[string]interface{}, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("query or q is required")
	}

	baseURL := strings.TrimSpace(t.BaseURL)
	if baseURL == "" {
		baseURL = defaultWebSearchBaseURL
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid search endpoint: %w", err)
	}

	finalQuery := query
	for _, domain := range domains {
		d := strings.TrimSpace(domain)
		if d == "" {
			continue
		}
		finalQuery += " site:" + d
	}

	q := parsed.Query()
	q.Set("q", finalQuery)
	if recencyQFT := bingRecencyQFT(recency); recencyQFT != "" {
		q.Set("qft", recencyQFT)
	}
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
		return nil, fmt.Errorf("search request failed: %s", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}

	results := parseBingSearchResults(string(body), maxResults)
	results = attachSearchRefs(results)

	return map[string]interface{}{
		"query":           query,
		"effective_query": finalQuery,
		"domains":         domains,
		"recency_days":    recency,
		"results":         results,
	}, nil
}

func normalizeQuery(query string, q string) string {
	if strings.TrimSpace(query) != "" {
		return strings.TrimSpace(query)
	}
	return strings.TrimSpace(q)
}

func inheritDomains(primary []string, fallback []string) []string {
	if len(primary) > 0 {
		return primary
	}
	return fallback
}

func effectiveRecency(primary int, fallback int) int {
	if primary > 0 {
		return primary
	}
	return fallback
}

func bingRecencyQFT(recencyDays int) string {
	if recencyDays <= 0 {
		return ""
	}

	minutes := recencyDays * 24 * 60
	if minutes <= 0 {
		return ""
	}

	maxMinutes := 10 * 365 * 24 * 60
	if minutes > maxMinutes {
		minutes = maxMinutes
	}

	return fmt.Sprintf("+filterui:age-lt%d", minutes)
}

func effectiveMaxResults(primary int, fallback int, toolDefault int) int {
	maxResults := primary
	if maxResults <= 0 {
		maxResults = fallback
	}
	if maxResults <= 0 {
		maxResults = toolDefault
	}
	if maxResults <= 0 {
		maxResults = defaultWebSearchMaxResults
	}
	if maxResults > maxWebSearchResultsHardCap {
		maxResults = maxWebSearchResultsHardCap
	}
	return maxResults
}

func attachSearchRefs(results []map[string]string) []map[string]string {
	if len(results) == 0 {
		return results
	}

	withRefs := make([]map[string]string, 0, len(results))
	for _, result := range results {
		title := strings.TrimSpace(result["title"])
		link := strings.TrimSpace(result["url"])
		entry := map[string]string{
			"title": title,
			"url":   link,
		}
		if link != "" {
			entry["ref_id"] = storeWebSearchRef(link, title)
		}
		withRefs = append(withRefs, entry)
	}
	return withRefs
}

func parseBingSearchResults(doc string, maxResults int) []map[string]string {
	if maxResults <= 0 {
		maxResults = defaultWebSearchMaxResults
	}

	matches := bingResultRe.FindAllStringSubmatch(doc, maxResults)
	out := make([]map[string]string, 0, len(matches))
	for _, m := range matches {
		if len(m) < 3 {
			continue
		}
		link := html.UnescapeString(strings.TrimSpace(m[1]))
		title := html.UnescapeString(strings.TrimSpace(htmlTagRe.ReplaceAllString(m[2], "")))
		if link == "" || title == "" {
			continue
		}
		out = append(out, map[string]string{
			"title": title,
			"url":   link,
		})
	}
	return out
}
