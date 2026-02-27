package command

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/harunnryd/heike/internal/orchestrator/session"
	"github.com/harunnryd/heike/internal/policy"
	"github.com/harunnryd/heike/internal/store"

	"github.com/google/shlex"
)

type Handler interface {
	CanHandle(input string) bool
	Execute(ctx context.Context, sessionID string, input string) error
}

type DefaultCommandHandler struct {
	policy  *policy.Engine
	session session.Manager
	store   *store.Worker
	output  commandOutput
}

type commandOutput interface {
	Send(ctx context.Context, sessionID string, content string) error
}

const commandOutputPrefix = "[CMD] "
const defaultCommandSessionSource = "cli"

func NewHandler(p *policy.Engine, s session.Manager, st *store.Worker, output commandOutput) *DefaultCommandHandler {
	return &DefaultCommandHandler{
		policy:  p,
		session: s,
		store:   st,
		output:  output,
	}
}

func (h *DefaultCommandHandler) CanHandle(input string) bool {
	return strings.HasPrefix(input, "/")
}

func (h *DefaultCommandHandler) Execute(ctx context.Context, sessionID string, input string) error {
	parts, parseErr := shlex.Split(input)
	if parseErr != nil {
		parts = strings.Fields(input)
	}
	if len(parts) == 0 {
		return nil
	}
	cmd := parts[0]
	args := parts[1:]

	slog.Info("Executing slash command", "cmd", cmd, "session", sessionID)

	var msg string
	var err error

	switch cmd {
	case "/approve":
		msg, err = h.handleApprove(args)
	case "/deny":
		msg, err = h.handleDeny(args)
	case "/clear":
		msg, err = h.handleClear(sessionID)
	case "/model":
		msg, err = h.handleModel(sessionID, args)
	case "/help":
		msg = h.helpText()
	default:
		msg = fmt.Sprintf("Unknown command: %s", cmd)
	}

	if err != nil {
		msg = fmt.Sprintf("Command failed: %v", err)
		slog.Error("Command execution failed", "cmd", cmd, "error", err)
	}

	if err := h.session.AppendInteraction(ctx, sessionID, "system", msg); err != nil {
		return err
	}
	if h.output != nil {
		if err := h.output.Send(ctx, sessionID, formatCommandOutput(msg)); err != nil {
			return fmt.Errorf("send command output: %w", err)
		}
	}

	return nil
}

func (h *DefaultCommandHandler) handleApprove(args []string) (string, error) {
	if len(args) < 1 {
		return "Usage: /approve <id>", nil
	}
	if h.policy == nil {
		return "", fmt.Errorf("policy engine not initialized")
	}
	id := args[0]
	if err := h.policy.Resolve(id, true); err != nil {
		return "", err
	}
	return fmt.Sprintf("Approved: %s. You can retry the action now.", id), nil
}

func (h *DefaultCommandHandler) handleDeny(args []string) (string, error) {
	if len(args) < 1 {
		return "Usage: /deny <id>", nil
	}
	if h.policy == nil {
		return "", fmt.Errorf("policy engine not initialized")
	}
	id := args[0]
	if err := h.policy.Resolve(id, false); err != nil {
		return "", err
	}
	return fmt.Sprintf("Denied: %s", id), nil
}

func (h *DefaultCommandHandler) handleClear(sessionID string) (string, error) {
	if sessionID == "" {
		return "", fmt.Errorf("session id is required")
	}
	if h.store == nil {
		return "", fmt.Errorf("store not initialized")
	}
	existing, err := h.store.GetSession(sessionID)
	if err != nil {
		return "", err
	}
	source := sessionSourceOrDefault(existing)
	title := "Session " + sessionID
	if existing != nil && strings.TrimSpace(existing.Title) != "" {
		title = existing.Title
	}

	if err := h.store.ResetSession(sessionID); err != nil {
		return "", err
	}
	if err := h.store.SaveSession(&store.SessionMeta{
		ID:        sessionID,
		Title:     title,
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  map[string]string{"source": source},
	}); err != nil {
		return "", err
	}

	return "Session cleared.", nil
}

func (h *DefaultCommandHandler) handleModel(sessionID string, args []string) (string, error) {
	if len(args) < 1 {
		return "Usage: /model <name>", nil
	}
	if sessionID == "" {
		return "", fmt.Errorf("session id is required")
	}
	if h.store == nil {
		return "", fmt.Errorf("store not initialized")
	}

	modelName := args[0]
	sess, err := h.store.GetSession(sessionID)
	if err != nil {
		return "", err
	}
	if sess == nil {
		sess = &store.SessionMeta{
			ID:        sessionID,
			Title:     "Session " + sessionID,
			Status:    "active",
			CreatedAt: time.Now(),
			Metadata:  map[string]string{"source": defaultCommandSessionSource},
		}
	}
	if sess.Metadata == nil {
		sess.Metadata = make(map[string]string)
	}
	if strings.TrimSpace(sess.Metadata["source"]) == "" {
		sess.Metadata["source"] = defaultCommandSessionSource
	}
	sess.Metadata["model"] = modelName
	sess.UpdatedAt = time.Now()

	if err := h.store.SaveSession(sess); err != nil {
		return "", err
	}
	return fmt.Sprintf("Model set to %s", modelName), nil
}

func (h *DefaultCommandHandler) helpText() string {
	return "Available commands: /help, /model <name>, /clear, /approve <id>, /deny <id>"
}

func formatCommandOutput(msg string) string {
	if strings.HasPrefix(msg, commandOutputPrefix) {
		return msg
	}
	return commandOutputPrefix + msg
}

func sessionSourceOrDefault(meta *store.SessionMeta) string {
	if meta != nil && meta.Metadata != nil {
		source := strings.TrimSpace(meta.Metadata["source"])
		if source != "" {
			return source
		}
	}
	return defaultCommandSessionSource
}
