package main

import (
	"fmt"

	"github.com/harunnryd/heike/internal/auth"

	"github.com/spf13/cobra"
)

var providerCmd = &cobra.Command{
	Use:   "provider",
	Short: "Manage LLM providers",
}

var loginCmd = &cobra.Command{
	Use:   "login [provider]",
	Short: "Authenticate with a provider (e.g. openai-codex)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		providerName := args[0]
		if providerName != "openai-codex" {
			return fmt.Errorf("currently only 'openai-codex' is supported for interactive login")
		}

		fmt.Printf("Initiating OAuth login for %s...\n", providerName)

		token, err := auth.LoginCodexOAuthInteractive(cmd.Context(), auth.CodexOAuthConfig{
			CallbackAddr: cfg.Auth.Codex.CallbackAddr,
			RedirectURI:  cfg.Auth.Codex.RedirectURI,
			OAuthTimeout: cfg.Auth.Codex.OAuthTimeout,
			TokenPath:    cfg.Auth.Codex.TokenPath,
		})
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		// Save Token
		if err := auth.SaveToken(token, cfg.Auth.Codex.TokenPath); err != nil {
			return fmt.Errorf("failed to save token: %w", err)
		}

		fmt.Printf("Successfully logged in to %s!\n", providerName)
		fmt.Printf("Access Token: %s... (expires in %d seconds)\n", token.AccessToken[:10], token.ExpiresIn)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(providerCmd)
	providerCmd.AddCommand(loginCmd)
}
