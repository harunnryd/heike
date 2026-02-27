package cognitive

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/harunnryd/heike/internal/config"
)

type UnifiedReflector struct {
	llm       LLMClient
	promptCfg ReflectorPromptConfig
}

type ReflectorPromptConfig struct {
	System     string
	Guidelines string
}

func NewReflector(llm LLMClient, promptCfg ReflectorPromptConfig) *UnifiedReflector {
	if strings.TrimSpace(promptCfg.System) == "" {
		promptCfg.System = config.DefaultReflectorSystemPrompt
	}
	if strings.TrimSpace(promptCfg.Guidelines) == "" {
		promptCfg.Guidelines = config.DefaultReflectorGuidelinesPrompt
	}

	return &UnifiedReflector{
		llm:       llm,
		promptCfg: promptCfg,
	}
}

func (r *UnifiedReflector) Reflect(ctx context.Context, goal string, action *Action, result *ExecutionResult) (*Reflection, error) {
	slog.Info("UnifiedReflector reflecting")

	prompt := r.buildPrompt(goal, action, result)

	response, err := r.llm.Complete(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("reflection failed: %w", err)
	}

	reflection, mode := parseReflectionResponse(response)
	if reflection == nil {
		return &Reflection{
			Content:    "No reflection content returned.",
			NextAction: SignalContinue,
		}, nil
	}
	if mode != reflectionParseModeJSON {
		slog.Debug("Reflector fallback parser used", "mode", mode)
	}
	return reflection, nil
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
