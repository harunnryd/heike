package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	toolcore "github.com/harunnryd/heike/internal/tool"
)

const (
	defaultSportsBaseURL  = "https://site.api.espn.com/apis/v2/sports"
	defaultSportsNumGames = 20
	maxSportsBatchSize    = 4
)

type sportsInput struct {
	Tool     string `json:"tool"`
	Fn       string `json:"fn"`
	League   string `json:"league"`
	Team     string `json:"team"`
	Opponent string `json:"opponent"`
	DateFrom string `json:"date_from"`
	DateTo   string `json:"date_to"`
	NumGames int    `json:"num_games"`
	Locale   string `json:"locale"`
}

type sportsRequest struct {
	Tool     string        `json:"tool"`
	Fn       string        `json:"fn"`
	League   string        `json:"league"`
	Team     string        `json:"team"`
	Opponent string        `json:"opponent"`
	DateFrom string        `json:"date_from"`
	DateTo   string        `json:"date_to"`
	NumGames int           `json:"num_games"`
	Locale   string        `json:"locale"`
	Sports   []sportsInput `json:"sports"`
}

func init() {
	toolcore.RegisterBuiltin("sports", func(options toolcore.BuiltinOptions) (toolcore.Tool, error) {
		timeout := options.SportsTimeout
		if timeout <= 0 {
			timeout = options.WebTimeout
		}
		if timeout <= 0 {
			timeout = toolcore.DefaultBuiltinWebTimeout
		}

		baseURL := strings.TrimSpace(options.SportsBaseURL)
		if baseURL == "" {
			baseURL = defaultSportsBaseURL
		}

		return &SportsTool{
			Client:  &http.Client{Timeout: timeout},
			BaseURL: baseURL,
		}, nil
	})
}

// SportsTool retrieves standings and schedule information by league.
type SportsTool struct {
	Client  *http.Client
	BaseURL string
}

func (t *SportsTool) Name() string { return "sports" }

func (t *SportsTool) Description() string {
	return "Look up sports standings or schedules by league."
}

func (t *SportsTool) ToolMetadata() toolcore.ToolMetadata {
	return toolcore.ToolMetadata{
		Source: "builtin",
		Capabilities: []string{
			"sports.lookup",
			"http.get",
			"research.live_data",
		},
		Risk: toolcore.RiskMedium,
	}
}

func (t *SportsTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"tool": map[string]interface{}{
				"type":        "string",
				"description": "Optional compatibility field, should be \"sports\"",
			},
			"fn": map[string]interface{}{
				"type":        "string",
				"description": "Function: schedule or standings",
			},
			"league": map[string]interface{}{
				"type":        "string",
				"description": "League code (nba, nfl, mlb, nhl, epl, ncaamb, ncaawb, wnba, ipl)",
			},
			"team": map[string]interface{}{
				"type":        "string",
				"description": "Optional team alias filter",
			},
			"opponent": map[string]interface{}{
				"type":        "string",
				"description": "Optional opponent filter (schedule)",
			},
			"date_from": map[string]interface{}{
				"type":        "string",
				"description": "Optional start date (YYYY-MM-DD)",
			},
			"date_to": map[string]interface{}{
				"type":        "string",
				"description": "Optional end date (YYYY-MM-DD)",
			},
			"num_games": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of schedule games to return (default 20)",
			},
			"locale": map[string]interface{}{
				"type":        "string",
				"description": "Optional locale (for example en-US)",
			},
			"sports": map[string]interface{}{
				"type":        "array",
				"description": "Batch mode",
				"items": map[string]interface{}{
					"type": "object",
				},
			},
		},
	}
}

func (t *SportsTool) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	var args sportsRequest
	if err := json.Unmarshal(input, &args); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	if len(args.Sports) > 0 {
		if len(args.Sports) > maxSportsBatchSize {
			return nil, fmt.Errorf("sports supports at most %d requests per call", maxSportsBatchSize)
		}
		results := make([]map[string]interface{}, 0, len(args.Sports))
		for _, req := range args.Sports {
			result, err := t.executeOne(ctx, req)
			if err != nil {
				return nil, err
			}
			results = append(results, result)
		}
		return json.Marshal(map[string]interface{}{"results": results})
	}

	result, err := t.executeOne(ctx, sportsInput{
		Tool:     args.Tool,
		Fn:       args.Fn,
		League:   args.League,
		Team:     args.Team,
		Opponent: args.Opponent,
		DateFrom: args.DateFrom,
		DateTo:   args.DateTo,
		NumGames: args.NumGames,
		Locale:   args.Locale,
	})
	if err != nil {
		return nil, err
	}
	return json.Marshal(result)
}

