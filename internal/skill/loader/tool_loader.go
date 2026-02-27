package loader

import (
	"path/filepath"
	"strings"

	"github.com/harunnryd/heike/internal/tool"
)

type ToolLoader interface {
	LoadFromSkill(skillPath string) ([]*tool.CustomTool, error)
	LoadFromSource(skillsPath string) ([]*tool.CustomTool, error)
}

type DefaultToolLoader struct{}

func NewToolLoader() *DefaultToolLoader {
	return &DefaultToolLoader{}
}

func (tl *DefaultToolLoader) LoadFromSkill(skillPath string) ([]*tool.CustomTool, error) {
	return tl.discoverTools(skillPath)
}

func (tl *DefaultToolLoader) LoadFromSource(skillsPath string) ([]*tool.CustomTool, error) {
	return tl.discoverToolsFromDirectory(filepath.Clean(strings.TrimSpace(skillsPath)))
}

func (tl *DefaultToolLoader) discoverTools(skillPath string) ([]*tool.CustomTool, error) {
	toolsPath := filepath.Join(skillPath, "tools")
	manifest, err := tl.loadManifest(filepath.Join(toolsPath, "tools.yaml"))
	if err != nil {
		return nil, err
	}

	var tools []*tool.CustomTool
	for _, toolDef := range manifest.Tools {
		customTool, err := toolDef.ToCustomTool(skillPath, toolsPath)
		if err != nil {
			return nil, err
		}
		if customTool.Source == "" {
			customTool.Source = "skill"
		}
		tools = append(tools, customTool)
	}

	if len(tools) > 0 {
		return tools, nil
	}

	// Fallback discovery: load any supported script directly if tools.yaml is missing.
	files, err := tl.scanToolFiles(toolsPath)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		name := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		tools = append(tools, &tool.CustomTool{
			Name:         tool.NormalizeToolName(name),
			Language:     tl.detectToolType(file),
			ScriptPath:   file,
			Description:  name,
			Source:       "skill",
			SandboxLevel: tool.SandboxBasic,
		})
	}

	return tools, nil
}
