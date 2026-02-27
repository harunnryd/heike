package skill

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	toolcore "github.com/harunnryd/heike/internal/tool"
	_ "github.com/harunnryd/heike/internal/tool/builtin"
)

func TestBundledSkills_LoadAndReferenceKnownBuiltinTools(t *testing.T) {
	root := repoRootFromThisFile(t)
	skillsPath := filepath.Join(root, "skills")
	if _, err := os.Stat(skillsPath); err != nil {
		t.Fatalf("skills path %q not readable: %v", skillsPath, err)
	}

	registry := NewRegistry()
	if err := registry.Load(skillsPath); err != nil {
		t.Fatalf("load bundled skills: %v", err)
	}
	if len(registry.skills) == 0 {
		t.Fatal("expected bundled skills to be loaded")
	}

	builtinNames := toolcore.BuiltinNames()
	allowedTools := make(map[string]struct{}, len(builtinNames))
	for _, toolName := range builtinNames {
		allowedTools[toolName] = struct{}{}
	}

	for skillName, item := range registry.skills {
		if item == nil {
			t.Fatalf("skill %q is nil", skillName)
		}
		if len(item.Metadata) == 0 {
			t.Fatalf("skill %q is missing metadata", skillName)
		}

		heikeMetaRaw, ok := item.Metadata["heike"]
		if !ok {
			t.Fatalf("skill %q metadata is missing heike key", skillName)
		}
		heikeMeta, ok := heikeMetaRaw.(map[string]interface{})
		if !ok {
			t.Fatalf("skill %q metadata.heike has unexpected type %T", skillName, heikeMetaRaw)
		}
		category, ok := heikeMeta["category"].(string)
		if !ok || strings.TrimSpace(category) == "" {
			t.Fatalf("skill %q metadata.heike.category must be non-empty", skillName)
		}
		icon, ok := heikeMeta["icon"].(string)
		if !ok || strings.TrimSpace(icon) == "" {
			t.Fatalf("skill %q metadata.heike.icon must be non-empty", skillName)
		}
		kindRaw, ok := heikeMeta["kind"].(string)
		if !ok || strings.TrimSpace(kindRaw) == "" {
			t.Fatalf("skill %q metadata.heike.kind must be non-empty", skillName)
		}
		kind := strings.TrimSpace(strings.ToLower(kindRaw))

		manifestPath := filepath.Join(skillsPath, skillName, "tools", "tools.yaml")
		_, manifestErr := os.Stat(manifestPath)
		hasManifest := manifestErr == nil
		if manifestErr != nil && !os.IsNotExist(manifestErr) {
			t.Fatalf("skill %q manifest check failed: %v", skillName, manifestErr)
		}

		switch kind {
		case "guidance":
			if hasManifest {
				t.Fatalf("skill %q has kind=guidance but defines runtime manifest at %s", skillName, manifestPath)
			}
		case "runtime":
			if !hasManifest {
				t.Fatalf("skill %q has kind=runtime but missing manifest at %s", skillName, manifestPath)
			}
		default:
			t.Fatalf("skill %q metadata.heike.kind must be one of: guidance|runtime (got %q)", skillName, kindRaw)
		}

		if len(item.Tools) == 0 {
			t.Fatalf("skill %q has no declared tools", skillName)
		}

		seen := make(map[string]struct{}, len(item.Tools))
		for _, toolName := range item.Tools {
			normalized := strings.TrimSpace(toolName)
			if normalized == "" {
				t.Fatalf("skill %q declares empty tool name", skillName)
			}
			if _, exists := seen[normalized]; exists {
				t.Fatalf("skill %q has duplicate tool entry %q", skillName, normalized)
			}
			seen[normalized] = struct{}{}

			if _, ok := allowedTools[normalized]; !ok {
				t.Fatalf("skill %q references unknown built-in tool %q", skillName, normalized)
			}
		}
	}
}

func repoRootFromThisFile(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