func (t *SportsTool) executeOne(ctx context.Context, req sportsInput) (map[string]interface{}, error) {
	fn := strings.ToLower(strings.TrimSpace(req.Fn))
	if fn == "" {
		return nil, fmt.Errorf("fn is required")
	}
	if fn != "schedule" && fn != "standings" {
		return nil, fmt.Errorf("unsupported sports fn: %s", fn)
	}

	sportPath, leaguePath, err := sportsLeaguePath(strings.ToLower(strings.TrimSpace(req.League)))
	if err != nil {
		return nil, err
	}

	parsedBase, err := url.Parse(strings.TrimSpace(t.BaseURL))
	if err != nil {
		return nil, fmt.Errorf("invalid sports endpoint: %w", err)
	}
	if parsedBase.Scheme == "" || parsedBase.Host == "" {
		return nil, fmt.Errorf("invalid sports endpoint")
	}

	pathSuffix := "/scoreboard"
	if fn == "standings" {
		pathSuffix = "/standings"
	}
	parsedBase.Path = strings.TrimSuffix(parsedBase.Path, "/") + "/" + sportPath + "/" + leaguePath + pathSuffix

	query := parsedBase.Query()
	if strings.TrimSpace(req.Team) != "" {
		query.Set("team", strings.TrimSpace(req.Team))
	}
	if strings.TrimSpace(req.Locale) != "" {
		query.Set("lang", strings.TrimSpace(req.Locale))
	}
	if dates := sportsDateRange(strings.TrimSpace(req.DateFrom), strings.TrimSpace(req.DateTo)); dates != "" {
		query.Set("dates", dates)
	}
	parsedBase.RawQuery = query.Encode()

	client := t.Client
	if client == nil {
		client = &http.Client{Timeout: toolcore.DefaultBuiltinWebTimeout}
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedBase.String(), nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("User-Agent", "Heike/1.0 (+https://example.invalid)")

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("sports request failed: %s", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode sports response: %w", err)
	}

	result := map[string]interface{}{
		"fn":        fn,
		"league":    strings.ToLower(strings.TrimSpace(req.League)),
		"team":      strings.TrimSpace(req.Team),
		"opponent":  strings.TrimSpace(req.Opponent),
		"date_from": strings.TrimSpace(req.DateFrom),
		"date_to":   strings.TrimSpace(req.DateTo),
	}

	switch fn {
	case "schedule":
		numGames := req.NumGames
		if numGames <= 0 {
			numGames = defaultSportsNumGames
		}
		result["results"] = parseSportsSchedule(payload, req.Team, req.Opponent, numGames)
	case "standings":
		result["results"] = parseSportsStandings(payload)
	}

	return result, nil
}

func sportsLeaguePath(league string) (string, string, error) {
	switch league {
	case "nba":
		return "basketball", "nba", nil
	case "wnba":
		return "basketball", "wnba", nil
	case "nfl":
		return "football", "nfl", nil
	case "nhl":
		return "hockey", "nhl", nil
	case "mlb":
		return "baseball", "mlb", nil
	case "epl":
		return "soccer", "eng.1", nil
	case "ncaamb":
		return "basketball", "mens-college-basketball", nil
	case "ncaawb":
		return "basketball", "womens-college-basketball", nil
	case "ipl":
		return "cricket", "ipl", nil
	default:
		return "", "", fmt.Errorf("unsupported league: %s", league)
	}
}

func sportsDateRange(dateFrom, dateTo string) string {
	from := strings.ReplaceAll(strings.TrimSpace(dateFrom), "-", "")
	to := strings.ReplaceAll(strings.TrimSpace(dateTo), "-", "")

	switch {
	case from != "" && to != "":
		return from + "-" + to
	case from != "":
		return from
	case to != "":
		return to
	default:
		return ""
	}
}

func parseSportsSchedule(payload map[string]interface{}, team, opponent string, maxGames int) []map[string]interface{} {
	events, _ := payload["events"].([]interface{})
	if len(events) == 0 {
		return []map[string]interface{}{}
	}

	teamFilter := strings.ToLower(strings.TrimSpace(team))
	opponentFilter := strings.ToLower(strings.TrimSpace(opponent))

	results := make([]map[string]interface{}, 0, len(events))
	for _, raw := range events {
		event, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}

		comps, _ := event["competitions"].([]interface{})
		if len(comps) == 0 {
			continue
		}
		comp, ok := comps[0].(map[string]interface{})
		if !ok {
			continue
		}

		competitors, _ := comp["competitors"].([]interface{})
		if len(competitors) < 2 {
			continue
		}

		homeName, homeScore, homeAlias := parseSportsCompetitor(competitors[0])
		awayName, awayScore, awayAlias := parseSportsCompetitor(competitors[1])

		if teamFilter != "" && !sportsTeamMatchesAny(homeName, homeAlias, teamFilter) && !sportsTeamMatchesAny(awayName, awayAlias, teamFilter) {
			continue
		}
		if opponentFilter != "" && !sportsTeamMatchesAny(homeName, homeAlias, opponentFilter) && !sportsTeamMatchesAny(awayName, awayAlias, opponentFilter) {
			continue
		}

		results = append(results, map[string]interface{}{
			"id":         stringValue(event["id"]),
			"name":       stringValue(event["name"]),
			"date":       stringValue(event["date"]),
			"status":     sportsStatusText(event["status"]),
			"home_team":  homeName,
			"away_team":  awayName,
			"home_score": homeScore,
			"away_score": awayScore,
		})

		if maxGames > 0 && len(results) >= maxGames {
			break
		}
	}

	return results
}

