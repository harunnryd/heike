package session

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/harunnryd/heike/internal/config"
	"github.com/oklog/ulid/v2"

	"github.com/harunnryd/heike/internal/cognitive"
	"github.com/harunnryd/heike/internal/model/contract"
	"github.com/harunnryd/heike/internal/store"
)

type Manager interface {
	GetContext(ctx context.Context, sessionID string) (*cognitive.CognitiveContext, error)
	AppendInteraction(ctx context.Context, sessionID string, role, content string) error
	PersistTool(ctx context.Context, sessionID, toolCallID, content string) error
}

type DefaultSessionManager struct {
	store        *store.Worker
	memory       cognitive.MemoryManager
	historyLimit int
}

func NewManager(s *store.Worker, m cognitive.MemoryManager, historyLimit int) *DefaultSessionManager {
	if historyLimit <= 0 {
		historyLimit = config.DefaultOrchestratorSessionHistoryLimit
	}

	return &DefaultSessionManager{
		store:        s,
		memory:       m,
		historyLimit: historyLimit,
	}
}

func (sm *DefaultSessionManager) GetContext(ctx context.Context, sessionID string) (*cognitive.CognitiveContext, error) {
	// Load History
	historyLines, err := sm.store.ReadTranscript(sessionID, sm.historyLimit)
	if err != nil {
		slog.Warn("Failed to read transcript", "error", err)
	}

	history := sm.parseHistoryLines(historyLines)

	// Load Memories (using last message as query if available)
	var memories []string
	if len(history) > 0 {
		lastMsg := history[len(history)-1].Content
		if sm.memory != nil && lastMsg != "" {
			mems, err := sm.memory.Retrieve(ctx, lastMsg)
			if err != nil {
				slog.Warn("Failed to retrieve memories", "error", err)
			} else {
				memories = mems
			}
		}
	}

	return &cognitive.CognitiveContext{
		SessionID: sessionID,
		History:   history,
		Memories:  memories,
		Metadata:  make(map[string]string),
	}, nil
}

func (sm *DefaultSessionManager) AppendInteraction(ctx context.Context, sessionID string, role, content string) error {
	evt := Event{
		ID:        ulid.Make().String(),
		Timestamp: time.Now(),
		Type:      EventType(role), // Simplified mapping
		Role:      role,
		Content:   content,
	}

	// Adjust EventType for system/user
	if role == "system" {
		evt.Type = EventTypeSystem
	} else if role == "user" {
		evt.Type = EventTypeUser
	} else if role == "assistant" {
		evt.Type = EventTypeAssistant
	}

	line, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal failed: %w", err)
	}
	return sm.store.WriteTranscript(sessionID, line)
}

func (sm *DefaultSessionManager) PersistTool(ctx context.Context, sessionID, toolCallID, content string) error {
	evt := Event{
		ID:         ulid.Make().String(),
		Timestamp:  time.Now(),
		Type:       EventTypeTool,
		Role:       "tool",
		Content:    content,
		ToolCallID: toolCallID,
	}

	line, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal tool failed: %w", err)
	}
	return sm.store.WriteTranscript(sessionID, line)
}

func (sm *DefaultSessionManager) parseHistoryLines(historyLines []string) []contract.Message {
	var messages []contract.Message
	for _, line := range historyLines {
		var evt Event
		if err := json.Unmarshal([]byte(line), &evt); err != nil {
			continue
		}

		messages = append(messages, evt.ToContractMessage())
	}
	return messages
}
