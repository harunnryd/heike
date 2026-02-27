package cognitive

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/model/contract"
)

// Error types for the Cognitive Engine
type ErrorType string

const (
	ErrTransient ErrorType = "transient"
	ErrLogic     ErrorType = "logic"
	ErrFatal     ErrorType = "fatal"
	ErrMaxTurns  ErrorType = "max_turns_reached"
)

type CognitiveError struct {
	Type    ErrorType
	Message string
	Cause   error
}

func (e *CognitiveError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// DefaultCognitiveEngine implements the OODA loop
type DefaultCognitiveEngine struct {
	planner     Planner
	thinker     Thinker
	actor       Actor
	reflector   Reflector
	memory      MemoryManager
	maxTurns    int
	tokenBudget int
}

func NewEngine(
	planner Planner,
	thinker Thinker,
	actor Actor,
	reflector Reflector,
	memory MemoryManager,
	maxTurns int,
	tokenBudget int,
) *DefaultCognitiveEngine {
	if maxTurns <= 0 {
		maxTurns = config.DefaultOrchestratorMaxTurns
	}
	if tokenBudget <= 0 {
		tokenBudget = config.DefaultOrchestratorTokenBudget
	}

	return &DefaultCognitiveEngine{
		planner:     planner,
		thinker:     thinker,
		actor:       actor,
		reflector:   reflector,
		memory:      memory,
		maxTurns:    maxTurns,
		tokenBudget: tokenBudget,
	}
}

func (e *DefaultCognitiveEngine) SetMaxTurns(n int) {
	if n > 0 {
		e.maxTurns = n
	}
}

func (e *DefaultCognitiveEngine) SetTokenBudget(n int) {
	if n > 0 {
		e.tokenBudget = n
	}
}

func (e *DefaultCognitiveEngine) Run(ctx context.Context, goal string, opts ...ExecutionOption) (*Result, error) {
	// Initialize Context
	cCtx := &CognitiveContext{
		Metadata:    make(map[string]string),
		Scratchpad:  []string{},
		History:     []contract.Message{},
		Memories:    []string{},
		TokenBudget: e.tokenBudget,
	}

	// Apply options to hydrate context
	for _, opt := range opts {
		opt(cCtx)
	}

	slog.Info("CognitiveEngine started", "goal", goal, "context_keys", len(cCtx.Metadata))

	// Plan (Observe & Orient)
	plan, err := e.planner.Plan(ctx, goal, cCtx)
	if err != nil {
		return nil, &CognitiveError{Type: ErrFatal, Message: "Planning failed", Cause: err}
	}
	cCtx.CurrentPlan = plan
	slog.Debug("Plan generated", "steps", len(plan.Steps))

	// Cognitive Loop (Decide & Act)
	for i := 0; i < e.maxTurns; i++ {
		// Check for cancellation
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		slog.Debug("Cognitive loop turn", "turn", i+1, "max", e.maxTurns)

		// Think (Decide)
		thought, err := e.thinker.Think(ctx, goal, cCtx.CurrentPlan, cCtx)
		if err != nil {
			return nil, &CognitiveError{Type: ErrLogic, Message: "Thinking failed", Cause: err}
		}

		// Append Assistant Thought to History
		asstMsg := contract.Message{
			Role:    "assistant",
			Content: thought.Content,
		}
		if thought.Action != nil && thought.Action.Type == ActionTypeToolCall {
			asstMsg.ToolCalls = thought.Action.ToolCalls
		}
		cCtx.History = append(cCtx.History, asstMsg)

		// Final Answer Check
		if thought.IsFinalAnswer() {
			slog.Info("Final answer reached", "turn", i+1)
			return &Result{
				Content: thought.Content,
				Meta:    map[string]interface{}{"turns": i + 1},
			}, nil
		}

		// Act
		result, err := e.actor.Execute(ctx, thought.Action)
		if err != nil {
			slog.Error("Action execution failed", "error", err)
			return nil, &CognitiveError{Type: ErrFatal, Message: "Action execution failed", Cause: err}
		}

		// Append Tool Outputs to History
		if thought.Action.Type == ActionTypeToolCall {
			for _, toolOut := range result.ToolOutputs {
				cCtx.History = append(cCtx.History, contract.Message{
					Role:       "tool",
					Content:    toolOut.Output,
					ToolCallID: toolOut.CallID,
				})
			}
		}

		// Auto-prune history if needed
		cCtx.Prune()

		// Reflect
		reflection, err := e.reflector.Reflect(ctx, goal, thought.Action, result)
		if err != nil {
			slog.Warn("Reflection failed", "error", err)
		} else {
			cCtx.Update(reflection)

			// Handle Control Signals
			switch reflection.NextAction {
			case SignalRetry:
				slog.Info("Reflector requested retry")
				// Logic to retry logic (decrement counter?)
				i-- // Naive retry: just don't count this turn? Or keep counting to avoid infinite loop?
				// Better: keep counting, but don't advance plan step.
			case SignalReplan:
				slog.Info("Reflector requested replan")
				newPlan, err := e.planner.Plan(ctx, goal, cCtx)
				if err == nil {
					cCtx.CurrentPlan = newPlan
				}
			case SignalStop:
				slog.Info("Reflector requested stop")
				return &Result{
					Content: "Stopped by reflector: " + reflection.Content,
					Meta:    map[string]interface{}{"turns": i + 1},
				}, nil
			}

			// Optional: Persist new memories if memory manager is available
			if e.memory != nil && len(reflection.NewMemories) > 0 {
				go func(mems []string) {
					for _, m := range mems {
						if err := e.memory.Remember(context.Background(), m); err != nil {
							slog.Warn("Failed to persist memory", "error", err)
						}
					}
				}(reflection.NewMemories)
			}
		}
	}

	return nil, &CognitiveError{Type: ErrMaxTurns, Message: "Max cognitive turns reached"}
}
