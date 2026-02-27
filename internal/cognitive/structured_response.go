package cognitive

import (
	"encoding/json"
	"fmt"
	"strings"
)

type plannerParseMode string

const (
	plannerParseModeJSONArray  plannerParseMode = "json_array"
	plannerParseModeJSONObject plannerParseMode = "json_object"
	plannerParseModeInvalid    plannerParseMode = "invalid"
)

type reflectionParseMode string

const (
	reflectionParseModeJSON    reflectionParseMode = "json_object"
	reflectionParseModeInvalid reflectionParseMode = "invalid"
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

func parsePlannerResponse(raw string, _ string) ([]PlanStep, plannerParseMode) {
	normalized := cleanModelJSON(raw)

	if steps, ok := parsePlanStepArrayJSON(normalized); ok {
		return steps, plannerParseModeJSONArray
	}
	if steps, ok := parsePlanStepObjectJSON(normalized); ok {
		return steps, plannerParseModeJSONObject
	}

	return nil, plannerParseModeInvalid
}

func parseReflectionResponse(raw string) (*Reflection, reflectionParseMode) {
	normalized := cleanModelJSON(raw)

	if reflection, ok := parseReflectionJSON(normalized, reflectionParseModeJSON); ok {
		return reflection, reflectionParseModeJSON
	}

	return nil, reflectionParseModeInvalid
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
		analysis = "No analysis provided."
	}
	nextAction, ok := parseControlSignal(payload.Action)
	if !ok {
		return nil, false
	}

	return &Reflection{
		Content:     analysis,
		NextAction:  nextAction,
		NewMemories: normalizeMemories(payload.NewMemories),
	}, true
}

func parseControlSignal(actionRaw string) (ControlSignal, bool) {
	switch strings.ToLower(strings.TrimSpace(actionRaw)) {
	case "retry":
		return SignalRetry, true
	case "replan":
		return SignalReplan, true
	case "stop":
		return SignalStop, true
	case "continue":
		return SignalContinue, true
	default:
		return "", false
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
