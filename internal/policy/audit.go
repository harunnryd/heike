package policy

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/harunnryd/heike/internal/logger"
	"github.com/harunnryd/heike/internal/store"
)

type AuditLogger interface {
	Log(ctx context.Context, entry *AuditEntry) error
	Query(ctx context.Context, filter *AuditFilter) ([]*AuditEntry, error)
}

type DefaultAuditLogger struct {
	mu             sync.RWMutex
	logPath        string
	enabled        bool
	redactPatterns []string
}

func NewAuditLogger(workspaceID string, workspaceRootPath string, policy *AuditPolicy) (*DefaultAuditLogger, error) {
	if policy == nil || !policy.Enabled {
		return &DefaultAuditLogger{
			enabled: false,
		}, nil
	}

	workspacePath, err := store.GetWorkspacePath(workspaceID, workspaceRootPath)
	if err != nil {
		return nil, err
	}

	baseDir := filepath.Join(workspacePath, "governance")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}

	logPath := filepath.Join(baseDir, "audit.log")

	return &DefaultAuditLogger{
		logPath:        logPath,
		enabled:        true,
		redactPatterns: policy.RedactPatterns,
	}, nil
}

func (al *DefaultAuditLogger) Log(ctx context.Context, entry *AuditEntry) error {
	if !al.enabled {
		return nil
	}
	if entry == nil {
		return fmt.Errorf("audit entry cannot be nil")
	}

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	if entry.TraceID == "" {
		entry.TraceID = logger.GetTraceID(ctx)
	}

	al.mu.Lock()
	defer al.mu.Unlock()

	redactedEntry := al.redact(entry)
	entryJSON, err := al.marshalEntry(redactedEntry)
	if err != nil {
		slog.Error("Failed to marshal audit entry", "error", err)
		return err
	}

	f, err := os.OpenFile(al.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		slog.Error("Failed to open audit log", "error", err)
		return err
	}
	defer f.Close()

	if _, err := f.Write(append(entryJSON, '\n')); err != nil {
		slog.Error("Failed to write audit entry", "error", err)
		return err
	}

	slog.Debug("Audit entry logged", "trace_id", entry.TraceID, "tool", entry.ToolName, "action", entry.Action)
	return nil
}

func (al *DefaultAuditLogger) Query(ctx context.Context, filter *AuditFilter) ([]*AuditEntry, error) {
	al.mu.RLock()
	defer al.mu.RUnlock()

	file, err := os.Open(al.logPath)
	if os.IsNotExist(err) {
		return []*AuditEntry{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []*AuditEntry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry AuditEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			slog.Warn("Failed to parse audit entry", "line", string(line), "error", err)
			continue
		}

		entries = append(entries, &entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if filter == nil {
		return entries, nil
	}

	return al.applyFilter(entries, filter), nil
}

func (al *DefaultAuditLogger) redact(entry *AuditEntry) *AuditEntry {
	redacted := *entry

	for _, pattern := range al.redactPatterns {
		redacted.Input = al.redactString(redacted.Input, pattern)
		redacted.Output = al.redactString(redacted.Output, pattern)
	}

	return &redacted
}

func (al *DefaultAuditLogger) redactString(data json.RawMessage, pattern string) json.RawMessage {
	dataStr := string(data)
	if dataStr == "" || pattern == "" {
		return json.RawMessage(dataStr)
	}

	if re, err := regexp.Compile(pattern); err == nil {
		return json.RawMessage(re.ReplaceAllString(dataStr, "[REDACTED]"))
	}

	redacted := strings.ReplaceAll(dataStr, pattern, "[REDACTED]")
	return json.RawMessage(redacted)
}

func (al *DefaultAuditLogger) marshalEntry(entry *AuditEntry) ([]byte, error) {
	return json.Marshal(entry)
}

func (al *DefaultAuditLogger) applyFilter(entries []*AuditEntry, filter *AuditFilter) []*AuditEntry {
	var filtered []*AuditEntry

	for _, entry := range entries {
		if !al.matchesFilter(entry, filter) {
			continue
		}
		filtered = append(filtered, entry)
	}

	return filtered
}

func (al *DefaultAuditLogger) matchesFilter(entry *AuditEntry, filter *AuditFilter) bool {
	if filter.WorkspaceID != "" && entry.WorkspaceID != filter.WorkspaceID {
		return false
	}

	if filter.ToolName != "" && entry.ToolName != filter.ToolName {
		return false
	}

	if !filter.StartTime.IsZero() && entry.Timestamp.Before(filter.StartTime) {
		return false
	}

	if !filter.EndTime.IsZero() && entry.Timestamp.After(filter.EndTime) {
		return false
	}

	if filter.Status != "" && entry.Status != filter.Status {
		return false
	}

	return true
}
