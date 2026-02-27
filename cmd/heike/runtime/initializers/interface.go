package initializers

import (
	"context"

	"github.com/harunnryd/heike/internal/config"
)

type ComponentInitializer interface {
	Name() string
	Dependencies() []string
	Initialize(ctx context.Context, cfg *config.Config, workspaceID string) (interface{}, error)
}
