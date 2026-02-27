package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harunnryd/heike/internal/config"

	"github.com/spf13/cobra"
)

func TestConfigInitCmd(t *testing.T) {
	tmpDir := t.TempDir()

	home := os.Getenv("HOME")
	defer func() {
		if home != "" {
			os.Setenv("HOME", home)
		}
	}()
	os.Setenv("HOME", tmpDir)

	cmd := &cobra.Command{}
	args := []string{}

	if err := configInitCmd.RunE(cmd, args); err != nil {
		t.Errorf("Config init failed: %v", err)
	}

	configPath := filepath.Join(tmpDir, ".heike", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("Config file not created at %s", configPath)
	}

	cmd2 := &cobra.Command{}
	args2 := []string{}
	if err := configInitCmd.RunE(cmd2, args2); err != nil {
		t.Errorf("Config init should succeed when config exists: %v", err)
	}
}

func TestConfigViewCmd(t *testing.T) {
	cmd := &cobra.Command{}
	args := []string{"view"}

	err := configViewCmd.RunE(cmd, args)
	if err == nil {
		t.Log("Config view succeeded (may have config)")
	} else {
		t.Logf("Config view failed (expected without config): %v", err)
	}
}

func TestRedactConfigSecrets(t *testing.T) {
	original := &config.Config{
		Models: config.ModelsConfig{
			Registry: []config.ModelRegistry{
				{Name: "m1", APIKey: "sk-secret-123456"},
				{Name: "m2", APIKey: "abcd"},
			},
		},
			Adapters: config.AdaptersConfig{
				Slack: config.SlackConfig{
					SigningSecret: "slack-signing-secret",
					BotToken:      "slack-bot-token",
				},
			Telegram: config.TelegramConfig{
				BotToken: "telegram-secret-token",
			},
		},
	}

	redacted := redactConfigSecrets(original)

	if redacted == nil {
		t.Fatal("redacted config should not be nil")
	}
	if redacted.Models.Registry[0].APIKey == original.Models.Registry[0].APIKey {
		t.Fatal("model API key should be masked")
	}
	if strings.Contains(redacted.Models.Registry[0].APIKey, "secret") {
		t.Fatal("masked model API key should not leak original value")
	}
	if redacted.Adapters.Slack.SigningSecret == original.Adapters.Slack.SigningSecret {
		t.Fatal("slack signing secret should be masked")
	}
	if redacted.Adapters.Slack.BotToken == original.Adapters.Slack.BotToken {
		t.Fatal("slack bot token should be masked")
	}
	if redacted.Adapters.Telegram.BotToken == original.Adapters.Telegram.BotToken {
		t.Fatal("telegram bot token should be masked")
	}

	// Ensure original struct is not mutated.
	if original.Models.Registry[0].APIKey != "sk-secret-123456" {
		t.Fatal("original config must not be modified")
	}
}

func TestMaskSecret(t *testing.T) {
	if got := maskSecret(""); got != "" {
		t.Fatalf("empty secret: got %q", got)
	}
	if got := maskSecret("abc"); got != "****" {
		t.Fatalf("short secret: got %q", got)
	}

	got := maskSecret("abcdef")
	if len(got) != len("abcdef") {
		t.Fatalf("masked secret length mismatch: got %d", len(got))
	}
	if got[:2] != "ab" || got[len(got)-2:] != "ef" {
		t.Fatalf("masked secret should preserve prefix/suffix: got %q", got)
	}
}
