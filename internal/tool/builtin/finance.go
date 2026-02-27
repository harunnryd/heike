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

const (
	defaultFinanceBaseURL = "https://query1.finance.yahoo.com/v7/finance/quote"
	maxFinanceBatchSize   = 20
)

type financeInput struct {
	Ticker string `json:"ticker"`
	Type   string `json:"type"`
	Market string `json:"market"`
}

type financeRequest struct {
	Ticker  string         `json:"ticker"`
	Type    string         `json:"type"`
	Market  string         `json:"market"`
	Finance []financeInput `json:"finance"`
}

type yahooQuoteResponse struct {
	QuoteResponse struct {
		Result []yahooQuote `json:"result"`
	} `json:"quoteResponse"`
}

type yahooQuote struct {
	Symbol                   string  `json:"symbol"`
	RegularMarketPrice       float64 `json:"regularMarketPrice"`
	RegularMarketChange      float64 `json:"regularMarketChange"`
	RegularMarketChangePct   float64 `json:"regularMarketChangePercent"`
	Currency                 string  `json:"currency"`
	RegularMarketTime        int64   `json:"regularMarketTime"`
	MarketState              string  `json:"marketState"`
	RegularMarketPreviousClo float64 `json:"regularMarketPreviousClose"`
}

func init() {
	toolcore.RegisterBuiltin("finance", func(options toolcore.BuiltinOptions) (toolcore.Tool, error) {
		timeout := options.FinanceTimeout
		if timeout <= 0 {
			timeout = options.WebTimeout
		}
		if timeout <= 0 {
			timeout = toolcore.DefaultBuiltinWebTimeout
		}

		baseURL := strings.TrimSpace(options.FinanceBaseURL)
		if baseURL == "" {
			baseURL = defaultFinanceBaseURL
		}

		return &FinanceTool{
			Client:  &http.Client{Timeout: timeout},
			BaseURL: baseURL,
		}, nil
	})
}

// FinanceTool retrieves market quote data by ticker symbol.
type FinanceTool struct {
	Client  *http.Client
	BaseURL string
}

func (t *FinanceTool) Name() string { return "finance" }

func (t *FinanceTool) Description() string {
	return "Look up market quote data for stocks, funds, crypto, and indexes."
}

func (t *FinanceTool) ToolMetadata() toolcore.ToolMetadata {
	return toolcore.ToolMetadata{
		Source: "builtin",
		Capabilities: []string{
			"finance.quote",
			"http.get",
			"research.market",
		},
		Risk: toolcore.RiskMedium,
	}
}

func (t *FinanceTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"ticker": map[string]interface{}{
				"type":        "string",
				"description": "Ticker symbol (for example: AMD, BTC)",
			},
			"type": map[string]interface{}{
				"type":        "string",
				"description": "Asset type: equity, fund, crypto, or index",
			},
			"market": map[string]interface{}{
				"type":        "string",
				"description": "Optional market code (for crypto use empty string)",
			},
			"finance": map[string]interface{}{
				"type":        "array",
				"description": "Batch lookup mode",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"ticker": map[string]interface{}{"type": "string"},
						"type":   map[string]interface{}{"type": "string"},
						"market": map[string]interface{}{"type": "string"},
					},
					"required": []string{"ticker", "type"},
				},
			},
		},
	}
}

func (t *FinanceTool) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	var args financeRequest
	if err := json.Unmarshal(input, &args); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	if len(args.Finance) > 0 {
		if len(args.Finance) > maxFinanceBatchSize {
			return nil, fmt.Errorf("finance supports at most %d tickers per call", maxFinanceBatchSize)
		}
		results, err := t.executeBatch(ctx, args.Finance)
		if err != nil {
			return nil, err
		}
		return json.Marshal(map[string]interface{}{"results": results})
	}

	if strings.TrimSpace(args.Ticker) == "" {
		return nil, fmt.Errorf("ticker is required")
	}
	if strings.TrimSpace(args.Type) == "" {
		return nil, fmt.Errorf("type is required")
	}

	results, err := t.executeBatch(ctx, []financeInput{{
		Ticker: args.Ticker,
		Type:   args.Type,
		Market: args.Market,
	}})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return json.Marshal(map[string]interface{}{})
	}
	return json.Marshal(results[0])
}

