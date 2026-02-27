package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/harunnryd/heike/internal/skill/domain"
	"github.com/harunnryd/heike/internal/skill/repository"
)

func setupTestService(t *testing.T) (SkillService, *repository.FileSkillRepository, string, func()) {
	tmpDir := t.TempDir()
	repo := repository.NewFileSkillRepository(tmpDir)
	service := NewSkillService(repo)

	cleanup := func() {
		t.Cleanup(func() {})
	}

	return service, repo, tmpDir, cleanup
}

func createTestSkill(t *testing.T, name string) *domain.Skill {
	return &domain.Skill{
		ID:          domain.SkillID(name),
		Name:        name,
		Description: "Test skill",
		Tags:        []domain.Tag{"test"},
		Tools:       []domain.ToolRef{"tool1"},
		Content:     "Test content",
		Version:     "1.0.0",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func TestNewSkillService(t *testing.T) {
	service, repo, _, cleanup := setupTestService(t)
	defer cleanup()

	if service == nil {
		t.Error("NewSkillService() returned nil")
	}
	if repo == nil {
		t.Error("repo is nil")
	}
}

func TestSkillServiceImpl_LoadSkills(t *testing.T) {
	service, _, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	if err := service.LoadSkills(ctx); err != nil {
		t.Fatalf("LoadSkills() failed: %v", err)
	}

	skills, err := service.ListSkills(ctx, repository.SkillFilter{})
	if err != nil {
		t.Fatalf("ListSkills() failed: %v", err)
	}

	if len(skills) > 0 {
		t.Error("should not load any skills from empty directory")
	}
}

func TestSkillServiceImpl_GetSkill(t *testing.T) {
	service, repo, _, cleanup := setupTestService(t)
	defer cleanup()

	testSkill := createTestSkill(t, "test_skill")

	ctx := context.Background()
	if err := repo.Store(ctx, testSkill); err != nil {
		t.Fatal(err)
	}
	if err := service.LoadSkills(ctx); err != nil {
		t.Fatal(err)
	}

	skill, err := service.GetSkill(ctx, testSkill.ID)
	if err != nil {
		t.Errorf("GetSkill() failed: %v", err)
	}
	if skill.Name != testSkill.Name {
		t.Errorf("skill.Name = %v, want %v", skill.Name, testSkill.Name)
	}
}

func TestSkillServiceImpl_GetSkill_InvalidID(t *testing.T) {
	service, _, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	_, err := service.GetSkill(ctx, domain.SkillID(""))
	if err == nil {
		t.Error("GetSkill() should error for invalid ID")
	}
}

func TestSkillServiceImpl_SearchSkills_EmptyQuery(t *testing.T) {
	service, _, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	skills, err := service.SearchSkills(ctx, "", 10)
	if err != nil {
		t.Errorf("SearchSkills() failed: %v", err)
	}
	if len(skills) > 0 {
		t.Error("should return empty list for empty query")
	}
}

func TestSkillServiceImpl_SearchSkills_WithQuery(t *testing.T) {
	service, repo, _, cleanup := setupTestService(t)
	defer cleanup()

	skill1 := createTestSkill(t, "web_scrape_skill")
	skill1.Tags = []domain.Tag{"web_scrape", "data_analysis"}

	skill2 := createTestSkill(t, "other_skill")
	skill2.Tags = []domain.Tag{"other"}

	ctx := context.Background()
	if err := repo.Store(ctx, skill1); err != nil {
		t.Fatal(err)
	}
	if err := repo.Store(ctx, skill2); err != nil {
		t.Fatal(err)
	}
	if err := service.LoadSkills(ctx); err != nil {
		t.Fatal(err)
	}

	skills, err := service.SearchSkills(ctx, "web", 10)
	if err != nil {
		t.Errorf("SearchSkills() failed: %v", err)
	}
	if len(skills) != 1 {
		t.Errorf("got %d skills, want 1", len(skills))
	}
	if skills[0].Name != "web_scrape_skill" {
		t.Errorf("first skill = %v, want web_scrape_skill", skills[0].Name)
	}
}

func TestSkillServiceImpl_SearchSkills_WithLimit(t *testing.T) {
	service, repo, _, cleanup := setupTestService(t)
	defer cleanup()

	for i := 1; i <= 5; i++ {
		skill := createTestSkill(t, "skill"+string(rune('0'+i)))
		ctx := context.Background()
		if err := repo.Store(ctx, skill); err != nil {
			t.Fatal(err)
		}
	}

	ctx := context.Background()
	if err := service.LoadSkills(ctx); err != nil {
		t.Fatal(err)
	}

	skills, err := service.SearchSkills(ctx, "", 3)
	if err != nil {
		t.Errorf("SearchSkills() failed: %v", err)
	}
	if len(skills) != 3 {
		t.Errorf("got %d skills, want 3 (limit)", len(skills))
	}
}

func TestSkillServiceImpl_ListSkills(t *testing.T) {
	service, repo, _, cleanup := setupTestService(t)
	defer cleanup()

	skill1 := createTestSkill(t, "zebra")
	skill2 := createTestSkill(t, "alpha")
	skill3 := createTestSkill(t, "beta")

	ctx := context.Background()
	if err := repo.Store(ctx, skill1); err != nil {
		t.Fatal(err)
	}
	if err := repo.Store(ctx, skill2); err != nil {
		t.Fatal(err)
	}
	if err := repo.Store(ctx, skill3); err != nil {
		t.Fatal(err)
	}
	if err := service.LoadSkills(ctx); err != nil {
		t.Fatal(err)
	}

	filter := repository.SkillFilter{
		SortBy:    "name",
		SortOrder: "asc",
	}

	skills, err := service.ListSkills(ctx, filter)
	if err != nil {
		t.Errorf("ListSkills() failed: %v", err)
	}
	if len(skills) != 3 {
		t.Errorf("got %d skills, want 3", len(skills))
	}
	if skills[0].Name != "alpha" {
		t.Errorf("first skill = %v, want alpha", skills[0].Name)
	}
}

func TestSkillServiceImpl_ValidateSkill(t *testing.T) {
	service, _, _, cleanup := setupTestService(t)
	defer cleanup()

	skill := createTestSkill(t, "valid_skill")

	ctx := context.Background()
	if err := service.ValidateSkill(ctx, skill); err != nil {
		t.Errorf("ValidateSkill() failed: %v", err)
	}

	if skill.ID == "" {
		t.Error("ID should be set after validation")
	}
	if skill.ID != domain.SkillID("valid_skill") {
		t.Errorf("ID = %v, want valid_skill", skill.ID)
	}
}

func TestSkillServiceImpl_ValidateSkill_Invalid(t *testing.T) {
	service, _, _, cleanup := setupTestService(t)
	defer cleanup()

	skill := createTestSkill(t, "")
	skill.Name = "invalid name!"

	ctx := context.Background()
	err := service.ValidateSkill(ctx, skill)
	if err == nil {
		t.Error("ValidateSkill() should error for invalid skill")
	}
}

func TestSkillServiceImpl_ValidateSkill_Nil(t *testing.T) {
	service, _, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	err := service.ValidateSkill(ctx, nil)
	if err == nil {
		t.Error("ValidateSkill() should error for nil skill")
	}
}

func TestSkillServiceImpl_InstallSkill(t *testing.T) {
	service, repo, _, cleanup := setupTestService(t)
	defer cleanup()

	skill := createTestSkill(t, "new_skill")

	ctx := context.Background()
	if err := service.InstallSkill(ctx, skill); err != nil {
		t.Errorf("InstallSkill() failed: %v", err)
	}

	if err := service.LoadSkills(ctx); err != nil {
		t.Fatal(err)
	}

	exists, err := repo.Exists(ctx, skill.ID)
	if err != nil {
		t.Errorf("Exists() failed: %v", err)
	}
	if !exists {
		t.Error("skill should exist after installation")
	}
}

func TestSkillServiceImpl_InstallSkill_Duplicate(t *testing.T) {
	service, _, _, cleanup := setupTestService(t)
	defer cleanup()

	skill := createTestSkill(t, "duplicate_skill")

	ctx := context.Background()
	if err := service.InstallSkill(ctx, skill); err != nil {
		t.Fatal(err)
	}

	err := service.InstallSkill(ctx, skill)
	if err == nil {
		t.Error("InstallSkill() should error for duplicate skill")
	}

	var existsErr *domain.SkillExistsError
	if !errors.As(err, &existsErr) {
		t.Errorf("expected SkillExistsError, got %T", err)
	}
}

func TestSkillServiceImpl_UninstallSkill(t *testing.T) {
	service, repo, _, cleanup := setupTestService(t)
	defer cleanup()

	skill := createTestSkill(t, "test_skill")

	ctx := context.Background()
	if err := service.InstallSkill(ctx, skill); err != nil {
		t.Fatal(err)
	}

	if err := service.UninstallSkill(ctx, skill.ID); err != nil {
		t.Errorf("UninstallSkill() failed: %v", err)
	}

	exists, err := repo.Exists(ctx, skill.ID)
	if err != nil {
		t.Errorf("Exists() failed: %v", err)
	}
	if exists {
		t.Error("skill should not exist after uninstallation")
	}
}

func TestSkillServiceImpl_UninstallSkill_NotFound(t *testing.T) {
	service, _, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	err := service.UninstallSkill(ctx, domain.SkillID("nonexistent"))
	if err == nil {
		t.Error("UninstallSkill() should error for nonexistent skill")
	}

	var notFoundErr *domain.SkillNotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Errorf("expected SkillNotFoundError, got %T", err)
	}
}
