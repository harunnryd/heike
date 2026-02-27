package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("ZAI_API_KEY", "")

	// We pass nil for cmd to skip flags
	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Server.Port != DefaultServerPort {
		t.Errorf("Expected default port %d, got %d", DefaultServerPort, cfg.Server.Port)
	}

	if cfg.Models.Default != DefaultModelDefault {
		t.Errorf("Expected default model %s, got %s", DefaultModelDefault, cfg.Models.Default)
	}
	if cfg.Models.Embedding != DefaultModelEmbedding {
		t.Errorf("Expected default embedding model %s, got %s", DefaultModelEmbedding, cfg.Models.Embedding)
	}
	if cfg.Orchestrator.MaxTurns != DefaultOrchestratorMaxTurns {
		t.Errorf("Expected default max turns %d, got %d", DefaultOrchestratorMaxTurns, cfg.Orchestrator.MaxTurns)
	}
	if cfg.Orchestrator.TokenBudget != DefaultOrchestratorTokenBudget {
		t.Errorf("Expected default token budget %d, got %d", DefaultOrchestratorTokenBudget, cfg.Orchestrator.TokenBudget)
	}
	if cfg.Governance.DailyToolLimit != DefaultGovernanceDailyToolLimit {
		t.Errorf("Expected default daily tool limit %d, got %d", DefaultGovernanceDailyToolLimit, cfg.Governance.DailyToolLimit)
	}
	if cfg.Tools.Web.MaxContentLength != DefaultWebToolMaxContentLength {
		t.Errorf("Expected default web max content length %d, got %d", DefaultWebToolMaxContentLength, cfg.Tools.Web.MaxContentLength)
	}
	if cfg.Tools.Web.BaseURL != DefaultWebToolBaseURL {
		t.Errorf("Expected default web base url %s, got %s", DefaultWebToolBaseURL, cfg.Tools.Web.BaseURL)
	}
	if cfg.Tools.Weather.BaseURL != DefaultWeatherToolBaseURL {
		t.Errorf("Expected default weather base url %s, got %s", DefaultWeatherToolBaseURL, cfg.Tools.Weather.BaseURL)
	}
	if cfg.Tools.Weather.Timeout != DefaultWeatherToolTimeout {
		t.Errorf("Expected default weather timeout %s, got %s", DefaultWeatherToolTimeout, cfg.Tools.Weather.Timeout)
	}
	if cfg.Tools.Finance.BaseURL != DefaultFinanceToolBaseURL {
		t.Errorf("Expected default finance base url %s, got %s", DefaultFinanceToolBaseURL, cfg.Tools.Finance.BaseURL)
	}
	if cfg.Tools.Finance.Timeout != DefaultFinanceToolTimeout {
		t.Errorf("Expected default finance timeout %s, got %s", DefaultFinanceToolTimeout, cfg.Tools.Finance.Timeout)
	}
	if cfg.Tools.Sports.BaseURL != DefaultSportsToolBaseURL {
		t.Errorf("Expected default sports base url %s, got %s", DefaultSportsToolBaseURL, cfg.Tools.Sports.BaseURL)
	}
	if cfg.Tools.Sports.Timeout != DefaultSportsToolTimeout {
		t.Errorf("Expected default sports timeout %s, got %s", DefaultSportsToolTimeout, cfg.Tools.Sports.Timeout)
	}
	if cfg.Tools.ImageQuery.BaseURL != DefaultImageQueryToolBaseURL {
		t.Errorf("Expected default image_query base url %s, got %s", DefaultImageQueryToolBaseURL, cfg.Tools.ImageQuery.BaseURL)
	}
	if cfg.Tools.ImageQuery.Timeout != DefaultImageQueryToolTimeout {
		t.Errorf("Expected default image_query timeout %s, got %s", DefaultImageQueryToolTimeout, cfg.Tools.ImageQuery.Timeout)
	}
	if cfg.Tools.Screenshot.Timeout != DefaultScreenshotToolTimeout {
		t.Errorf("Expected default screenshot timeout %s, got %s", DefaultScreenshotToolTimeout, cfg.Tools.Screenshot.Timeout)
	}
	if cfg.Tools.Screenshot.Renderer != DefaultScreenshotToolRenderer {
		t.Errorf("Expected default screenshot renderer %s, got %s", DefaultScreenshotToolRenderer, cfg.Tools.Screenshot.Renderer)
	}
	if cfg.Tools.ApplyPatch.Command != DefaultApplyPatchToolCommand {
		t.Errorf("Expected default apply_patch command %s, got %s", DefaultApplyPatchToolCommand, cfg.Tools.ApplyPatch.Command)
	}
	if cfg.Worker.ShutdownTimeout != DefaultWorkerShutdownTimeout {
		t.Errorf("Expected default worker shutdown timeout %s, got %s", DefaultWorkerShutdownTimeout, cfg.Worker.ShutdownTimeout)
	}
	if cfg.Scheduler.TickInterval != DefaultSchedulerTickInterval {
		t.Errorf("Expected default scheduler tick interval %s, got %s", DefaultSchedulerTickInterval, cfg.Scheduler.TickInterval)
	}
	if cfg.Scheduler.ShutdownTimeout != DefaultSchedulerShutdownTimeout {
		t.Errorf("Expected default scheduler shutdown timeout %s, got %s", DefaultSchedulerShutdownTimeout, cfg.Scheduler.ShutdownTimeout)
	}
	if cfg.Scheduler.MaxCatchupRuns != DefaultSchedulerMaxCatchupRuns {
		t.Errorf("Expected default scheduler max catchup runs %d, got %d", DefaultSchedulerMaxCatchupRuns, cfg.Scheduler.MaxCatchupRuns)
	}
	if cfg.Daemon.PreflightTimeout != DefaultDaemonPreflightTimeout {
		t.Errorf("Expected default daemon preflight timeout %s, got %s", DefaultDaemonPreflightTimeout, cfg.Daemon.PreflightTimeout)
	}
	if cfg.Store.LockTimeout != DefaultStoreLockTimeout {
		t.Errorf("Expected default store lock timeout %s, got %s", DefaultStoreLockTimeout, cfg.Store.LockTimeout)
	}
	if cfg.Store.LockRetry != DefaultStoreLockRetry {
		t.Errorf("Expected default store lock retry %s, got %s", DefaultStoreLockRetry, cfg.Store.LockRetry)
	}
	if cfg.Store.LockMaxRetry != DefaultStoreLockMaxRetry {
		t.Errorf("Expected default store lock max retry %d, got %d", DefaultStoreLockMaxRetry, cfg.Store.LockMaxRetry)
	}
	if cfg.Store.InboxSize != DefaultStoreInboxSize {
		t.Errorf("Expected default store inbox size %d, got %d", DefaultStoreInboxSize, cfg.Store.InboxSize)
	}
	if cfg.Store.TranscriptRotateMaxBytes != DefaultStoreTranscriptRotateMaxBytes {
		t.Errorf("Expected default transcript rotate max bytes %d, got %d", DefaultStoreTranscriptRotateMaxBytes, cfg.Store.TranscriptRotateMaxBytes)
	}
	if cfg.Orchestrator.DecomposeWordThreshold != DefaultOrchestratorDecomposeWordThresh {
		t.Errorf("Expected default decompose threshold %d, got %d", DefaultOrchestratorDecomposeWordThresh, cfg.Orchestrator.DecomposeWordThreshold)
	}
	if cfg.Orchestrator.SessionHistoryLimit != DefaultOrchestratorSessionHistoryLimit {
		t.Errorf("Expected default session history limit %d, got %d", DefaultOrchestratorSessionHistoryLimit, cfg.Orchestrator.SessionHistoryLimit)
	}
	if cfg.Orchestrator.SubTaskRetryMax != DefaultOrchestratorSubTaskRetryMax {
		t.Errorf("Expected default subtask retry max %d, got %d", DefaultOrchestratorSubTaskRetryMax, cfg.Orchestrator.SubTaskRetryMax)
	}
	if cfg.Orchestrator.MaxParallelSubTasks != DefaultOrchestratorMaxParallelSubTasks {
		t.Errorf("Expected default max parallel subtasks %d, got %d", DefaultOrchestratorMaxParallelSubTasks, cfg.Orchestrator.MaxParallelSubTasks)
	}
	if cfg.Orchestrator.StructuredRetryMax != DefaultOrchestratorStructuredRetryMax {
		t.Errorf("Expected default structured retry max %d, got %d", DefaultOrchestratorStructuredRetryMax, cfg.Orchestrator.StructuredRetryMax)
	}
	if cfg.Orchestrator.SubTaskRetryBackoff != DefaultOrchestratorSubTaskRetryBackoff {
		t.Errorf("Expected default subtask retry backoff %s, got %s", DefaultOrchestratorSubTaskRetryBackoff, cfg.Orchestrator.SubTaskRetryBackoff)
	}
	if cfg.Prompts.Thinker.System != DefaultThinkerSystemPrompt {
		t.Errorf("Expected default thinker system prompt, got %s", cfg.Prompts.Thinker.System)
	}
	if cfg.Auth.Codex.CallbackAddr != DefaultCodexAuthCallbackAddr {
		t.Errorf("Expected default codex callback addr %s, got %s", DefaultCodexAuthCallbackAddr, cfg.Auth.Codex.CallbackAddr)
	}
	if cfg.Auth.Codex.RedirectURI != DefaultCodexAuthRedirectURI {
		t.Errorf("Expected default codex redirect uri %s, got %s", DefaultCodexAuthRedirectURI, cfg.Auth.Codex.RedirectURI)
	}
	if cfg.Auth.Codex.OAuthTimeout != DefaultCodexAuthOAuthTimeout {
		t.Errorf("Expected default codex oauth timeout %s, got %s", DefaultCodexAuthOAuthTimeout, cfg.Auth.Codex.OAuthTimeout)
	}
	if cfg.Adapters.Telegram.UpdateTimeout != DefaultTelegramUpdateTimeout {
		t.Errorf("Expected default telegram update timeout %d, got %d", DefaultTelegramUpdateTimeout, cfg.Adapters.Telegram.UpdateTimeout)
	}
}

