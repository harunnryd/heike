package policy

import (
	"fmt"
	"strings"
)

type WorkspacePolicyManager struct {
	rules *WorkspacePolicy
}

func NewWorkspacePolicyManager(rules *WorkspacePolicy) *WorkspacePolicyManager {
	return &WorkspacePolicyManager{
		rules: rules,
	}
}

func (wpm *WorkspacePolicyManager) Check(toolName string) error {
	if wpm.rules == nil {
		return nil
	}

	for _, denied := range wpm.rules.DeniedTools {
		if strings.EqualFold(toolName, denied) {
			return fmt.Errorf("tool %s is denied by workspace policy", toolName)
		}
	}

	if len(wpm.rules.AllowedTools) > 0 {
		allowed := false
		for _, allowedTool := range wpm.rules.AllowedTools {
			if strings.EqualFold(toolName, allowedTool) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("tool %s is not in workspace allow list", toolName)
		}
	}

	return nil
}

func (wpm *WorkspacePolicyManager) GetApprovalRule(toolName string) *ApprovalRule {
	if wpm.rules == nil || wpm.rules.ApprovalRules == nil {
		return nil
	}

	for pattern, rule := range wpm.rules.ApprovalRules {
		if strings.EqualFold(toolName, pattern) {
			return rule
		}
	}

	return nil
}

func (wpm *WorkspacePolicyManager) GetResourceLimits() *ResourceLimits {
	if wpm.rules == nil {
		return nil
	}
	return wpm.rules.ResourceLimits
}
