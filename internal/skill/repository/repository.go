package repository

import (
	"context"

	"github.com/harunnryd/heike/internal/skill/domain"
)

type SkillRepository interface {
	LoadAll(ctx context.Context) error
	Store(ctx context.Context, skill *domain.Skill) error
	Get(ctx context.Context, id domain.SkillID) (*domain.Skill, error)
	List(ctx context.Context, filter SkillFilter) ([]*domain.Skill, error)
	Delete(ctx context.Context, id domain.SkillID) error
	Exists(ctx context.Context, id domain.SkillID) (bool, error)
}

type SkillFilter struct {
	Tags      []domain.Tag
	Tools     []domain.ToolRef
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}
