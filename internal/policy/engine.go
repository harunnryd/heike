package policy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/harunnryd/heike/internal/config"
	heikeErrors "github.com/harunnryd/heike/internal/errors"
	"github.com/harunnryd/heike/internal/store"

	"github.com/natefinch/atomic"
	"github.com/oklog/ulid/v2"
)

type ApprovalStatus string

const (
	StatusPending ApprovalStatus = "PENDING"
	StatusGranted ApprovalStatus = "GRANTED"
	StatusDenied  ApprovalStatus = "DENIED"
)

const (
	sandboxPermissionUseDefault       = "use_default"
	sandboxPermissionRequireEscalated = "require_escalated"
)

type Approval struct {
	ID        string         `json:"id"`
	Tool      string         `json:"tool"`
	Input     string         `json:"input"`
	Status    ApprovalStatus `json:"status"`
	CreatedAt time.Time      `json:"created_at"`
}

type DomainList struct {
	Allowed []string `json:"allowed"`
}

type Engine struct {
	config         config.GovernanceConfig
	storePath      string
	domainPath     string
	approvals      map[string]Approval
	allowedDomains []string
	mu             sync.RWMutex
	store          *store.Worker
	// Quota limits
	dailyLimit int
	usage      map[string]int // tool -> count
}

func NewEngine(cfg config.GovernanceConfig, workspaceID string, workspaceRootPath string) (*Engine, error) {
	workspacePath, err := store.GetWorkspacePath(workspaceID, workspaceRootPath)
	if err != nil {
		return nil, err
	}
	base := filepath.Join(workspacePath, "governance")
	storePath := filepath.Join(base, "approvals.json")
	domainPath := filepath.Join(base, "domains.json")

	if err := os.MkdirAll(base, 0755); err != nil {
		return nil, fmt.Errorf("failed to create governance dir: %w", err)
	}

	e := &Engine{
		config:     cfg,
		storePath:  storePath,
		domainPath: domainPath,
		approvals:  make(map[string]Approval),
		usage:      make(map[string]int),
		dailyLimit: cfg.DailyToolLimit,
	}
	if e.dailyLimit <= 0 {
		e.dailyLimit = config.DefaultGovernanceDailyToolLimit
	}
	if err := e.load(); err != nil {
		return nil, err
	}
	return e, nil
}

func (e *Engine) load() error {
	// Load Approvals
	data, err := os.ReadFile(e.storePath)
	if err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &e.approvals); err != nil {
			return err
		}
	}

	// Load Domains
	dData, err := os.ReadFile(e.domainPath)
	if err == nil && len(dData) > 0 {
		var dl DomainList
		if err := json.Unmarshal(dData, &dl); err == nil {
			e.allowedDomains = dl.Allowed
		}
	}
	return nil
}

func (e *Engine) save() error {
	data, err := json.MarshalIndent(e.approvals, "", "  ")
	if err != nil {
		return err
	}
	return atomic.WriteFile(e.storePath, bytes.NewReader(data))
}

func (e *Engine) saveDomains() error {
	dl := DomainList{Allowed: e.allowedDomains}
	data, err := json.MarshalIndent(dl, "", "  ")
	if err != nil {
		return err
	}
	return atomic.WriteFile(e.domainPath, bytes.NewReader(data))
}

// Check evaluates whether a tool call is allowed.
func (e *Engine) Check(toolName string, input json.RawMessage) (bool, string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	toolName = normalizeToolName(toolName)

	// Explicit sandbox permission policy.
	if sandboxPerm, ok := extractSandboxPermissionsFromInput(input); ok {
		switch sandboxPerm {
		case "", sandboxPermissionUseDefault:
			// continue
		case sandboxPermissionRequireEscalated:
			return e.createApproval(toolName, input)
		default:
			return false, "", fmt.Errorf("sandbox_permissions %q is denied: %w", sandboxPerm, heikeErrors.ErrPermissionDenied)
		}
	}

	// Quota Check
	if count := e.usage[toolName]; count >= e.dailyLimit {
		return false, "", fmt.Errorf("quota exceeded for tool %s", toolName)
	}

	// Domain allowlist applies to any tool input that carries a URL.
	if host, ok := extractHostFromInput(input); ok {
		if !containsDomain(e.allowedDomains, host) {
			return e.createApproval(toolName, input)
		}
		e.consumeQuotaLocked(toolName)
		return true, "", nil
	}

	// Check Auto-Allow List
	for _, allowed := range e.config.AutoAllow {
		if normalizeToolName(allowed) == toolName {
			e.consumeQuotaLocked(toolName)
			return true, "", nil
		}
	}

	// Check Require-Approval List
	requiresApproval := false
	for _, restricted := range e.config.RequireApproval {
		if normalizeToolName(restricted) == toolName {
			requiresApproval = true
			break
		}
	}

	// Create Approval Request
	if !requiresApproval {
		e.consumeQuotaLocked(toolName)
		return true, "", nil
	}

	return e.createApproval(toolName, input)
}

