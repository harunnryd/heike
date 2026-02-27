package cognitive

import "testing"

func TestParsePlannerResponse_JSONArray(t *testing.T) {
	steps, mode := parsePlannerResponse(`[{"id":"a","description":"first","status":"pending"}]`, "goal")
	if mode != plannerParseModeJSONArray {
		t.Fatalf("mode = %s, want %s", mode, plannerParseModeJSONArray)
	}
	if len(steps) != 1 || steps[0].Description != "first" {
		t.Fatalf("unexpected steps: %+v", steps)
	}
}

func TestParsePlannerResponse_JSONObject(t *testing.T) {
	steps, mode := parsePlannerResponse(`{"steps":[{"id":1,"description":"from object"}]}`, "goal")
	if mode != plannerParseModeJSONObject {
		t.Fatalf("mode = %s, want %s", mode, plannerParseModeJSONObject)
	}
	if len(steps) != 1 || steps[0].Description != "from object" {
		t.Fatalf("unexpected steps: %+v", steps)
	}
}

func TestParsePlannerResponse_ExtractedJSON(t *testing.T) {
	raw := "Plan:\n```json\n[{\"id\":\"x\",\"description\":\"inside fence\"}]\n```"
	steps, mode := parsePlannerResponse(raw, "goal")
	if mode != plannerParseModeJSONArray && mode != plannerParseModeExtracted {
		t.Fatalf("unexpected mode: %s", mode)
	}
	if len(steps) != 1 || steps[0].Description != "inside fence" {
		t.Fatalf("unexpected steps: %+v", steps)
	}
}

func TestParsePlannerResponse_ControlTokenUsesGoalDefault(t *testing.T) {
	goal := "Audit repository and summarize findings."
	steps, mode := parsePlannerResponse("SKILL_CODEBASE_STATS_DONE", goal)
	if mode != plannerParseModeGoalDefault {
		t.Fatalf("mode = %s, want %s", mode, plannerParseModeGoalDefault)
	}
	if len(steps) != 1 || steps[0].Description != goal {
		t.Fatalf("unexpected steps: %+v", steps)
	}
}

func TestParseReflectionResponse_JSON(t *testing.T) {
	raw := `{"analysis":"ok","next_action":"retry","new_memories":["m1","m1","m2"]}`
	reflection, mode := parseReflectionResponse(raw)
	if mode != reflectionParseModeJSON {
		t.Fatalf("mode = %s, want %s", mode, reflectionParseModeJSON)
	}
	if reflection.NextAction != SignalRetry {
		t.Fatalf("next action = %s, want %s", reflection.NextAction, SignalRetry)
	}
	if len(reflection.NewMemories) != 2 {
		t.Fatalf("unexpected memories: %+v", reflection.NewMemories)
	}
}

func TestParseReflectionResponse_ExtractedJSON(t *testing.T) {
	raw := "Result:\n```json\n{\"analysis\":\"need plan update\",\"next_action\":\"replan\"}\n```"
	reflection, mode := parseReflectionResponse(raw)
	if mode != reflectionParseModeJSON && mode != reflectionParseModeExtracted {
		t.Fatalf("unexpected mode: %s", mode)
	}
	if reflection.NextAction != SignalReplan {
		t.Fatalf("next action = %s, want %s", reflection.NextAction, SignalReplan)
	}
}

func TestParseReflectionResponse_HeuristicFallback(t *testing.T) {
	reflection, mode := parseReflectionResponse("Temporary network issue, please retry.")
	if mode != reflectionParseModeHeuristic {
		t.Fatalf("mode = %s, want %s", mode, reflectionParseModeHeuristic)
	}
	if reflection.NextAction != SignalRetry {
		t.Fatalf("next action = %s, want %s", reflection.NextAction, SignalRetry)
	}
}

func TestParseReflectionResponse_ControlTokenDefaultsContinue(t *testing.T) {
	reflection, mode := parseReflectionResponse("SKILL_CODEBASE_STATS_DONE")
	if mode != reflectionParseModeHeuristic {
		t.Fatalf("mode = %s, want %s", mode, reflectionParseModeHeuristic)
	}
	if reflection.NextAction != SignalContinue {
		t.Fatalf("next action = %s, want %s", reflection.NextAction, SignalContinue)
	}
}
