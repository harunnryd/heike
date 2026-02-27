package initializers

import (
	"context"
	"fmt"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/policy"
	"github.com/harunnryd/heike/internal/store"
	"github.com/harunnryd/heike/internal/tool"
	"github.com/harunnryd/heike/internal/tooling"
)

type ToolsInitializer struct {
	storeWorker  *store.Worker
	policyEngine *policy.Engine
}

func NewToolsInitializer(storeWorker *store.Worker, policyEngine *policy.Engine) *ToolsInitializer {
	return &ToolsInitializer{
		storeWorker:  storeWorker,
		policyEngine: policyEngine,
	}
}

func (ti *ToolsInitializer) Name() string {
	return "tools"
}

func (ti *ToolsInitializer) Dependencies() []string {
	return []string{"store", "policy"}
}

func (ti *ToolsInitializer) Initialize(ctx context.Context, cfg *config.Config, workspaceID string) (interface{}, error) {
	_ = ctx
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	if ti.storeWorker == nil {
		return nil, fmt.Errorf("store worker not initialized")
	}
	if ti.policyEngine == nil {
		return nil, fmt.Errorf("policy engine not initialized")
	}

	toolingComponents, err := tooling.Build(workspaceID, ti.policyEngine, "", cfg)
	if err != nil {
		return nil, fmt.Errorf("build tooling: %w", err)
	}

	return struct {
		Registry *tool.Registry
		Runner   *tool.Runner
	}{
		Registry: toolingComponents.Registry,
		Runner:   toolingComponents.Runner,
	}, nil
}
