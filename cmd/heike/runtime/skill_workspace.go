package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/harunnryd/heike/internal/skill/domain"
	"github.com/harunnryd/heike/internal/skill/repository"
	"github.com/harunnryd/heike/internal/store"
)

type SkillWorkspaceManager interface {
	GetGlobalSkillsRepo(ctx context.Context) (repository.SkillRepository, error)
	GetWorkspaceSkillsRepo(ctx context.Context, workspaceID string) (repository.SkillRepository, error)
	LoadSkills(ctx context.Context, workspaceID string) error
	GetSkill(ctx context.Context, workspaceID string, id domain.SkillID) (*domain.Skill, error)
	SearchSkills(ctx context.Context, workspaceID string, query string, limit int) ([]*domain.Skill, error)
	ListSkills(ctx context.Context, workspaceID string, filter repository.SkillFilter) ([]*domain.Skill, error)
	InstallSkill(ctx context.Context, workspaceID string, skill *domain.Skill) error
	UninstallSkill(ctx context.Context, workspaceID string, id domain.SkillID) error
}

type DefaultSkillWorkspaceManager struct {
	globalRepo     repository.SkillRepository
	workspaceRepos map[string]repository.SkillRepository
}

func NewSkillWorkspaceManager() SkillWorkspaceManager {
	return &DefaultSkillWorkspaceManager{
		workspaceRepos: make(map[string]repository.SkillRepository),
	}
}

func (m *DefaultSkillWorkspaceManager) GetGlobalSkillsRepo(ctx context.Context) (repository.SkillRepository, error) {
	if m.globalRepo == nil {
		globalSkillsDir, err := store.GetSkillsDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get global skills directory: %w", err)
		}
		m.globalRepo = repository.NewFileSkillRepository(globalSkillsDir)
	}

	if err := m.globalRepo.LoadAll(ctx); err != nil {
		return nil, fmt.Errorf("failed to load global skills: %w", err)
	}

	return m.globalRepo, nil
}

func (m *DefaultSkillWorkspaceManager) GetWorkspaceSkillsRepo(ctx context.Context, workspaceID string) (repository.SkillRepository, error) {
	if repo, exists := m.workspaceRepos[workspaceID]; exists {
		return repo, nil
	}

	workspaceSkillsDir, err := store.GetWorkspaceSkillsDir(workspaceID, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace skills directory: %w", err)
	}

	repo := repository.NewFileSkillRepository(workspaceSkillsDir)
	m.workspaceRepos[workspaceID] = repo

	if err := repo.LoadAll(ctx); err != nil {
		return nil, fmt.Errorf("failed to load workspace skills: %w", err)
	}

	return repo, nil
}

func (m *DefaultSkillWorkspaceManager) LoadSkills(ctx context.Context, workspaceID string) error {
	globalRepo, err := m.GetGlobalSkillsRepo(ctx)
	if err != nil {
		return err
	}

	workspaceRepo, err := m.GetWorkspaceSkillsRepo(ctx, workspaceID)
	if err != nil {
		return err
	}

	if err := globalRepo.LoadAll(ctx); err != nil {
		return fmt.Errorf("failed to load global skills: %w", err)
	}

	if err := workspaceRepo.LoadAll(ctx); err != nil {
		return fmt.Errorf("failed to load workspace skills: %w", err)
	}

	return nil
}

func (m *DefaultSkillWorkspaceManager) GetSkill(ctx context.Context, workspaceID string, id domain.SkillID) (*domain.Skill, error) {
	workspaceRepo, err := m.GetWorkspaceSkillsRepo(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	skill, err := workspaceRepo.Get(ctx, id)
	if err == nil {
		return skill, nil
	}

	globalRepo, err := m.GetGlobalSkillsRepo(ctx)
	if err != nil {
		return nil, err
	}

	return globalRepo.Get(ctx, id)
}

func (m *DefaultSkillWorkspaceManager) SearchSkills(ctx context.Context, workspaceID string, query string, limit int) ([]*domain.Skill, error) {
	workspaceRepo, err := m.GetWorkspaceSkillsRepo(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	workspaceSkills, err := workspaceRepo.List(ctx, repository.SkillFilter{Limit: limit})
	if err != nil {
		return nil, fmt.Errorf("failed to search workspace skills: %w", err)
	}

	globalRepo, err := m.GetGlobalSkillsRepo(ctx)
	if err != nil {
		return nil, err
	}

	globalSkills, err := globalRepo.List(ctx, repository.SkillFilter{})
	if err != nil {
		return nil, fmt.Errorf("failed to search global skills: %w", err)
	}

	allSkills := append(workspaceSkills, globalSkills...)

	skills := m.scoreAndFilterSkills(allSkills, query, limit)
	return skills, nil
}

func (m *DefaultSkillWorkspaceManager) ListSkills(ctx context.Context, workspaceID string, filter repository.SkillFilter) ([]*domain.Skill, error) {
	workspaceRepo, err := m.GetWorkspaceSkillsRepo(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	workspaceSkills, err := workspaceRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspace skills: %w", err)
	}

	globalRepo, err := m.GetGlobalSkillsRepo(ctx)
	if err != nil {
		return nil, err
	}

	globalSkills, err := globalRepo.List(ctx, repository.SkillFilter{})
	if err != nil {
		return nil, fmt.Errorf("failed to list global skills: %w", err)
	}

	combined := append(workspaceSkills, globalSkills...)
	return combined, nil
}

func (m *DefaultSkillWorkspaceManager) InstallSkill(ctx context.Context, workspaceID string, skill *domain.Skill) error {
	workspaceRepo, err := m.GetWorkspaceSkillsRepo(ctx, workspaceID)
	if err != nil {
		return err
	}

	return workspaceRepo.Store(ctx, skill)
}

func (m *DefaultSkillWorkspaceManager) UninstallSkill(ctx context.Context, workspaceID string, id domain.SkillID) error {
	workspaceRepo, err := m.GetWorkspaceSkillsRepo(ctx, workspaceID)
	if err != nil {
		return err
	}

	return workspaceRepo.Delete(ctx, id)
}

func (m *DefaultSkillWorkspaceManager) scoreAndFilterSkills(skills []*domain.Skill, query string, limit int) []*domain.Skill {
	var scored []struct {
		skill *domain.Skill
		score float64
	}

	for _, skill := range skills {
		score := m.calculateRelevanceScore(query, skill)
		if score > 0 {
			scored = append(scored, struct {
				skill *domain.Skill
				score float64
			}{skill: skill, score: score})
		}
	}

	for i := 0; i < len(scored)-1; i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[i].score < scored[j].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	result := make([]*domain.Skill, 0, len(scored))
	for i, s := range scored {
		if limit > 0 && i >= limit {
			break
		}
		result = append(result, s.skill)
	}

	return result
}

func (m *DefaultSkillWorkspaceManager) calculateRelevanceScore(query string, skill *domain.Skill) float64 {
	var score float64
	queryLower := query

	for _, tag := range skill.Tags {
		if string(tag) == queryLower {
			score += 1.0
		} else if strings.Contains(string(tag), queryLower) {
			score += 0.8
		}
	}

	if skill.Name == queryLower {
		score += 0.9
	} else if strings.Contains(skill.Name, queryLower) {
		score += 0.7
	}

	return score
}