func (t *FinanceTool) executeBatch(ctx context.Context, requests []financeInput) ([]map[string]interface{}, error) {
	type normalized struct {
		Original financeInput
		Symbol   string
	}

	normalizedReqs := make([]normalized, 0, len(requests))
	symbols := make([]string, 0, len(requests))
	seen := make(map[string]struct{}, len(requests))

	for _, req := range requests {
		symbol, err := resolveFinanceSymbol(req)
		if err != nil {
			return nil, err
		}
		normalizedReqs = append(normalizedReqs, normalized{Original: req, Symbol: symbol})

		if _, exists := seen[symbol]; exists {
			continue
		}
		seen[symbol] = struct{}{}
		symbols = append(symbols, symbol)
	}

	quotes, err := t.fetchQuotes(ctx, symbols)
	if err != nil {
		return nil, err
	}

	results := make([]map[string]interface{}, 0, len(normalizedReqs))
	for _, req := range normalizedReqs {
		entry := map[string]interface{}{
			"ticker":          strings.ToUpper(strings.TrimSpace(req.Original.Ticker)),
			"type":            strings.ToLower(strings.TrimSpace(req.Original.Type)),
			"market":          strings.TrimSpace(req.Original.Market),
			"resolved_symbol": req.Symbol,
			"found":           false,
		}

		quote, ok := quotes[req.Symbol]
		if !ok {
			results = append(results, entry)
			continue
		}

		entry["found"] = true
		entry["symbol"] = quote.Symbol
		entry["price"] = quote.RegularMarketPrice
		entry["change"] = quote.RegularMarketChange
		entry["change_percent"] = quote.RegularMarketChangePct
		entry["currency"] = strings.TrimSpace(quote.Currency)
		entry["market_state"] = strings.TrimSpace(quote.MarketState)
		if quote.RegularMarketTime > 0 {
			entry["timestamp"] = time.Unix(quote.RegularMarketTime, 0).UTC().Format(time.RFC3339)
		}
		if quote.RegularMarketPreviousClo != 0 {
			entry["previous_close"] = quote.RegularMarketPreviousClo
		}

		results = append(results, entry)
	}

	return results, nil
}

func (t *FinanceTool) fetchQuotes(ctx context.Context, symbols []string) (map[string]yahooQuote, error) {
	if len(symbols) == 0 {
		return map[string]yahooQuote{}, nil
	}

	baseURL := strings.TrimSpace(t.BaseURL)
	if baseURL == "" {
		baseURL = defaultFinanceBaseURL
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid finance endpoint: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid finance endpoint")
	}

	q := parsed.Query()
	q.Set("symbols", strings.Join(symbols, ","))
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
		return nil, fmt.Errorf("finance request failed: %s", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}

	var payload yahooQuoteResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode finance response: %w", err)
	}

	results := make(map[string]yahooQuote, len(payload.QuoteResponse.Result))
	for _, quote := range payload.QuoteResponse.Result {
		key := strings.ToUpper(strings.TrimSpace(quote.Symbol))
		if key == "" {
			continue
		}
		results[key] = quote
	}
	return results, nil
}

func resolveFinanceSymbol(input financeInput) (string, error) {
	ticker := strings.ToUpper(strings.TrimSpace(input.Ticker))
	if ticker == "" {
		return "", fmt.Errorf("ticker is required")
	}

	assetType := strings.ToLower(strings.TrimSpace(input.Type))
	switch assetType {
	case "equity", "fund", "index":
		return ticker, nil
	case "crypto":
		ticker = strings.ReplaceAll(ticker, "/", "-")
		if strings.Contains(ticker, "-") {
			return ticker, nil
		}
		return ticker + "-USD", nil
	default:
		return "", fmt.Errorf("unsupported finance type: %s", strconv.Quote(assetType))
	}
}
