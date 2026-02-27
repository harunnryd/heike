package cognitive

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
)

type plannerParseMode string

const (
	plannerParseModeJSONArray   plannerParseMode = "json_array"
	plannerParseModeJSONObject  plannerParseMode = "json_object"
	plannerParseModeExtracted   plannerParseMode = "json_extracted"
	plannerParseModeLineSplit   plannerParseMode = "line_split"
	plannerParseModeGoalDefault plannerParseMode = "goal_default"
)

type reflectionParseMode string

const (
	reflectionParseModeJSON      reflectionParseMode = "json_object"
	reflectionParseModeExtracted reflectionParseMode = "json_extracted"
	reflectionParseModeHeuristic reflectionParseMode = "heuristic_fallback"
)

type reflectionPayload struct {
	Analysis    string   `json:"analysis"`
	Action      string   `json:"next_action"`
	NewMemories []string `json:"new_memories"`
}

type plannerPayload struct {
	Steps []PlanStep `json:"steps"`
	Plan  []PlanStep `json:"plan"`
	Items []PlanStep `json:"items"`
	Tasks []PlanStep `json:"tasks"`
}

func parsePlannerResponse(raw string, goal string) ([]PlanStep, plannerParseMode) {
	normalized := cleanModelJSON(raw)

	if steps, ok := parsePlanStepArrayJSON(normalized); ok {
		return steps, plannerParseModeJSONArray
	}
	if steps, ok := parsePlanStepObjectJSON(normalized); ok {
		return steps, plannerParseModeJSONObject
	}

	if extracted := extractFirstBalancedJSON(normalized, '[', ']'); extracted != "" {
		if steps, ok := parsePlanStepArrayJSON(extracted); ok {
			return steps, plannerParseModeExtracted
		}
	}
	if extracted := extractFirstBalancedJSON(normalized, '{', '}'); extracted != "" {
		if steps, ok := parsePlanStepObjectJSON(extracted); ok {
			return steps, plannerParseModeExtracted
		}
	}

	if steps := parsePlanStepLines(normalized); len(steps) > 0 {
		if len(steps) == 1 && looksLikeControlToken(steps[0].Description) {
			return defaultPlanSteps(goal), plannerParseModeGoalDefault
		}
		return steps, plannerParseModeLineSplit
	}

	return defaultPlanSteps(goal), plannerParseModeGoalDefault
}

func parseReflectionResponse(raw string) (*Reflection, reflectionParseMode) {
	normalized := cleanModelJSON(raw)

	if reflection, ok := parseReflectionJSON(normalized, reflectionParseModeJSON); ok {
		return reflection, reflectionParseModeJSON
	}

	if extracted := extractFirstBalancedJSON(normalized, '{', '}'); extracted != "" {
		if reflection, ok := parseReflectionJSON(extracted, reflectionParseModeExtracted); ok {
			return reflection, reflectionParseModeExtracted
		}
	}

	return buildReflectionFallback(normalized), reflectionParseModeHeuristic
}

func parsePlanStepArrayJSON(raw string) ([]PlanStep, bool) {
	if strings.TrimSpace(raw) == "" {
		return nil, false
	}
	var steps []PlanStep
	if err := json.Unmarshal([]byte(raw), &steps); err != nil {
		return nil, false
	}
	steps = normalizePlanSteps(steps)
	if len(steps) == 0 {
		return nil, false
	}
	return steps, true
}

func parsePlanStepObjectJSON(raw string) ([]PlanStep, bool) {
	if strings.TrimSpace(raw) == "" {
		return nil, false
	}
	var payload plannerPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, false
	}

	candidates := [][]PlanStep{payload.Steps, payload.Plan, payload.Items, payload.Tasks}
	for _, candidate := range candidates {
		steps := normalizePlanSteps(candidate)
		if len(steps) > 0 {
			return steps, true
		}
	}
	return nil, false
}

func parsePlanStepLines(raw string) []PlanStep {
	lines := strings.Split(raw, "\n")
	out := make([]PlanStep, 0, len(lines))
	for _, line := range lines {
		description := normalizePlanLine(line)
		if description == "" {
			continue
		}
		out = append(out, PlanStep{
			ID:          fmt.Sprintf("step-%d", len(out)+1),
			Description: description,
			Status:      "pending",
		})
	}
	return out
}

func normalizePlanLine(line string) string {
	clean := strings.TrimSpace(line)
	if clean == "" {
		return ""
	}

	for {
		updated := false
		for _, prefix := range []string{"- ", "* ", "• ", "> "} {
			if strings.HasPrefix(clean, prefix) {
				clean = strings.TrimSpace(clean[len(prefix):])
				updated = true
			}
		}
		if !updated {
			break
		}
	}

	clean = trimNumericPrefix(clean)
	clean = strings.TrimSpace(clean)
	if clean == "" {
		return ""
	}
	return clean
}

