package formatter

import (
	"fmt"
	"strings"

	"github.com/harunnryd/heike/internal/skill/domain"
)

type OutputFormat string

const (
	OutputFormatTable OutputFormat = "table"
	OutputFormatJSON  OutputFormat = "json"
	OutputFormatYAML  OutputFormat = "yaml"
)

type SkillFormatter interface {
	FormatSkills([]*domain.Skill) (string, error)
	FormatSkill(*domain.Skill) (string, error)
}

type FormatterFactory struct{}

func NewFormatterFactory() *FormatterFactory {
	return &FormatterFactory{}
}

func (f *FormatterFactory) Create(format OutputFormat) (SkillFormatter, error) {
	switch format {
	case OutputFormatTable:
		return NewTableFormatter(), nil
	case OutputFormatJSON:
		return NewJSONFormatter(), nil
	case OutputFormatYAML:
		return NewYAMLFormatter(), nil
	default:
		return nil, fmt.Errorf("unsupported output format: %s (supported: table, json, yaml)", format)
	}
}

func ParseOutputFormat(s string) (OutputFormat, error) {
	format := OutputFormat(strings.ToLower(s))
	switch format {
	case OutputFormatTable, OutputFormatJSON, OutputFormatYAML:
		return format, nil
	default:
		return "", fmt.Errorf("invalid output format: %s (supported: table, json, yaml)", s)
	}
}
