package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	toolcore "github.com/harunnryd/heike/internal/tool"
)

const (
	defaultWeatherBaseURL      = "https://wttr.in"
	defaultWeatherDurationDays = 7
	maxWeatherDurationDays     = 14
	maxWeatherBatchSize        = 4
)

type weatherQuery struct {
	Location string `json:"location"`
	Start    string `json:"start"`
	Duration int    `json:"duration"`
}

type weatherRequest struct {
	Location string         `json:"location"`
	Start    string         `json:"start"`
	Duration int            `json:"duration"`
	Weather  []weatherQuery `json:"weather"`
}

type wttrNamedValue struct {
	Value string `json:"value"`
}

type wttrCurrentCondition struct {
	TempC           string           `json:"temp_C"`
	FeelsLikeC      string           `json:"FeelsLikeC"`
	WeatherDesc     []wttrNamedValue `json:"weatherDesc"`
	Humidity        string           `json:"humidity"`
	WindspeedKmph   string           `json:"windspeedKmph"`
	ObservationTime string           `json:"observation_time"`
}

type wttrNearestArea struct {
	AreaName []wttrNamedValue `json:"areaName"`
	Region   []wttrNamedValue `json:"region"`
	Country  []wttrNamedValue `json:"country"`
}

type wttrHourly struct {
	WeatherDesc []wttrNamedValue `json:"weatherDesc"`
}

type wttrWeatherDay struct {
	Date     string       `json:"date"`
	MaxTempC string       `json:"maxtempC"`
	MinTempC string       `json:"mintempC"`
	AvgTempC string       `json:"avgtempC"`
	Hourly   []wttrHourly `json:"hourly"`
}

type wttrResponse struct {
	CurrentCondition []wttrCurrentCondition `json:"current_condition"`
	NearestArea      []wttrNearestArea      `json:"nearest_area"`
	Weather          []wttrWeatherDay       `json:"weather"`
}

func init() {
	toolcore.RegisterBuiltin("weather", func(options toolcore.BuiltinOptions) (toolcore.Tool, error) {
		timeout := options.WeatherTimeout
		if timeout <= 0 {
			timeout = options.WebTimeout
		}
		if timeout <= 0 {
			timeout = toolcore.DefaultBuiltinWebTimeout
		}

		baseURL := strings.TrimSpace(options.WeatherBaseURL)
		if baseURL == "" {
			baseURL = defaultWeatherBaseURL
		}

		return &WeatherTool{
			Client:  &http.Client{Timeout: timeout},
			BaseURL: baseURL,
		}, nil
	})
}

// WeatherTool fetches weather data by location.
type WeatherTool struct {
	Client  *http.Client
	BaseURL string
}

func (t *WeatherTool) Name() string { return "weather" }

func (t *WeatherTool) Description() string {
	return "Get weather forecast by location."
}

func (t *WeatherTool) ToolMetadata() toolcore.ToolMetadata {
	return toolcore.ToolMetadata{
		Source: "builtin",
		Capabilities: []string{
			"weather.query",
			"http.get",
			"research.web",
		},
		Risk: toolcore.RiskMedium,
	}
}

func (t *WeatherTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"location": map[string]interface{}{
				"type":        "string",
				"description": "Location in text format (for example: San Francisco, CA)",
			},
			"start": map[string]interface{}{
				"type":        "string",
				"description": "Optional start date in YYYY-MM-DD format",
			},
			"duration": map[string]interface{}{
				"type":        "integer",
				"description": "Optional number of days to include (default 7)",
			},
			"weather": map[string]interface{}{
				"type":        "array",
				"description": "Batch weather queries",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"location": map[string]interface{}{"type": "string"},
						"start":    map[string]interface{}{"type": "string"},
						"duration": map[string]interface{}{"type": "integer"},
					},
				},
			},
		},
		"required": []string{"location"},
	}
}

func (t *WeatherTool) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	var args weatherRequest
	if err := json.Unmarshal(input, &args); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	if len(args.Weather) > 0 {
		if len(args.Weather) > maxWeatherBatchSize {
			return nil, fmt.Errorf("weather supports at most %d queries per call", maxWeatherBatchSize)
		}

		results := make([]map[string]interface{}, 0, len(args.Weather))
		for _, query := range args.Weather {
			result, err := t.executeOne(ctx, query)
			if err != nil {
				return nil, err
			}
			results = append(results, result)
		}

		return json.Marshal(map[string]interface{}{
			"results": results,
		})
	}

	result, err := t.executeOne(ctx, weatherQuery{
		Location: args.Location,
		Start:    args.Start,
		Duration: args.Duration,
	})
	if err != nil {
		return nil, err
	}

	return json.Marshal(result)
}

