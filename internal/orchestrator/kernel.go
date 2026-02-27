package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/harunnryd/heike/internal/cognitive"
	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/egress"
	"github.com/harunnryd/heike/internal/ingress"
	"github.com/harunnryd/heike/internal/logger"
	"github.com/harunnryd/heike/internal/model"
	"github.com/harunnryd/heike/internal/model/contract"
	"github.com/harunnryd/heike/internal/orchestrator/command"
	"github.com/harunnryd/heike/internal/orchestrator/memory"
	"github.com/harunnryd/heike/internal/orchestrator/session"
	"github.com/harunnryd/heike/internal/orchestrator/task"
	"github.com/harunnryd/heike/internal/policy"
	"github.com/harunnryd/heike/internal/skill"
	"github.com/harunnryd/heike/internal/store"
	"github.com/harunnryd/heike/internal/tool"
)

// Kernel orchestrates the high-level request flow
type Kernel interface {
	Execute(ctx context.Context, evt *ingress.Event) error
	Init(ctx context.Context) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Health(ctx context.Context) (*ComponentHealth, error)
}

type ComponentHealth struct {
	Name    string
	Healthy bool
	Error   error
}

type DefaultKernel struct {
	cfg     config.Config
	running bool
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc

	// Managers
	session session.Manager
	task    task.Manager
	command command.Handler
	memory  cognitive.MemoryManager
}

func NewKernel(
	cfg config.Config,
	store *store.Worker,
	runner *tool.Runner,
	policy *policy.Engine,
	skills *skill.Registry,
	egress egress.Egress,
) (*DefaultKernel, error) {
	// Initialize Core Services
	router, err := model.NewModelRouter(cfg.Models)
	if err != nil {
		return nil, fmt.Errorf("model router init: %w", err)
	}

	llmExecutor := NewLLMAdapter(router, cfg.Models.Default) // Adapter for Cognitive Engine

	// Initialize Memory
	memMgr := memory.NewManager(store, router, cfg.Models.Embedding)

	// Initialize Cognitive Engine
	planner := cognitive.NewPlanner(llmExecutor, cognitive.PlannerPromptConfig{
		System: cfg.Prompts.Planner.System,
		Output: cfg.Prompts.Planner.Output,
	}, cfg.Orchestrator.StructuredRetryMax)
	thinker := cognitive.NewThinker(llmExecutor, cognitive.ThinkerPromptConfig{
		System:      cfg.Prompts.Thinker.System,
		Instruction: cfg.Prompts.Thinker.Instruction,
	})

	// Adapter for Actor (ToolRunner + Egress)
	actorAdapter := NewActorAdapter(runner)
	actor := cognitive.NewActor(actorAdapter)

	reflector := cognitive.NewReflector(llmExecutor, cognitive.ReflectorPromptConfig{
		System:     cfg.Prompts.Reflector.System,
		Guidelines: cfg.Prompts.Reflector.Guidelines,
	}, cfg.Orchestrator.StructuredRetryMax)

	engine := cognitive.NewEngine(
		planner,
		thinker,
		actor,
		reflector,
		memMgr,
		cfg.Orchestrator.MaxTurns,
		cfg.Orchestrator.TokenBudget,
	)

	subTaskRetryBackoff, err := config.DurationOrDefault(
		cfg.Orchestrator.SubTaskRetryBackoff,
		config.DefaultOrchestratorSubTaskRetryBackoff,
	)
	if err != nil {
		return nil, fmt.Errorf("parse orchestrator subtask retry backoff: %w", err)
	}

	// Initialize Managers
	sessMgr := session.NewManager(store, memMgr, cfg.Orchestrator.SessionHistoryLimit)
	cmdHandler := command.NewHandler(policy, sessMgr, store, egress)

	decomposer := task.NewDecomposer(llmExecutor, cfg.Orchestrator.DecomposeWordThreshold, task.DecomposerPromptConfig{
		System:       cfg.Prompts.Decomposer.System,
		Requirements: cfg.Prompts.Decomposer.Requirements,
	})
	toolBroker := task.NewDefaultToolBroker(cfg.Orchestrator.MaxToolsPerTurn)
	taskMgr := task.NewManager(
		engine,
		decomposer,
		sessMgr,
		runner.GetDescriptors(),
		toolBroker,
		skills,
		cfg.Orchestrator.SubTaskRetryMax,
		subTaskRetryBackoff,
		cfg.Orchestrator.MaxSubTasks,
		cfg.Orchestrator.MaxParallelSubTasks,
		egress,
	)

	return &DefaultKernel{
		cfg:     cfg,
		session: sessMgr,
		task:    taskMgr,
		command: cmdHandler,
		memory:  memMgr,
	}, nil
}