func (e *Engine) createApproval(toolName string, input json.RawMessage) (bool, string, error) {
	toolName = normalizeToolName(toolName)
	id := ulid.Make().String()
	app := Approval{
		ID:        id,
		Tool:      toolName,
		Input:     string(input),
		Status:    StatusPending,
		CreatedAt: time.Now(),
	}
	e.approvals[id] = app
	if err := e.save(); err != nil {
		return false, "", fmt.Errorf("failed to persist approval: %w", err)
	}

	slog.Info("Approval required", "id", id, "tool", toolName)
	return false, id, heikeErrors.ErrApprovalRequired
}

// Resolve updates the status of an approval.
func (e *Engine) Resolve(id string, approve bool) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	app, ok := e.approvals[id]
	if !ok {
		return fmt.Errorf("approval request not found: %s", id)
	}

	if app.Status != StatusPending {
		return fmt.Errorf("approval %s is already %s", id, app.Status)
	}

	if approve {
		app.Status = StatusGranted

		// If input contains URL, persist domain into allowlist after approval.
		if host, ok := extractHostFromInput(json.RawMessage(app.Input)); ok {
			if !containsDomain(e.allowedDomains, host) {
				e.allowedDomains = append(e.allowedDomains, host)
				e.saveDomains()
				slog.Info("Domain added to allowlist", "domain", host)
			}
		}

	} else {
		app.Status = StatusDenied
	}
	e.approvals[id] = app

	return e.save()
}

func normalizeToolName(name string) string {
	return strings.TrimSpace(name)
}

func extractHostFromInput(input json.RawMessage) (string, bool) {
	var args struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", false
	}
	parsed, err := url.Parse(strings.TrimSpace(args.URL))
	if err != nil {
		return "", false
	}
	host := strings.TrimSpace(strings.ToLower(parsed.Hostname()))
	if host == "" {
		return "", false
	}
	return host, true
}

func extractSandboxPermissionsFromInput(input json.RawMessage) (string, bool) {
	var args struct {
		SandboxPermissions string `json:"sandbox_permissions"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", false
	}
	if strings.TrimSpace(args.SandboxPermissions) == "" {
		return "", false
	}
	return strings.TrimSpace(strings.ToLower(args.SandboxPermissions)), true
}

func containsDomain(domains []string, host string) bool {
	for _, domain := range domains {
		if strings.EqualFold(strings.TrimSpace(domain), host) {
			return true
		}
	}
	return false
}

// IsGranted checks if a specific approval ID has been granted.
func (e *Engine) IsGranted(id string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	app, ok := e.approvals[id]
	return ok && app.Status == StatusGranted
}

func (e *Engine) consumeQuotaLocked(toolName string) {
	e.usage[normalizeToolName(toolName)]++
}

func (e *Engine) ConsumeQuota(toolName string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	normalized := normalizeToolName(toolName)
	if e.usage[normalized] >= e.dailyLimit {
		return fmt.Errorf("quota exceeded for tool %s", normalized)
	}
	e.consumeQuotaLocked(normalized)
	return nil
}

func (e *Engine) ListApprovals(statuses ...ApprovalStatus) []Approval {
	e.mu.RLock()
	defer e.mu.RUnlock()

	filter := make(map[ApprovalStatus]struct{}, len(statuses))
	for _, status := range statuses {
		filter[status] = struct{}{}
	}

	approvals := make([]Approval, 0, len(e.approvals))
	for _, approval := range e.approvals {
		if len(filter) > 0 {
			if _, ok := filter[approval.Status]; !ok {
				continue
			}
		}
		approvals = append(approvals, approval)
	}

	sort.Slice(approvals, func(i, j int) bool {
		return approvals[i].CreatedAt.After(approvals[j].CreatedAt)
	})
	return approvals
}
