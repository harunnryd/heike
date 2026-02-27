package cognitive

import (
	"context"
	"testing"

	"github.com/harunnryd/heike/internal/model/contract"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUnifiedThinker_ReturnsAnswerWhenModelDoesNotCallTool(t *testing.T) {
	mockLLM := new(MockLLMClient)
	thinker := NewThinker(mockLLM, ThinkerPromptConfig{})

	ctx := context.Background()
	goal := "Please investigate tool registry behavior and explain its purpose."
	cCtx := &CognitiveContext{
		AvailableTools: []contract.ToolDef{
			{Name: "exec_command", Description: "Run command"},
			{Name: "open", Description: "Open URL"},
		},
	}

	mockLLM.
		On("ChatComplete", ctx, mock.Anything, mock.Anything).
		Return("Sorry—I'm having trouble accessing the file with the tool call. Please allow me to retry.", []*contract.ToolCall{}, nil).
		Once()

	thought, err := thinker.Think(ctx, goal, nil, cCtx)
	assert.NoError(t, err)
	assert.NotNil(t, thought)
	assert.NotNil(t, thought.Action)
	assert.Equal(t, ActionTypeAnswer, thought.Action.Type)
	assert.Equal(t, "Sorry—I'm having trouble accessing the file with the tool call. Please allow me to retry.", thought.Action.Content)

	mockLLM.AssertExpectations(t)
}

func TestUnifiedThinker_DoesNotInjectFallbackAfterToolOutput(t *testing.T) {
	mockLLM := new(MockLLMClient)
	thinker := NewThinker(mockLLM, ThinkerPromptConfig{})

	ctx := context.Background()
	goal := "Please investigate tool registry behavior and explain its purpose."
	cCtx := &CognitiveContext{
		History: []contract.Message{
			{Role: "tool", Content: `{"content":"package tool"}`},
		},
		AvailableTools: []contract.ToolDef{
			{Name: "exec_command", Description: "Run command"},
		},
	}

	mockLLM.
		On("ChatComplete", ctx, mock.Anything, mock.Anything).
		Return("The file defines the core tool interfaces.", []*contract.ToolCall{}, nil).
		Once()

	thought, err := thinker.Think(ctx, goal, nil, cCtx)
	assert.NoError(t, err)
	assert.NotNil(t, thought)
	assert.NotNil(t, thought.Action)
	assert.Equal(t, ActionTypeAnswer, thought.Action.Type)
	assert.Equal(t, "The file defines the core tool interfaces.", thought.Action.Content)

	mockLLM.AssertExpectations(t)
}

func TestUnifiedThinker_BuildSystemPrompt_IncludesSkillContext(t *testing.T) {
	thinker := NewThinker(new(MockLLMClient), ThinkerPromptConfig{})
	ctx := &CognitiveContext{
		AvailableSkills: []string{"web_research"},
		Metadata: map[string]string{
			"skills_context": "- web_research: Find and verify web sources",
		},
	}

	prompt := thinker.buildSystemPrompt("Research latest AI tooling", nil, ctx)
	assert.Contains(t, prompt, "AVAILABLE SKILLS:")
	assert.Contains(t, prompt, "- web_research")
	assert.Contains(t, prompt, "SKILL CONTEXT:")
	assert.Contains(t, prompt, "Find and verify web sources")
}