func (k *DefaultKernel) Init(ctx context.Context) error {
	k.ctx, k.cancel = context.WithCancel(ctx)
	slog.Info("Kernel initialized")
	return nil
}

func (k *DefaultKernel) Start(ctx context.Context) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.running {
		return nil
	}
	k.running = true
	slog.Info("Kernel started")
	return nil
}

func (k *DefaultKernel) Stop(ctx context.Context) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if !k.running {
		return nil
	}
	k.running = false
	k.cancel()
	slog.Info("Kernel stopped")
	return nil
}

func (k *DefaultKernel) Health(ctx context.Context) (*ComponentHealth, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	status := &ComponentHealth{
		Name:    "Kernel",
		Healthy: k.running,
	}
	if !k.running {
		status.Error = fmt.Errorf("kernel not running")
	}
	return status, nil
}

func (k *DefaultKernel) Execute(ctx context.Context, evt *ingress.Event) error {
	ctx = logger.WithTraceID(ctx, evt.ID)
	ctx = logger.WithSessionID(ctx, evt.SessionID)
	slog.Info("Kernel executing event", "id", evt.ID, "type", evt.Type)

	// Slash Commands
	if evt.Type == ingress.TypeCommand || (evt.Type == ingress.TypeUserMessage && k.command.CanHandle(evt.Content)) {
		return k.command.Execute(ctx, evt.SessionID, evt.Content)
	}

	// Task Execution
	if evt.Type == ingress.TypeUserMessage {
		// Persist user message first
		if err := k.session.AppendInteraction(ctx, evt.SessionID, "user", evt.Content); err != nil {
			slog.Warn("Failed to persist user message", "error", err)
		}

		return k.task.HandleRequest(ctx, evt.SessionID, evt.Content)
	}

	return nil
}

// ActorAdapter adapts ToolRunner and Egress to Cognitive Actor interfaces
type ActorAdapter struct {
	runner *tool.Runner
}

func NewActorAdapter(r *tool.Runner) *ActorAdapter {
	return &ActorAdapter{runner: r}
}

func (a *ActorAdapter) Execute(ctx context.Context, name string, args json.RawMessage, input string) (json.RawMessage, error) {
	return a.runner.Execute(ctx, name, args, input)
}

// LLMExecutorAdapter adapts Orchestrator LLMExecutor to Cognitive LLMClient
type LLMExecutorAdapter struct {
	router    model.ModelRouter
	modelName string
}

func NewLLMAdapter(router model.ModelRouter, modelName string) *LLMExecutorAdapter {
	return &LLMExecutorAdapter{
		router:    router,
		modelName: modelName,
	}
}

func (l *LLMExecutorAdapter) Complete(ctx context.Context, prompt string) (string, error) {
	req := contract.CompletionRequest{
		Model: l.modelName,
		Messages: []contract.Message{
			{Role: "user", Content: prompt},
		},
	}

	resp, err := l.router.Route(ctx, l.modelName, req)
	if err != nil {
		return "", fmt.Errorf("LLM execution failed: %w", err)
	}

	return resp.Content, nil
}

func (l *LLMExecutorAdapter) ChatComplete(ctx context.Context, messages []contract.Message, tools []contract.ToolDef) (string, []*contract.ToolCall, error) {
	req := contract.CompletionRequest{
		Model:    l.modelName,
		Messages: messages,
		Tools:    tools,
	}

	resp, err := l.router.Route(ctx, l.modelName, req)
	if err != nil {
		return "", nil, fmt.Errorf("LLM execution with tools failed: %w", err)
	}

	return resp.Content, resp.ToolCalls, nil
}
