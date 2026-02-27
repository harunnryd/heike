package repository

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	skillmodel "github.com/harunnryd/heike/internal/skill"
	"github.com/harunnryd/heike/internal/skill/domain"
)

type FileSkillRepository struct {
	basePath string
	registry *skillmodel.Registry
}

func NewFileSkillRepository(basePath string) *FileSkillRepository {
	return &FileSkillRepository{
		basePath: basePath,
		registry: skillmodel.NewRegistry(),
	}
}

func (r *FileSkillRepository) LoadAll(ctx context.Context) error {
	if err := r.registry.Load(r.basePath); err != nil {
		return fmt.Errorf("failed to load skills: %w", err)
	}
	return nil
}

func (r *FileSkillRepository) Store(ctx context.Context, skill *domain.Skill) error {
	skillDir := filepath.Join(r.basePath, string(skill.Name))
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	skillPath := filepath.Join(skillDir, "SKILL.md")

	registrySkill := r.domainToRegistrySkill(skill)
	content := r.formatSkillContent(registrySkill)

	if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write skill file: %w", err)
	}

	r.registry.Register(registrySkill)

	slog.Info("Stored skill", "name", skill.Name, "path", skillPath)
	return nil
}

func (r *FileSkillRepository) Get(ctx context.Context, id domain.SkillID) (*domain.Skill, error) {
	if !id.IsValid() {
		return nil, fmt.Errorf("invalid skill ID: %s", id)
	}

	registrySkill, err := r.registry.Get(string(id))
	if err != nil {
		return nil, fmt.Errorf("skill not found: %w", err)
	}

	return r.registrySkillToDomain(registrySkill), nil
}

func (r *FileSkillRepository) List(ctx context.Context, filter SkillFilter) ([]*domain.Skill, error) {
	skills, err := r.registry.GetRelevant("", 0)
	if err != nil {
		return nil, fmt.Errorf("failed to list skills: %w", err)
	}

	filtered := r.filterSkills(skills, filter)
	sorted := r.sortSkills(filtered, filter.SortBy, filter.SortOrder)

	if filter.Limit > 0 && filter.Limit < len(sorted) {
		sorted = sorted[:filter.Limit]
	}

	if filter.Offset > 0 && filter.Offset < len(sorted) {
		sorted = sorted[filter.Offset:]
	}

	result := make([]*domain.Skill, len(sorted))
	for i, s := range sorted {
		result[i] = r.registrySkillToDomain(s)
	}

	return result, nil
}

func (r *FileSkillRepository) Delete(ctx context.Context, id domain.SkillID) error {
	if !id.IsValid() {
		return fmt.Errorf("invalid skill ID: %s", id)
	}

	skillDir := filepath.Join(r.basePath, string(id))
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		return fmt.Errorf("skill directory not found: %s", id)
	}

	if err := os.RemoveAll(skillDir); err != nil {
		return fmt.Errorf("failed to delete skill: %w", err)
	}

	r.registry.Remove(string(id))

	slog.Info("Deleted skill", "name", id)
	return nil
}

func (r *FileSkillRepository) Exists(ctx context.Context, id domain.SkillID) (bool, error) {
	if !id.IsValid() {
		return false, nil
	}

	skillPath := filepath.Join(r.basePath, string(id), "SKILL.md")
	if _, err := os.Stat(skillPath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check skill existence: %w", err)
	}

	return true, nil
}

func (r *FileSkillRepository) filterSkills(skills []*skillmodel.Skill, filter SkillFilter) []*skillmodel.Skill {
	var result []*skillmodel.Skill

	for _, s := range skills {
		if r.matchesFilter(s, filter) {
			result = append(result, s)
		}
	}

	return result
}

func (r *FileSkillRepository) matchesFilter(s *skillmodel.Skill, filter SkillFilter) bool {
	if len(filter.Tags) > 0 {
		matchesTag := false
		for _, tag := range filter.Tags {
			tagStr := string(tag)
			for _, skillTag := range s.Tags {
				if skillTag == tagStr {
					matchesTag = true
					break
				}
			}
		}
		if !matchesTag {
			return false
		}
	}

	if len(filter.Tools) > 0 {
		matchesTool := false
		for _, tool := range filter.Tools {
			toolStr := string(tool)
			for _, skillTool := range s.Tools {
				if skillTool == toolStr {
					matchesTool = true
					break
				}
			}
		}
		if !matchesTool {
			return false
		}
	}

	return true
}

func (r *FileSkillRepository) sortSkills(skills []*skillmodel.Skill, sortBy, sortOrder string) []*skillmodel.Skill {
	if sortBy == "" {
		sortBy = "name"
	}

	sort.Slice(skills, func(i, j int) bool {
		var iVal, jVal string

		switch sortBy {
		case "name":
			iVal = skills[i].Name
			jVal = skills[j].Name
		default:
			iVal = skills[i].Name
			jVal = skills[j].Name
		}

		if sortOrder == "desc" {
			return strings.Compare(iVal, jVal) > 0
		}
		return strings.Compare(iVal, jVal) < 0
	})

	return skills
}

func (r *FileSkillRepository) domainToRegistrySkill(skill *domain.Skill) *skillmodel.Skill {
	tags := make([]string, len(skill.Tags))
	for i, tag := range skill.Tags {
		tags[i] = string(tag)
	}

	tools := make([]string, len(skill.Tools))
	for i, tool := range skill.Tools {
		tools[i] = string(tool)
	}

	return &skillmodel.Skill{
		Name:        skill.Name,
		Description: skill.Description,
		Tags:        tags,
		Tools:       tools,
		Content:     skill.Content,
	}
}

func (r *FileSkillRepository) registrySkillToDomain(registrySkill *skillmodel.Skill) *domain.Skill {
	tags := make([]domain.Tag, len(registrySkill.Tags))
	for i, tag := range registrySkill.Tags {
		tags[i] = domain.Tag(tag)
	}

	tools := make([]domain.ToolRef, len(registrySkill.Tools))
	for i, tool := range registrySkill.Tools {
		tools[i] = domain.ToolRef(tool)
	}

	return &domain.Skill{
		ID:          domain.SkillID(registrySkill.Name),
		Name:        registrySkill.Name,
		Description: registrySkill.Description,
		Tags:        tags,
		Tools:       tools,
		Content:     registrySkill.Content,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func (r *FileSkillRepository) formatSkillContent(s *skillmodel.Skill) string {
	var builder strings.Builder

	builder.WriteString("---\n")
	builder.WriteString("name: " + s.Name + "\n")
	builder.WriteString("description: " + s.Description + "\n")
	builder.WriteString("tags:\n")
	for _, tag := range s.Tags {
		builder.WriteString("  - " + tag + "\n")
	}
	if len(s.Tools) > 0 {
		builder.WriteString("tools:\n")
		for _, tool := range s.Tools {
			builder.WriteString("  - " + tool + "\n")
		}
	}
	builder.WriteString("---\n")
	builder.WriteString(s.Content)

	return builder.String()
}
