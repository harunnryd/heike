package policy

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestAuditLoggerAppendsEntries(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	logger, err := NewAuditLogger("ws-audit-append", "", &AuditPolicy{
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}

	ctx := context.Background()
	if err := logger.Log(ctx, &AuditEntry{
		ToolName: "file.read",
		Action:   "execute",
		Status:   "ok",
		Input:    json.RawMessage(`{"path":"a.txt"}`),
	}); err != nil {
		t.Fatalf("first Log failed: %v", err)
	}

	if err := logger.Log(ctx, &AuditEntry{
		ToolName: "exec.command",
		Action:   "execute",
		Status:   "ok",
		Input:    json.RawMessage(`{"command":"pwd"}`),
	}); err != nil {
		t.Fatalf("second Log failed: %v", err)
	}

	entries, err := logger.Query(ctx, nil)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
}

func TestAuditLoggerRedactsByRegex(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	logger, err := NewAuditLogger("ws-audit-redact", "", &AuditPolicy{
		Enabled:        true,
		RedactPatterns: []string{`secret-[0-9]+`},
	})
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}

	ctx := context.Background()
	if err := logger.Log(ctx, &AuditEntry{
		Timestamp: time.Now(),
		ToolName:  "exec.command",
		Action:    "execute",
		Status:    "ok",
		Input:     json.RawMessage(`{"token":"secret-12345"}`),
		Output:    json.RawMessage(`{"result":"ok secret-67890"}`),
	}); err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	entries, err := logger.Query(ctx, nil)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	input := string(entries[0].Input)
	output := string(entries[0].Output)

	if input == `{"token":"secret-12345"}` {
		t.Fatalf("input was not redacted: %s", input)
	}
	if output == `{"result":"ok secret-67890"}` {
		t.Fatalf("output was not redacted: %s", output)
	}
}
