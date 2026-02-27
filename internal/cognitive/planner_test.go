package cognitive

import (
	"context"
	stdErrors "errors"
	"testing"

	heikeErrors "github.com/harunnryd/heike/internal/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUnifiedPlanner_BuildPrompt_IncludesSkillContext(t *testing.T) {
	planner := NewPlanner(new(MockLLMClient), PlannerPromptConfig{}, 1)
	ctx := &CognitiveContext{
		AvailableSkills: []string{"web_research"},
		Metadata: map[string]string{
			"skills_context": "- web_research: Validate sources before answering",
		},
	}

	prompt := planner.buildPrompt("Research latest AI assistants", ctx)
	assert.Contains(t, prompt, "AVAILABLE SKILLS:")
	assert.Contains(t, prompt, "- web_research")
	assert.Contains(t, prompt, "SKILL CONTEXT:")
	assert.Contains(t, prompt, "Validate sources before answering")
}

func TestUnifiedPlanner_InvalidStructuredOutputReturnsTypedError(t *testing.T) {
	mockLLM := new(MockLLMClient)
	planner := NewPlanner(mockLLM, PlannerPromptConfig{}, 1)

	ctx := context.Background()
	goal := "Summarize release notes"
	cCtx := &CognitiveContext{Metadata: map[string]string{}}

	mockLLM.On("Complete", ctx, mock.Anything).Return("NOT_JSON", nil).Twice()

	_, err := planner.Plan(ctx, goal, cCtx)
	assert.Error(t, err)
	assert.True(t, stdErrors.Is(err, heikeErrors.ErrInvalidModelOutput))
	mockLLM.AssertExpectations(t)
}
