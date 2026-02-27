package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/policy"
	"github.com/harunnryd/heike/internal/store"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Manage workspace policies",
	Long:  `Manage and view workspace governance policies for tool access control.`,
}

var policyShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current workspace policy",
	Long:  `Display current workspace policy configuration including allowed tools, denied tools, and approval rules.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		workspaceID := filepath.Base(wd)

		cfg, err := config.Load(cmd)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		workspacePath, err := store.GetWorkspacePath(workspaceID, cfg.Daemon.WorkspacePath)
		if err != nil {
			return fmt.Errorf("failed to resolve workspace path: %w", err)
		}
		baseDir := filepath.Join(workspacePath, "governance")

		fmt.Println("=== Workspace Policy ===")
		fmt.Printf("Auto-Allow Tools: %v\n", cfg.Governance.AutoAllow)
		fmt.Printf("Require Approval: %v\n", cfg.Governance.RequireApproval)

		domainPath := filepath.Join(baseDir, "domains.json")
		dData, err := os.ReadFile(domainPath)
		if err == nil && len(dData) > 0 {
			var dl policy.DomainList
			if err := json.Unmarshal(dData, &dl); err == nil {
				fmt.Printf("Domain Allowlist: %v\n", dl.Allowed)
			}
		}

		return nil
	},
}

var policySetCmd = &cobra.Command{
	Use:   "set [tool]",
	Short: "Set tool policy",
	Long:  `Add a tool to auto-allow list or require-approval list.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		toolName := args[0]
		autoAllow, _ := cmd.Flags().GetBool("allow")
		requireApproval, _ := cmd.Flags().GetBool("require-approval")

		configPath, err := resolveGovernanceConfigPath(cmd)
		if err != nil {
			return fmt.Errorf("failed to resolve config path: %w", err)
		}

		cfg, err := config.Load(cmd)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if autoAllow {
			cfg.Governance.AutoAllow = append(cfg.Governance.AutoAllow, toolName)
		} else if requireApproval {
			cfg.Governance.RequireApproval = append(cfg.Governance.RequireApproval, toolName)
		} else {
			return fmt.Errorf("must specify --allow or --require-approval")
		}

		if err := saveGovernanceConfig(configPath, cfg.Governance); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("✓ Policy updated for tool: %s\n", toolName)
		return nil
	},
}

var policyDenyCmd = &cobra.Command{
	Use:   "deny [tool]",
	Short: "Deny a tool",
	Long:  `Add a tool to the require-approval list (acts as deny).`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		toolName := args[0]

		configPath, err := resolveGovernanceConfigPath(cmd)
		if err != nil {
			return fmt.Errorf("failed to resolve config path: %w", err)
		}

		cfg, err := config.Load(cmd)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		cfg.Governance.RequireApproval = append(cfg.Governance.RequireApproval, toolName)

		if err := saveGovernanceConfig(configPath, cfg.Governance); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("✓ Tool denied (requires approval): %s\n", toolName)
		return nil
	},
}

var policyRequireApprovalCmd = &cobra.Command{
	Use:   "require-approval [tool]",
	Short: "Require approval for a tool",
	Long:  `Add a tool to the require-approval list. These tools will need explicit approval before execution.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		toolName := args[0]

		configPath, err := resolveGovernanceConfigPath(cmd)
		if err != nil {
			return fmt.Errorf("failed to resolve config path: %w", err)
		}

		cfg, err := config.Load(cmd)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		cfg.Governance.RequireApproval = append(cfg.Governance.RequireApproval, toolName)

		if err := saveGovernanceConfig(configPath, cfg.Governance); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("✓ Tool requires approval: %s\n", toolName)
		return nil
	},
}

var policyAuditCmd = &cobra.Command{
	Use:   "audit",
	Short: "View audit logs",
	Long:  `Display audit logs of tool executions for security and compliance monitoring.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		workspaceID := filepath.Base(wd)

		cfg, err := config.Load(cmd)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		auditLogger, err := policy.NewAuditLogger(workspaceID, cfg.Daemon.WorkspacePath, &policy.AuditPolicy{
			Enabled: true,
		})
		if err != nil {
			return fmt.Errorf("failed to load audit logger: %w", err)
		}

		ctx := context.Background()
		entries, err := auditLogger.Query(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to query audit logs: %w", err)
		}

		if len(entries) == 0 {
			fmt.Println("No audit entries found.")
			return nil
		}

		fmt.Println("=== Audit Logs ===")
		for _, entry := range entries {
			fmt.Printf("Time: %s | Tool: %s | Action: %s | Status: %s\n",
				entry.Timestamp.Format("2006-01-02 15:04:05"),
				entry.ToolName,
				entry.Action,
				entry.Status)
		}

		return nil
	},
}

var policyStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show policy statistics",
	Long:  `Display statistics about tool usage, approvals, and policy enforcement.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		cfg, err := config.Load(cmd)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		fmt.Println("=== Policy Statistics ===")
		fmt.Printf("Auto-Allow Tools: %d\n", len(cfg.Governance.AutoAllow))
		fmt.Printf("Require Approval Tools: %d\n", len(cfg.Governance.RequireApproval))

		workspaceID := filepath.Base(wd)
		workspacePath, err := store.GetWorkspacePath(workspaceID, cfg.Daemon.WorkspacePath)
		if err != nil {
			return fmt.Errorf("failed to resolve workspace path: %w", err)
		}
		domainPath := filepath.Join(workspacePath, "governance", "domains.json")
		dData, err := os.ReadFile(domainPath)
		if err == nil && len(dData) > 0 {
			var dl policy.DomainList
			if err := json.Unmarshal(dData, &dl); err == nil {
				fmt.Printf("Allowed Domains: %d\n", len(dl.Allowed))
			}
		}

		return nil
	},
}

func init() {
	policySetCmd.Flags().BoolP("allow", "a", false, "Add to auto-allow list")
	policySetCmd.Flags().BoolP("require-approval", "r", false, "Add to require-approval list")

	policyCmd.AddCommand(policyShowCmd)
	policyCmd.AddCommand(policySetCmd)
	policyCmd.AddCommand(policyDenyCmd)
	policyCmd.AddCommand(policyRequireApprovalCmd)
	policyCmd.AddCommand(policyAuditCmd)
	policyCmd.AddCommand(policyStatsCmd)
	rootCmd.AddCommand(policyCmd)
}

func saveGovernanceConfig(configPath string, governance config.GovernanceConfig) error {
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	cfgData := map[string]interface{}{}
	if data, err := os.ReadFile(configPath); err == nil && len(data) > 0 {
		if err := yaml.Unmarshal(data, &cfgData); err != nil {
			return err
		}
	}

	cfgData["governance"] = map[string]interface{}{
		"require_approval": governance.RequireApproval,
		"auto_allow":       governance.AutoAllow,
		"idempotency_ttl":  governance.IdempotencyTTL,
	}

	data, err := yaml.Marshal(cfgData)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

func resolveGovernanceConfigPath(cmd *cobra.Command) (string, error) {
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return "", err
	}

	if configPath != "" {
		return configPath, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".heike", "config.yaml"), nil
}
