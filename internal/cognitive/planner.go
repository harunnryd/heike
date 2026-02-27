package cognitive

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/harunnryd/heike/internal/config"
)

type UnifiedPlanner struct {
	llm       LLMClient
	promptCfg PlannerPromptConfig
}

type PlannerPromptConfig struct {
	System string
	Output string
}

func NewPlanner(llm LLMClient, promptCfg PlannerPromptConfig) *UnifiedPlanner {
	if strings.TrimSpace(promptCfg.System) == "" {
		promptCfg.System = config.DefaultPlannerSystemPrompt
	}
	if strings.TrimSpace(promptCfg.Output) == "" {
		promptCfg.Output = config.DefaultPlannerOutputPrompt
	}

	return &UnifiedPlanner{
		llm:       llm,
		promptCfg: promptCfg,
	}
}

func (p *UnifiedPlanner) Plan(ctx context.Context, goal string, c *CognitiveContext) (*Plan, error) {
	slog.Info("UnifiedPlanner planning", "goal", goal)

	prompt := p.buildPrompt(goal, c)

	response, err := p.llm.Complete(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("planning failed: %w", err)
	}

	normalized := cleanModelJSON(response)
	steps, mode := parsePlannerResponse(normalized, goal)
	if mode != plannerParseModeJSONArray {
		slog.Debug("Planner fallback parser used", "mode", mode, "steps", len(steps))
	}

	return &Plan{
		Raw:   normalized,
		Steps: steps,
	}, nil
}

func (p *UnifiedPlanner) buildPrompt(goal string, c *CognitiveContext) string {
	var sb strings.Builder
	sb.WriteString(p.promptCfg.System + "\n")

	if len(c.AvailableTools) > 0 {
		sb.WriteString("\nAVAILABLE TOOLS:\n")
		for _, t := range c.AvailableTools {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", t.Name, t.Description))
		}
	}

	if len(c.Memories) > 0 {
		sb.WriteString("\nRELEVANT CONTEXT:\n")
		for _, m := range c.Memories {
			sb.WriteString(fmt.Sprintf("- %s\n", m))
		}
	}
	if len(c.AvailableSkills) > 0 {
		sb.WriteString("\nAVAILABLE SKILLS:\n")
		for _, name := range c.AvailableSkills {
			sb.WriteString(fmt.Sprintf("- %s\n", name))
		}
	}
	if skillContext := strings.TrimSpace(c.Metadata["skills_context"]); skillContext != "" {
		sb.WriteString("\nSKILL CONTEXT:\n")
		sb.WriteString(skillContext + "\n")
	}

	// Add scratchpad/previous thoughts if this is a replanning step
	if len(c.Scratchpad) > 0 {
		sb.WriteString("\nPREVIOUS THOUGHTS:\n")
		for _, t := range c.Scratchpad {
			sb.WriteString(fmt.Sprintf("> %s\n", t))
		}
	}

	sb.WriteString(fmt.Sprintf("\nGOAL: %s\n", goal))
	sb.WriteString("\n" + p.promptCfg.Output)

	return sb.String()
}
