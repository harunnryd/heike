package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/harunnryd/heike/internal/pathutil"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/cobra"
)

type Config struct {
	Server       ServerConfig       `koanf:"server"`
	Models       ModelsConfig       `koanf:"models"`
	Governance   GovernanceConfig   `koanf:"governance"`
	Auth         AuthConfig         `koanf:"auth"`
	Adapters     AdaptersConfig     `koanf:"adapters"`
	Discovery    DiscoveryConfig    `koanf:"discovery"`
	Tools        ToolsConfig        `koanf:"tools"`
	Ingress      IngressConfig      `koanf:"ingress"`
	Prompts      PromptsConfig      `koanf:"prompts"`
	Store        StoreConfig        `koanf:"store"`
	Orchestrator OrchestratorConfig `koanf:"orchestrator"`
	Worker       WorkerConfig       `koanf:"worker"`
	Scheduler    SchedulerConfig    `koanf:"scheduler"`
	Zanshin      ZanshinConfig      `koanf:"zanshin"`
	Daemon       DaemonConfig       `koanf:"daemon"`
}

type PromptsConfig struct {
	Planner    PlannerPromptConfig    `koanf:"planner"`
	Thinker    ThinkerPromptConfig    `koanf:"thinker"`
	Reflector  ReflectorPromptConfig  `koanf:"reflector"`
	Decomposer DecomposerPromptConfig `koanf:"decomposer"`
}

type PlannerPromptConfig struct {
	System string `koanf:"system"`
	Output string `koanf:"output"`
}

type ThinkerPromptConfig struct {
	System      string `koanf:"system"`
	Instruction string `koanf:"instruction"`
}

type ReflectorPromptConfig struct {
	System     string `koanf:"system"`
	Guidelines string `koanf:"guidelines"`
}

type DecomposerPromptConfig struct {
	System       string `koanf:"system"`
	Requirements string `koanf:"requirements"`
}

type StoreConfig struct {
	LockTimeout              string `koanf:"lock_timeout"`
	LockRetry                string `koanf:"lock_retry"`
	LockMaxRetry             int    `koanf:"lock_max_retry"`
	InboxSize                int    `koanf:"inbox_size"`
	TranscriptRotateMaxBytes int64  `koanf:"transcript_rotate_max_bytes"`
}

type WorkerConfig struct {
	ShutdownTimeout string `koanf:"shutdown_timeout"`
}

type SchedulerConfig struct {
	TickInterval         string `koanf:"tick_interval"`
	ShutdownTimeout      string `koanf:"shutdown_timeout"`
	LeaseDuration        string `koanf:"lease_duration"`
	MaxCatchupRuns       int    `koanf:"max_catchup_runs"`
	InFlightPollInterval string `koanf:"in_flight_poll_interval"`
	HeartbeatWorkspaceID string `koanf:"heartbeat_workspace_id"`
}

type DaemonConfig struct {
	ShutdownTimeout        string `koanf:"shutdown_timeout"`
	HealthCheckInterval    string `koanf:"health_check_interval"`
	StartupShutdownTimeout string `koanf:"startup_shutdown_timeout"`
	PreflightTimeout       string `koanf:"preflight_timeout"`
	StaleLockTTL           string `koanf:"stale_lock_ttl"`
	WorkspacePath          string `koanf:"workspace_path"`
}

type ZanshinConfig struct {
	Enabled           bool    `koanf:"enabled"`
	TriggerThreshold  float64 `koanf:"trigger_threshold"`
	PruneThreshold    float64 `koanf:"prune_threshold"`
	SimilarityEpsilon float64 `koanf:"similarity_epsilon"`
	ClusterCount      int     `koanf:"cluster_count"`
	MaxIdleTime       string  `koanf:"max_idle_time"`
}

type AdaptersConfig struct {
	Slack    SlackConfig    `koanf:"slack"`
	Telegram TelegramConfig `koanf:"telegram"`
}

type AuthConfig struct {
	Codex CodexAuthConfig `koanf:"codex"`
}

type CodexAuthConfig struct {
	CallbackAddr string `koanf:"callback_addr"`
	RedirectURI  string `koanf:"redirect_uri"`
	OAuthTimeout string `koanf:"oauth_timeout"`
	TokenPath    string `koanf:"token_path"`
}

type DiscoveryConfig struct {
	ProjectPath  string   `koanf:"project_path"`
	SkillSources []string `koanf:"skill_sources"`
	ToolSources  []string `koanf:"tool_sources"`
}

