package task

import (
	"fmt"
	"sort"
	"strings"

	"github.com/harunnryd/heike/internal/tool"
)

type ToolBroker interface {
	Select(goal string, tools []tool.ToolDescriptor) []tool.ToolDescriptor
}

type ExplainableToolBroker interface {
	ToolBroker
	SelectWithMetadata(goal string, tools []tool.ToolDescriptor) ToolSelectionResult
}

type ToolSelectionResult struct {
	Tools   []tool.ToolDescriptor
	Details []ToolSelectionDetail
}

type ToolSelectionDetail struct {
	Name    string
	Score   int
	Reasons []string
}

type DefaultToolBroker struct {
	maxTools int
}

func NewDefaultToolBroker(maxTools int) *DefaultToolBroker {
	return &DefaultToolBroker{maxTools: maxTools}
}

func (b *DefaultToolBroker) Select(goal string, tools []tool.ToolDescriptor) []tool.ToolDescriptor {
	result := b.SelectWithMetadata(goal, tools)
	return result.Tools
}

func (b *DefaultToolBroker) SelectWithMetadata(goal string, tools []tool.ToolDescriptor) ToolSelectionResult {
	if len(tools) == 0 {
		return ToolSelectionResult{}
	}
	if b.maxTools <= 0 || len(tools) <= b.maxTools {
		selected := cloneToolDescriptors(tools)
		details := make([]ToolSelectionDetail, 0, len(selected))
		for _, descriptor := range selected {
			details = append(details, ToolSelectionDetail{
				Name:    descriptor.Definition.Name,
				Score:   0,
				Reasons: []string{"within_budget"},
			})
		}
		return ToolSelectionResult{
			Tools:   selected,
			Details: details,
		}
	}

	catalog := BuildCapabilityCatalog(tools)
	goalTokens := tokenSet(goal)
	lowerGoal := strings.ToLower(goal)

	type scoredTool struct {
		tool    tool.ToolDescriptor
		score   int
		reasons []string
	}

	candidates := make([]scoredTool, 0, len(tools))
	for _, descriptor := range tools {
		name := strings.ToLower(strings.TrimSpace(descriptor.Definition.Name))
		capability, _ := catalog.Find(name)
		score, reasons := scoreCapability(capability, lowerGoal, goalTokens)
		candidates = append(candidates, scoredTool{
			tool:    descriptor,
			score:   score,
			reasons: reasons,
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return strings.ToLower(candidates[i].tool.Definition.Name) < strings.ToLower(candidates[j].tool.Definition.Name)
	})

	limit := b.maxTools
	if limit > len(candidates) {
		limit = len(candidates)
	}

	selected := make([]tool.ToolDescriptor, 0, limit)
	details := make([]ToolSelectionDetail, 0, limit)
	for i := 0; i < limit; i++ {
		selected = append(selected, candidates[i].tool)
		details = append(details, ToolSelectionDetail{
			Name:    candidates[i].tool.Definition.Name,
			Score:   candidates[i].score,
			Reasons: append([]string(nil), candidates[i].reasons...),
		})
	}

	return ToolSelectionResult{
		Tools:   selected,
		Details: details,
	}
}

func cloneToolDescriptors(in []tool.ToolDescriptor) []tool.ToolDescriptor {
	out := make([]tool.ToolDescriptor, len(in))
	copy(out, in)
	return out
}

func tokenSet(text string) map[string]struct{} {
	tokens := tokenize(text)
	out := make(map[string]struct{}, len(tokens))
	for _, t := range tokens {
		out[t] = struct{}{}
	}
	return out
}

func scoreCapability(c Capability, lowerGoal string, goalTokens map[string]struct{}) (int, []string) {
	score := 0
	reasons := []string{}

	tagMatches := countTokenMatches(c.Tags, goalTokens)
	if tagMatches > 0 {
		score += tagMatches * 4
		reasons = append(reasons, fmt.Sprintf("tag_match:%d", tagMatches))
	}

	capabilityTokens := capabilityTokenSet(c.Capabilities)
	capabilityMatches := countSetMatches(capabilityTokens, goalTokens)
	if capabilityMatches > 0 {
		score += capabilityMatches * 6
		reasons = append(reasons, fmt.Sprintf("capability_match:%d", capabilityMatches))
	}

	if strings.Contains(lowerGoal, c.Name) {
		score += 10
		reasons = append(reasons, "direct_name_match")
	}

	if strings.Contains(lowerGoal, "/") || strings.Contains(lowerGoal, "\\") {
		if _, hasFileToken := capabilityTokens["file"]; hasFileToken {
			score += 6
			reasons = append(reasons, "path_context_match")
		}
	}

	if !containsGoalToken(goalTokens, "write", "create", "update", "edit", "modify", "patch", "save", "delete", "exec", "run", "command", "shell") {
		switch c.Risk {
		case tool.RiskHigh:
			score -= 6
			reasons = append(reasons, "risk_penalty_high")
		case tool.RiskMedium:
			score -= 2
			reasons = append(reasons, "risk_penalty_medium")
		}
	}

	// Light boost for builtin/runtime tools when nothing else is obvious.
	if score == 0 {
		switch c.Source {
		case "builtin", "runtime":
			score = 1
			reasons = append(reasons, "safe_baseline")
		}
	}

	if len(reasons) == 0 {
		reasons = append(reasons, "default_scoring")
	}

	return score, reasons
}

func containsGoalToken(goalTokens map[string]struct{}, candidates ...string) bool {
	for _, candidate := range candidates {
		if _, ok := goalTokens[strings.TrimSpace(strings.ToLower(candidate))]; ok {
			return true
		}
	}
	return false
}

func countTokenMatches(tokens []string, goalTokens map[string]struct{}) int {
	seen := map[string]struct{}{}
	matches := 0
	for _, token := range tokens {
		token = strings.TrimSpace(strings.ToLower(token))
		if token == "" {
			continue
		}
		if _, already := seen[token]; already {
			continue
		}
		seen[token] = struct{}{}
		if _, ok := goalTokens[token]; ok {
			matches++
		}
	}
	return matches
}

func capabilityTokenSet(capabilities []string) map[string]struct{} {
	set := make(map[string]struct{})
	for _, capability := range capabilities {
		for _, token := range tokenize(capability) {
			token = strings.TrimSpace(strings.ToLower(token))
			if token == "" {
				continue
			}
			set[token] = struct{}{}
		}
	}
	return set
}

func countSetMatches(tokens map[string]struct{}, goalTokens map[string]struct{}) int {
	matches := 0
	for token := range tokens {
		if _, ok := goalTokens[token]; ok {
			matches++
		}
	}
	return matches
}
