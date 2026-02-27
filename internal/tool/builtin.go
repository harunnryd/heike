package tool

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// BuiltinOptions carries runtime dependencies needed by built-in tool factories.
type BuiltinOptions struct {
	WebTimeout          time.Duration
	WebBaseURL          string
	WebMaxContentLength int
	WeatherBaseURL      string
	WeatherTimeout      time.Duration
	FinanceBaseURL      string
	FinanceTimeout      time.Duration
	SportsBaseURL       string
	SportsTimeout       time.Duration
	ImageQueryBaseURL   string
	ImageQueryTimeout   time.Duration
	ScreenshotTimeout   time.Duration
	ScreenshotRenderer  string
	ApplyPatchCommand   string
}

const (
	DefaultBuiltinWebTimeout          = 10 * time.Second
	DefaultBuiltinWebMaxContentLength = 5000
)

type BuiltinFactory func(options BuiltinOptions) (Tool, error)

var builtinCatalog = struct {
	mu        sync.RWMutex
	factories map[string]BuiltinFactory
}{
	factories: map[string]BuiltinFactory{},
}

// RegisterBuiltin registers a built-in tool factory under a tool name.
// Intended to be called in init() from built-in tool files.
func RegisterBuiltin(name string, factory BuiltinFactory) {
	normalized := NormalizeToolName(name)
	if normalized == "" {
		panic("tool: built-in name cannot be empty")
	}
	if factory == nil {
		panic(fmt.Sprintf("tool: built-in factory cannot be nil (%s)", normalized))
	}

	builtinCatalog.mu.Lock()
	defer builtinCatalog.mu.Unlock()

	if _, exists := builtinCatalog.factories[normalized]; exists {
		panic(fmt.Sprintf("tool: built-in already registered: %s", normalized))
	}
	builtinCatalog.factories[normalized] = factory
}

// BuiltinNames returns all registered built-in names in deterministic order.
func BuiltinNames() []string {
	builtinCatalog.mu.RLock()
	defer builtinCatalog.mu.RUnlock()

	names := make([]string, 0, len(builtinCatalog.factories))
	for name := range builtinCatalog.factories {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// IsBuiltinName reports whether a tool name maps to a registered built-in tool.
func IsBuiltinName(name string) bool {
	normalized := NormalizeToolName(name)
	if normalized == "" {
		return false
	}

	builtinCatalog.mu.RLock()
	defer builtinCatalog.mu.RUnlock()
	_, ok := builtinCatalog.factories[normalized]
	return ok
}

// InstantiateBuiltins constructs all built-in tools using their registered factories.
func InstantiateBuiltins(options BuiltinOptions) ([]Tool, error) {
	names := BuiltinNames()

	builtinCatalog.mu.RLock()
	factories := make(map[string]BuiltinFactory, len(builtinCatalog.factories))
	for name, factory := range builtinCatalog.factories {
		factories[name] = factory
	}
	builtinCatalog.mu.RUnlock()

	tools := make([]Tool, 0, len(names))
	for _, name := range names {
		toolFactory, ok := factories[name]
		if !ok {
			continue
		}

		t, err := toolFactory(options)
		if err != nil {
			return nil, fmt.Errorf("instantiate built-in %q: %w", name, err)
		}
		tools = append(tools, t)
	}

	return tools, nil
}
