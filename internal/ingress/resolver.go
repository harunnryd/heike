package ingress

import (
	"context"
	"fmt"
	"time"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/store"

	"github.com/oklog/ulid/v2"
)

type Resolver interface {
	ResolveWorkspace(ctx context.Context, event *Event) (string, error)
	ResolveSession(ctx context.Context, event *Event) (string, error)
}

type StandardResolver struct {
	store *store.Worker
}

func NewStandardResolver(store *store.Worker) *StandardResolver {
	return &StandardResolver{store: store}
}

func (r *StandardResolver) ResolveWorkspace(ctx context.Context, event *Event) (string, error) {
	if event == nil {
		return "", fmt.Errorf("event is nil")
	}
	// If already set, return it
	if event.WorkspaceID != "" {
		return event.WorkspaceID, nil
	}

	// Try metadata
	if ws, ok := event.Metadata["workspace_id"]; ok && ws != "" {
		return ws, nil
	}

	// Default
	return config.DefaultWorkspaceID, nil
}

func (r *StandardResolver) ResolveSession(ctx context.Context, event *Event) (string, error) {
	if event == nil {
		return "", fmt.Errorf("event is nil")
	}
	if r.store == nil {
		return "", fmt.Errorf("store is nil")
	}

	// Ensure metadata exists and has source
	if event.Metadata == nil {
		event.Metadata = make(map[string]string)
	}
	if _, ok := event.Metadata["source"]; !ok {
		event.Metadata["source"] = event.Source
	}

	if event.SessionID != "" {
		if err := r.ensureSession(event.SessionID, event.Metadata, "Session "+event.SessionID); err != nil {
			return "", err
		}
		return event.SessionID, nil
	}

	var sessionID string
	switch event.Source {
	case "slack":
		if thread, ok := event.Metadata["thread_ts"]; ok && thread != "" {
			sessionID = thread
		} else if channel, ok := event.Metadata["channel_id"]; ok && channel != "" {
			sessionID = channel
		}
	case "telegram":
		if chatID, ok := event.Metadata["chat_id"]; ok && chatID != "" {
			sessionID = chatID
		}
	case "scheduler":
		workspaceID := event.Metadata["workspace_id"]
		if workspaceID == "" {
			workspaceID = config.DefaultWorkspaceID
		}
		sessionID = "scheduler:" + workspaceID
	case "cli":
		sessionID = "cli:" + ulid.Make().String()
	}

	if sessionID == "" {
		sessionID = "sess_" + ulid.Make().String()
	}

	if err := r.ensureSession(sessionID, event.Metadata, "New Session"); err != nil {
		return "", err
	}

	return sessionID, nil
}

func (r *StandardResolver) ensureSession(sessionID string, metadata map[string]string, title string) error {
	sess, err := r.store.GetSession(sessionID)
	if err != nil {
		return err
	}
	if sess != nil {
		return nil
	}
	return r.store.SaveSession(&store.SessionMeta{
		ID:        sessionID,
		Title:     title,
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  metadata,
	})
}
