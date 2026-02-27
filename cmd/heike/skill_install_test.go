package main

import (
	"os"
	"path/filepath"
	"testing"

	skillmodel "github.com/harunnryd/heike/internal/skill"
)

func TestResolveSkillSource(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	skillFile := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillFile, []byte("---\nname: demo\ndescription: test\ntags: [x]\n---\nbody"), 0644); err != nil {
		t.Fatalf("write skill file: %v", err)
	}

	gotDir, gotFile, err := resolveSkillSource(skillDir)
	if err != nil {
		t.Fatalf("resolve dir failed: %v", err)
	}
	if gotDir != skillDir || gotFile != skillFile {
		t.Fatalf("resolved (%s, %s), want (%s, %s)", gotDir, gotFile, skillDir, skillFile)
	}

	gotDir, gotFile, err = resolveSkillSource(skillFile)
	if err != nil {
		t.Fatalf("resolve file failed: %v", err)
	}
	if gotDir != skillDir || gotFile != skillFile {
		t.Fatalf("resolved (%s, %s), want (%s, %s)", gotDir, gotFile, skillDir, skillFile)
	}
}

func TestCopySkillDirectory(t *testing.T) {
	src := t.TempDir()
	dest := t.TempDir()

	if err := os.MkdirAll(filepath.Join(src, "scripts"), 0755); err != nil {
		t.Fatalf("mkdir src scripts: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("demo"), 0644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "scripts", "run.sh"), []byte("echo ok\n"), 0755); err != nil {
		t.Fatalf("write run.sh: %v", err)
	}

	target := filepath.Join(dest, "installed")
	if err := copySkillDirectory(src, target); err != nil {
		t.Fatalf("copySkillDirectory failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(target, "SKILL.md")); err != nil {
		t.Fatalf("installed SKILL.md missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "scripts", "run.sh")); err != nil {
		t.Fatalf("installed scripts/run.sh missing: %v", err)
	}
}

func TestLoadSkillFromFile_MetadataIsPreserved(t *testing.T) {
	root := t.TempDir()
	skillFile := filepath.Join(root, "SKILL.md")
	content := `---
name: demo
description: test
tags:
  - x
tools:
  - exec_command
metadata:
  heike:
    requires:
      bins:
        - python3
---
body`
	if err := os.WriteFile(skillFile, []byte(content), 0644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	loaded, err := skillmodel.LoadSkillFromFile(skillFile)
	if err != nil {
		t.Fatalf("LoadSkillFromFile failed: %v", err)
	}
	if len(loaded.Metadata) == 0 {
		t.Fatal("expected metadata to be preserved")
	}
}
