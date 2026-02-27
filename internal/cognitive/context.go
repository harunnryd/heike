package cognitive

import (
	"fmt"
	"strings"
	"sync"

	"github.com/harunnryd/heike/internal/model/contract"
)

// CognitiveContext holds the dynamic state of the cognitive process
type CognitiveContext struct {
	SessionID   string
	WorkspaceID string

	// Static Configuration (Injected at start)
	AvailableTools  []contract.ToolDef
	AvailableSkills []string // Simplified for now

	// Dynamic State (Updated during loop)
	History    []contract.Message // Full conversation history
	Memories   []string           // Retrieved relevant memories
	Scratchpad []string           // Internal monologue / reasoning trace

	// Execution State
	CurrentPlan *Plan             // The active plan
	StepIndex   int               // Current step in the plan
	Metadata    map[string]string // Arbitrary k/v for extensions

	// Token Management
	TokenBudget int // Max tokens allowed for context
	TokenUsage  int // Current estimated usage
}

// Prune optimizes context to fit within TokenBudget
// This is a naive implementation; a real one would use a tokenizer
func (c *CognitiveContext) Prune() {
	if c.TokenBudget <= 0 {
		return
	}

	// Naive estimation: 1 char ~= 0.25 tokens (4 chars/token)
	estimate := func(s string) int { return len(s) / 4 }

	currentTokens := 0

	// Always keep plan and scratchpad (high priority)
	if c.CurrentPlan != nil {
		currentTokens += estimate(c.CurrentPlan.Raw)
	}
	for _, s := range c.Scratchpad {
		currentTokens += estimate(s)
	}

	// Calculate remaining budget for History and Memories
	remaining := c.TokenBudget - currentTokens
	if remaining < 0 {
		// Critical: context is already full with internal state.
		// Keep only the latest scratchpad entries that still fit.
		budgetForScratchpad := c.TokenBudget
		if c.CurrentPlan != nil {
			budgetForScratchpad -= estimate(c.CurrentPlan.Raw)
		}
		if budgetForScratchpad < 0 {
			c.Scratchpad = nil
			c.Memories = nil
			c.History = nil
			c.TokenUsage = c.TokenBudget
			return
		}

		prunedScratchpad := []string{}
		remaining = budgetForScratchpad
		for i := len(c.Scratchpad) - 1; i >= 0; i-- {
			entry := c.Scratchpad[i]
			cost := estimate(entry)
			if remaining >= cost {
				prunedScratchpad = append([]string{entry}, prunedScratchpad...)
				remaining -= cost
			} else {
				break
			}
		}
		c.Scratchpad = prunedScratchpad
	}

	// Prune Memories (Low Priority)
	// We keep top N memories that fit
	validMemories := []string{}
	for _, m := range c.Memories {
		cost := estimate(m)
		if remaining >= cost {
			validMemories = append(validMemories, m)
			remaining -= cost
		} else {
			break // Stop if we can't fit anymore
		}
	}
	c.Memories = validMemories

	// Prune History (Medium Priority)
	// We keep latest N messages
	// We iterate backwards
	validHistory := []contract.Message{}
	for i := len(c.History) - 1; i >= 0; i-- {
		msg := c.History[i]
		cost := estimate(msg.Content)
		if remaining >= cost {
			// Prepend to maintain order
			validHistory = append([]contract.Message{msg}, validHistory...)
			remaining -= cost
		} else {
			break
		}
	}
	c.History = validHistory
	c.TokenUsage = c.TokenBudget - remaining
}

// Update merges a Reflection into the context
func (c *CognitiveContext) Update(r *Reflection) {
	if r == nil {
		return
	}

	if r.Content != "" {
		c.Scratchpad = append(c.Scratchpad, r.Content)
	}

	if len(r.NewMemories) > 0 {
		c.Memories = append(c.Memories, r.NewMemories...)
	}

	// Auto-prune after update
	c.Prune()
}

func (c *CognitiveContext) String() string {
	var sb strings.Builder

	if len(c.Memories) > 0 {
		sb.WriteString("MEMORIES:\n")
		for _, m := range c.Memories {
			sb.WriteString(fmt.Sprintf("- %s\n", m))
		}
		sb.WriteString("\n")
	}

	if len(c.History) > 0 {
		sb.WriteString("HISTORY:\n")
		for _, msg := range c.History {
			sb.WriteString(fmt.Sprintf("[%s]: %s\n", msg.Role, msg.Content))
		}
		sb.WriteString("\n")
	}

	if c.CurrentPlan != nil {
		sb.WriteString("CURRENT PLAN:\n")
		for _, step := range c.CurrentPlan.Steps {
			sb.WriteString(fmt.Sprintf("- [%s] %s (%s)\n", step.ID, step.Description, step.Status))
		}
		sb.WriteString("\n")
	}

	if len(c.Scratchpad) > 0 {
		sb.WriteString("THOUGHTS:\n")
		for _, t := range c.Scratchpad {
			sb.WriteString(fmt.Sprintf("> %s\n", t))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// ContextManager handles context creation and hydration
type DefaultContextManager struct {
	mu sync.RWMutex
}

func NewContextManager() *DefaultContextManager {
	return &DefaultContextManager{}
}

func (cm *DefaultContextManager) BuildContext(goal string) *CognitiveContext {
	return &CognitiveContext{
		Metadata:   make(map[string]string),
		Scratchpad: []string{},
		History:    []contract.Message{},
		Memories:   []string{},
	}
}
