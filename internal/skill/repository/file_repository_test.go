package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/harunnryd/heike/internal/skill/domain"
)

func setupTestRepository(t *testing.T) (*FileSkillRepository, string, func()) {
	tmpDir := t.TempDir()
	repo := NewFileSkillRepository(tmpDir)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return repo, tmpDir, cleanup
}

func createTestSkill(t *testing.T, basePath, name string) *domain.Skill {
	skill := &domain.Skill{
		ID:          domain.SkillID(name),
		Name:        name,
		Description: "Test skill description",
		Tags:        []domain.Tag{"test"},
		Tools:       []domain.ToolRef{"tool1"},
		Content:     "Test skill content",
		Version:     "1.0.0",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	skillDir := filepath.Join(basePath, name)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	skillPath := filepath.Join(skillDir, "SKILL.md")
	content := `---
name: ` + name + `
description: Test skill description
tags:
  - test
tools:
  - tool1
---
Test skill content`
	if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	return skill
}

func TestNewFileSkillRepository(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	if repo == nil {
		t.Error("NewFileSkillRepository() returned nil")
	}
	if repo.basePath == "" {
		t.Error("basePath is empty")
	}
}

func TestFileSkillRepository_LoadAll(t *testing.T) {
	repo, tmpDir, cleanup := setupTestRepository(t)
	defer cleanup()

	createTestSkill(t, tmpDir, "skill1")
	createTestSkill(t, tmpDir, "skill2")

	ctx := context.Background()
	if err := repo.LoadAll(ctx); err != nil {
		t.Fatalf("LoadAll() failed: %v", err)
	}

	skills, err := repo.registry.List("name")
	if err != nil {
		t.Fatalf("registry.List() failed: %v", err)
	}

	if len(skills) != 2 {
		t.Errorf("loaded %d skills, want 2", len(skills))
	}
}

func TestFileSkillRepository_Get(t *testing.T) {
	repo, tmpDir, cleanup := setupTestRepository(t)
	defer cleanup()

	testSkill := createTestSkill(t, tmpDir, "test_skill")

	ctx := context.Background()
	if err := repo.LoadAll(ctx); err != nil {
		t.Fatal(err)
	}

	skill, err := repo.Get(ctx, testSkill.ID)
	if err != nil {
		t.Errorf("Get() failed: %v", err)
	}
	if skill.Name != testSkill.Name {
		t.Errorf("skill.Name = %v, want %v", skill.Name, testSkill.Name)
	}
}

func TestFileSkillRepository_Get_NotFound(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx := context.Background()
	_, err := repo.Get(ctx, domain.SkillID("nonexistent"))
	if err == nil {
		t.Error("Get() should error for nonexistent skill")
	}
}

func TestFileSkillRepository_List(t *testing.T) {
	repo, tmpDir, cleanup := setupTestRepository(t)
	defer cleanup()

	createTestSkill(t, tmpDir, "zebra")
	createTestSkill(t, tmpDir, "alpha")
	createTestSkill(t, tmpDir, "beta")

	ctx := context.Background()
	if err := repo.LoadAll(ctx); err != nil {
		t.Fatal(err)
	}

	filter := SkillFilter{
		SortBy:    "name",
		SortOrder: "asc",
	}

	skills, err := repo.List(ctx, filter)
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if len(skills) != 3 {
		t.Errorf("got %d skills, want 3", len(skills))
	}

	if skills[0].Name != "alpha" {
		t.Errorf("first skill = %v, want alpha", skills[0].Name)
	}
	if skills[1].Name != "beta" {
		t.Errorf("second skill = %v, want beta", skills[1].Name)
	}
	if skills[2].Name != "zebra" {
		t.Errorf("third skill = %v, want zebra", skills[2].Name)
	}
}

func TestFileSkillRepository_List_WithLimit(t *testing.T) {
	repo, tmpDir, cleanup := setupTestRepository(t)
	defer cleanup()

	for i := 1; i <= 5; i++ {
		createTestSkill(t, tmpDir, "skill"+string(rune('0'+i)))
	}

	ctx := context.Background()
	if err := repo.LoadAll(ctx); err != nil {
		t.Fatal(err)
	}

	filter := SkillFilter{
		Limit: 3,
	}

	skills, err := repo.List(ctx, filter)
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if len(skills) != 3 {
		t.Errorf("got %d skills, want 3 (limit)", len(skills))
	}
}

func TestFileSkillRepository_List_WithOffset(t *testing.T) {
	repo, tmpDir, cleanup := setupTestRepository(t)
	defer cleanup()

	for i := 1; i <= 5; i++ {
		createTestSkill(t, tmpDir, "skill"+string(rune('0'+i)))
	}

	ctx := context.Background()
	if err := repo.LoadAll(ctx); err != nil {
		t.Fatal(err)
	}

	filter := SkillFilter{
		Offset: 2,
	}

	skills, err := repo.List(ctx, filter)
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if len(skills) != 3 {
		t.Errorf("got %d skills, want 3 (5 - 2 offset)", len(skills))
	}
}

func TestFileSkillRepository_List_WithTagFilter(t *testing.T) {
	repo, tmpDir, cleanup := setupTestRepository(t)
	defer cleanup()

	createTestSkill(t, tmpDir, "skill1")
	createTestSkill(t, tmpDir, "skill2")

	ctx := context.Background()
	if err := repo.LoadAll(ctx); err != nil {
		t.Fatal(err)
	}

	filter := SkillFilter{
		Tags: []domain.Tag{"test"},
	}

	skills, err := repo.List(ctx, filter)
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if len(skills) != 2 {
		t.Errorf("got %d skills, want 2 (both have 'test' tag)", len(skills))
	}
}

func TestFileSkillRepository_List_WithToolFilter(t *testing.T) {
	repo, tmpDir, cleanup := setupTestRepository(t)
	defer cleanup()

	createTestSkill(t, tmpDir, "skill1")

	ctx := context.Background()
	if err := repo.LoadAll(ctx); err != nil {
		t.Fatal(err)
	}

	filter := SkillFilter{
		Tools: []domain.ToolRef{"tool1"},
	}

	skills, err := repo.List(ctx, filter)
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if len(skills) != 1 {
		t.Errorf("got %d skills, want 1 (has 'tool1')", len(skills))
	}
}

func TestFileSkillRepository_Exists(t *testing.T) {
	repo, tmpDir, cleanup := setupTestRepository(t)
	defer cleanup()

	testSkill := createTestSkill(t, tmpDir, "test_skill")

	ctx := context.Background()
	if err := repo.LoadAll(ctx); err != nil {
		t.Fatal(err)
	}

	exists, err := repo.Exists(ctx, testSkill.ID)
	if err != nil {
		t.Errorf("Exists() failed: %v", err)
	}
	if !exists {
		t.Error("Exists() should return true for existing skill")
	}

	exists, err = repo.Exists(ctx, domain.SkillID("nonexistent"))
	if err != nil {
		t.Errorf("Exists() failed: %v", err)
	}
	if exists {
		t.Error("Exists() should return false for nonexistent skill")
	}
}

func TestFileSkillRepository_Store(t *testing.T) {
	repo, tmpDir, cleanup := setupTestRepository(t)
	defer cleanup()

	skill := &domain.Skill{
		ID:          domain.SkillID("new_skill"),
		Name:        "new_skill",
		Description: "A new skill",
		Tags:        []domain.Tag{"new"},
		Tools:       []domain.ToolRef{"new_tool"},
		Content:     "New skill content",
		Version:     "1.0.0",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	ctx := context.Background()
	if err := repo.Store(ctx, skill); err != nil {
		t.Fatalf("Store() failed: %v", err)
	}

	skillPath := filepath.Join(tmpDir, "new_skill", "SKILL.md")
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		t.Error("Store() should create skill file")
	}

	exists, err := repo.Exists(ctx, skill.ID)
	if err != nil {
		t.Errorf("Exists() failed: %v", err)
	}
	if !exists {
		t.Error("Exists() should return true after Store()")
	}
}

func TestFileSkillRepository_Delete(t *testing.T) {
	repo, tmpDir, cleanup := setupTestRepository(t)
	defer cleanup()

	testSkill := createTestSkill(t, tmpDir, "test_skill")

	ctx := context.Background()
	if err := repo.LoadAll(ctx); err != nil {
		t.Fatal(err)
	}

	if err := repo.Delete(ctx, testSkill.ID); err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	skillPath := filepath.Join(tmpDir, "test_skill")
	if _, err := os.Stat(skillPath); !os.IsNotExist(err) {
		t.Error("Delete() should remove skill directory")
	}

	exists, err := repo.Exists(ctx, testSkill.ID)
	if err != nil {
		t.Errorf("Exists() failed: %v", err)
	}
	if exists {
		t.Error("Exists() should return false after Delete()")
	}
}

func TestFileSkillRepository_Delete_NotFound(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx := context.Background()
	err := repo.Delete(ctx, domain.SkillID("nonexistent"))
	if err == nil {
		t.Error("Delete() should error for nonexistent skill")
	}
}
