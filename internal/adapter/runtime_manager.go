package adapter

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/harunnryd/heike/internal/config"
)

type RuntimeAdapterOptions struct {
	IncludeCLI          bool
	IncludeSystemNull   bool
	RequireSlackSecrets bool
}

type RuntimeManager struct {
	mu      sync.RWMutex
	inputs  []InputAdapter
	outputs []OutputAdapter
	started bool
}

func NewRuntimeManager(cfg config.AdaptersConfig, eventHandler EventHandler, opts RuntimeAdapterOptions) (*RuntimeManager, error) {
	m := &RuntimeManager{}

	if opts.IncludeCLI {
		m.outputs = append(m.outputs, NewCLIAdapter())
	}
	if opts.IncludeSystemNull {
		m.outputs = append(m.outputs, NewNullAdapter("scheduler"), NewNullAdapter("system"))
	}

	if cfg.Slack.Enabled {
		if opts.RequireSlackSecrets {
			if strings.TrimSpace(cfg.Slack.SigningSecret) == "" && strings.TrimSpace(os.Getenv("SLACK_SIGNING_SECRET")) == "" {
				return nil, fmt.Errorf("adapters.slack.signing_secret is required when slack adapter is enabled")
			}
		}
		if strings.TrimSpace(cfg.Slack.BotToken) == "" && strings.TrimSpace(os.Getenv("SLACK_BOT_TOKEN")) == "" {
			return nil, fmt.Errorf("adapters.slack.bot_token is required when slack adapter is enabled")
		}

		slackAdapter := NewSlackAdapter(cfg.Slack.Port, cfg.Slack.SigningSecret, cfg.Slack.BotToken, eventHandler)
		m.inputs = append(m.inputs, slackAdapter)
		m.outputs = append(m.outputs, slackAdapter)
	}

	if cfg.Telegram.Enabled {
		token := strings.TrimSpace(cfg.Telegram.BotToken)
		if token == "" {
			return nil, fmt.Errorf("adapters.telegram.bot_token is required when telegram adapter is enabled")
		}

		telegramAdapter := NewTelegramAdapter(token, eventHandler, cfg.Telegram.UpdateTimeout)
		m.inputs = append(m.inputs, telegramAdapter)
		m.outputs = append(m.outputs, telegramAdapter)
	}

	m.outputs = dedupeOutputAdapters(m.outputs)
	return m, nil
}

func (m *RuntimeManager) OutputAdapters() []OutputAdapter {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]OutputAdapter, len(m.outputs))
	copy(out, m.outputs)
	return out
}

func (m *RuntimeManager) Start(ctx context.Context) {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return
	}
	m.started = true
	inputs := make([]InputAdapter, len(m.inputs))
	copy(inputs, m.inputs)
	m.mu.Unlock()

	for _, input := range inputs {
		adapter := input
		go func() {
			slog.Info("Starting input adapter", "adapter", adapter.Name())
			if err := adapter.Start(ctx); err != nil && ctx.Err() == nil {
				slog.Error("Input adapter stopped with error", "adapter", adapter.Name(), "error", err)
			}
		}()
	}
}

func (m *RuntimeManager) Stop(ctx context.Context) error {
	m.mu.Lock()
	if !m.started {
		m.mu.Unlock()
		return nil
	}
	m.started = false
	inputs := make([]InputAdapter, len(m.inputs))
	copy(inputs, m.inputs)
	m.mu.Unlock()

	var errs []string
	for _, input := range inputs {
		if err := input.Stop(ctx); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", input.Name(), err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to stop adapters: %s", strings.Join(errs, "; "))
	}
	return nil
}

func (m *RuntimeManager) Health(ctx context.Context) error {
	m.mu.RLock()
	inputs := make([]InputAdapter, len(m.inputs))
	copy(inputs, m.inputs)
	outputs := make([]OutputAdapter, len(m.outputs))
	copy(outputs, m.outputs)
	m.mu.RUnlock()

	for _, input := range inputs {
		if err := input.Health(ctx); err != nil {
			return fmt.Errorf("input adapter %s unhealthy: %w", input.Name(), err)
		}
	}
	for _, output := range outputs {
		if err := output.Health(ctx); err != nil {
			return fmt.Errorf("output adapter %s unhealthy: %w", output.Name(), err)
		}
	}
	return nil
}

func dedupeOutputAdapters(adapters []OutputAdapter) []OutputAdapter {
	if len(adapters) == 0 {
		return nil
	}
	indexByName := make(map[string]int, len(adapters))
	ordered := make([]OutputAdapter, 0, len(adapters))
	for _, adapter := range adapters {
		if adapter == nil {
			continue
		}
		name := strings.TrimSpace(adapter.Name())
		if name == "" {
			continue
		}
		if idx, exists := indexByName[name]; exists {
			ordered[idx] = adapter
			continue
		}
		indexByName[name] = len(ordered)
		ordered = append(ordered, adapter)
	}
	return ordered
}
