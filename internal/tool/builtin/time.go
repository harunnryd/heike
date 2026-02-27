package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	toolcore "github.com/harunnryd/heike/internal/tool"
)

func init() {
	toolcore.RegisterBuiltin("time", func(options toolcore.BuiltinOptions) (toolcore.Tool, error) {
		return &TimeTool{}, nil
	})
}

// TimeTool returns the current time.
type TimeTool struct{}

func (t *TimeTool) Name() string {
	return "time"
}

func (t *TimeTool) Description() string {
	return "Get the current time."
}

func (t *TimeTool) ToolMetadata() toolcore.ToolMetadata {
	return toolcore.ToolMetadata{
		Source: "builtin",
		Capabilities: []string{
			"time.query",
			"clock.now",
		},
		Risk: toolcore.RiskLow,
	}
}

func (t *TimeTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"utc_offset": map[string]interface{}{
				"type":        "string",
				"description": "UTC offset like +07:00 (optional)",
			},
			"time": map[string]interface{}{
				"type":        "array",
				"description": "Batch mode input list of utc offsets",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"utc_offset": map[string]interface{}{
							"type":        "string",
							"description": "UTC offset like +07:00 (optional)",
						},
					},
				},
			},
		},
	}
}

func (t *TimeTool) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	_ = ctx

	type timeQuery struct {
		UTCOffset string `json:"utc_offset"`
	}
	var args struct {
		UTCOffset string      `json:"utc_offset"`
		Time      []timeQuery `json:"time"`
	}
	if len(input) > 0 {
		if err := json.Unmarshal(input, &args); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}
	}

	if len(args.Time) > 0 {
		results := make([]map[string]string, 0, len(args.Time))
		for _, q := range args.Time {
			entry, err := currentTimePayload(q.UTCOffset)
			if err != nil {
				return nil, err
			}
			results = append(results, entry)
		}
		return json.Marshal(map[string]interface{}{
			"results": results,
		})
	}

	entry, err := currentTimePayload(args.UTCOffset)
	if err != nil {
		return nil, err
	}
	return json.Marshal(entry)
}

func currentTimePayload(utcOffset string) (map[string]string, error) {
	now := time.Now().UTC()
	offset := strings.TrimSpace(utcOffset)
	if offset != "" {
		parsedOffset, err := parseUTCOffset(offset)
		if err != nil {
			return nil, err
		}
		now = now.Add(time.Duration(parsedOffset) * time.Second)
	}

	return map[string]string{
		"time":       now.Format(time.RFC3339),
		"utc_offset": offsetOrUTC(offset),
	}, nil
}

func parseUTCOffset(offset string) (int, error) {
	if len(offset) != 6 {
		return 0, fmt.Errorf("invalid utc_offset format")
	}
	if offset[0] != '+' && offset[0] != '-' {
		return 0, fmt.Errorf("invalid utc_offset sign")
	}
	if offset[3] != ':' {
		return 0, fmt.Errorf("invalid utc_offset format")
	}
	if offset[1] < '0' || offset[1] > '9' ||
		offset[2] < '0' || offset[2] > '9' ||
		offset[4] < '0' || offset[4] > '9' ||
		offset[5] < '0' || offset[5] > '9' {
		return 0, fmt.Errorf("invalid utc_offset format")
	}

	hours := int(offset[1]-'0')*10 + int(offset[2]-'0')
	minutes := int(offset[4]-'0')*10 + int(offset[5]-'0')
	if hours > 23 || minutes > 59 {
		return 0, fmt.Errorf("invalid utc_offset value")
	}

	totalSeconds := hours*3600 + minutes*60
	if offset[0] == '-' {
		totalSeconds = -totalSeconds
	}
	return totalSeconds, nil
}

func offsetOrUTC(in string) string {
	if strings.TrimSpace(in) == "" {
		return "+00:00"
	}
	return in
}
