package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestSkillDir(t *testing.T) string {
	tmpDir := t.TempDir()
	return tmpDir
}

func createSkillFile(t *testing.T, dir, name string, content string) {
	skillDir := filepath.Join(dir, name)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Error("NewRegistry() returned nil")
	}
	if r.skills == nil {
		t.Error("skills map not initialized")
	}
}

func TestNewRegistryWithPath(t *testing.T) {
	r := NewRegistryWithPath("/some/path")
	if r == nil {
		t.Error("NewRegistryWithPath() returned nil")
	}
	if r.loadPath != "/some/path" {
		t.Errorf("loadPath = %v, want /some/path", r.loadPath)
	}
}

func TestRegistry_Load(t *testing.T) {
	tmpDir := setupTestSkillDir(t)

	content := `---
name: test_skill
description: A test skill
tags:
  - tag1
  - tag2
tools:
  - tool1
  - tool2
---
This is the skill content.
`
	createSkillFile(t, tmpDir, "test_skill", content)

	r := NewRegistry()
	if err := r.Load(tmpDir); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if len(r.skills) != 1 {
		t.Errorf("loaded %d skills, want 1", len(r.skills))
	}

	skill, ok := r.skills["test_skill"]
	if !ok {
		t.Error("test_skill not found in registry")
	}
	if skill.Name != "test_skill" {
		t.Errorf("skill.Name = %v, want test_skill", skill.Name)
	}
}

func TestRegistry_Load_IncludesMetadata(t *testing.T) {
	tmpDir := setupTestSkillDir(t)

	content := `---
name: test_skill
description: A test skill
tags:
  - tag1
tools:
  - tool1
metadata:
  heike:
    requires:
      bins:
        - python3
      env:
        - OPENAI_API_KEY
---
This is the skill content.
`
	createSkillFile(t, tmpDir, "test_skill", content)

	r := NewRegistry()
	if err := r.Load(tmpDir); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	sk, ok := r.skills["test_skill"]
	if !ok {
		t.Fatal("test_skill not found")
	}
	if len(sk.Metadata) == 0 {
		t.Fatal("metadata should be loaded")
	}
	heikeMeta, ok := sk.Metadata["heike"].(map[string]interface{})
	if !ok || len(heikeMeta) == 0 {
		t.Fatalf("heike metadata should exist, got: %#v", sk.Metadata)
	}
}

func TestRegistry_Load_NonExistentDir(t *testing.T) {
	r := NewRegistry()
	if err := r.Load("/nonexistent/path"); err != nil {
		t.Errorf("Load() should not error for non-existent dir, got: %v", err)
	}
}

func TestRegistry_Load_EmptyPath(t *testing.T) {
	r := NewRegistry()
	if err := r.Load(""); err == nil {
		t.Error("Load() should error for empty path")
	}
}

func TestRegistry_Load_InvalidFrontmatter(t *testing.T) {
	tmpDir := setupTestSkillDir(t)

	content := `invalid frontmatter
name: test_skill
---
content`
	createSkillFile(t, tmpDir, "invalid_skill", content)

	r := NewRegistry()
	if err := r.Load(tmpDir); err == nil {
		t.Error("Load() should error for invalid frontmatter")
	}
}

func TestRegistry_Load_InvalidYAML(t *testing.T) {
	tmpDir := setupTestSkillDir(t)

	content := `---
name: test_skill
description: [invalid yaml
tags:
  - tag1
tools:
  - tool1
---
content`
	createSkillFile(t, tmpDir, "invalid_yaml", content)

	r := NewRegistry()
	if err := r.Load(tmpDir); err == nil {
		t.Error("Load() should error for invalid YAML")
	}
}

func TestRegistry_Load_DuplicateNames(t *testing.T) {
	tmpDir := setupTestSkillDir(t)

	content1 := `---
name: duplicate_skill
description: First skill
tags:
  - tag1
tools:
  - tool1
---
content1`

	content2 := `---
name: duplicate_skill
description: Second skill
tags:
  - tag2
tools:
  - tool2
---
content2`

	createSkillFile(t, tmpDir, "skill1", content1)
	createSkillFile(t, tmpDir, "skill2", content2)

	r := NewRegistry()
	if err := r.Load(tmpDir); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if len(r.skills) != 1 {
		t.Errorf("loaded %d skills, want 1 (duplicate should be overwritten)", len(r.skills))
	}

	skill := r.skills["duplicate_skill"]
	if skill.Description != "Second skill" {
		t.Error("second skill should have overwritten first skill")
	}
}

