package tool

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"github.com/harunnryd/heike/internal/model/contract"
)

var (
	ErrToolNotFound = errors.New("tool not found")
	ErrToolFailed   = errors.New("tool execution failed")
)

// Tool represents an executable capability.
type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]interface{}
	Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error)
}

// Registry holds all available tools.
type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

func (r *Registry) Register(t Tool) {
	name := NormalizeToolName(t.Name())
	if name == "" {
		panic("tool: empty tool name")
	}

	r.tools[name] = t
}

func (r *Registry) Get(name string) (Tool, bool) {
	name = NormalizeToolName(name)
	t, ok := r.tools[name]
	return t, ok
}

func (r *Registry) GetDescriptors() []ToolDescriptor {
	unique := make(map[string]ToolDescriptor)
	for _, t := range r.tools {
		name := DefinitionToolName(t.Name())
		if _, exists := unique[name]; exists {
			continue
		}

		meta := normalizeToolMetadata(ToolMetadata{})
		if provider, ok := t.(MetadataProvider); ok {
			meta = normalizeToolMetadata(provider.ToolMetadata())
		}

		unique[name] = ToolDescriptor{
			Definition: contract.ToolDef{
				Name:        name,
				Description: t.Description(),
				Parameters:  t.Parameters(),
			},
			Metadata: meta,
		}
	}

	names := make([]string, 0, len(unique))
	for name := range unique {
		names = append(names, name)
	}
	sort.Strings(names)

	descriptors := make([]ToolDescriptor, 0, len(names))
	for _, name := range names {
		descriptors = append(descriptors, unique[name])
	}
	return descriptors
}

// Common input structs

type FilePathInput struct {
	Path string `json:"path"`
}

type ExecCommandInput struct {
	Cmd             string   `json:"cmd"`
	Command         string   `json:"command"`
	Args            []string `json:"args"`
	TTY             bool     `json:"tty"`
	YieldTimeMS     int      `json:"yield_time_ms"`
	MaxOutputTokens int      `json:"max_output_tokens"`
	Workdir         string   `json:"workdir"`
	Shell           string   `json:"shell"`
	Login           *bool    `json:"login"`
	Justification   string   `json:"justification"`
	PrefixRule      []string `json:"prefix_rule"`
	SandboxPerms    string   `json:"sandbox_permissions"`
}

func NormalizeToolName(name string) string {
	return strings.TrimSpace(name)
}

func DefinitionToolName(name string) string {
	return NormalizeToolName(name)
}