func (t *WeatherTool) executeOne(ctx context.Context, query weatherQuery) (map[string]interface{}, error) {
	location := strings.TrimSpace(query.Location)
	if location == "" {
		return nil, fmt.Errorf("location is required")
	}

	endpoint, err := weatherEndpoint(t.BaseURL, location)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
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

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("weather request failed: %s", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}

	var payload wttrResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode weather response: %w", err)
	}
	if len(payload.CurrentCondition) == 0 {
		return nil, fmt.Errorf("weather response missing current condition")
	}

	selectedDays, effectiveStart, err := selectForecastDays(payload.Weather, query.Start, query.Duration)
	if err != nil {
		return nil, err
	}

	forecast := make([]map[string]string, 0, len(selectedDays))
	for _, day := range selectedDays {
		forecast = append(forecast, map[string]string{
			"date":       strings.TrimSpace(day.Date),
			"min_temp_c": strings.TrimSpace(day.MinTempC),
			"max_temp_c": strings.TrimSpace(day.MaxTempC),
			"avg_temp_c": strings.TrimSpace(day.AvgTempC),
			"condition":  firstHourlyWeatherDescription(day.Hourly),
		})
	}

	current := payload.CurrentCondition[0]
	return map[string]interface{}{
		"query_location": location,
		"location":       resolveWeatherLocation(payload.NearestArea, location),
		"start":          effectiveStart,
		"duration":       normalizeWeatherDuration(query.Duration),
		"current": map[string]string{
			"temperature_c":        strings.TrimSpace(current.TempC),
			"feels_like_c":         strings.TrimSpace(current.FeelsLikeC),
			"condition":            firstNamedValue(current.WeatherDesc),
			"humidity_pct":         strings.TrimSpace(current.Humidity),
			"wind_kmph":            strings.TrimSpace(current.WindspeedKmph),
			"observation_time_utc": strings.TrimSpace(current.ObservationTime),
		},
		"forecast": forecast,
	}, nil
}

func weatherEndpoint(baseURL string, location string) (string, error) {
	base := strings.TrimSpace(baseURL)
	if base == "" {
		base = defaultWeatherBaseURL
	}

	parsed, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid weather endpoint: %w", err)
	}
	if strings.TrimSpace(parsed.Scheme) == "" || strings.TrimSpace(parsed.Host) == "" {
		return "", fmt.Errorf("invalid weather endpoint")
	}

	parsed.Path = strings.TrimSuffix(parsed.Path, "/") + "/" + url.PathEscape(strings.TrimSpace(location))
	q := parsed.Query()
	q.Set("format", "j1")
	parsed.RawQuery = q.Encode()

	return parsed.String(), nil
}

func normalizeWeatherDuration(duration int) int {
	if duration <= 0 {
		return defaultWeatherDurationDays
	}
	if duration > maxWeatherDurationDays {
		return maxWeatherDurationDays
	}
	return duration
}

func selectForecastDays(days []wttrWeatherDay, start string, duration int) ([]wttrWeatherDay, string, error) {
	if len(days) == 0 {
		return []wttrWeatherDay{}, "", nil
	}

	effectiveDuration := normalizeWeatherDuration(duration)
	effectiveStart := strings.TrimSpace(days[0].Date)
	startDateStr := strings.TrimSpace(start)

	startIdx := 0
	if startDateStr != "" {
		startDate, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			return nil, "", fmt.Errorf("start must use YYYY-MM-DD format")
		}

		found := false
		for i, day := range days {
			dayDate, err := time.Parse("2006-01-02", strings.TrimSpace(day.Date))
			if err != nil {
				continue
			}
			if !dayDate.Before(startDate) {
				startIdx = i
				effectiveStart = strings.TrimSpace(day.Date)
				found = true
				break
			}
		}

		if !found {
			return []wttrWeatherDay{}, startDateStr, nil
		}
	}

	endIdx := startIdx + effectiveDuration
	if endIdx > len(days) {
		endIdx = len(days)
	}

	selected := days[startIdx:endIdx]
	if len(selected) > 0 {
		effectiveStart = strings.TrimSpace(selected[0].Date)
	}

	return selected, effectiveStart, nil
}

func resolveWeatherLocation(nearest []wttrNearestArea, fallback string) string {
	if len(nearest) == 0 {
		return strings.TrimSpace(fallback)
	}

	parts := []string{
		firstNamedValue(nearest[0].AreaName),
		firstNamedValue(nearest[0].Region),
		firstNamedValue(nearest[0].Country),
	}

	nonEmpty := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
		}
		nonEmpty = append(nonEmpty, part)
	}
	if len(nonEmpty) == 0 {
		return strings.TrimSpace(fallback)
	}
	return strings.Join(nonEmpty, ", ")
}

func firstNamedValue(values []wttrNamedValue) string {
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(values[0].Value)
}

func firstHourlyWeatherDescription(hourly []wttrHourly) string {
	if len(hourly) == 0 {
		return ""
	}
	return firstNamedValue(hourly[0].WeatherDesc)
}
