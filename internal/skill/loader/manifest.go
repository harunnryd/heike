package loader

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/harunnryd/heike/internal/tool"

	"gopkg.in/yaml.v3"
)

type ToolManifest struct {
	Tools []ToolDefinition `yaml:"tools"`
}

type ToolDefinition struct {
	Name         string        `yaml:"name"`
	Language     tool.ToolType `yaml:"language"`
	Script       string        `yaml:"script"`
	Description  string        `yaml:"description"`
	Parameters   interface{}   `yaml:"parameters"`
	Capabilities []string      `yaml:"capabilities"`
	Source       string        `yaml:"source"`
	Risk         string        `yaml:"risk"`
	Sandbox      string        `yaml:"sandbox"`
	Dependencies []string      `yaml:"dependencies"`
}

func (td *ToolDefinition) ToCustomTool(skillPath, toolsPath string) (*tool.CustomTool, error) {
	sandboxLevel := tool.SandboxBasic
	if td.Sandbox != "" {
		sandboxLevel = tool.SandboxLevel(td.Sandbox)
	}

	scriptPath, err := resolveScriptPath(skillPath, toolsPath, td.Script)
	if err != nil {
		return nil, err
	}

	var params json.RawMessage
	if td.Parameters != nil {
		b, err := json.Marshal(td.Parameters)
		if err != nil {
			return nil, fmt.Errorf("marshal tool parameters for %s: %w", td.Name, err)
		}
		params = b
	}

	return &tool.CustomTool{
		Name:         tool.NormalizeToolName(td.Name),
		Language:     td.Language,
		ScriptPath:   scriptPath,
		Description:  td.Description,
		Parameters:   params,
		Capabilities: append([]string(nil), td.Capabilities...),
		Source:       td.Source,
		Risk:         tool.RiskLevel(td.Risk),
		SandboxLevel: sandboxLevel,
	}, nil
}

func (tl *DefaultToolLoader) loadManifest(manifestPath string) (*ToolManifest, error) {
	data, err := os.ReadFile(manifestPath)
	if os.IsNotExist(err) {
		return &ToolManifest{Tools: []ToolDefinition{}}, nil
	}
	if err != nil {
		return nil, err
	}

	var manifest ToolManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

func resolveScriptPath(skillPath, toolsPath, script string) (string, error) {
	if script == "" {
		return "", fmt.Errorf("tool script path is required")
	}

	if filepath.IsAbs(script) {
		if _, err := os.Stat(script); err != nil {
			return "", fmt.Errorf("tool script not found: %w", err)
		}
		return filepath.Clean(script), nil
	}

	candidates := []string{
		filepath.Join(skillPath, script),
		filepath.Join(toolsPath, script),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return filepath.Clean(candidate), nil
		}
	}

	return "", fmt.Errorf("tool script not found: %s", script)
}
