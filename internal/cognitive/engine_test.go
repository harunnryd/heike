package cognitive

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/model/contract"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLLMClient is a mock of LLMClient interface
type MockLLMClient struct {
	mock.Mock
}

func (m *MockLLMClient) Complete(ctx context.Context, prompt string) (string, error) {
	args := m.Called(ctx, prompt)
	return args.String(0), args.Error(1)
}

func (m *MockLLMClient) ChatComplete(ctx context.Context, messages []contract.Message, tools []contract.ToolDef) (string, []*contract.ToolCall, error) {
	args := m.Called(ctx, messages, tools)
	return args.String(0), args.Get(1).([]*contract.ToolCall), args.Error(2)
}

// MockToolExecutor is a mock of ToolExecutor interface
type MockToolExecutor struct {
	mock.Mock
}

func (m *MockToolExecutor) Execute(ctx context.Context, name string, args json.RawMessage, input string) (json.RawMessage, error) {
	argsMock := m.Called(ctx, name, args, input)
	return argsMock.Get(0).(json.RawMessage), argsMock.Error(1)
}

func TestCognitiveEngine_Run_Simple(t *testing.T) {
	mockLLM := new(MockLLMClient)
	mockToolExec := new(MockToolExecutor)

	planner := NewPlanner(mockLLM, PlannerPromptConfig{}, 1)
	thinker := NewThinker(mockLLM, ThinkerPromptConfig{})
	actor := NewActor(mockToolExec)
	reflector := NewReflector(mockLLM, ReflectorPromptConfig{}, 1)

	engine := NewEngine(planner, thinker, actor, reflector, nil, config.DefaultOrchestratorMaxTurns, config.DefaultOrchestratorTokenBudget)

	ctx := context.Background()
	goal := "Say hello"

	// Mock Planning
	mockLLM.On("Complete", ctx, mock.Anything).Return(`[{"id":"1","description":"Say hello"}]`, nil).Once()

	// Mock Thinking (Final Answer)
	mockLLM.On("ChatComplete", ctx, mock.Anything, mock.Anything).Return("Hello!", []*contract.ToolCall{}, nil).Once()

	// Run
	result, err := engine.Run(ctx, goal)

	assert.NoError(t, err)
	assert.Equal(t, "Hello!", result.Content)
	mockLLM.AssertExpectations(t)
}

func TestCognitiveEngine_Run_WithTool(t *testing.T) {
	mockLLM := new(MockLLMClient)
	mockToolExec := new(MockToolExecutor)

	planner := NewPlanner(mockLLM, PlannerPromptConfig{}, 1)
	thinker := NewThinker(mockLLM, ThinkerPromptConfig{})
	actor := NewActor(mockToolExec)
	reflector := NewReflector(mockLLM, ReflectorPromptConfig{}, 1)

	engine := NewEngine(planner, thinker, actor, reflector, nil, config.DefaultOrchestratorMaxTurns, config.DefaultOrchestratorTokenBudget)

	ctx := context.Background()
	goal := "Get weather"

	// Planning
	mockLLM.On("Complete", ctx, mock.Anything).Return(`[{"id":"1","description":"Check weather tool"}]`, nil).Once()

	// Thinking (Tool Call)
	toolCall := &contract.ToolCall{Name: "weather", Input: "{}"}
	mockLLM.On("ChatComplete", ctx, mock.Anything, mock.Anything).Return("", []*contract.ToolCall{toolCall}, nil).Once()

	// Acting (Tool Execution)
	mockToolExec.On("Execute", ctx, "weather", mock.Anything, "").Return(json.RawMessage(`"Sunny"`), nil).Once()

	// Reflecting
	mockLLM.On("Complete", ctx, mock.Anything).Return(`{"analysis":"tool worked","next_action":"continue","new_memories":[]}`, nil).Once()

	// Thinking (Final Answer)
	mockLLM.On("ChatComplete", ctx, mock.Anything, mock.Anything).Return("It is Sunny", []*contract.ToolCall{}, nil).Once()

	// Run
	result, err := engine.Run(ctx, goal)

	assert.NoError(t, err)
	assert.Equal(t, "It is Sunny", result.Content)
	mockLLM.AssertExpectations(t)
	mockToolExec.AssertExpectations(t)
}
