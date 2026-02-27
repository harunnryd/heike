package cognitive

import (
	"context"
	"fmt"

	"github.com/harunnryd/heike/internal/model/contract"
)

// CognitiveEngine drives the OODA (Observe-Orient-Decide-Act) loop
type Engine interface {
	// Run executes the cognitive loop for a given goal
	Run(ctx context.Context, goal string, opts ...ExecutionOption) (*Result, error)
}

// Planner generates a step-by-step plan based on goal and context
type Planner interface {
	// Plan creates a sequence of steps to achieve the goal.
	// It MUST consider the current context (history, memories, available tools).
	Plan(ctx context.Context, goal string, context *CognitiveContext) (*Plan, error)
}

// Thinker reasons about the current state and decides the next action
type Thinker interface {
	// Think evaluates the current plan progress and context to decide the next step.
	// It returns a Thought containing reasoning and a specific Action (ToolCall or Answer).
	Think(ctx context.Context, goal string, plan *Plan, context *CognitiveContext) (*Thought, error)
}

// Actor executes the decided actions
type Actor interface {
	// Execute performs the action decided by the Thinker.
	// This abstracts over ToolRunner (for tools) and Egress (for answers).
	Execute(ctx context.Context, action *Action) (*ExecutionResult, error)
}

// Reflector analyzes the execution result and updates the context
type Reflector interface {
	// Reflect analyzes the outcome of an action.
	// It determines if the action succeeded, failed, or requires a retry.
	// It also extracts new memories or insights to update the context.
	Reflect(ctx context.Context, goal string, action *Action, result *ExecutionResult) (*Reflection, error)
}

// MemoryManager handles semantic recall (optional dependency for Engine)
type MemoryManager interface {
	Retrieve(ctx context.Context, query string) ([]string, error)
	Remember(ctx context.Context, fact string) error
}

// Plan represents a structured plan
type Plan struct {
	Steps []PlanStep
	Raw   string
}

type PlanStep struct {
	ID          interface{} `json:"id"`
	Description string      `json:"description"`
	Status      string      `json:"status"` // pending, completed, failed
}

func (s *PlanStep) GetID() string {
	return fmt.Sprintf("%v", s.ID)
}

// Thought represents the output of the thinking process
type Thought struct {
	Content string
	Action  *Action
}

func (t *Thought) IsFinalAnswer() bool {
	return t.Action == nil || t.Action.Type == ActionTypeAnswer
}

// Action represents a decision to do something
type Action struct {
	Type      ActionType
	ToolCalls []*contract.ToolCall
	Content   string // For final answer
}

type ActionType string

const (
	ActionTypeToolCall ActionType = "tool_call"
	ActionTypeAnswer   ActionType = "answer"
)

// ExecutionResult represents the outcome of an action
type ExecutionResult struct {
	Success     bool
	Output      string
	Error       error
	ToolOutputs []ToolOutput
}

type ToolOutput struct {
	CallID string
	Name   string
	Output string
}

// Reflection represents the analysis of an execution
type Reflection struct {
	Content     string
	NextAction  ControlSignal // What to do next
	NewMemories []string
}

type ControlSignal string

const (
	SignalContinue ControlSignal = "continue" // Proceed to next step/turn
	SignalRetry    ControlSignal = "retry"    // Retry current action
	SignalReplan   ControlSignal = "replan"   // Plan is invalid, regenerate
	SignalStop     ControlSignal = "stop"     // Goal achieved or impossible
)

// Result is the final output of the cognitive engine
type Result struct {
	Content string
	Meta    map[string]interface{}
}

// ExecutionOption allows configuring the engine run
type ExecutionOption func(*CognitiveContext)

// LLMClient abstracts the LLM provider
type LLMClient interface {
	Complete(ctx context.Context, prompt string) (string, error)
	// ChatComplete sends a list of messages to the LLM
	ChatComplete(ctx context.Context, messages []contract.Message, tools []contract.ToolDef) (string, []*contract.ToolCall, error)
}
