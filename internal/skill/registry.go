package skill

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Skill struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Tags        []string               `yaml:"tags"`
	Tools       []string               `yaml:"tools"`
	Metadata    map[string]interface{} `yaml:"metadata"`
	Content     string                 `yaml:"-"`
}

type SkillLoadError struct {
	Path    string
	Message string
	Cause   error
}

func (e *SkillLoadError) Error() string {
	return fmt.Sprintf("failed to load skill from %s: %s", e.Path, e.Message)
}

func (e *SkillLoadError) Unwrap() error {
	return e.Cause
}

type SkillValidationError struct {
	Field   string
	Message string
}

func (e *SkillValidationError) Error() string {
	return fmt.Sprintf("validation failed for field %s: %s", e.Field, e.Message)
}

type Registry struct {
	skills      map[string]*Skill
	loadPath    string
	frontmatter *FrontmatterParser
}

type RegistryStats struct {
	TotalSkills  int
	LoadErrors   int
	LastLoadTime string
	SkillsByTag  map[string]int
}

type FrontmatterParser struct{}

func NewRegistry() *Registry {
	return &Registry{
		skills:      make(map[string]*Skill),
		frontmatter: &FrontmatterParser{},
	}
}

func NewRegistryWithPath(path string) *Registry {
	return &Registry{
		skills:      make(map[string]*Skill),
		loadPath:    path,
		frontmatter: &FrontmatterParser{},
	}
}

func (r *Registry) Load(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	entries, err := os.ReadDir(path)
	if os.IsNotExist(err) {
		slog.Debug("Skills directory does not exist", "path", path)
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read skills directory %s: %w", path, err)
	}

	var loadErrors []error
	loadedCount := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillPath := filepath.Join(path, entry.Name(), "SKILL.md")
		if err := r.loadSkill(skillPath); err != nil {
			loadErrors = append(loadErrors, &SkillLoadError{
				Path:    skillPath,
				Message: "load failed",
				Cause:   err,
			})
			continue
		}
		loadedCount++
	}

	slog.Info("Skills loaded", "count", loadedCount, "errors", len(loadErrors), "path", path)

	if len(loadErrors) > 0 {
		return fmt.Errorf("loaded %d skills with %d errors: %w", loadedCount, len(loadErrors), joinErrors(loadErrors))
	}

	r.loadPath = path
	return nil
}

func (r *Registry) loadSkill(path string) error {
	skill, err := LoadSkillFromFile(path)
	if err != nil {
		return err
	}

	if err := r.Validate(skill); err != nil {
		return err
	}

	if _, exists := r.skills[skill.Name]; exists {
		slog.Warn("Duplicate skill detected, overwriting", "name", skill.Name, "path", path)
	}

	r.skills[skill.Name] = skill
	slog.Debug("Loaded skill", "name", skill.Name, "path", path)
	return nil
}

func (r *Registry) Register(skill *Skill) {
	if skill == nil {
		return
	}

	r.skills[skill.Name] = skill
	slog.Debug("Registered skill", "name", skill.Name)
}

func (r *Registry) Remove(name string) {
	delete(r.skills, name)
	slog.Debug("Removed skill", "name", name)
}

func (r *Registry) Validate(skill *Skill) error {
	if skill.Name == "" {
		return &SkillValidationError{Field: "name", Message: "cannot be empty"}
	}

	if !isValidSkillName(skill.Name) {
		return &SkillValidationError{
			Field:   "name",
			Message: "must only contain alphanumeric characters, underscores, and hyphens",
		}
	}

	if skill.Description == "" {
		return &SkillValidationError{Field: "description", Message: "cannot be empty"}
	}

	if len(skill.Tags) == 0 {
		return &SkillValidationError{Field: "tags", Message: "must have at least one tag"}
	}

	for i, tag := range skill.Tags {
		if tag == "" {
			return &SkillValidationError{
				Field:   "tags",
				Message: fmt.Sprintf("tag[%d] cannot be empty", i),
			}
		}
	}

	return nil
}

