package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	toolcore "github.com/harunnryd/heike/internal/tool"
)

type clickInput struct {
	RefID string `json:"ref_id"`
	ID    int    `json:"id"`
}

type clickRequest struct {
	RefID string       `json:"ref_id"`
	ID    int          `json:"id"`
	Click []clickInput `json:"click"`
}

type findInput struct {
	RefID         string `json:"ref_id"`
	Pattern       string `json:"pattern"`
	CaseSensitive bool   `json:"case_sensitive"`
}

type findRequest struct {
	RefID         string      `json:"ref_id"`
	Pattern       string      `json:"pattern"`
	CaseSensitive bool        `json:"case_sensitive"`
	Find          []findInput `json:"find"`
}

func init() {
	toolcore.RegisterBuiltin("click", func(options toolcore.BuiltinOptions) (toolcore.Tool, error) {
		return &ClickTool{}, nil
	})
	toolcore.RegisterBuiltin("find", func(options toolcore.BuiltinOptions) (toolcore.Tool, error) {
		return &FindTool{}, nil
	})
}

// ClickTool resolves a link from a previously opened page reference.
type ClickTool struct{}

func (t *ClickTool) Name() string { return "click" }

func (t *ClickTool) Description() string {
	return "Resolve a link from a previously opened page reference."
}

func (t *ClickTool) ToolMetadata() toolcore.ToolMetadata {
	return toolcore.ToolMetadata{
		Source: "builtin",
		Capabilities: []string{
			"web.navigate.link",
			"web.resolve.link",
			"research.web",
		},
		Risk: toolcore.RiskLow,
	}
}

func (t *ClickTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"ref_id": map[string]interface{}{
				"type":        "string",
				"description": "Reference id from open tool",
			},
			"id": map[string]interface{}{
				"type":        "integer",
				"description": "Link id from open response",
			},
			"click": map[string]interface{}{
				"type":        "array",
				"description": "Batch click mode",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"ref_id": map[string]interface{}{"type": "string"},
						"id":     map[string]interface{}{"type": "integer"},
					},
					"required": []string{"ref_id", "id"},
				},
			},
		},
	}
}

func (t *ClickTool) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	_ = ctx

	var args clickRequest
	if err := json.Unmarshal(input, &args); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	if len(args.Click) > 0 {
		results := make([]map[string]interface{}, 0, len(args.Click))
		for _, req := range args.Click {
			result, err := t.executeOne(req)
			if err != nil {
				return nil, err
			}
			results = append(results, result)
		}
		return json.Marshal(map[string]interface{}{"results": results})
	}

	result, err := t.executeOne(clickInput{RefID: args.RefID, ID: args.ID})
	if err != nil {
		return nil, err
	}
	return json.Marshal(result)
}

func (t *ClickTool) executeOne(args clickInput) (map[string]interface{}, error) {
	if strings.TrimSpace(args.RefID) == "" {
		return nil, fmt.Errorf("ref_id is required")
	}
	if args.ID <= 0 {
		return nil, fmt.Errorf("id must be >= 1")
	}

	ref, ok := getWebPage(strings.TrimSpace(args.RefID))
	if !ok {
		return nil, fmt.Errorf("ref_id not found")
	}

	for _, link := range ref.Links {
		idValue, ok := link["id"].(int)
		if !ok {
			if idFloat, okFloat := link["id"].(float64); okFloat {
				idValue = int(idFloat)
				ok = true
			}
		}
		if !ok || idValue != args.ID {
			continue
		}

		urlValue, _ := link["url"].(string)
		textValue, _ := link["text"].(string)
		return map[string]interface{}{
			"ref_id":    ref.RefID,
			"id":        args.ID,
			"url":       urlValue,
			"text":      textValue,
			"next_step": "call open with ref_id=url",
		}, nil
	}

	return nil, fmt.Errorf("link id not found")
}

// FindTool searches a text pattern inside previously opened page content.
type FindTool struct{}

func (t *FindTool) Name() string { return "find" }

func (t *FindTool) Description() string {
	return "Find text pattern inside a page referenced by ref_id."
}

func (t *FindTool) ToolMetadata() toolcore.ToolMetadata {
	return toolcore.ToolMetadata{
		Source: "builtin",
		Capabilities: []string{
			"web.find.text",
			"document.find",
			"research.web",
		},
		Risk: toolcore.RiskLow,
	}
}

func (t *FindTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"ref_id": map[string]interface{}{
				"type":        "string",
				"description": "Reference id from open tool",
			},
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "Pattern text to find",
			},
			"case_sensitive": map[string]interface{}{
				"type":        "boolean",
				"description": "Optional case-sensitive search flag",
			},
			"find": map[string]interface{}{
				"type":        "array",
				"description": "Batch find mode",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"ref_id":         map[string]interface{}{"type": "string"},
						"pattern":        map[string]interface{}{"type": "string"},
						"case_sensitive": map[string]interface{}{"type": "boolean"},
					},
					"required": []string{"ref_id", "pattern"},
				},
			},
		},
	}
}

func (t *FindTool) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	_ = ctx

	var args findRequest
	if err := json.Unmarshal(input, &args); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	if len(args.Find) > 0 {
		results := make([]map[string]interface{}, 0, len(args.Find))
		for _, req := range args.Find {
			result, err := t.executeOne(req)
			if err != nil {
				return nil, err
			}
			results = append(results, result)
		}
		return json.Marshal(map[string]interface{}{"results": results})
	}

	result, err := t.executeOne(findInput{
		RefID:         args.RefID,
		Pattern:       args.Pattern,
		CaseSensitive: args.CaseSensitive,
	})
	if err != nil {
		return nil, err
	}
	return json.Marshal(result)
}

func (t *FindTool) executeOne(args findInput) (map[string]interface{}, error) {
	refID := strings.TrimSpace(args.RefID)
	if refID == "" {
		return nil, fmt.Errorf("ref_id is required")
	}
	pattern := strings.TrimSpace(args.Pattern)
	if pattern == "" {
		return nil, fmt.Errorf("pattern is required")
	}

	ref, ok := getWebPage(refID)
	if !ok {
		return nil, fmt.Errorf("ref_id not found")
	}

	query := pattern
	if !args.CaseSensitive {
		query = strings.ToLower(query)
	}

	lines := strings.Split(ref.Content, "\n")
	matches := make([]map[string]interface{}, 0, 10)
	for idx, line := range lines {
		comp := line
		if !args.CaseSensitive {
			comp = strings.ToLower(comp)
		}
		col := strings.Index(comp, query)
		if col < 0 {
			continue
		}
		matches = append(matches, map[string]interface{}{
			"line":   idx + 1,
			"column": col + 1,
			"text":   line,
		})
		if len(matches) >= 50 {
			break
		}
	}

	return map[string]interface{}{
		"ref_id":  refID,
		"pattern": pattern,
		"found":   len(matches) > 0,
		"matches": matches,
	}, nil
}
