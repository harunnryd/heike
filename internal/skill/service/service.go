package service

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/harunnryd/heike/internal/skill/domain"
	"github.com/harunnryd/heike/internal/skill/repository"
)

type SkillService interface {
	LoadSkills(ctx context.Context) error
	GetSkill(ctx context.Context, id domain.SkillID) (*domain.Skill, error)
	SearchSkills(ctx context.Context, query string, limit int) ([]*domain.Skill, error)
	ListSkills(ctx context.Context, filter repository.SkillFilter) ([]*domain.Skill, error)
	ValidateSkill(ctx context.Context, skill *domain.Skill) error
	InstallSkill(ctx context.Context, skill *domain.Skill) error
	UninstallSkill(ctx context.Context, id domain.SkillID) error
}

type SkillServiceImpl struct {
	repo     repository.SkillRepository
	validate func(*domain.Skill) error
}

func NewSkillService(repo repository.SkillRepository) SkillService {
	return &SkillServiceImpl{
		repo:     repo,
		validate: domain.ValidateSkill,
	}
}

func (s *SkillServiceImpl) LoadSkills(ctx context.Context) error {
	slog.Info("Loading skills", "context", ctx)

	if err := s.repo.LoadAll(ctx); err != nil {
		return fmt.Errorf("failed to load skills: %w", err)
	}

	return nil
}

func (s *SkillServiceImpl) GetSkill(ctx context.Context, id domain.SkillID) (*domain.Skill, error) {
	if !id.IsValid() {
		return nil, &domain.SkillValidationError{
			Field:   "id",
			Message: "invalid skill ID",
			Code:    domain.CodeInvalidValue,
		}
	}

	return s.repo.Get(ctx, id)
}

func (s *SkillServiceImpl) SearchSkills(ctx context.Context, query string, limit int) ([]*domain.Skill, error) {
	if query == "" {
		filter := repository.SkillFilter{
			Limit: limit,
		}
		return s.repo.List(ctx, filter)
	}

	allSkills, err := s.repo.List(ctx, repository.SkillFilter{})
	if err != nil {
		return nil, fmt.Errorf("failed to list skills for search: %w", err)
	}

	filteredSkills := s.searchByQuery(allSkills, query)
	sort.Slice(filteredSkills, func(i, j int) bool {
		return s.calculateRelevanceScore(query, filteredSkills[i]) > s.calculateRelevanceScore(query, filteredSkills[j])
	})

	if limit > 0 && limit < len(filteredSkills) {
		filteredSkills = filteredSkills[:limit]
	}

	return filteredSkills, nil
}

func (s *SkillServiceImpl) ListSkills(ctx context.Context, filter repository.SkillFilter) ([]*domain.Skill, error) {
	return s.repo.List(ctx, filter)
}

func (s *SkillServiceImpl) ValidateSkill(ctx context.Context, skill *domain.Skill) error {
	if skill == nil {
		return &domain.SkillValidationError{
			Field:   "skill",
			Message: "skill cannot be nil",
			Code:    domain.CodeInvalidValue,
		}
	}

	return s.validate(skill)
}

func (s *SkillServiceImpl) InstallSkill(ctx context.Context, skill *domain.Skill) error {
	if err := s.ValidateSkill(ctx, skill); err != nil {
		return fmt.Errorf("skill validation failed: %w", err)
	}

	skill.ID = domain.SkillID(skill.Name)
	skill.CreatedAt = time.Now()
	skill.UpdatedAt = time.Now()

	if exists, err := s.repo.Exists(ctx, skill.ID); err != nil {
		return fmt.Errorf("failed to check skill existence: %w", err)
	} else if exists {
		return &domain.SkillExistsError{ID: skill.ID}
	}

	if err := s.repo.Store(ctx, skill); err != nil {
		return fmt.Errorf("failed to store skill: %w", err)
	}

	slog.Info("Skill installed", "id", skill.ID, "name", skill.Name)
	return nil
}

func (s *SkillServiceImpl) UninstallSkill(ctx context.Context, id domain.SkillID) error {
	if !id.IsValid() {
		return &domain.SkillValidationError{
			Field:   "id",
			Message: "invalid skill ID",
			Code:    domain.CodeInvalidValue,
		}
	}

	exists, err := s.repo.Exists(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check skill existence: %w", err)
	}
	if !exists {
		return &domain.SkillNotFoundError{ID: id}
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete skill: %w", err)
	}

	slog.Info("Skill uninstalled", "id", id)
	return nil
}

func (s *SkillServiceImpl) searchByQuery(skills []*domain.Skill, query string) []*domain.Skill {
	var result []*domain.Skill

	for _, skill := range skills {
		if s.matchesQuery(skill, query) {
			result = append(result, skill)
		}
	}

	return result
}

func (s *SkillServiceImpl) matchesQuery(skill *domain.Skill, query string) bool {
	query = strings.ToLower(query)

	for _, tag := range skill.Tags {
		tagLower := strings.ToLower(string(tag))
		if strings.Contains(tagLower, query) {
			return true
		}
	}

	if strings.Contains(strings.ToLower(skill.Name), query) {
		return true
	}

	if strings.Contains(strings.ToLower(skill.Description), query) {
		return true
	}

	for _, tool := range skill.Tools {
		toolLower := strings.ToLower(string(tool))
		if strings.Contains(toolLower, query) {
			return true
		}
	}

	return false
}

func (s *SkillServiceImpl) calculateRelevanceScore(query string, skill *domain.Skill) float64 {
	var score float64
	query = strings.ToLower(query)

	for _, tag := range skill.Tags {
		tagLower := strings.ToLower(string(tag))
		if tagLower == query {
			score += 1.0
		} else if strings.Contains(tagLower, query) {
			score += 0.8
		}
	}

	nameLower := strings.ToLower(skill.Name)
	if nameLower == query {
		score += 0.9
	} else if strings.Contains(nameLower, query) {
		score += 0.7
	}

	descLower := strings.ToLower(skill.Description)
	if strings.Contains(descLower, query) {
		score += 0.5
	}

	return score
}
