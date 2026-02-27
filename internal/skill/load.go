package skill

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadSkillFromFile reads and parses a SKILL.md document.
func LoadSkillFromFile(path string) (*Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read failed: %w", err)
	}

	skill, err := ParseSkillContent(data)
	if err != nil {
		return nil, err
	}
	return skill, nil
}

// ParseSkillContent parses frontmatter + body into a Skill instance.
func ParseSkillContent(content []byte) (*Skill, error) {
	parser := &FrontmatterParser{}

	frontmatter, body, err := parser.Parse(content)
	if err != nil {
		return nil, fmt.Errorf("frontmatter parse failed: %w", err)
	}

	var skill Skill
	if err := yaml.Unmarshal([]byte(frontmatter), &skill); err != nil {
		return nil, fmt.Errorf("yaml unmarshal failed: %w", err)
	}
	skill.Content = body

	return &skill, nil
}
