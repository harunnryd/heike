package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/harunnryd/heike/internal/skill/domain"
	"github.com/harunnryd/heike/internal/skill/repository"
)

func TestNewSkillWorkspaceManager(t *testing.T) {
	manager := NewSkillWorkspaceManager()

	if manager == nil {
		t.Error("NewSkillWorkspaceManager() returned nil")
	}
}

func TestDefaultSkillWorkspaceManager_GetGlobalSkillsRepo(t *testing.T) {
	manager := &DefaultSkillWorkspaceManager{
		workspaceRepos: make(map[string]repository.SkillRepository),
	}

	ctx := context.Background()
	repo, err := manager.GetGlobalSkillsRepo(ctx)
	if err != nil {
		t.Fatalf("GetGlobalSkillsRepo() failed: %v", err)
	}

	if repo == nil {
		t.Error("GetGlobalSkillsRepo() returned nil repository")
	}

	if manager.globalRepo != repo {
		t.Error("GetGlobalSkillsRepo() should cache the global repo")
	}
}

func TestDefaultSkillWorkspaceManager_GetWorkspaceSkillsRepo(t *testing.T) {
	manager := &DefaultSkillWorkspaceManager{
		workspaceRepos: make(map[string]repository.SkillRepository),
	}

	ctx := context.Background()
	repo, err := manager.GetWorkspaceSkillsRepo(ctx, "test_workspace")
	if err != nil {
		t.Fatalf("GetWorkspaceSkillsRepo() failed: %v", err)
	}

	if repo == nil {
		t.Error("GetWorkspaceSkillsRepo() returned nil repository")
	}

	cachedRepo, err := manager.GetWorkspaceSkillsRepo(ctx, "test_workspace")
	if err != nil {
		t.Fatalf("GetWorkspaceSkillsRepo() second call failed: %v", err)
	}

	if cachedRepo != repo {
		t.Error("GetWorkspaceSkillsRepo() should cache workspace repos")
	}
}

func TestDefaultSkillWorkspaceManager_LoadSkills(t *testing.T) {
	manager := &DefaultSkillWorkspaceManager{
		workspaceRepos: make(map[string]repository.SkillRepository),
	}

	ctx := context.Background()
	if err := manager.LoadSkills(ctx, "test_workspace"); err != nil {
		t.Fatalf("LoadSkills() failed: %v", err)
	}

	if manager.globalRepo == nil {
		t.Error("LoadSkills() should initialize global repo")
	}

	if _, exists := manager.workspaceRepos["test_workspace"]; !exists {
		t.Error("LoadSkills() should initialize workspace repo")
	}
}

func TestDefaultSkillWorkspaceManager_InstallSkill(t *testing.T) {
	manager := &DefaultSkillWorkspaceManager{
		workspaceRepos: make(map[string]repository.SkillRepository),
	}

	skill := &domain.Skill{
		ID:          domain.SkillID("test_skill"),
		Name:        "test_skill",
		Description: "Test skill",
		Tags:        []domain.Tag{"test"},
	}

	ctx := context.Background()
	if err := manager.InstallSkill(ctx, "test_workspace", skill); err != nil {
		t.Fatalf("InstallSkill() failed: %v", err)
	}

	if _, exists := manager.workspaceRepos["test_workspace"]; !exists {
		t.Error("InstallSkill() should initialize workspace repo")
	}

	repo := manager.workspaceRepos["test_workspace"]
	retrievedSkill, err := repo.Get(ctx, skill.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve installed skill: %v", err)
	}

	if retrievedSkill.Name != skill.Name {
		t.Errorf("Retrieved skill name = %v, want %v", retrievedSkill.Name, skill.Name)
	}
}

func TestDefaultSkillWorkspaceManager_GetSkill_Workspace(t *testing.T) {
	manager := &DefaultSkillWorkspaceManager{
		workspaceRepos: make(map[string]repository.SkillRepository),
	}

	skill := &domain.Skill{
		ID:          domain.SkillID("workspace_skill"),
		Name:        "workspace_skill",
		Description: "Workspace skill",
		Tags:        []domain.Tag{"workspace"},
	}

	ctx := context.Background()
	if err := manager.InstallSkill(ctx, "test_workspace", skill); err != nil {
		t.Fatal(err)
	}

	retrievedSkill, err := manager.GetSkill(ctx, "test_workspace", skill.ID)
	if err != nil {
		t.Fatalf("GetSkill() failed: %v", err)
	}

	if retrievedSkill.Name != skill.Name {
		t.Errorf("Retrieved skill name = %v, want %v", retrievedSkill.Name, skill.Name)
	}
}