type ToolsConfig struct {
	Web        WebToolConfig        `koanf:"web"`
	Weather    WeatherToolConfig    `koanf:"weather"`
	Finance    FinanceToolConfig    `koanf:"finance"`
	Sports     SportsToolConfig     `koanf:"sports"`
	ImageQuery ImageQueryToolConfig `koanf:"image_query"`
	Screenshot ScreenshotToolConfig `koanf:"screenshot"`
	ApplyPatch ApplyPatchToolConfig `koanf:"apply_patch"`
}

type WebToolConfig struct {
	BaseURL          string `koanf:"base_url"`
	Timeout          string `koanf:"timeout"`
	MaxContentLength int    `koanf:"max_content_length"`
}

type WeatherToolConfig struct {
	BaseURL string `koanf:"base_url"`
	Timeout string `koanf:"timeout"`
}

type FinanceToolConfig struct {
	BaseURL string `koanf:"base_url"`
	Timeout string `koanf:"timeout"`
}

type SportsToolConfig struct {
	BaseURL string `koanf:"base_url"`
	Timeout string `koanf:"timeout"`
}

type ImageQueryToolConfig struct {
	BaseURL string `koanf:"base_url"`
	Timeout string `koanf:"timeout"`
}

type ScreenshotToolConfig struct {
	Timeout  string `koanf:"timeout"`
	Renderer string `koanf:"renderer"`
}

type ApplyPatchToolConfig struct {
	Command string `koanf:"command"`
}

type IngressConfig struct {
	InteractiveQueueSize     int    `koanf:"interactive_queue_size"`
	BackgroundQueueSize      int    `koanf:"background_queue_size"`
	InteractiveSubmitTimeout string `koanf:"interactive_submit_timeout"`
	DrainTimeout             string `koanf:"drain_timeout"`
	DrainPollInterval        string `koanf:"drain_poll_interval"`
}

type SlackConfig struct {
	Enabled       bool   `koanf:"enabled"`
	Port          int    `koanf:"port"`
	SigningSecret string `koanf:"signing_secret"`
	BotToken      string `koanf:"bot_token"`
}

type TelegramConfig struct {
	Enabled       bool   `koanf:"enabled"`
	BotToken      string `koanf:"bot_token"`
	UpdateTimeout int    `koanf:"update_timeout"`
}

type ServerConfig struct {
	Port            int    `koanf:"port"`
	LogLevel        string `koanf:"log_level"`
	ReadTimeout     string `koanf:"read_timeout"`
	WriteTimeout    string `koanf:"write_timeout"`
	IdleTimeout     string `koanf:"idle_timeout"`
	ShutdownTimeout string `koanf:"shutdown_timeout"`
}

type ModelsConfig struct {
	Default             string          `koanf:"default"`
	Fallback            string          `koanf:"fallback"`
	Embedding           string          `koanf:"embedding"`
	MaxFallbackAttempts int             `koanf:"max_fallback_attempts"`
	Registry            []ModelRegistry `koanf:"registry"`
}

type ModelRegistry struct {
	Name                   string `koanf:"name"`
	Provider               string `koanf:"provider"`
	BaseURL                string `koanf:"base_url"`
	APIKey                 string `koanf:"api_key"`
	AuthFile               string `koanf:"auth_file"`
	RequestTimeout         string `koanf:"request_timeout"`
	EmbeddingInputMaxChars int    `koanf:"embedding_input_max_chars"`
}

type GovernanceConfig struct {
	RequireApproval []string `koanf:"require_approval"`
	AutoAllow       []string `koanf:"auto_allow"`
	IdempotencyTTL  string   `koanf:"idempotency_ttl"`
	DailyToolLimit  int      `koanf:"daily_tool_limit"`
}

type OrchestratorConfig struct {
	Verbose                bool   `koanf:"verbose"`
	MaxSubTasks            int    `koanf:"max_sub_tasks"`
	MaxParallelSubTasks    int    `koanf:"max_parallel_subtasks"`
	MaxToolsPerTurn        int    `koanf:"max_tools_per_turn"`
	MaxTurns               int    `koanf:"max_turns"`
	TokenBudget            int    `koanf:"token_budget"`
	DecomposeWordThreshold int    `koanf:"decompose_word_threshold"`
	SessionHistoryLimit    int    `koanf:"session_history_limit"`
	StructuredRetryMax     int    `koanf:"structured_retry_max"`
	SubTaskRetryMax        int    `koanf:"subtask_retry_max"`
	SubTaskRetryBackoff    string `koanf:"subtask_retry_backoff"`
}

