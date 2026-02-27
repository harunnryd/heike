package tooling

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/harunnryd/heike/internal/config"
	customexec "github.com/harunnryd/heike/internal/executor"
	"github.com/harunnryd/heike/internal/executor/runtimes"
	"github.com/harunnryd/heike/internal/policy"
	"github.com/harunnryd/heike/internal/sandbox"
	"github.com/harunnryd/heike/internal/skill/loader"
	"github.com/harunnryd/heike/internal/store"
	"github.com/harunnryd/heike/internal/tool"
	_ "github.com/harunnryd/heike/internal/tool/builtin"
)

type Components struct {
	Registry *tool.Registry
	Runner   *tool.Runner
}

func Build(workspaceID string, policyEngine *policy.Engine, workspacePath string, cfg *config.Config) (*Components, error) {
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace ID cannot be empty")
	}
	if policyEngine == nil {
		return nil, fmt.Errorf("policy engine cannot be nil")
	}
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	workspaceRoot, err := store.ResolveWorkspaceRootPath(cfg.Daemon.WorkspacePath)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace root path: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home directory: %w", err)
	}

	builtinOptions, err := resolveBuiltinOptions(cfg)
	if err != nil {
		return nil, err
	}

	toolRegistry := tool.NewRegistry()

	builtins, err := tool.InstantiateBuiltins(builtinOptions)
	if err != nil {
		return nil, fmt.Errorf("instantiate built-in tools: %w", err)
	}
	for _, builtin := range builtins {
		toolRegistry.Register(builtin)
	}
	slog.Info("Built-in tools registered", "count", len(builtins), "workspace", workspaceID)

	sandboxBasePath := filepath.Join(filepath.Dir(workspaceRoot), "sandboxes")
	if err := registerCustomTools(toolRegistry, workspaceID, home, workspacePath, sandboxBasePath); err != nil {
		return nil, fmt.Errorf("register custom tools: %w", err)
	}

	return &Components{
		Registry: toolRegistry,
		Runner:   tool.NewRunner(toolRegistry, policyEngine),
	}, nil
}

func registerCustomTools(registry *tool.Registry, workspaceID, home, workspacePath, sandboxBasePath string) error {
	toolLoader := loader.NewToolLoader(home)

	runtimeRegistry, err := runtimes.NewRuntimeRegistry()
	if err != nil {
		return fmt.Errorf("initialize runtime registry: %w", err)
	}

	runtimeExecutor := customexec.NewRuntimeBasedExecutor(runtimeRegistry)
	runtimeExecutor.SetWorkspaceID(workspaceID)

	sbManager, err := sandbox.NewBasicSandboxManager(sandboxBasePath, true)
	if err == nil {
		runtimeExecutor.SetSandbox(sbManager)
	}

	globalTools, err := toolLoader.LoadFromGlobal()
	if err != nil {
		return fmt.Errorf("load global custom tools: %w", err)
	}

	if workspacePath == "" {
		workspacePath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("get current workspace path: %w", err)
		}
	}

	bundledTools, err := toolLoader.LoadFromBundled(workspacePath)
	if err != nil {
		return fmt.Errorf("load bundled custom tools: %w", err)
	}

	workspaceTools, err := toolLoader.LoadFromWorkspace(workspacePath)
	if err != nil {
		return fmt.Errorf("load workspace custom tools: %w", err)
	}

	allTools := append(globalTools, bundledTools...)
	allTools = append(allTools, workspaceTools...)
	dedupedTools := dedupeCustomToolsByName(allTools)
	for _, ct := range dedupedTools {
		if err := runtimeExecutor.Validate(ct); err != nil {
			return fmt.Errorf("validate custom tool %q: %w", ct.Name, err)
		}

		adapter, err := newCustomToolAdapter(ct, runtimeExecutor)
		if err != nil {
			return fmt.Errorf("create custom tool adapter for %q: %w", ct.Name, err)
		}

		registry.Register(adapter)
	}

	if len(dedupedTools) > 0 {
		slog.Info("Custom tools registered", "count", len(dedupedTools), "workspace", workspaceID)
	}

	return nil
}

