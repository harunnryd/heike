package task

import (
	"context"
	"errors"
	"testing"

	"github.com/harunnryd/heike/internal/model/contract"

	"github.com/stretchr/testify/assert"
)

type decomposerLLMStub struct {
	response string
	err      error
}

func (s *decomposerLLMStub) Complete(ctx context.Context, prompt string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.response, nil
}

func (s *decomposerLLMStub) ChatComplete(ctx context.Context, messages []contract.Message, tools []contract.ToolDef) (string, []*contract.ToolCall, error) {
	return "", nil, errors.New("not used")
}

func TestParseDecompositionResponse_JSONArray(t *testing.T) {
	tasks, mode := parseDecompositionResponse(`[{"id":"a","description":"first","priority":2}]`, "goal")
	if assert.Len(t, tasks, 1) {
		assert.Equal(t, "a", tasks[0].ID)
		assert.Equal(t, "first", tasks[0].Description)
		assert.Equal(t, 2, tasks[0].Priority)
	}
	assert.Equal(t, decompositionParseModeJSONArray, mode)
}

func TestParseDecompositionResponse_JSONObject(t *testing.T) {
	tasks, mode := parseDecompositionResponse(`{"sub_tasks":[{"description":"from object"}]}`, "goal")
	if assert.Len(t, tasks, 1) {
		assert.Equal(t, "task-1", tasks[0].ID)
		assert.Equal(t, "from object", tasks[0].Description)
		assert.Equal(t, 1, tasks[0].Priority)
	}
	assert.Equal(t, decompositionParseModeJSONObject, mode)
}

func TestParseDecompositionResponse_ExtractedJSON(t *testing.T) {
	raw := "Use this:\n```json\n[{\"description\":\"wrapped\"}]\n```"
	tasks, mode := parseDecompositionResponse(raw, "goal")
	if assert.Len(t, tasks, 1) {
		assert.Equal(t, "wrapped", tasks[0].Description)
	}
	assert.Equal(t, decompositionParseModeExtracted, mode)
}

func TestParseDecompositionResponse_ControlToken_DefaultGoal(t *testing.T) {
	goal := "Audit repository quickly"
	tasks, mode := parseDecompositionResponse("SKILL_CODEBASE_STATS_DONE", goal)
	if assert.Len(t, tasks, 1) {
		assert.Equal(t, goal, tasks[0].Description)
		assert.Equal(t, "task-1", tasks[0].ID)
	}
	assert.Equal(t, decompositionParseModeDefault, mode)
}

func TestLLMDecomposer_Decompose_NonJSONFallback(t *testing.T) {
	d := NewDecomposer(&decomposerLLMStub{response: "SKILL_DONE"}, 1, DecomposerPromptConfig{})

	tasks, err := d.Decompose(context.Background(), "Summarize codebase")
	assert.NoError(t, err)
	if assert.Len(t, tasks, 1) {
		assert.Equal(t, "Summarize codebase", tasks[0].Description)
		assert.Equal(t, "task-1", tasks[0].ID)
	}
}
