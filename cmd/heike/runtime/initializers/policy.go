package initializers

import (
	"context"
	"fmt"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/policy"
)

type PolicyInitializer struct{}

func NewPolicyInitializer() *PolicyInitializer {
	return &PolicyInitializer{}
}

func (pi *PolicyInitializer) Name() string {
	return "policy"
}

func (pi *PolicyInitializer) Dependencies() []string {
	return []string{}
}

func (pi *PolicyInitializer) Initialize(ctx context.Context, cfg *config.Config, workspaceID string) (interface{}, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	workspaceRootPath := ""
	workspaceRootPath = cfg.Daemon.WorkspacePath

	engine, err := policy.NewEngine(cfg.Governance, workspaceID, workspaceRootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create policy engine: %w", err)
	}
	return engine, nil
}