func TestRegistry_Load_MultipleSkills(t *testing.T) {
	tmpDir := setupTestSkillDir(t)

	createSkillFile(t, tmpDir, "skill1", `---
name: skill1
description: First skill
tags:
  - tag1
tools:
  - tool1
---
content1`)

	createSkillFile(t, tmpDir, "skill2", `---
name: skill2
description: Second skill
tags:
  - tag2
tools:
  - tool2
---
content2`)

	createSkillFile(t, tmpDir, "skill3", `---
name: skill3
description: Third skill
tags:
  - tag3
tools:
  - tool3
---
content3`)

	r := NewRegistry()
	if err := r.Load(tmpDir); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if len(r.skills) != 3 {
		t.Errorf("loaded %d skills, want 3", len(r.skills))
	}
}

func TestRegistry_Load_SkipFiles(t *testing.T) {
	tmpDir := setupTestSkillDir(t)

	content := `---
name: test_skill
description: A test skill
tags:
  - tag1
tools:
  - tool1
---
content`

	createSkillFile(t, tmpDir, "test_skill", content)

	if err := os.WriteFile(filepath.Join(tmpDir, "not_a_skill.md"), []byte("not a skill"), 0644); err != nil {
		t.Fatal(err)
	}

	r := NewRegistry()
	if err := r.Load(tmpDir); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if len(r.skills) != 1 {
		t.Errorf("loaded %d skills, want 1 (should skip files)", len(r.skills))
	}
}

func TestRegistry_Get(t *testing.T) {
	tmpDir := setupTestSkillDir(t)

	content := `---
name: test_skill
description: A test skill
tags:
  - tag1
tools:
  - tool1
---
content`
	createSkillFile(t, tmpDir, "test_skill", content)

	r := NewRegistry()
	if err := r.Load(tmpDir); err != nil {
		t.Fatal(err)
	}

	skill, err := r.Get("test_skill")
	if err != nil {
		t.Errorf("Get() failed: %v", err)
	}
	if skill.Name != "test_skill" {
		t.Errorf("skill.Name = %v, want test_skill", skill.Name)
	}

	_, err = r.Get("nonexistent")
	if err == nil {
		t.Error("Get() should error for nonexistent skill")
	}
}

func TestRegistry_GetRelevant_EmptyQuery(t *testing.T) {
	tmpDir := setupTestSkillDir(t)

	createSkillFile(t, tmpDir, "skill1", `---
name: skill1
description: First skill
tags:
  - tag1
tools:
  - tool1
---
content1`)

	createSkillFile(t, tmpDir, "skill2", `---
name: skill2
description: Second skill
tags:
  - tag2
tools:
  - tool2
---
content2`)

	r := NewRegistry()
	if err := r.Load(tmpDir); err != nil {
		t.Fatal(err)
	}

	skills, err := r.GetRelevant("", 10)
	if err != nil {
		t.Errorf("GetRelevant() failed: %v", err)
	}
	if len(skills) != 2 {
		t.Errorf("got %d skills, want 2", len(skills))
	}
}

func TestRegistry_GetRelevant_ExactTagMatch(t *testing.T) {
	tmpDir := setupTestSkillDir(t)

	createSkillFile(t, tmpDir, "skill1", `---
name: skill1
description: First skill
tags:
  - web_scrape
tools:
  - tool1
---
content1`)

	createSkillFile(t, tmpDir, "skill2", `---
name: skill2
description: Second skill
tags:
  - data_analysis
tools:
  - tool2
---
content2`)

	r := NewRegistry()
	if err := r.Load(tmpDir); err != nil {
		t.Fatal(err)
	}

	skills, err := r.GetRelevant("web_scrape", 10)
	if err != nil {
		t.Errorf("GetRelevant() failed: %v", err)
	}
	if len(skills) != 1 {
		t.Errorf("got %d skills, want 1", len(skills))
	}
	if skills[0].Name != "skill1" {
		t.Errorf("got %v, want skill1", skills[0].Name)
	}
}

