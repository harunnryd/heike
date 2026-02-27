package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/harunnryd/heike/internal/config"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

//go:embed templates/config.yaml
var embeddedDefaultConfig []byte

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  `Manage Heike configuration file.`,
}

var configViewCmd = &cobra.Command{
	Use:   "view",
	Short: "Dump fully resolved configuration",
	Long:  `Display current configuration with all defaults applied and environment variables resolved.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		loadedCfg, err := loadConfigForCommand(cmd)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if loadedCfg == nil {
			return fmt.Errorf("config is not initialized; run 'heike config init' first")
		}

		redacted := redactConfigSecrets(loadedCfg)

		enc := yaml.NewEncoder(os.Stdout)
		enc.SetIndent(2)
		if err := enc.Encode(redacted); err != nil {
			return fmt.Errorf("failed to encode config: %w", err)
		}
		return nil
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize default configuration",
	Long:  `Create a default configuration file at $HOME/.heike/config.yaml if it doesn't exist.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		configDir := filepath.Join(home, ".heike")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory %s: %w", configDir, err)
		}

		configPath := filepath.Join(configDir, "config.yaml")
		if _, err := os.Stat(configPath); err == nil {
			fmt.Printf("Config already exists at %s\n", configPath)
			fmt.Println("Use 'heike config view' to see current configuration.")
			fmt.Println("To reinitialize, remove the existing config file first.")
			return nil
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("failed to check config file: %w", err)
		}

		defaultConfig := strings.TrimSpace(string(embeddedDefaultConfig)) + "\n"
		if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
			return fmt.Errorf("failed to write config to %s: %w", configPath, err)
		}

		fmt.Printf("✓ Initialized config at %s\n", configPath)
		fmt.Println("\nNext steps:")
		fmt.Println("1. Set OPENAI_API_KEY or ANTHROPIC_API_KEY environment variable (recommended)")
		fmt.Println("2. Or edit config.yaml to add your API key directly")
		fmt.Println("3. Run 'heike config view' to verify your configuration")
		return nil
	},
}

func loadConfigForCommand(cmd *cobra.Command) (*config.Config, error) {
	if cfg != nil {
		return cfg, nil
	}

	loadedCfg, err := config.Load(cmd)
	if err != nil {
		return nil, err
	}

	return loadedCfg, nil
}

func redactConfigSecrets(in *config.Config) *config.Config {
	if in == nil {
		return nil
	}

	out := *in

	if len(in.Models.Registry) > 0 {
		out.Models.Registry = make([]config.ModelRegistry, len(in.Models.Registry))
		copy(out.Models.Registry, in.Models.Registry)
		for i := range out.Models.Registry {
			out.Models.Registry[i].APIKey = maskSecret(out.Models.Registry[i].APIKey)
		}
	}

	out.Adapters.Slack.SigningSecret = maskSecret(out.Adapters.Slack.SigningSecret)
	out.Adapters.Slack.BotToken = maskSecret(out.Adapters.Slack.BotToken)
	out.Adapters.Telegram.BotToken = maskSecret(out.Adapters.Telegram.BotToken)

	return &out
}

func maskSecret(secret string) string {
	if secret == "" {
		return ""
	}
	if len(secret) <= 4 {
		return "****"
	}
	return secret[:2] + strings.Repeat("*", len(secret)-4) + secret[len(secret)-2:]
}

func init() {
	configCmd.AddCommand(configViewCmd)
	configCmd.AddCommand(configInitCmd)
	rootCmd.AddCommand(configCmd)
}
