package domain

import (
	"fmt"
	"regexp"
)

var skillNameRegex = regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)

func ValidateSkillName(name string) error {
	if name == "" {
		return &SkillValidationError{
			Field:   "name",
			Message: "cannot be empty",
			Code:    CodeMissingField,
		}
	}

	if len(name) < 3 {
		return &SkillValidationError{
			Field:   "name",
			Message: "must be at least 3 characters",
			Code:    CodeInvalidLength,
		}
	}

	if len(name) > 100 {
		return &SkillValidationError{
			Field:   "name",
			Message: "must not exceed 100 characters",
			Code:    CodeInvalidLength,
		}
	}

	if !skillNameRegex.MatchString(name) {
		return &SkillValidationError{
			Field:   "name",
			Message: fmt.Sprintf("must only contain alphanumeric characters, underscores, and hyphens (got: %s)", name),
			Code:    CodeInvalidFormat,
		}
	}

	return nil
}

func ValidateSkillDescription(description string) error {
	if description == "" {
		return &SkillValidationError{
			Field:   "description",
			Message: "cannot be empty",
			Code:    CodeMissingField,
		}
	}

	if len(description) > 1000 {
		return &SkillValidationError{
			Field:   "description",
			Message: "must not exceed 1000 characters",
			Code:    CodeInvalidLength,
		}
	}

	return nil
}

func ValidateSkillTags(tags []Tag) error {
	if len(tags) == 0 {
		return &SkillValidationError{
			Field:   "tags",
			Message: "must have at least one tag",
			Code:    CodeMissingField,
		}
	}

	for i, tag := range tags {
		if !tag.IsValid() {
			return &SkillValidationError{
				Field:   "tags",
				Message: fmt.Sprintf("tag[%d] cannot be empty", i),
				Code:    CodeInvalidValue,
			}
		}

		if len(tag.String()) > 50 {
			return &SkillValidationError{
				Field:   "tags",
				Message: fmt.Sprintf("tag[%d] must not exceed 50 characters", i),
				Code:    CodeInvalidLength,
			}
		}
	}

	return nil
}

func ValidateSkillTools(tools []ToolRef) error {
	if len(tools) == 0 {
		return nil
	}

	seen := make(map[ToolRef]bool)
	for i, tool := range tools {
		if !tool.IsValid() {
			return &SkillValidationError{
				Field:   "tools",
				Message: fmt.Sprintf("tool[%d] cannot be empty", i),
				Code:    CodeInvalidValue,
			}
		}

		if seen[tool] {
			return &SkillValidationError{
				Field:   "tools",
				Message: fmt.Sprintf("tool[%d] is duplicated: %s", i, tool.String()),
				Code:    CodeInvalidValue,
			}
		}
		seen[tool] = true
	}

	return nil
}

func ValidateSkillVersion(version string) error {
	if version == "" {
		return nil
	}

	versionRegex := regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	if !versionRegex.MatchString(version) {
		return &SkillValidationError{
			Field:   "version",
			Message: "must follow semantic versioning (e.g., 1.0.0)",
			Code:    CodeInvalidFormat,
		}
	}

	return nil
}

func ValidateSkill(skill *Skill) error {
	if err := ValidateSkillName(skill.Name); err != nil {
		return err
	}

	if err := ValidateSkillDescription(skill.Description); err != nil {
		return err
	}

	if err := ValidateSkillTags(skill.Tags); err != nil {
		return err
	}

	if err := ValidateSkillTools(skill.Tools); err != nil {
		return err
	}

	if err := ValidateSkillVersion(skill.Version); err != nil {
		return err
	}

	metadataValid := true
	if skill.Metadata.Complexity != "" {
		metadataValid = skill.Metadata.IsValid()
	}

	if !metadataValid {
		return &SkillValidationError{
			Field:   "metadata",
			Message: "contains invalid values",
			Code:    CodeInvalidValue,
		}
	}

	return nil
}
