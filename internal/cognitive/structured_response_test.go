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

func TestParsePlannerResponse_InvalidReturnsNil(t *testing.T) {
	steps, mode := parsePlannerResponse("SKILL_CODEBASE_STATS_DONE", "goal")
	if mode != plannerParseModeInvalid {
		t.Fatalf("mode = %s, want %s", mode, plannerParseModeInvalid)
	}
	if len(steps) != 0 {
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

func TestParseReflectionResponse_InvalidReturnsNil(t *testing.T) {
	reflection, mode := parseReflectionResponse("Temporary network issue, please retry.")
	if mode != reflectionParseModeInvalid {
		t.Fatalf("mode = %s, want %s", mode, reflectionParseModeInvalid)
	}
	if reflection != nil {
		t.Fatalf("expected nil reflection, got %+v", reflection)
	}
}

func TestParseReflectionResponse_InvalidControlSignalReturnsNil(t *testing.T) {
	reflection, mode := parseReflectionResponse(`{"analysis":"x","next_action":"unknown","new_memories":[]}`)
	if mode != reflectionParseModeInvalid {
		t.Fatalf("mode = %s, want %s", mode, reflectionParseModeInvalid)
	}
	if reflection != nil {
		t.Fatalf("expected nil reflection, got %+v", reflection)
	}
}

func TestParseReflectionResponse_ControlTokenReturnsInvalid(t *testing.T) {
	reflection, mode := parseReflectionResponse("SKILL_CODEBASE_STATS_DONE")
	if mode != reflectionParseModeInvalid {
		t.Fatalf("mode = %s, want %s", mode, reflectionParseModeInvalid)
	}
	if reflection != nil {
		t.Fatalf("expected nil reflection, got %+v", reflection)
	}
}
