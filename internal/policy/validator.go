package policy

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

type PolicyValidator struct {
	workspaceRules *WorkspacePolicy
	toolRules      map[string]*ToolPolicy
}

func NewPolicyValidator(p *Policy) (*PolicyValidator, error) {
	return &PolicyValidator{
		workspaceRules: p.WorkspaceRules,
		toolRules:      p.ToolRules,
	}, nil
}

func (pv *PolicyValidator) ValidateToolPolicy(toolName string, policy *ToolPolicy) error {
	if policy == nil {
		return nil
	}

	for _, path := range policy.AllowedPaths {
		if !filepath.IsAbs(path) {
			return fmt.Errorf("allowed path must be absolute: %s", path)
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("allowed path does not exist: %s", path)
		}
	}

	for _, pattern := range policy.DeniedPatterns {
		if _, err := regexp.Compile(pattern); err != nil {
			return fmt.Errorf("invalid denied pattern: %s: %w", pattern, err)
		}
	}

	if policy.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	return nil
}

func (pv *PolicyValidator) ValidateWorkspacePolicy(wp *WorkspacePolicy) error {
	if wp == nil {
		return nil
	}

	for _, toolName := range wp.AllowedTools {
		if toolName == "" {
			return fmt.Errorf("allowed tool name cannot be empty")
		}
	}

	for _, toolName := range wp.DeniedTools {
		if toolName == "" {
			return fmt.Errorf("denied tool name cannot be empty")
		}
	}

	if err := pv.validateResourceLimits(wp.ResourceLimits); err != nil {
		return err
	}

	return nil
}

func (pv *PolicyValidator) validateResourceLimits(rl *ResourceLimits) error {
	if rl == nil {
		return nil
	}

	if rl.MaxMemory < 0 {
		return fmt.Errorf("max memory cannot be negative")
	}

	if rl.MaxCPU < 0 {
		return fmt.Errorf("max CPU cannot be negative")
	}

	if rl.MaxDuration < 0 {
		return fmt.Errorf("max duration cannot be negative")
	}

	if rl.MaxProcesses < 0 {
		return fmt.Errorf("max processes cannot be negative")
	}

	return nil
}

func (pv *PolicyValidator) ValidateInput(toolName string, input string, policy *ToolPolicy) error {
	if policy == nil {
		return nil
	}

	for _, pattern := range policy.DeniedPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		if re.MatchString(input) {
			return fmt.Errorf("input matches denied pattern: %s", pattern)
		}
	}

	return nil
}

func (pv *PolicyValidator) ValidateTimeout(timeout time.Duration, policy *ToolPolicy) error {
	if policy == nil || policy.Timeout == 0 {
		return nil
	}

	if timeout > policy.Timeout {
		return fmt.Errorf("execution timeout %v exceeds policy limit %v", timeout, policy.Timeout)
	}

	return nil
}