const (
	DefaultWorkspaceID                     = "default"
	DefaultServerPort                      = 8080
	DefaultServerLogLevel                  = "info"
	DefaultServerReadTimeout               = "10s"
	DefaultServerWriteTimeout              = "10s"
	DefaultServerIdleTimeout               = "60s"
	DefaultServerShutdownTimeout           = "5s"
	DefaultModelDefault                    = "gpt-4-turbo"
	DefaultModelFallback                   = "claude-3-haiku"
	DefaultModelEmbedding                  = "nomic-embed-text"
	DefaultModelMaxFallbackAttempts        = 2
	DefaultOpenAIBaseURL                   = "https://api.openai.com/v1"
	DefaultOllamaBaseURL                   = "http://localhost:11434/v1"
	DefaultOllamaAPIKey                    = "ollama"
	DefaultCodexBaseURL                    = "https://chatgpt.com/backend-api"
	DefaultGovernanceIdempotencyTTL        = "24h"
	DefaultGovernanceDailyToolLimit        = 100
	DefaultCodexAuthCallbackAddr           = "localhost:1455"
	DefaultCodexAuthRedirectURI            = "http://localhost:1455/auth/callback"
	DefaultCodexAuthOAuthTimeout           = "5m"
	DefaultCodexRequestTimeout             = "120s"
	DefaultCodexEmbeddingInputMaxChars     = 8000
	DefaultDiscoveryProjectPath            = ""
	DefaultPlannerSystemPrompt             = "You are a strategic planning agent. Create a concise, step-by-step plan to achieve the goal."
	DefaultPlannerOutputPrompt             = "Output the plan as a JSON array of objects with 'id' and 'description' fields. Do not include other text."
	DefaultThinkerSystemPrompt             = "You are Heike, an intelligent agent executing a task."
	DefaultThinkerInstructionPrompt        = "Think step-by-step. If you need to use a tool, do so. If you have the final answer, provide it clearly."
	DefaultReflectorSystemPrompt           = "You are a reflective agent. Analyze the last action and its result."
	DefaultReflectorGuidelinesPrompt       = "Analyze what happened. Did it succeed? What did we learn? What should be the next step?\n\nReturn a JSON object with:\n- \"analysis\": string (your reasoning)\n- \"next_action\": string (\"continue\", \"retry\", \"replan\", \"stop\")\n- \"new_memories\": array of strings (facts to remember)\n\nGuidelines:\n- \"retry\": if the tool failed transiently.\n- \"replan\": if the current plan is impossible or invalid.\n- \"stop\": if the goal is achieved or impossible.\n- \"continue\": otherwise."
	DefaultDecomposerSystemPrompt          = "You are a task decomposition expert. Break down the following high-level goal into a list of specific, executable sub-tasks."
	DefaultDecomposerRequirementsPrompt    = "Requirements:\n1. Each sub-task must be clear and actionable.\n2. Return the result as a JSON array of objects with:\n   - 'id' (string): unique identifier\n   - 'description' (string): actionable instruction\n   - 'priority' (int): 1 (high) to 5 (low)\n   - 'dependencies' (array of strings): list of IDs that must be completed BEFORE this task can start.\n3. Analyze dependencies carefully. If Task B requires output from Task A, Task B must list Task A's ID in 'dependencies'.\n4. Do not include markdown formatting or explanations, just the raw JSON."
	DefaultStoreLockTimeout                = "30s"
	DefaultStoreLockRetry                  = "100ms"
	DefaultStoreLockMaxRetry               = 300
	DefaultStoreInboxSize                  = 100
	DefaultStoreTranscriptRotateMaxBytes   = 10 * 1024 * 1024
	DefaultOrchestratorVerbose             = false
	DefaultOrchestratorMaxSubTasks         = 10
	DefaultOrchestratorMaxParallelSubTasks = 4
	DefaultOrchestratorMaxToolsPerTurn     = 12
	DefaultOrchestratorMaxTurns            = 10
	DefaultOrchestratorTokenBudget         = 8000
	DefaultOrchestratorDecomposeWordThresh = 20
	DefaultOrchestratorSessionHistoryLimit = 20
	DefaultOrchestratorStructuredRetryMax  = 1
	DefaultOrchestratorSubTaskRetryMax     = 3
	DefaultOrchestratorSubTaskRetryBackoff = "1s"
	DefaultSlackPort                       = 3000
	DefaultTelegramUpdateTimeout           = 60
	DefaultIngressInteractiveQueue         = 100
	DefaultIngressBackgroundQueue          = 1000
	DefaultIngressInteractiveSubmitTimeout = "500ms"
	DefaultIngressDrainTimeout             = "5s"
	DefaultIngressDrainPollInterval        = "100ms"
	DefaultWebToolTimeout                  = "10s"
	DefaultWebToolBaseURL                  = "https://www.bing.com/search"
	DefaultWebToolMaxContentLength         = 5000
	DefaultWeatherToolBaseURL              = "https://wttr.in"
	DefaultWeatherToolTimeout              = "10s"
	DefaultFinanceToolBaseURL              = "https://query1.finance.yahoo.com/v7/finance/quote"
	DefaultFinanceToolTimeout              = "10s"
	DefaultSportsToolBaseURL               = "https://site.api.espn.com/apis/v2/sports"
	DefaultSportsToolTimeout               = "10s"
	DefaultImageQueryToolBaseURL           = "https://commons.wikimedia.org/w/api.php"
	DefaultImageQueryToolTimeout           = "10s"
	DefaultScreenshotToolTimeout           = "20s"
	DefaultScreenshotToolRenderer          = "pdftoppm"
	DefaultApplyPatchToolCommand           = "apply_patch"
	DefaultWorkerShutdownTimeout           = "30s"
	DefaultSchedulerTickInterval           = "1m"
	DefaultSchedulerShutdownTimeout        = "30s"
	DefaultSchedulerLeaseDuration          = "5m"
	DefaultSchedulerMaxCatchupRuns         = 1
	DefaultSchedulerInFlightPollInterval   = "100ms"
	DefaultSchedulerHeartbeatWorkspaceID   = DefaultWorkspaceID
	DefaultDaemonShutdownTimeout           = "30s"
	DefaultDaemonHealthCheckInterval       = "30s"
	DefaultDaemonStartupShutdownTimeout    = "10s"
	DefaultDaemonPreflightTimeout          = "10s"
	DefaultDaemonStaleLockTTL              = "15m"
	DefaultZanshinEnabled                  = true
	DefaultZanshinTriggerThreshold         = 0.5
	DefaultZanshinPruneThreshold           = 0.3
	DefaultZanshinSimilarityEpsilon        = 0.85
	DefaultZanshinClusterCount             = 10
	DefaultZanshinMaxIdleTime              = "30m"
)

