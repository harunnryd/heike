package runtime

import (
	"context"
	"fmt"

	"github.com/harunnryd/heike/internal/config"
)

type RuntimeBuilder interface {
	WithContext(ctx context.Context) RuntimeBuilder
	WithConfig(cfg *config.Config) RuntimeBuilder
	WithWorkspace(workspaceID string) RuntimeBuilder
	WithAdapterOptions(opts AdapterBuildOptions) RuntimeBuilder
	Build() (*RuntimeComponents, error)
}

type DefaultRuntimeBuilder struct {
	ctx         context.Context
	cfg         *config.Config
	workspaceID string
	adapterOpts AdapterBuildOptions
}

func NewRuntimeBuilder() RuntimeBuilder {
	return &DefaultRuntimeBuilder{
		adapterOpts: DefaultAdapterBuildOptions(),
	}
}

func (b *DefaultRuntimeBuilder) WithContext(ctx context.Context) RuntimeBuilder {
	b.ctx = ctx
	return b
}

func (b *DefaultRuntimeBuilder) WithConfig(cfg *config.Config) RuntimeBuilder {
	b.cfg = cfg
	return b
}

func (b *DefaultRuntimeBuilder) WithWorkspace(workspaceID string) RuntimeBuilder {
	b.workspaceID = workspaceID
	return b
}

func (b *DefaultRuntimeBuilder) WithAdapterOptions(opts AdapterBuildOptions) RuntimeBuilder {
	b.adapterOpts = opts
	return b
}

func (b *DefaultRuntimeBuilder) Build() (*RuntimeComponents, error) {
	if b.ctx == nil {
		b.ctx = context.Background()
	}

	if b.cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	if b.workspaceID == "" {
		b.workspaceID = DefaultWorkspaceID
	}

	components, err := NewRuntimeComponentsWithOptions(b.ctx, b.cfg, b.workspaceID, b.adapterOpts)
	if err != nil {
		return nil, err
	}

	return components, nil
}