func dedupeCustomToolsByName(tools []*tool.CustomTool) []*tool.CustomTool {
	if len(tools) == 0 {
		return nil
	}

	indexByName := make(map[string]int, len(tools))
	ordered := make([]*tool.CustomTool, 0, len(tools))

	for _, ct := range tools {
		if ct == nil {
			continue
		}
		name := tool.NormalizeToolName(ct.Name)
		if name == "" {
			continue
		}

		if idx, exists := indexByName[name]; exists {
			ordered[idx] = ct
			continue
		}

		indexByName[name] = len(ordered)
		ordered = append(ordered, ct)
	}

	return ordered
}

type customToolAdapter struct {
	custom   *tool.CustomTool
	executor customToolExecutor
	params   map[string]interface{}
	meta     tool.ToolMetadata
}

type customToolExecutor interface {
	Execute(ctx context.Context, ct *tool.CustomTool, input json.RawMessage) (json.RawMessage, error)
	Validate(ct *tool.CustomTool) error
}

func newCustomToolAdapter(ct *tool.CustomTool, exec customToolExecutor) (*customToolAdapter, error) {
	params := map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
	if len(ct.Parameters) > 0 {
		if err := json.Unmarshal(ct.Parameters, &params); err != nil {
			return nil, fmt.Errorf("parse custom tool parameters: %w", err)
		}
	}

	return &customToolAdapter{
		custom:   ct,
		executor: exec,
		params:   params,
		meta:     buildCustomToolMetadata(ct),
	}, nil
}

func (cta *customToolAdapter) Name() string {
	return tool.NormalizeToolName(cta.custom.Name)
}

func (cta *customToolAdapter) Description() string {
	return cta.custom.Description
}

func (cta *customToolAdapter) Parameters() map[string]interface{} {
	return cta.params
}

func (cta *customToolAdapter) ToolMetadata() tool.ToolMetadata {
	return cta.meta
}

func (cta *customToolAdapter) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	return cta.executor.Execute(ctx, cta.custom, input)
}

func buildCustomToolMetadata(ct *tool.CustomTool) tool.ToolMetadata {
	if ct == nil {
		return tool.ToolMetadata{}
	}

	source := ct.Source
	if strings.TrimSpace(source) == "" {
		source = inferCustomToolSource(ct.ScriptPath)
	}

	risk := ct.Risk
	if risk == "" {
		risk = inferRiskFromSandbox(ct.SandboxLevel)
	}

	capabilities := append([]string(nil), ct.Capabilities...)
	if len(capabilities) == 0 {
		capabilities = inferCapabilitiesFromCustomTool(ct)
	}

	return tool.ToolMetadata{
		Source:       source,
		Capabilities: capabilities,
		Risk:         risk,
	}
}

func inferCustomToolSource(scriptPath string) string {
	path := strings.ToLower(scriptPath)
	switch {
	case strings.Contains(path, string(filepath.Separator)+".heike"+string(filepath.Separator)+"skills"+string(filepath.Separator)):
		return "skill"
	default:
		return "runtime"
	}
}

func inferRiskFromSandbox(level tool.SandboxLevel) tool.RiskLevel {
	switch level {
	case tool.SandboxContainer:
		return tool.RiskLow
	case tool.SandboxAdvanced, tool.SandboxMedium:
		return tool.RiskMedium
	default:
		return tool.RiskMedium
	}
}

func inferCapabilitiesFromCustomTool(ct *tool.CustomTool) []string {
	name := strings.TrimSpace(ct.Name)
	if name == "" {
		return []string{"custom.run"}
	}

	caps := []string{"custom.run"}
	if ct.Language != "" {
		caps = append(caps, fmt.Sprintf("language.%s", strings.ToLower(string(ct.Language))))
	}
	caps = append(caps, strings.ReplaceAll(name, "_", "."))
	return caps
}
