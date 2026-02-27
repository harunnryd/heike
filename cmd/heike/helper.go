package main

import (
	"context"
	"fmt"

	"github.com/harunnryd/heike/cmd/heike/runtime"

	"github.com/harunnryd/heike/internal/config"

	"github.com/spf13/cobra"
)

func executeWithRuntime(cmd *cobra.Command, fn func(*runtime.RuntimeComponents) error) error {
	workspaceID := runtime.ResolveWorkspaceID(cmd)

	cfg, err := config.Load(cmd)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := runtime.NewRuntimeBuilder().
		WithContext(ctx).
		WithConfig(cfg).
		WithWorkspace(workspaceID)

	components, err := builder.Build()
	if err != nil {
		return fmt.Errorf("failed to initialize runtime: %w", err)
	}
	defer components.Stop()

	return fn(components)
}