func TestRegistry_GetRelevant_PartialTagMatch(t *testing.T) {
	tmpDir := setupTestSkillDir(t)

	createSkillFile(t, tmpDir, "skill1", `---
name: skill1
description: First skill
tags:
  - web_scrape
tools:
  - tool1
---
content1`)

	createSkillFile(t, tmpDir, "skill2", `---
name: skill2
description: Second skill
tags:
  - web_query
tools:
  - tool2
---
content2`)

	r := NewRegistry()
	if err := r.Load(tmpDir); err != nil {
		t.Fatal(err)
	}

	skills, err := r.GetRelevant("web", 10)
	if err != nil {
		t.Errorf("GetRelevant() failed: %v", err)
	}
	if len(skills) != 2 {
		t.Errorf("got %d skills, want 2", len(skills))
	}
}

func TestRegistry_GetRelevant_NameMatch(t *testing.T) {
	tmpDir := setupTestSkillDir(t)

	createSkillFile(t, tmpDir, "web_scrape_skill", `---
name: web_scrape_skill
description: A skill for web scraping
tags:
  - web
tools:
  - http
---
content`)

	createSkillFile(t, tmpDir, "other_skill", `---
name: other_skill
description: Another skill
tags:
  - other
tools:
  - other
---
content`)

	r := NewRegistry()
	if err := r.Load(tmpDir); err != nil {
		t.Fatal(err)
	}

	skills, err := r.GetRelevant("web_scrape", 10)
	if err != nil {
		t.Errorf("GetRelevant() failed: %v", err)
	}
	if len(skills) != 1 {
		t.Errorf("got %d skills, want 1", len(skills))
	}
}

func TestRegistry_GetRelevant_Limit(t *testing.T) {
	tmpDir := setupTestSkillDir(t)

	createSkillFile(t, tmpDir, "skill1", `---
name: skill1
description: Skill 1
tags:
  - tag1
tools:
  - tool1
---
content1`)

	createSkillFile(t, tmpDir, "skill2", `---
name: skill2
description: Skill 2
tags:
  - tag2
tools:
  - tool2
---
content2`)

	createSkillFile(t, tmpDir, "skill3", `---
name: skill3
description: Skill 3
tags:
  - tag3
tools:
  - tool3
---
content3`)

	createSkillFile(t, tmpDir, "skill4", `---
name: skill4
description: Skill 4
tags:
  - tag4
tools:
  - tool4
---
content4`)

	createSkillFile(t, tmpDir, "skill5", `---
name: skill5
description: Skill 5
tags:
  - tag5
tools:
  - tool5
---
content5`)

	r := NewRegistry()
	if err := r.Load(tmpDir); err != nil {
		t.Fatal(err)
	}

	skills, err := r.GetRelevant("", 3)
	if err != nil {
		t.Errorf("GetRelevant() failed: %v", err)
	}
	if len(skills) != 3 {
		t.Errorf("got %d skills, want 3 (limit)", len(skills))
	}
}

func TestRegistry_List(t *testing.T) {
	tmpDir := setupTestSkillDir(t)

	createSkillFile(t, tmpDir, "zebra", `---
name: zebra
description: Zebra skill
tags:
  - zebra
tools:
  - z
---
content`)

	createSkillFile(t, tmpDir, "alpha", `---
name: alpha
description: Alpha skill
tags:
  - alpha
tools:
  - a
---
content`)

	createSkillFile(t, tmpDir, "beta", `---
name: beta
description: Beta skill
tags:
  - beta
tools:
  - b
---
content`)

	r := NewRegistry()
	if err := r.Load(tmpDir); err != nil {
		t.Fatal(err)
	}

	names, err := r.List("name")
	if err != nil {
		t.Errorf("List() failed: %v", err)
	}
	if len(names) != 3 {
		t.Errorf("got %d names, want 3", len(names))
	}

	if names[0] != "alpha" || names[1] != "beta" || names[2] != "zebra" {
		t.Errorf("names = %v, want [alpha beta zebra]", names)
	}
}

