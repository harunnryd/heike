package policy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/harunnryd/heike/internal/config"
)

func TestPolicyEngine(t *testing.T) {
	// Setup Temp Dir
	tmpDir, err := os.MkdirTemp("", "heike_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	os.Setenv("HOME", tmpDir)

	wsID := "test_ws"
	// Ensure directories exist (simulating StoreWorker init)
	wsDir := filepath.Join(tmpDir, ".heike", "workspaces", wsID, "governance")
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := config.GovernanceConfig{
		AutoAllow:       []string{"ls", "exec_command"},
		RequireApproval: []string{"rm", "open"},
	}

	engine, err := NewEngine(cfg, wsID, "")
	if err != nil {
		t.Fatalf("Failed to init engine: %v", err)
	}

	// Auto Allow
	allowed, _, err := engine.Check("ls", nil)
	if err != nil {
		t.Errorf("Auto-allow failed: %v", err)
	}
	if !allowed {
		t.Error("Expected ls to be allowed")
	}

	// Require Approval
	allowed, id, err := engine.Check("rm", nil)
	if err != ErrApprovalRequired {
		t.Errorf("Expected ErrApprovalRequired, got %v", err)
	}
	if allowed {
		t.Error("Expected rm to be denied initially")
	}
	if id == "" {
		t.Error("Expected approval ID")
	}

	// Resolve Approval
	if err := engine.Resolve(id, true); err != nil {
		t.Fatalf("Failed to resolve approval: %v", err)
	}

	if !engine.IsGranted(id) {
		t.Error("Expected approval to be granted")
	}

	// Open Tool Domain Check
	input := json.RawMessage(`{"url": "https://google.com"}`)
	allowed, id2, err := engine.Check("open", input)
	if err != ErrApprovalRequired {
		t.Errorf("Expected open to require approval for new domain, got %v", err)
	}

	// Approve
	if err := engine.Resolve(id2, true); err != nil {
		t.Fatal(err)
	}

	// Check again - should be allowed (if logic adds to whitelist)
	allowed, _, err = engine.Check("open", input)
	if err != nil {
		t.Errorf("Expected open to be allowed after approval, got %v", err)
	}
	if !allowed {
		t.Error("Expected open to be allowed after domain whitelisting")
	}

	// URL-based tools should share the same domain allowlist behavior.
	allowed, _, err = engine.Check("open", json.RawMessage(`{"url":"https://google.com/api"}`))
	if err != nil {
		t.Errorf("Expected open to be allowed for whitelisted domain, got %v", err)
	}
	if !allowed {
		t.Error("Expected open to be allowed after domain whitelisting")
	}

	// Explicit sandbox escalation should force approval workflow.
	allowed, escalatedID, err := engine.Check("exec_command", json.RawMessage(`{"cmd":"echo test","sandbox_permissions":"require_escalated"}`))
	if err != ErrApprovalRequired {
		t.Errorf("Expected require_escalated to require approval, got %v", err)
	}
	if allowed {
		t.Error("Expected require_escalated request to be blocked pending approval")
	}
	if escalatedID == "" {
		t.Error("Expected approval ID for require_escalated")
	}

	// Unsupported sandbox mode should fail fast.
	allowed, _, err = engine.Check("exec_command", json.RawMessage(`{"cmd":"echo test","sandbox_permissions":"forbidden_mode"}`))
	if err == nil {
		t.Error("Expected unsupported sandbox_permissions to return error")
	}
	if allowed {
		t.Error("Expected unsupported sandbox_permissions to be denied")
	}
}