func TestDefaultSkillWorkspaceManager_SearchSkills(t *testing.T) {
	manager := &DefaultSkillWorkspaceManager{
		workspaceRepos: make(map[string]repository.SkillRepository),
	}

	workspaceSkill := &domain.Skill{
		ID:          domain.SkillID("web_scrape_skill"),
		Name:        "web_scrape_skill",
		Description: "Web scrape skill",
		Tags:        []domain.Tag{"web_scrape", "data_analysis"},
	}

	ctx := context.Background()
	if err := manager.InstallSkill(ctx, "test_workspace", workspaceSkill); err != nil {
		t.Fatal(err)
	}

	skills, err := manager.SearchSkills(ctx, "test_workspace", "web", 10)
	if err != nil {
		t.Fatalf("SearchSkills() failed: %v", err)
	}

	if len(skills) == 0 {
		t.Error("SearchSkills() should find skills")
	}

	if len(skills) > 0 && skills[0].Name != "web_scrape_skill" {
		t.Errorf("First skill name = %v, want web_scrape_skill", skills[0].Name)
	}
}

func TestDefaultSkillWorkspaceManager_ListSkills(t *testing.T) {
	manager := &DefaultSkillWorkspaceManager{
		workspaceRepos: make(map[string]repository.SkillRepository),
	}

	skill1 := &domain.Skill{
		ID:          domain.SkillID("alpha_workspace_test"),
		Name:        "alpha_workspace_test",
		Description: "Alpha skill",
		Tags:        []domain.Tag{"test"},
	}

	skill2 := &domain.Skill{
		ID:          domain.SkillID("beta_workspace_test"),
		Name:        "beta_workspace_test",
		Description: "Beta skill",
		Tags:        []domain.Tag{"test"},
	}

	ctx := context.Background()
	if err := manager.InstallSkill(ctx, "test_workspace", skill1); err != nil {
		t.Fatal(err)
	}
	if err := manager.InstallSkill(ctx, "test_workspace", skill2); err != nil {
		t.Fatal(err)
	}

	filter := repository.SkillFilter{
		SortBy:    "name",
		SortOrder: "asc",
	}

	skills, err := manager.ListSkills(ctx, "test_workspace", filter)
	if err != nil {
		t.Fatalf("ListSkills() failed: %v", err)
	}

	if len(skills) < 2 {
		t.Errorf("ListSkills() returned %d skills, want at least 2", len(skills))
	}

	workspaceSkills := filterWorkspaceSkills(skills, "test_workspace")
	if len(workspaceSkills) < 2 {
		t.Errorf("Workspace should have at least 2 skills, got %d", len(workspaceSkills))
	}
}

func TestDefaultSkillWorkspaceManager_UninstallSkill(t *testing.T) {
	manager := &DefaultSkillWorkspaceManager{
		workspaceRepos: make(map[string]repository.SkillRepository),
	}

	skill := &domain.Skill{
		ID:          domain.SkillID("test_skill"),
		Name:        "test_skill",
		Description: "Test skill",
		Tags:        []domain.Tag{"test"},
	}

	ctx := context.Background()
	if err := manager.InstallSkill(ctx, "test_workspace", skill); err != nil {
		t.Fatal(err)
	}

	if err := manager.UninstallSkill(ctx, "test_workspace", skill.ID); err != nil {
		t.Fatalf("UninstallSkill() failed: %v", err)
	}

	_, err := manager.GetSkill(ctx, "test_workspace", skill.ID)
	if err == nil {
		t.Error("UninstallSkill() should remove skill from workspace")
	}
}

func filterWorkspaceSkills(skills []*domain.Skill, workspaceID string) []*domain.Skill {
	var filtered []*domain.Skill
	for _, skill := range skills {
		if strings.Contains(skill.Name, "workspace_test") {
			filtered = append(filtered, skill)
		}
	}
	return filtered
}
