package session

import (
	"time"

	"github.com/harunnryd/heike/internal/model/contract"
)

type EventType string

const (
	EventTypeUser      EventType = "user"
	EventTypeAssistant EventType = "assistant"
	EventTypeTool      EventType = "tool"
	EventTypeSystem    EventType = "system"
)

// Event represents a persisted interaction in the session history
type Event struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"ts"`
	Type      EventType `json:"type"` // Maps to Role usually, but more explicit

	// Core Content (compatible with contract.Message)
	Role       string               `json:"role"`
	Content    string               `json:"content"`
	ToolCalls  []*contract.ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string               `json:"tool_call_id,omitempty"`

	// Extended Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (e Event) ToContractMessage() contract.Message {
	return contract.Message{
		Role:       e.Role,
		Content:    e.Content,
		ToolCalls:  e.ToolCalls,
		ToolCallID: e.ToolCallID,
	}
}
