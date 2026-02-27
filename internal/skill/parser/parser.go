package parser

import (
	"fmt"

	"github.com/harunnryd/heike/internal/skill/domain"
)

type SkillParser interface {
	Parse(content string) (*domain.Skill, error)
	Validate(content string) error
}

type ParseError struct {
	Line     int
	Message  string
	Code     ParseErrorCode
	Original error
}

type ParseErrorCode string

const (
	CodeInvalidYAML        ParseErrorCode = "INVALID_YAML"
	CodeMissingFrontmatter ParseErrorCode = "MISSING_FRONTMATTER"
	CodeInvalidField       ParseErrorCode = "INVALID_FIELD"
	CodeDuplicateField     ParseErrorCode = "DUPLICATE_FIELD"
)

func (e *ParseError) Error() string {
	if e.Line > 0 {
		return string(e.Code) + " at line " + fmt.Sprintf("%d", e.Line) + ": " + e.Message
	}
	return string(e.Code) + ": " + e.Message
}

func (e *ParseError) Unwrap() error {
	return e.Original
}
