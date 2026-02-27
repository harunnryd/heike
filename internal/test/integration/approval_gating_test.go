package integration_test

import (
	"encoding/json"
	"testing"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/policy"
)

func TestApprovalGating(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg := config.GovernanceConfig{
		RequireApproval: []string{"exec.command", "file.write"},
		AutoAllow:       []string{"file.read", "file.list"},
		IdempotencyTTL:  "24h",
	}

	workspaceID := "approval-test-ws-" + t.Name()
	engine, err := policy.NewEngine(cfg, workspaceID, "")
	if err != nil {
		t.Fatalf("Failed to create policy engine: %v", err)
	}

	t.Run("Auto-allow tools should pass", func(t *testing.T) {
		allowed, approvalID, err := engine.Check("file.read", json.RawMessage(`{"path":"test.txt"}`))
		if err != nil {
			t.Errorf("Auto-allow should not return error: %v", err)
		}
		if !allowed {
			t.Error("Auto-allow tool should be allowed")
		}
		if approvalID != "" {
			t.Error("Auto-allow should not create approval ID")
		}
	})

	t.Run("Tools requiring approval should be gated", func(t *testing.T) {
		allowed, approvalID, err := engine.Check("exec.command", json.RawMessage(`{"command":"ls"}`))
		if err != policy.ErrApprovalRequired {
			t.Errorf("Expected ErrApprovalRequired, got: %v", err)
		}
		if allowed {
			t.Error("Restricted tool should not be allowed")
		}
		if approvalID == "" {
			t.Error("Restricted tool should create approval ID")
		}
	})

	t.Run("Unconfigured tools should be allowed", func(t *testing.T) {
		allowed, approvalID, err := engine.Check("unknown.tool", json.RawMessage(`{}`))
		if err != nil {
			t.Errorf("Unconfigured tool should not return error: %v", err)
		}
		if !allowed {
			t.Error("Unconfigured tool should be allowed")
		}
		if approvalID != "" {
			t.Error("Unconfigured tool should not create approval ID")
		}
	})
}

func TestApprovalFlow(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg := config.GovernanceConfig{
		RequireApproval: []string{"exec.command"},
		AutoAllow:       []string{},
		IdempotencyTTL:  "24h",
	}

	workspaceID := "approval-flow-test-ws-" + t.Name()
	engine, err := policy.NewEngine(cfg, workspaceID, "")
	if err != nil {
		t.Fatalf("Failed to create policy engine: %v", err)
	}

	t.Run("Approval request creation", func(t *testing.T) {
		allowed, approvalID, err := engine.Check("exec.command", json.RawMessage(`{"command":"rm -rf /"}`))
		if err != policy.ErrApprovalRequired {
			t.Errorf("Expected ErrApprovalRequired, got: %v", err)
		}
		if allowed {
			t.Error("Should not be allowed initially")
		}
		if approvalID == "" {
			t.Error("Should have approval ID")
		}
	})

	t.Run("Grant approval", func(t *testing.T) {
		_, approvalID, _ := engine.Check("exec.command", json.RawMessage(`{"command":"echo hello"}`))
		if approvalID == "" {
			t.Error("Should have approval ID")
		}

		err := engine.Resolve(approvalID, true)
		if err != nil {
			t.Errorf("Failed to grant approval: %v", err)
		}

		if !engine.IsGranted(approvalID) {
			t.Error("Approval should be granted")
		}
	})

	t.Run("Deny approval", func(t *testing.T) {
		_, approvalID, _ := engine.Check("exec.command", json.RawMessage(`{"command":"cat /etc/passwd"}`))
		if approvalID == "" {
			t.Error("Should have approval ID")
		}

		err := engine.Resolve(approvalID, false)
		if err != nil {
			t.Errorf("Failed to deny approval: %v", err)
		}

		if engine.IsGranted(approvalID) {
			t.Error("Approval should be denied, not granted")
		}
	})

	t.Run("Resolve non-existent approval", func(t *testing.T) {
		err := engine.Resolve("non-existent-id", true)
		if err == nil {
			t.Error("Resolving non-existent approval should return error")
		}
	})

	t.Run("Resolve already resolved approval", func(t *testing.T) {
		_, approvalID, _ := engine.Check("exec.command", json.RawMessage(`{"command":"date"}`))
		engine.Resolve(approvalID, true)

		err := engine.Resolve(approvalID, false)
		if err == nil {
			t.Error("Resolving already resolved approval should return error")
		}
	})
}

func TestWebBrowseDomainGating(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg := config.GovernanceConfig{
		RequireApproval: []string{"web.browse"},
		AutoAllow:       []string{},
		IdempotencyTTL:  "24h",
	}

	workspaceID := "domain-gating-test-ws-" + t.Name()
	engine, err := policy.NewEngine(cfg, workspaceID, "")
	if err != nil {
		t.Fatalf("Failed to create policy engine: %v", err)
	}

	t.Run("Unknown domain should require approval", func(t *testing.T) {
		allowed, approvalID, err := engine.Check("web.browse", json.RawMessage(`{"url":"https://unknown-domain.com"}`))
		if err != policy.ErrApprovalRequired {
			t.Errorf("Expected ErrApprovalRequired, got: %v", err)
		}
		if allowed {
			t.Error("Unknown domain should require approval")
		}
		if approvalID == "" {
			t.Error("Should have approval ID")
		}
	})

	t.Run("Approve and add domain to allowlist", func(t *testing.T) {
		_, approvalID, _ := engine.Check("web.browse", json.RawMessage(`{"url":"https://example.com"}`))
		engine.Resolve(approvalID, true)

		if !engine.IsGranted(approvalID) {
			t.Error("Domain approval should be granted")
		}

		allowed2, approvalID2, err := engine.Check("web.browse", json.RawMessage(`{"url":"https://example.com/page"}`))
		if err != nil {
			t.Errorf("Second browse to allowed domain should not error: %v", err)
		}
		if !allowed2 {
			t.Error("Second browse to allowed domain should be auto-allowed")
		}
		if approvalID2 != "" {
			t.Error("Second browse to allowed domain should not create approval")
		}
	})
}