func TestRegistry_List_Empty(t *testing.T) {
	r := NewRegistry()
	names, err := r.List("name")
	if err != nil {
		t.Errorf("List() failed: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("got %d names, want 0", len(names))
	}
}

func TestRegistry_List_Desc(t *testing.T) {
	tmpDir := setupTestSkillDir(t)

	createSkillFile(t, tmpDir, "alpha", `---
name: alpha
description: Alpha skill
tags:
  - alpha
tools:
  - a
---
content`)

	createSkillFile(t, tmpDir, "zebra", `---
name: zebra
description: Zebra skill
tags:
  - zebra
tools:
  - z
---
content`)

	r := NewRegistry()
	if err := r.Load(tmpDir); err != nil {
		t.Fatal(err)
	}

	names, err := r.List("name-desc")
	if err != nil {
		t.Errorf("List() failed: %v", err)
	}

	if names[0] != "zebra" || names[1] != "alpha" {
		t.Errorf("names = %v, want [zebra alpha]", names)
	}
}

func TestRegistry_Stats(t *testing.T) {
	tmpDir := setupTestSkillDir(t)

	createSkillFile(t, tmpDir, "skill1", `---
name: skill1
description: First skill
tags:
  - web_scrape
  - data_analysis
tools:
  - tool1
---
content1`)

	createSkillFile(t, tmpDir, "skill2", `---
name: skill2
description: Second skill
tags:
  - web_scrape
  - file_io
tools:
  - tool2
---
content2`)

	r := NewRegistry()
	if err := r.Load(tmpDir); err != nil {
		t.Fatal(err)
	}

	stats := r.Stats()
	if stats.TotalSkills != 2 {
		t.Errorf("TotalSkills = %d, want 2", stats.TotalSkills)
	}

	if stats.SkillsByTag["web_scrape"] != 2 {
		t.Errorf("web_scrape count = %d, want 2", stats.SkillsByTag["web_scrape"])
	}
	if stats.SkillsByTag["data_analysis"] != 1 {
		t.Errorf("data_analysis count = %d, want 1", stats.SkillsByTag["data_analysis"])
	}
}

func TestRegistry_Reload(t *testing.T) {
	tmpDir := setupTestSkillDir(t)

	createSkillFile(t, tmpDir, "skill1", `---
name: skill1
description: First skill
tags:
  - tag1
tools:
  - tool1
---
content1`)

	r := NewRegistryWithPath(tmpDir)
	if err := r.Load(tmpDir); err != nil {
		t.Fatal(err)
	}

	if len(r.skills) != 1 {
		t.Fatalf("loaded %d skills, want 1", len(r.skills))
	}

	createSkillFile(t, tmpDir, "skill2", `---
name: skill2
description: Second skill
tags:
  - tag2
tools:
  - tool2
---
content2`)

	if err := r.Reload(); err != nil {
		t.Errorf("Reload() failed: %v", err)
	}

	if len(r.skills) != 2 {
		t.Errorf("after reload, loaded %d skills, want 2", len(r.skills))
	}

	_, ok := r.skills["skill2"]
	if !ok {
		t.Error("skill2 should be loaded after reload")
	}
}

func TestRegistry_Reload_NoPath(t *testing.T) {
	r := NewRegistry()
	if err := r.Reload(); err == nil {
		t.Error("Reload() should error when no previous path")
	}
}

func TestRegistry_Validate_EmptyName(t *testing.T) {
	r := NewRegistry()
	skill := &Skill{
		Name:        "",
		Description: "description",
		Tags:        []string{"tag1"},
	}

	err := r.Validate(skill)
	if err == nil {
		t.Error("Validate() should error for empty name")
	}
	if !strings.Contains(err.Error(), "name") {
		t.Errorf("error should mention 'name', got: %v", err)
	}
}

func TestRegistry_Validate_InvalidName(t *testing.T) {
	r := NewRegistry()
	skill := &Skill{
		Name:        "invalid name!",
		Description: "description",
		Tags:        []string{"tag1"},
	}

	err := r.Validate(skill)
	if err == nil {
		t.Error("Validate() should error for invalid name")
	}
}

func TestRegistry_Validate_EmptyDescription(t *testing.T) {
	r := NewRegistry()
	skill := &Skill{
		Name:        "valid_name",
		Description: "",
		Tags:        []string{"tag1"},
	}

	err := r.Validate(skill)
	if err == nil {
		t.Error("Validate() should error for empty description")
	}
}

func TestRegistry_Validate_EmptyTags(t *testing.T) {
	r := NewRegistry()
	skill := &Skill{
		Name:        "valid_name",
		Description: "description",
		Tags:        []string{},
	}

	err := r.Validate(skill)
	if err == nil {
		t.Error("Validate() should error for empty tags")
	}
}

func TestRegistry_Validate_EmptyTag(t *testing.T) {
	r := NewRegistry()
	skill := &Skill{
		Name:        "valid_name",
		Description: "description",
		Tags:        []string{"", "tag2"},
	}

	err := r.Validate(skill)
	if err == nil {
		t.Error("Validate() should error for empty tag")
	}
}

func TestRegistry_Validate_Valid(t *testing.T) {
	r := NewRegistry()
	skill := &Skill{
		Name:        "valid_name-123",
		Description: "A valid skill",
		Tags:        []string{"tag1", "tag2"},
	}

	err := r.Validate(skill)
	if err != nil {
		t.Errorf("Validate() should not error for valid skill, got: %v", err)
	}
}

func TestFrontmatterParser_Parse_Valid(t *testing.T) {
	fp := &FrontmatterParser{}

	content := []byte(`---
name: test
description: desc
tags:
  - tag1
tools:
  - tool1
---
body content`)

	frontmatter, body, err := fp.Parse(content)
	if err != nil {
		t.Errorf("Parse() failed: %v", err)
	}

	if !strings.Contains(frontmatter, "name: test") {
		t.Errorf("frontmatter missing name")
	}

	if body != "body content" {
		t.Errorf("body = %v, want 'body content'", body)
	}
}

func TestFrontmatterParser_Parse_NoFrontmatter(t *testing.T) {
	fp := &FrontmatterParser{}

	content := []byte(`no frontmatter here`)

	_, _, err := fp.Parse(content)
	if err == nil {
		t.Error("Parse() should error for missing frontmatter")
	}
}

func TestFrontmatterParser_Parse_EmptyFrontmatter(t *testing.T) {
	fp := &FrontmatterParser{}

	content := []byte(`---
---
body`)

	_, _, err := fp.Parse(content)
	if err == nil {
		t.Error("Parse() should error for empty frontmatter")
	}
}

func TestFrontmatterParser_Parse_WithTrailingWhitespace(t *testing.T) {
	fp := &FrontmatterParser{}

	content := []byte(`---
name: test
description: desc
tags:
  - tag1
tools:
  - tool1
---
body content   `)

	frontmatter, body, err := fp.Parse(content)
	if err != nil {
		t.Errorf("Parse() failed: %v", err)
	}

	if !strings.Contains(frontmatter, "name: test") {
		t.Errorf("frontmatter missing name")
	}

	if body != "body content" {
		t.Errorf("body = %v, want 'body content'", body)
	}
}

func TestSkillLoadError_Error(t *testing.T) {
	err := &SkillLoadError{
		Path:    "/path/to/skill",
		Message: "load failed",
		Cause:   nil,
	}

	msg := err.Error()
	if !strings.Contains(msg, "/path/to/skill") {
		t.Errorf("error message should contain path, got: %v", msg)
	}
	if !strings.Contains(msg, "load failed") {
		t.Errorf("error message should contain message, got: %v", msg)
	}
}

func TestSkillValidationError_Error(t *testing.T) {
	err := &SkillValidationError{
		Field:   "name",
		Message: "cannot be empty",
	}

	msg := err.Error()
	if !strings.Contains(msg, "name") {
		t.Errorf("error message should contain field, got: %v", msg)
	}
	if !strings.Contains(msg, "cannot be empty") {
		t.Errorf("error message should contain message, got: %v", msg)
	}
}
