package ingress

import (
	"context"
	"strings"
	"sync"

	"github.com/google/shlex"
)

// Destination represents the target of a routed event.
type DestinationType int

const (
	DestPipeline DestinationType = iota // Continue to Resolvers -> Queue
	DestCommand                         // Handle as direct command
	DestDrop                            // Drop the event
)

type Destination struct {
	Type    DestinationType
	Handler func(context.Context, *Event) error // For DestCommand
}

// Router determines the destination of an event.
type Router interface {
	Route(ctx context.Context, event *Event) Destination
}

type StandardRouter struct {
	commands map[string]func(context.Context, *Event) error
	mu       sync.RWMutex
}

func NewStandardRouter() *StandardRouter {
	r := &StandardRouter{
		commands: make(map[string]func(context.Context, *Event) error),
	}
	return r
}

func (r *StandardRouter) Route(ctx context.Context, event *Event) Destination {
	if !strings.HasPrefix(event.Content, "/") {
		return Destination{Type: DestPipeline}
	}

	parts, err := shlex.Split(event.Content)
	if err != nil || len(parts) == 0 {
		return Destination{Type: DestPipeline}
	}

	cmd := parts[0]

	r.mu.RLock()
	handler, exists := r.commands[cmd]
	r.mu.RUnlock()

	if exists {
		return Destination{
			Type:    DestCommand,
			Handler: handler,
		}
	}

	// Route unknown slash commands through pipeline so orchestrator command handler
	// can generate user-visible responses.
	event.Type = TypeCommand
	return Destination{Type: DestPipeline}
}

func (r *StandardRouter) RegisterCommand(name string, handler func(context.Context, *Event) error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.commands[name] = handler
}
