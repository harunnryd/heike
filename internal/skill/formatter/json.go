package formatter

import (
	"encoding/json"

	"github.com/harunnryd/heike/internal/skill/domain"
)

type JSONFormatter struct{}

func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{}
}

func (f *JSONFormatter) FormatSkills(skills []*domain.Skill) (string, error) {
	data, err := json.MarshalIndent(skills, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (f *JSONFormatter) FormatSkill(skill *domain.Skill) (string, error) {
	if skill == nil {
		return "null", nil
	}
	data, err := json.MarshalIndent(skill, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
