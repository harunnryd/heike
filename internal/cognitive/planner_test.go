package cognitive

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnifiedPlanner_BuildPrompt_IncludesSkillContext(t *testing.T) {
	planner := NewPlanner(new(MockLLMClient), PlannerPromptConfig{})
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
