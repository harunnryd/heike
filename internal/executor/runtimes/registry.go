package runtimes

import (
	"fmt"

	"github.com/harunnryd/heike/internal/tool"
)

type RuntimeRegistry struct {
	runtimes map[tool.ToolType]LanguageRuntime
}

func NewRuntimeRegistry() (*RuntimeRegistry, error) {
	registry := &RuntimeRegistry{
		runtimes: make(map[tool.ToolType]LanguageRuntime),
	}

	if err := registry.registerRuntimes(); err != nil {
		return nil, err
	}

	return registry, nil
}

func (rr *RuntimeRegistry) registerRuntimes() error {
	pythonRuntime, err := NewPythonRuntime()
	if err == nil {
		rr.runtimes[tool.ToolTypePython] = pythonRuntime
	}

	shellRuntime, err := NewShellRuntime()
	if err == nil {
		rr.runtimes[tool.ToolTypeShell] = shellRuntime
	}

	goRuntime, err := NewGoRuntime()
	if err == nil {
		rr.runtimes[tool.ToolTypeGo] = goRuntime
	}

	nodeRuntime, err := NewNodeRuntime()
	if err == nil {
		rr.runtimes[tool.ToolTypeJS] = nodeRuntime
	}

	rubyRuntime, err := NewRubyRuntime()
	if err == nil {
		rr.runtimes[tool.ToolTypeRuby] = rubyRuntime
	}

	rustRuntime, err := NewRustRuntime()
	if err == nil {
		rr.runtimes[tool.ToolTypeRust] = rustRuntime
	}

	return nil
}

func (rr *RuntimeRegistry) Get(toolType tool.ToolType) (LanguageRuntime, error) {
	runtime, ok := rr.runtimes[toolType]
	if !ok {
		return nil, fmt.Errorf("runtime not found for type: %s", toolType)
	}
	return runtime, nil
}

func (rr *RuntimeRegistry) GetAll() map[tool.ToolType]LanguageRuntime {
	return rr.runtimes
}

func (rr *RuntimeRegistry) IsAvailable(toolType tool.ToolType) bool {
	_, ok := rr.runtimes[toolType]
	return ok
}
