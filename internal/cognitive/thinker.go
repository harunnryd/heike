package cognitive

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/model/contract"
)

type UnifiedThinker struct {
	llm       LLMClient
	promptCfg ThinkerPromptConfig
}

type ThinkerPromptConfig struct {
	System      string
	Instruction string
}

func NewThinker(llm LLMClient, promptCfg ThinkerPromptConfig) *UnifiedThinker {
	if strings.TrimSpace(promptCfg.System) == "" {
		promptCfg.System = config.DefaultThinkerSystemPrompt
	}
	if strings.TrimSpace(promptCfg.Instruction) == "" {
		promptCfg.Instruction = config.DefaultThinkerInstructionPrompt
	}

	return &UnifiedThinker{
		llm:       llm,
		promptCfg: promptCfg,
	}
}

func (t *UnifiedThinker) Think(ctx context.Context, goal string, plan *Plan, c *CognitiveContext) (*Thought, error) {
	slog.Info("UnifiedThinker thinking", "goal", goal)

	// Build message history
	var messages []contract.Message

	// System Prompt
	messages = append(messages, contract.Message{
		Role:    "system",
		Content: t.buildSystemPrompt(goal, plan, c),
	})

	// Conversation History (if any)
	if len(c.History) > 0 {
		messages = append(messages, c.History...)
	}

	// User Prompt (Next Step Trigger)
	// Only add if the last message wasn't a tool output (which acts as a trigger)
	// But actually, we usually need to re-prompt the model to "continue" or "think".
	// Ideally, we'd just let the conversation flow.
	// But for robustness, we can add a transient "system" reminder or just rely on history.
	// Let's add a user prompt if history is empty OR just to force action.
	if len(c.History) == 0 || c.History[len(c.History)-1].Role != "user" {
		// messages = append(messages, contract.Message{
		// 	Role:    "user",
		// 	Content: "Execute the next step of the plan.",
		// })
		// Actually, if we have a goal, that's the user prompt.
		// If history is empty, we must add the goal.
		if len(c.History) == 0 {
			messages = append(messages, contract.Message{
				Role:    "user",
				Content: goal,
			})
		}
	}

	content, toolCalls, err := t.llm.ChatComplete(ctx, messages, c.AvailableTools)
	if err != nil {
		return nil, fmt.Errorf("thinking failed: %w", err)
	}

	// Trace LLM Response
	slog.Debug("LLM Response received", "content_len", len(content), "tool_calls", len(toolCalls))
	if len(content) > 0 {
		slog.Debug("LLM Content Preview", "content", content[:min(len(content), 200)]+"...")
	}

	thought := &Thought{
		Content: content,
	}

	if len(toolCalls) > 0 {
		thought.Action = &Action{
			Type:      ActionTypeToolCall,
			ToolCalls: toolCalls,
		}
	} else {
		// If no tools called, assume it's a final answer or just text response
		thought.Action = &Action{
			Type:    ActionTypeAnswer,
			Content: content,
		}
	}

	return thought, nil
}

func (t *UnifiedThinker) buildSystemPrompt(goal string, plan *Plan, c *CognitiveContext) string {
	var sb strings.Builder
	sb.WriteString(t.promptCfg.System + "\n")
	sb.WriteString(fmt.Sprintf("GOAL: %s\n", goal))

	if plan != nil {
		sb.WriteString(fmt.Sprintf("PLAN:\n%s\n", plan.Raw))
	}

	if len(c.Memories) > 0 {
		sb.WriteString("CONTEXT:\n")
		for _, m := range c.Memories {
			sb.WriteString(fmt.Sprintf("- %s\n", m))
		}
	}

	if len(c.Scratchpad) > 0 {
		sb.WriteString("HISTORY OF THOUGHTS:\n")
		for _, thought := range c.Scratchpad {
			sb.WriteString(fmt.Sprintf("%s\n", thought))
		}
	}
	if len(c.AvailableSkills) > 0 {
		sb.WriteString("AVAILABLE SKILLS:\n")
		for _, skillName := range c.AvailableSkills {
			sb.WriteString(fmt.Sprintf("- %s\n", skillName))
		}
	}
	if context := strings.TrimSpace(c.Metadata["skills_context"]); context != "" {
		sb.WriteString("SKILL CONTEXT:\n")
		sb.WriteString(context)
		sb.WriteString("\n")
	}

	sb.WriteString("\n" + t.promptCfg.Instruction)
	return sb.String()
}
