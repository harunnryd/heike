package tooling

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/policy"
)

func TestBuildRegistersBuiltInTools(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	workspaceID := "test-workspace-" + t.Name()
	policyEngine, err := policy.NewEngine(config.GovernanceConfig{}, workspaceID, "")
	if err != nil {
		t.Fatalf("create policy engine: %v", err)
	}
	cfg := &config.Config{}

	components, err := Build(workspaceID, policyEngine, t.TempDir(), cfg)
	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}
	if components == nil || components.Registry == nil || components.Runner == nil {
		t.Fatalf("Build() returned incomplete tooling components: %#v", components)
	}

	required := []string{
		"apply_patch",
		"click",
		"exec_command",
		"finance",
		"find",
		"image_query",
		"open",
		"screenshot",
		"search_query",
		"sports",
		"time",
		"view_image",
		"weather",
		"write_stdin",
	}
	for _, name := range required {
		if _, ok := components.Registry.Get(name); !ok {
			t.Fatalf("expected built-in tool %q to be registered", name)
		}
	}
}

func TestBuildLoadsWorkspaceCustomTools(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	workspaceID := "test-custom-tool-" + t.Name()
	workspacePath := t.TempDir()

	toolDir := filepath.Join(workspacePath, ".heike", "skills", "sample", "tools")
	if err := os.MkdirAll(toolDir, 0755); err != nil {
		t.Fatalf("create tool dir: %v", err)
	}

	manifest := `tools:
  - name: custom.echo
    language: shell
    script: echo.sh
    description: Echo test tool
`
	if err := os.WriteFile(filepath.Join(toolDir, "tools.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatalf("write tools.yaml: %v", err)
	}

	script := "#!/usr/bin/env sh\n" + `echo '{"message":"ok"}'` + "\n"
	if err := os.WriteFile(filepath.Join(toolDir, "echo.sh"), []byte(script), 0755); err != nil {
		t.Fatalf("write custom script: %v", err)
	}

	policyEngine, err := policy.NewEngine(config.GovernanceConfig{}, workspaceID, "")
	if err != nil {
		t.Fatalf("create policy engine: %v", err)
	}
	cfg := &config.Config{}

	components, err := Build(workspaceID, policyEngine, workspacePath, cfg)
	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}

	result, err := components.Runner.Execute(context.Background(), "custom.echo", json.RawMessage(`{}`), "")
	if err != nil {
		t.Fatalf("execute custom tool: %v", err)
	}

	var payload map[string]string
	if err := json.Unmarshal(result, &payload); err != nil {
		t.Fatalf("parse custom tool output: %v", err)
	}
	if payload["message"] != "ok" {
		t.Fatalf("unexpected custom tool output: %v", payload)
	}
}

func TestBuildLoadsBundledCustomTools(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	workspaceID := "test-bundled-tool-" + t.Name()
	workspacePath := t.TempDir()

	toolDir := filepath.Join(workspacePath, "skills", "sample", "tools")
	if err := os.MkdirAll(toolDir, 0755); err != nil {
		t.Fatalf("create tool dir: %v", err)
	}

	manifest := `tools:
  - name: bundled.echo
    language: shell
    script: echo.sh
    description: Bundled echo tool
`
	if err := os.WriteFile(filepath.Join(toolDir, "tools.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatalf("write tools.yaml: %v", err)
	}

	script := "#!/usr/bin/env sh\n" + `echo '{"message":"bundled"}'` + "\n"
	if err := os.WriteFile(filepath.Join(toolDir, "echo.sh"), []byte(script), 0755); err != nil {
		t.Fatalf("write custom script: %v", err)
	}

	policyEngine, err := policy.NewEngine(config.GovernanceConfig{}, workspaceID, "")
	if err != nil {
		t.Fatalf("create policy engine: %v", err)
	}
	cfg := &config.Config{}

	components, err := Build(workspaceID, policyEngine, workspacePath, cfg)
	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}

	result, err := components.Runner.Execute(context.Background(), "bundled.echo", json.RawMessage(`{}`), "")
	if err != nil {
		t.Fatalf("execute bundled tool: %v", err)
	}

	var payload map[string]string
	if err := json.Unmarshal(result, &payload); err != nil {
		t.Fatalf("parse bundled tool output: %v", err)
	}
	if payload["message"] != "bundled" {
		t.Fatalf("unexpected bundled tool output: %v", payload)
	}
}

func TestBuildDedupesCustomToolsByNameWorkspaceOverridesBundled(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	workspaceID := "test-dedupe-custom-tool-" + t.Name()
	workspacePath := t.TempDir()

	bundledToolDir := filepath.Join(workspacePath, "skills", "sample", "tools")
	if err := os.MkdirAll(bundledToolDir, 0755); err != nil {
		t.Fatalf("create bundled tool dir: %v", err)
	}
	bundledManifest := `tools:
  - name: dup.echo
    language: shell
    script: echo.sh
    description: Bundled duplicate tool
`
	if err := os.WriteFile(filepath.Join(bundledToolDir, "tools.yaml"), []byte(bundledManifest), 0644); err != nil {
		t.Fatalf("write bundled tools.yaml: %v", err)
	}
	bundledScript := "#!/usr/bin/env sh\n" + `echo '{"source":"bundled"}'` + "\n"
	if err := os.WriteFile(filepath.Join(bundledToolDir, "echo.sh"), []byte(bundledScript), 0755); err != nil {
		t.Fatalf("write bundled script: %v", err)
	}

	workspaceToolDir := filepath.Join(workspacePath, ".heike", "skills", "sample", "tools")
	if err := os.MkdirAll(workspaceToolDir, 0755); err != nil {
		t.Fatalf("create workspace tool dir: %v", err)
	}
	workspaceManifest := `tools:
  - name: dup.echo
    language: shell
    script: echo.sh
    description: Workspace duplicate tool
`
	if err := os.WriteFile(filepath.Join(workspaceToolDir, "tools.yaml"), []byte(workspaceManifest), 0644); err != nil {
		t.Fatalf("write workspace tools.yaml: %v", err)
	}
	workspaceScript := "#!/usr/bin/env sh\n" + `echo '{"source":"workspace"}'` + "\n"
	if err := os.WriteFile(filepath.Join(workspaceToolDir, "echo.sh"), []byte(workspaceScript), 0755); err != nil {
		t.Fatalf("write workspace script: %v", err)
	}

	policyEngine, err := policy.NewEngine(config.GovernanceConfig{}, workspaceID, "")
	if err != nil {
		t.Fatalf("create policy engine: %v", err)
	}
	cfg := &config.Config{}

	components, err := Build(workspaceID, policyEngine, workspacePath, cfg)
	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}

	result, err := components.Runner.Execute(context.Background(), "dup.echo", json.RawMessage(`{}`), "")
	if err != nil {
		t.Fatalf("execute duplicate tool: %v", err)
	}

	var payload map[string]string
	if err := json.Unmarshal(result, &payload); err != nil {
		t.Fatalf("parse duplicate tool output: %v", err)
	}
	if payload["source"] != "workspace" {
		t.Fatalf("expected workspace override, got: %v", payload)
	}
}
