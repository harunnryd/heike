package cognitive

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/harunnryd/heike/internal/config"
	heikeErrors "github.com/harunnryd/heike/internal/errors"
)

type UnifiedReflector struct {
	llm                LLMClient
	promptCfg          ReflectorPromptConfig
	structuredRetryMax int
}

type ReflectorPromptConfig struct {
	System     string
	Guidelines string
}

func NewReflector(llm LLMClient, promptCfg ReflectorPromptConfig, structuredRetryMax int) *UnifiedReflector {
	if strings.TrimSpace(promptCfg.System) == "" {
		promptCfg.System = config.DefaultReflectorSystemPrompt
	}
	if strings.TrimSpace(promptCfg.Guidelines) == "" {
		promptCfg.Guidelines = config.DefaultReflectorGuidelinesPrompt
	}
	if structuredRetryMax < 0 {
		structuredRetryMax = 0
	}

	return &UnifiedReflector{
		llm:                llm,
		promptCfg:          promptCfg,
		structuredRetryMax: structuredRetryMax,
	}
}

func (r *UnifiedReflector) Reflect(ctx context.Context, goal string, action *Action, result *ExecutionResult) (*Reflection, error) {
	slog.Info("UnifiedReflector reflecting")

	prompt := r.buildPrompt(goal, action, result)

	for attempt := 0; attempt <= r.structuredRetryMax; attempt++ {
		response, err := r.llm.Complete(ctx, prompt)
		if err != nil {
			return nil, fmt.Errorf("reflection failed: %w", err)
		}

		reflection, mode := parseReflectionResponse(response)
		if reflection != nil {
			return reflection, nil
		}

		slog.Warn("Reflector returned invalid structured output",
			"attempt", attempt+1,
			"max_attempts", r.structuredRetryMax+1,
			"mode", mode)
	}

	return nil, heikeErrors.InvalidModelOutput("reflector returned invalid JSON output")
}

func (r *UnifiedReflector) buildPrompt(goal string, action *Action, result *ExecutionResult) string {
	var sb strings.Builder
	sb.WriteString(r.promptCfg.System + "\n")
	sb.WriteString(fmt.Sprintf("GOAL: %s\n", goal))

	if action.Type == ActionTypeToolCall {
		sb.WriteString("ACTION: Tool Execution\n")
	} else {
		sb.WriteString("ACTION: Answer User\n")
	}

	sb.WriteString(fmt.Sprintf("RESULT:\n%s\n", result.Output))

	sb.WriteString("\n" + r.promptCfg.Guidelines + "\n")
	return sb.String()
}