func Load(cmd *cobra.Command) (*Config, error) {
	k := koanf.New(".")

	// Hardcoded Defaults
	defaults := map[string]interface{}{
		"server.port":                  DefaultServerPort,
		"server.log_level":             DefaultServerLogLevel,
		"server.read_timeout":          DefaultServerReadTimeout,
		"server.write_timeout":         DefaultServerWriteTimeout,
		"server.idle_timeout":          DefaultServerIdleTimeout,
		"server.shutdown_timeout":      DefaultServerShutdownTimeout,
		"models.default":               DefaultModelDefault,
		"models.fallback":              DefaultModelFallback,
		"models.embedding":             DefaultModelEmbedding,
		"models.max_fallback_attempts": DefaultModelMaxFallbackAttempts,
		"models.registry": []ModelRegistry{
			{Name: DefaultModelDefault, Provider: "openai"},
			{Name: DefaultModelFallback, Provider: "anthropic"}, // Not implemented yet, will be skipped
			{Name: "local-llama", Provider: "ollama", BaseURL: DefaultOllamaBaseURL},
		},
		"governance.require_approval":           []string{"exec_command", "write_stdin", "apply_patch"},
		"governance.auto_allow":                 []string{"time", "search_query", "open", "click", "find", "weather", "finance", "sports", "image_query", "screenshot"},
		"governance.idempotency_ttl":            DefaultGovernanceIdempotencyTTL,
		"governance.daily_tool_limit":           DefaultGovernanceDailyToolLimit,
		"auth.codex.callback_addr":              DefaultCodexAuthCallbackAddr,
		"auth.codex.redirect_uri":               DefaultCodexAuthRedirectURI,
		"auth.codex.oauth_timeout":              DefaultCodexAuthOAuthTimeout,
		"auth.codex.token_path":                 filepath.Join(os.Getenv("HOME"), ".heike", "auth", "codex.json"),
		"discovery.project_path":                DefaultDiscoveryProjectPath,
		"discovery.skill_sources":               []string{"bundled", "global", "workspace", "project"},
		"discovery.tool_sources":                []string{"global", "bundled", "workspace", "project"},
		"prompts.planner.system":                DefaultPlannerSystemPrompt,
		"prompts.planner.output":                DefaultPlannerOutputPrompt,
		"prompts.thinker.system":                DefaultThinkerSystemPrompt,
		"prompts.thinker.instruction":           DefaultThinkerInstructionPrompt,
		"prompts.reflector.system":              DefaultReflectorSystemPrompt,
		"prompts.reflector.guidelines":          DefaultReflectorGuidelinesPrompt,
		"prompts.decomposer.system":             DefaultDecomposerSystemPrompt,
		"prompts.decomposer.requirements":       DefaultDecomposerRequirementsPrompt,
		"store.lock_timeout":                    DefaultStoreLockTimeout,
		"store.lock_retry":                      DefaultStoreLockRetry,
		"store.lock_max_retry":                  DefaultStoreLockMaxRetry,
		"store.inbox_size":                      DefaultStoreInboxSize,
		"store.transcript_rotate_max_bytes":     DefaultStoreTranscriptRotateMaxBytes,
		"tools.web.base_url":                    DefaultWebToolBaseURL,
		"tools.web.timeout":                     DefaultWebToolTimeout,
		"tools.web.max_content_length":          DefaultWebToolMaxContentLength,
		"tools.weather.base_url":                DefaultWeatherToolBaseURL,
		"tools.weather.timeout":                 DefaultWeatherToolTimeout,
		"tools.finance.base_url":                DefaultFinanceToolBaseURL,
		"tools.finance.timeout":                 DefaultFinanceToolTimeout,
		"tools.sports.base_url":                 DefaultSportsToolBaseURL,
		"tools.sports.timeout":                  DefaultSportsToolTimeout,
		"tools.image_query.base_url":            DefaultImageQueryToolBaseURL,
		"tools.image_query.timeout":             DefaultImageQueryToolTimeout,
		"tools.screenshot.timeout":              DefaultScreenshotToolTimeout,
		"tools.screenshot.renderer":             DefaultScreenshotToolRenderer,
		"tools.apply_patch.command":             DefaultApplyPatchToolCommand,
		"orchestrator.verbose":                  DefaultOrchestratorVerbose,
		"orchestrator.max_sub_tasks":            DefaultOrchestratorMaxSubTasks,
		"orchestrator.max_parallel_subtasks":    DefaultOrchestratorMaxParallelSubTasks,
		"orchestrator.max_tools_per_turn":       DefaultOrchestratorMaxToolsPerTurn,
		"orchestrator.max_turns":                DefaultOrchestratorMaxTurns,
		"orchestrator.token_budget":             DefaultOrchestratorTokenBudget,
		"orchestrator.decompose_word_threshold": DefaultOrchestratorDecomposeWordThresh,
		"orchestrator.session_history_limit":    DefaultOrchestratorSessionHistoryLimit,
		"orchestrator.structured_retry_max":     DefaultOrchestratorStructuredRetryMax,
		"orchestrator.subtask_retry_max":        DefaultOrchestratorSubTaskRetryMax,
		"orchestrator.subtask_retry_backoff":    DefaultOrchestratorSubTaskRetryBackoff,
		"adapters.slack.port":                   DefaultSlackPort,
		"adapters.telegram.update_timeout":      DefaultTelegramUpdateTimeout,
		"ingress.interactive_queue_size":        DefaultIngressInteractiveQueue,
		"ingress.background_queue_size":         DefaultIngressBackgroundQueue,
		"ingress.interactive_submit_timeout":    DefaultIngressInteractiveSubmitTimeout,
		"ingress.drain_timeout":                 DefaultIngressDrainTimeout,
		"ingress.drain_poll_interval":           DefaultIngressDrainPollInterval,
		"worker.shutdown_timeout":               DefaultWorkerShutdownTimeout,
		"scheduler.tick_interval":               DefaultSchedulerTickInterval,
		"scheduler.shutdown_timeout":            DefaultSchedulerShutdownTimeout,
		"scheduler.lease_duration":              DefaultSchedulerLeaseDuration,
		"scheduler.max_catchup_runs":            DefaultSchedulerMaxCatchupRuns,
		"scheduler.in_flight_poll_interval":     DefaultSchedulerInFlightPollInterval,
		"scheduler.heartbeat_workspace_id":      DefaultSchedulerHeartbeatWorkspaceID,
		"daemon.shutdown_timeout":               DefaultDaemonShutdownTimeout,
		"daemon.health_check_interval":          DefaultDaemonHealthCheckInterval,
		"daemon.startup_shutdown_timeout":       DefaultDaemonStartupShutdownTimeout,
		"daemon.preflight_timeout":              DefaultDaemonPreflightTimeout,
		"daemon.stale_lock_ttl":                 DefaultDaemonStaleLockTTL,
		"daemon.workspace_path":                 filepath.Join(os.Getenv("HOME"), ".heike", "workspaces"),
		"zanshin.enabled":                       DefaultZanshinEnabled,
		"zanshin.trigger_threshold":             DefaultZanshinTriggerThreshold,
		"zanshin.prune_threshold":               DefaultZanshinPruneThreshold,
		"zanshin.similarity_epsilon":            DefaultZanshinSimilarityEpsilon,
		"zanshin.cluster_count":                 DefaultZanshinClusterCount,
		"zanshin.max_idle_time":                 DefaultZanshinMaxIdleTime,
	}
	for key, value := range defaults {
		k.Set(key, value)
	}

	// Config file loading
	configPath := ""
	if cmd != nil {
		if flag := cmd.Flags().Lookup("config"); flag != nil {
			configPath = strings.TrimSpace(flag.Value.String())
		}
	}

	if configPath != "" {
		if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
			return nil, err
		}
	} else {
		home, err := os.UserHomeDir()
		if err == nil {
			globalPath := filepath.Join(home, ".heike", "config.yaml")
			if err := k.Load(file.Provider(globalPath), yaml.Parser()); err != nil {
				slog.Debug("Global config not found or invalid", "path", globalPath, "error", err)
			}
		}
	}

	// Environment Variables
	k.Load(env.Provider("HEIKE_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(strings.TrimPrefix(s, "HEIKE_")), "_", ".", -1)
	}), nil)

	// CLI Flags
	if cmd != nil {
		k.Load(posflag.Provider(cmd.Flags(), ".", k), nil)
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, err
	}

	for i, m := range cfg.Models.Registry {
		if m.Provider == "" {
			cfg.Models.Registry[i].Provider = "openai"
		}
	}

	if err := normalizePathFields(&cfg); err != nil {
		return nil, err
	}

	// Post-Process: Inject standard Env Vars if missing
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		for i, m := range cfg.Models.Registry {
			if m.Provider == "openai" && m.APIKey == "" {
				cfg.Models.Registry[i].APIKey = key
			}
		}
	}
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		for i, m := range cfg.Models.Registry {
			if m.Provider == "anthropic" && m.APIKey == "" {
				cfg.Models.Registry[i].APIKey = key
			}
		}
	}
	if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		for i, m := range cfg.Models.Registry {
			if m.Provider == "gemini" && m.APIKey == "" {
				cfg.Models.Registry[i].APIKey = key
			}
		}
	}
	if key := os.Getenv("ZAI_API_KEY"); key != "" {
		for i, m := range cfg.Models.Registry {
			if m.Provider == "zai" && m.APIKey == "" {
				cfg.Models.Registry[i].APIKey = key
			}
		}
	}

	return &cfg, nil
}

func normalizePathFields(cfg *Config) error {
	if cfg == nil {
		return nil
	}

	projectPath, err := expandConfiguredPath(cfg.Discovery.ProjectPath)
	if err != nil {
		return err
	}
	if projectPath != "" {
		cfg.Discovery.ProjectPath = projectPath
	}

	workspacePath, err := expandConfiguredPath(cfg.Daemon.WorkspacePath)
	if err != nil {
		return err
	}
	if workspacePath != "" {
		cfg.Daemon.WorkspacePath = workspacePath
	}

	tokenPath, err := expandConfiguredPath(cfg.Auth.Codex.TokenPath)
	if err != nil {
		return err
	}
	if tokenPath != "" {
		cfg.Auth.Codex.TokenPath = tokenPath
	}

	for i := range cfg.Models.Registry {
		authFile, err := expandConfiguredPath(cfg.Models.Registry[i].AuthFile)
		if err != nil {
			return err
		}
		if authFile != "" {
			cfg.Models.Registry[i].AuthFile = authFile
		}
	}

	return nil
}

func expandConfiguredPath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", nil
	}
	expanded, err := pathutil.Expand(trimmed)
	if err != nil {
		return "", err
	}
	return expanded, nil
}
