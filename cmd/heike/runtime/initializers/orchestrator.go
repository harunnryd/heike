package initializers

import (
	"context"
	"fmt"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/egress"
	"github.com/harunnryd/heike/internal/orchestrator"
	"github.com/harunnryd/heike/internal/policy"
	"github.com/harunnryd/heike/internal/skill"
	"github.com/harunnryd/heike/internal/store"
	"github.com/harunnryd/heike/internal/tool"
)

type OrchestratorInitializer struct {
	storeWorker   *store.Worker
	toolRunner    *tool.Runner
	policyEngine  *policy.Engine
	skillRegistry *skill.Registry
	egress        egress.Egress
}

func NewOrchestratorInitializer(storeWorker *store.Worker, toolRunner *tool.Runner, policyEngine *policy.Engine, skillRegistry *skill.Registry, egress egress.Egress) *OrchestratorInitializer {
	return &OrchestratorInitializer{
		storeWorker:   storeWorker,
		toolRunner:    toolRunner,
		policyEngine:  policyEngine,
		skillRegistry: skillRegistry,
		egress:        egress,
	}
}

func (oi *OrchestratorInitializer) Name() string {
	return "orchestrator"
}

func (oi *OrchestratorInitializer) Dependencies() []string {
	return []string{"store", "tools", "policy"}
}

func (oi *OrchestratorInitializer) Initialize(ctx context.Context, cfg *config.Config, workspaceID string) (interface{}, error) {
	if oi.storeWorker == nil {
		return nil, fmt.Errorf("store worker not initialized")
	}
	if oi.toolRunner == nil {
		return nil, fmt.Errorf("tool runner not initialized")
	}
	if oi.policyEngine == nil {
		return nil, fmt.Errorf("policy engine not initialized")
	}

	if oi.egress == nil {
		return nil, fmt.Errorf("egress not initialized")
	}

	if oi.skillRegistry == nil {
		oi.skillRegistry = skill.NewRegistry()
	}

	orch, err := orchestrator.NewKernel(
		*cfg,
		oi.storeWorker,
		oi.toolRunner,
		oi.policyEngine,
		oi.skillRegistry,
		oi.egress,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create kernel: %w", err)
	}
	if err := orch.Init(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize kernel: %w", err)
	}
	return orch, nil
}
