package formatter

import (
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/harunnryd/heike/internal/skill/domain"
)

type YAMLFormatter struct{}

func NewYAMLFormatter() *YAMLFormatter {
	return &YAMLFormatter{}
}

func (f *YAMLFormatter) FormatSkills(skills []*domain.Skill) (string, error) {
	data, err := yaml.Marshal(skills)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func (f *YAMLFormatter) FormatSkill(skill *domain.Skill) (string, error) {
	if skill == nil {
		return "null", nil
	}
	data, err := yaml.Marshal(skill)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