func TestLoadWithConfigFlag(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	content := []byte(`
server:
  port: 9090
models:
  default: custom-model
`)
	if err := os.WriteFile(configPath, content, 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cmd := &cobra.Command{}
	cmd.Flags().String("config", "", "config file path")
	if err := cmd.Flags().Set("config", configPath); err != nil {
		t.Fatalf("failed to set config flag: %v", err)
	}

	cfg, err := Load(cmd)
	if err != nil {
		t.Fatalf("failed to load config with --config: %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Fatalf("expected port 9090, got %d", cfg.Server.Port)
	}
	if cfg.Models.Default != "custom-model" {
		t.Fatalf("expected default model custom-model, got %s", cfg.Models.Default)
	}
}

func TestLoadWithMissingConfigFlagReturnsError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("config", "", "config file path")
	if err := cmd.Flags().Set("config", filepath.Join(t.TempDir(), "missing.yaml")); err != nil {
		t.Fatalf("failed to set config flag: %v", err)
	}

	if _, err := Load(cmd); err == nil {
		t.Fatal("expected error when --config points to missing file")
	}
}

func TestLoad_ExpandsConfiguredPaths(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configPath := filepath.Join(tmpDir, "config.yaml")
	content := []byte(`
daemon:
  workspace_path: ~/.heike/workspaces
auth:
  codex:
    token_path: ~/.heike/auth/codex.json
models:
  registry:
    - name: gpt-5.2-codex
      provider: openai-codex
      auth_file: ~/.heike/auth/codex.json
`)
	if err := os.WriteFile(configPath, content, 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	cmd := &cobra.Command{}
	cmd.Flags().String("config", "", "config file path")
	if err := cmd.Flags().Set("config", configPath); err != nil {
		t.Fatalf("set config flag: %v", err)
	}

	cfg, err := Load(cmd)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	wantWorkspacePath := filepath.Join(tmpDir, ".heike", "workspaces")
	if cfg.Daemon.WorkspacePath != wantWorkspacePath {
		t.Fatalf("workspace path = %q, want %q", cfg.Daemon.WorkspacePath, wantWorkspacePath)
	}

	wantTokenPath := filepath.Join(tmpDir, ".heike", "auth", "codex.json")
	if cfg.Auth.Codex.TokenPath != wantTokenPath {
		t.Fatalf("token path = %q, want %q", cfg.Auth.Codex.TokenPath, wantTokenPath)
	}
	if len(cfg.Models.Registry) != 1 {
		t.Fatalf("expected 1 model registry, got %d", len(cfg.Models.Registry))
	}
	if cfg.Models.Registry[0].AuthFile != wantTokenPath {
		t.Fatalf("model auth file = %q, want %q", cfg.Models.Registry[0].AuthFile, wantTokenPath)
	}
}