func parseSportsCompetitor(raw interface{}) (name string, score string, alias string) {
	competitor, ok := raw.(map[string]interface{})
	if !ok {
		return "", "", ""
	}

	team, _ := competitor["team"].(map[string]interface{})
	name = stringValue(team["shortDisplayName"])
	if strings.TrimSpace(name) == "" {
		name = stringValue(team["displayName"])
	}
	if strings.TrimSpace(name) == "" {
		name = stringValue(team["abbreviation"])
	}
	alias = stringValue(team["abbreviation"])

	score = stringValue(competitor["score"])
	return name, score, alias
}

func sportsStatusText(raw interface{}) string {
	status, ok := raw.(map[string]interface{})
	if !ok {
		return ""
	}

	statusType, _ := status["type"].(map[string]interface{})
	text := stringValue(statusType["description"])
	if strings.TrimSpace(text) == "" {
		text = stringValue(statusType["name"])
	}
	return text
}

func sportsTeamMatches(value string, query string) bool {
	v := strings.ToLower(strings.TrimSpace(value))
	q := strings.ToLower(strings.TrimSpace(query))
	if v == "" || q == "" {
		return false
	}
	return strings.Contains(v, q)
}

func sportsTeamMatchesAny(name, alias, query string) bool {
	return sportsTeamMatches(name, query) || sportsTeamMatches(alias, query)
}

func parseSportsStandings(payload map[string]interface{}) []map[string]interface{} {
	entries := extractSportsStandingEntries(payload)
	results := make([]map[string]interface{}, 0, len(entries))

	for _, raw := range entries {
		entry, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}

		teamName := ""
		if team, ok := entry["team"].(map[string]interface{}); ok {
			teamName = stringValue(team["displayName"])
			if strings.TrimSpace(teamName) == "" {
				teamName = stringValue(team["shortDisplayName"])
			}
			if strings.TrimSpace(teamName) == "" {
				teamName = stringValue(team["abbreviation"])
			}
		}

		rank := ""
		record := ""
		points := ""
		if stats, ok := entry["stats"].([]interface{}); ok {
			for _, rawStat := range stats {
				stat, ok := rawStat.(map[string]interface{})
				if !ok {
					continue
				}
				name := strings.ToLower(strings.TrimSpace(stringValue(stat["name"])))
				display := strings.TrimSpace(stringValue(stat["displayValue"]))
				switch name {
				case "rank":
					rank = display
				case "record", "overall":
					record = display
				case "points", "pts":
					points = display
				}
			}
		}

		results = append(results, map[string]interface{}{
			"team":   teamName,
			"rank":   rank,
			"record": record,
			"points": points,
		})
	}

	return results
}

func extractSportsStandingEntries(payload map[string]interface{}) []interface{} {
	if standings, ok := payload["standings"].([]interface{}); ok && len(standings) > 0 {
		if first, ok := standings[0].(map[string]interface{}); ok {
			if entries, ok := first["entries"].([]interface{}); ok {
				return entries
			}
		}
	}

	if children, ok := payload["children"].([]interface{}); ok {
		for _, rawChild := range children {
			child, ok := rawChild.(map[string]interface{})
			if !ok {
				continue
			}
			standings, ok := child["standings"].(map[string]interface{})
			if !ok {
				continue
			}
			if entries, ok := standings["entries"].([]interface{}); ok {
				return entries
			}
		}
	}

	return []interface{}{}
}

func stringValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%.0f", v)
	default:
		return ""
	}
}