func trimNumericPrefix(line string) string {
	if line == "" || !unicode.IsDigit(rune(line[0])) {
		return line
	}

	i := 0
	for i < len(line) && unicode.IsDigit(rune(line[i])) {
		i++
	}
	if i >= len(line) {
		return line
	}

	switch line[i] {
	case '.', ')', '-', ':':
		i++
	default:
		return line
	}

	for i < len(line) && unicode.IsSpace(rune(line[i])) {
		i++
	}
	if i >= len(line) {
		return ""
	}
	return line[i:]
}

func normalizePlanSteps(steps []PlanStep) []PlanStep {
	out := make([]PlanStep, 0, len(steps))
	for _, step := range steps {
		description := strings.TrimSpace(step.Description)
		if description == "" {
			continue
		}

		id := strings.TrimSpace(step.GetID())
		if id == "" || id == "<nil>" {
			id = fmt.Sprintf("step-%d", len(out)+1)
		}
		status := strings.TrimSpace(step.Status)
		if status == "" {
			status = "pending"
		}

		out = append(out, PlanStep{
			ID:          id,
			Description: description,
			Status:      status,
		})
	}
	return out
}

func defaultPlanSteps(goal string) []PlanStep {
	description := strings.TrimSpace(goal)
	if description == "" {
		description = "Execute the user goal safely."
	}
	return []PlanStep{
		{
			ID:          "step-1",
			Description: description,
			Status:      "pending",
		},
	}
}

func parseReflectionJSON(raw string, _ reflectionParseMode) (*Reflection, bool) {
	if strings.TrimSpace(raw) == "" {
		return nil, false
	}

	var payload reflectionPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, false
	}

	analysis := strings.TrimSpace(payload.Analysis)
	if analysis == "" {
		analysis = strings.TrimSpace(raw)
	}

	return &Reflection{
		Content:     analysis,
		NextAction:  parseControlSignal(payload.Action, analysis),
		NewMemories: normalizeMemories(payload.NewMemories),
	}, true
}

func buildReflectionFallback(raw string) *Reflection {
	analysis := strings.TrimSpace(raw)
	if analysis == "" {
		analysis = "No reflection content returned."
	}
	return &Reflection{
		Content:    analysis,
		NextAction: inferControlSignalFromText(analysis),
	}
}

func parseControlSignal(actionRaw string, analysis string) ControlSignal {
	switch strings.ToLower(strings.TrimSpace(actionRaw)) {
	case "retry":
		return SignalRetry
	case "replan":
		return SignalReplan
	case "stop":
		return SignalStop
	case "continue":
		return SignalContinue
	default:
		return inferControlSignalFromText(analysis)
	}
}

func inferControlSignalFromText(text string) ControlSignal {
	lower := strings.ToLower(strings.TrimSpace(text))
	switch {
	case containsAny(lower, "retry", "try again", "transient"):
		return SignalRetry
	case containsAny(lower, "replan", "new plan", "different plan"):
		return SignalReplan
	case containsAny(lower, "goal achieved", "task complete", "cannot continue", "impossible"):
		return SignalStop
	default:
		return SignalContinue
	}
}

func normalizeMemories(memories []string) []string {
	if len(memories) == 0 {
		return nil
	}

	out := make([]string, 0, len(memories))
	seen := make(map[string]struct{}, len(memories))
	for _, memory := range memories {
		clean := strings.TrimSpace(memory)
		if clean == "" {
			continue
		}
		if _, exists := seen[clean]; exists {
			continue
		}
		seen[clean] = struct{}{}
		out = append(out, clean)
	}
	return out
}

func cleanModelJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

func extractFirstBalancedJSON(input string, open, close byte) string {
	start := -1
	depth := 0
	inString := false
	escaped := false

	for i := 0; i < len(input); i++ {
		ch := input[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
		case open:
			if depth == 0 {
				start = i
			}
			depth++
		case close:
			if depth == 0 {
				continue
			}
			depth--
			if depth == 0 && start >= 0 {
				return strings.TrimSpace(input[start : i+1])
			}
		}
	}
	return ""
}

func looksLikeControlToken(s string) bool {
	token := strings.TrimSpace(s)
	if token == "" || strings.Contains(token, " ") || len(token) > 80 {
		return false
	}
	for _, r := range token {
		if unicode.IsUpper(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if needle != "" && strings.Contains(text, needle) {
			return true
		}
	}
	return false
}