func isValidSkillName(name string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, name)
	return matched
}

func (r *Registry) Get(name string) (*Skill, error) {
	skill, ok := r.skills[name]
	if !ok {
		return nil, fmt.Errorf("skill not found: %s", name)
	}
	return skill, nil
}

func (r *Registry) GetRelevant(query string, limit int) ([]*Skill, error) {
	if query == "" {
		return r.listAllSkills(limit), nil
	}

	query = strings.ToLower(query)

	var scoredSkills []struct {
		skill *Skill
		score float64
	}

	for _, skill := range r.skills {
		score := r.calculateRelevanceScore(query, skill)
		if score > 0 {
			scoredSkills = append(scoredSkills, struct {
				skill *Skill
				score float64
			}{skill, score})
		}
	}

	sort.Slice(scoredSkills, func(i, j int) bool {
		return scoredSkills[i].score > scoredSkills[j].score
	})

	if limit <= 0 || limit > len(scoredSkills) {
		limit = len(scoredSkills)
	}

	result := make([]*Skill, limit)
	for i := 0; i < limit; i++ {
		result[i] = scoredSkills[i].skill
	}

	return result, nil
}

func (r *Registry) calculateRelevanceScore(query string, skill *Skill) float64 {
	var score float64

	for _, tag := range skill.Tags {
		tagLower := strings.ToLower(tag)
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

func (r *Registry) listAllSkills(limit int) []*Skill {
	skills := make([]*Skill, 0, len(r.skills))
	for _, skill := range r.skills {
		skills = append(skills, skill)
	}

	if limit > 0 && limit < len(skills) {
		skills = skills[:limit]
	}

	return skills
}

func (r *Registry) List(sortBy string) ([]string, error) {
	if len(r.skills) == 0 {
		return []string{}, nil
	}

	names := make([]string, 0, len(r.skills))
	for name := range r.skills {
		names = append(names, name)
	}

	switch sortBy {
	case "name":
		sort.Strings(names)
	case "name-desc":
		sort.Sort(sort.Reverse(sort.StringSlice(names)))
	default:
		sort.Strings(names)
	}

	return names, nil
}

func (r *Registry) Stats() RegistryStats {
	stats := RegistryStats{
		TotalSkills:  len(r.skills),
		LastLoadTime: "never",
		SkillsByTag:  make(map[string]int),
	}

	for _, skill := range r.skills {
		for _, tag := range skill.Tags {
			stats.SkillsByTag[tag]++
		}
	}

	return stats
}

func (r *Registry) Reload() error {
	if r.loadPath == "" {
		return fmt.Errorf("cannot reload: no previous load path")
	}

	r.skills = make(map[string]*Skill)
	return r.Load(r.loadPath)
}

func (fp *FrontmatterParser) Parse(content []byte) (string, string, error) {
	contentStr := string(content)
	contentStr = strings.TrimSpace(contentStr)

	if !strings.HasPrefix(contentStr, "---") {
		return "", "", fmt.Errorf("invalid frontmatter: must start with ---")
	}

	parts := strings.SplitN(contentStr, "---", 3)
	if len(parts) < 3 {
		return "", "", fmt.Errorf("invalid frontmatter: expected 3 parts separated by ---")
	}

	frontmatter := strings.TrimSpace(parts[1])
	if frontmatter == "" {
		return "", "", fmt.Errorf("invalid frontmatter: frontmatter is empty")
	}

	body := strings.TrimSpace(parts[2])

	return frontmatter, body, nil
}

func joinErrors(errors []error) error {
	if len(errors) == 0 {
		return nil
	}
	var msgs []string
	for _, err := range errors {
		msgs = append(msgs, err.Error())
	}
	return fmt.Errorf("multiple errors: %s", strings.Join(msgs, "; "))
}
