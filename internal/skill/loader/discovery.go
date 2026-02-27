package loader

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/harunnryd/heike/internal/tool"
)

func (tl *DefaultToolLoader) discoverToolsFromDirectory(skillsPath string) ([]*tool.CustomTool, error) {
	entries, err := os.ReadDir(skillsPath)
	if os.IsNotExist(err) {
		return []*tool.CustomTool{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read skills directory: %w", err)
	}

	var allTools []*tool.CustomTool

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillPath := filepath.Join(skillsPath, entry.Name())
		tools, err := tl.discoverTools(skillPath)
		if err != nil {
			slog.Warn("Failed to discover tools from skill", "skill", entry.Name(), "error", err)
			continue
		}

		allTools = append(allTools, tools...)
	}

	return allTools, nil
}

func (tl *DefaultToolLoader) scanToolFiles(toolsPath string) ([]string, error) {
	var toolFiles []string

	extensions := map[tool.ToolType]string{
		tool.ToolTypePython: ".py",
		tool.ToolTypeGo:     ".go",
		tool.ToolTypeJS:     ".js",
		tool.ToolTypeShell:  ".sh",
		tool.ToolTypeRuby:   ".rb",
		tool.ToolTypeRust:   ".rs",
	}

	entries, err := os.ReadDir(toolsPath)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := filepath.Ext(entry.Name())
		for _, toolExt := range extensions {
			if ext == toolExt {
				toolFiles = append(toolFiles, filepath.Join(toolsPath, entry.Name()))
				break
			}
		}
	}

	return toolFiles, nil
}

func (tl *DefaultToolLoader) detectToolType(filename string) tool.ToolType {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".py":
		return tool.ToolTypePython
	case ".go":
		return tool.ToolTypeGo
	case ".js":
		return tool.ToolTypeJS
	case ".sh":
		return tool.ToolTypeShell
	case ".rb":
		return tool.ToolTypeRuby
	case ".rs":
		return tool.ToolTypeRust
	default:
		return tool.ToolTypeShell
	}
}
