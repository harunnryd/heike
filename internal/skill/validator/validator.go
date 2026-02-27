package validator

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/harunnryd/heike/internal/skill/domain"
)

type SkillValidator interface {
	Validate(skill *domain.Skill) error
	ValidateName(name string) error
	ValidateDescription(desc string) error
	ValidateTags(tags []domain.Tag) error
	ValidateTools(tools []domain.ToolRef) error
	ValidateVersion(version string) error
	ValidateMetadata(metadata domain.SkillMetadata) error
}

type DefaultSkillValidator struct {
	nameRegex    *regexp.Regexp
	tagRegex     *regexp.Regexp
	toolRegex    *regexp.Regexp
	versionRegex *regexp.Regexp
	minNameLen   int
	maxNameLen   int
	maxDescLen   int
	maxTags      int
	maxTools     int
	maxTagLen    int
	maxToolLen   int
}

type ValidationRule string

const (
	RuleRequiredName    ValidationRule = "REQUIRED_NAME"
	RuleRequiredDesc    ValidationRule = "REQUIRED_DESCRIPTION"
	RuleRequiredTags    ValidationRule = "REQUIRED_TAGS"
	RuleNameFormat      ValidationRule = "NAME_FORMAT"
	RuleNameLength      ValidationRule = "NAME_LENGTH"
	RuleDescLength      ValidationRule = "DESCRIPTION_LENGTH"
	RuleTagLength       ValidationRule = "TAG_LENGTH"
	RuleTagFormat       ValidationRule = "TAG_FORMAT"
	RuleToolLength      ValidationRule = "TOOL_LENGTH"
	RuleToolFormat      ValidationRule = "TOOL_FORMAT"
	RuleTagCount        ValidationRule = "TAG_COUNT"
	RuleToolCount       ValidationRule = "TOOL_COUNT"
	RuleVersionFormat   ValidationRule = "VERSION_FORMAT"
	RuleMetadataInvalid ValidationRule = "METADATA_INVALID"
)

type ValidationError struct {
	Field   string
	Message string
	Rule    ValidationRule
	Details map[string]string
}

func (e *ValidationError) Error() string {
	if e.Details != nil {
		details := make([]string, 0, len(e.Details))
		for k, v := range e.Details {
			details = append(details, fmt.Sprintf("%s=%s", k, v))
		}
		return string(e.Rule) + ": " + e.Field + " - " + e.Message + " (" + strings.Join(details, ", ") + ")"
	}
	return string(e.Rule) + ": " + e.Field + " - " + e.Message
}

func NewSkillValidator() SkillValidator {
	return &DefaultSkillValidator{
		nameRegex:    regexp.MustCompile(`^[a-z][a-z0-9_]*$`),
		tagRegex:     regexp.MustCompile(`^[a-z][a-z0-9_]*$`),
		toolRegex:    regexp.MustCompile(`^[a-z][a-z0-9_]*$`),
		versionRegex: regexp.MustCompile(`^\d+\.\d+\.\d+$`),
		minNameLen:   3,
		maxNameLen:   100,
		maxDescLen:   500,
		maxTags:      10,
		maxTools:     20,
		maxTagLen:    50,
		maxToolLen:   100,
	}
}

func (v *DefaultSkillValidator) Validate(skill *domain.Skill) error {
	if skill == nil {
		return &ValidationError{
			Field:   "skill",
			Message: "skill cannot be nil",
			Rule:    RuleRequiredName,
		}
	}

	if err := v.ValidateName(skill.Name); err != nil {
		return err
	}

	if err := v.ValidateDescription(skill.Description); err != nil {
		return err
	}

	if err := v.ValidateTags(skill.Tags); err != nil {
		return err
	}

	if err := v.ValidateTools(skill.Tools); err != nil {
		return err
	}

	if err := v.ValidateVersion(skill.Version); err != nil {
		return err
	}

	if err := v.ValidateMetadata(skill.Metadata); err != nil {
		return err
	}

	return nil
}

func (v *DefaultSkillValidator) ValidateName(name string) error {
	if name == "" {
		return &ValidationError{
			Field:   "name",
			Message: "name is required",
			Rule:    RuleRequiredName,
		}
	}

	if len(name) < v.minNameLen {
		return &ValidationError{
			Field:   "name",
			Message: fmt.Sprintf("name must be at least %d characters", v.minNameLen),
			Rule:    RuleNameLength,
			Details: map[string]string{
				"actual": fmt.Sprintf("%d", len(name)),
				"min":    fmt.Sprintf("%d", v.minNameLen),
			},
		}
	}

	if len(name) > v.maxNameLen {
		return &ValidationError{
			Field:   "name",
			Message: fmt.Sprintf("name must not exceed %d characters", v.maxNameLen),
			Rule:    RuleNameLength,
			Details: map[string]string{
				"actual": fmt.Sprintf("%d", len(name)),
				"max":    fmt.Sprintf("%d", v.maxNameLen),
			},
		}
	}

	if !v.nameRegex.MatchString(name) {
		return &ValidationError{
			Field:   "name",
			Message: "name must contain only lowercase letters, numbers, and underscores, starting with a letter",
			Rule:    RuleNameFormat,
			Details: map[string]string{
				"pattern": v.nameRegex.String(),
			},
		}
	}

	if unicode.IsPunct(rune(name[0])) {
		return &ValidationError{
			Field:   "name",
			Message: "name must not start with punctuation",
			Rule:    RuleNameFormat,
		}
	}

	return nil
}

func (v *DefaultSkillValidator) ValidateDescription(desc string) error {
	if desc == "" {
		return &ValidationError{
			Field:   "description",
			Message: "description is required",
			Rule:    RuleRequiredDesc,
		}
	}

	if len(desc) > v.maxDescLen {
		return &ValidationError{
			Field:   "description",
			Message: fmt.Sprintf("description must not exceed %d characters", v.maxDescLen),
			Rule:    RuleDescLength,
			Details: map[string]string{
				"actual": fmt.Sprintf("%d", len(desc)),
				"max":    fmt.Sprintf("%d", v.maxDescLen),
			},
		}
	}

	return nil
}

func (v *DefaultSkillValidator) ValidateTags(tags []domain.Tag) error {
	if len(tags) == 0 {
		return &ValidationError{
			Field:   "tags",
			Message: "at least one tag is required",
			Rule:    RuleRequiredTags,
		}
	}

	if len(tags) > v.maxTags {
		return &ValidationError{
			Field:   "tags",
			Message: fmt.Sprintf("number of tags must not exceed %d", v.maxTags),
			Rule:    RuleTagCount,
			Details: map[string]string{
				"actual": fmt.Sprintf("%d", len(tags)),
				"max":    fmt.Sprintf("%d", v.maxTags),
			},
		}
	}

	seenTags := make(map[string]bool)

	for i, tag := range tags {
		tagStr := string(tag)

		if tagStr == "" {
			return &ValidationError{
				Field:   "tags",
				Message: fmt.Sprintf("tag at index %d cannot be empty", i),
				Rule:    RuleTagFormat,
				Details: map[string]string{
					"index": fmt.Sprintf("%d", i),
				},
			}
		}

		if len(tagStr) > v.maxTagLen {
			return &ValidationError{
				Field:   "tags",
				Message: fmt.Sprintf("tag at index %d must not exceed %d characters", i, v.maxTagLen),
				Rule:    RuleTagLength,
				Details: map[string]string{
					"index":  fmt.Sprintf("%d", i),
					"actual": fmt.Sprintf("%d", len(tagStr)),
					"max":    fmt.Sprintf("%d", v.maxTagLen),
				},
			}
		}

		if !v.tagRegex.MatchString(tagStr) {
			return &ValidationError{
				Field:   "tags",
				Message: fmt.Sprintf("tag at index %d must contain only lowercase letters, numbers, and underscores, starting with a letter", i),
				Rule:    RuleTagFormat,
				Details: map[string]string{
					"index":   fmt.Sprintf("%d", i),
					"tag":     tagStr,
					"pattern": v.tagRegex.String(),
				},
			}
		}

		if seenTags[tagStr] {
			return &ValidationError{
				Field:   "tags",
				Message: fmt.Sprintf("duplicate tag at index %d", i),
				Rule:    RuleTagFormat,
				Details: map[string]string{
					"index": fmt.Sprintf("%d", i),
					"tag":   tagStr,
				},
			}
		}

		seenTags[tagStr] = true
	}

	return nil
}

func (v *DefaultSkillValidator) ValidateTools(tools []domain.ToolRef) error {
	if len(tools) > v.maxTools {
		return &ValidationError{
			Field:   "tools",
			Message: fmt.Sprintf("number of tools must not exceed %d", v.maxTools),
			Rule:    RuleToolCount,
			Details: map[string]string{
				"actual": fmt.Sprintf("%d", len(tools)),
				"max":    fmt.Sprintf("%d", v.maxTools),
			},
		}
	}

	seenTools := make(map[string]bool)

	for i, tool := range tools {
		toolStr := string(tool)

		if toolStr == "" {
			return &ValidationError{
				Field:   "tools",
				Message: fmt.Sprintf("tool at index %d cannot be empty", i),
				Rule:    RuleToolFormat,
				Details: map[string]string{
					"index": fmt.Sprintf("%d", i),
				},
			}
		}

		if len(toolStr) > v.maxToolLen {
			return &ValidationError{
				Field:   "tools",
				Message: fmt.Sprintf("tool at index %d must not exceed %d characters", i, v.maxToolLen),
				Rule:    RuleToolLength,
				Details: map[string]string{
					"index":  fmt.Sprintf("%d", i),
					"actual": fmt.Sprintf("%d", len(toolStr)),
					"max":    fmt.Sprintf("%d", v.maxToolLen),
				},
			}
		}

		if !v.toolRegex.MatchString(toolStr) {
			return &ValidationError{
				Field:   "tools",
				Message: fmt.Sprintf("tool at index %d must contain only lowercase letters, numbers, and underscores, starting with a letter", i),
				Rule:    RuleToolFormat,
				Details: map[string]string{
					"index":   fmt.Sprintf("%d", i),
					"tool":    toolStr,
					"pattern": v.toolRegex.String(),
				},
			}
		}

		if seenTools[toolStr] {
			return &ValidationError{
				Field:   "tools",
				Message: fmt.Sprintf("duplicate tool at index %d", i),
				Rule:    RuleToolFormat,
				Details: map[string]string{
					"index": fmt.Sprintf("%d", i),
					"tool":  toolStr,
				},
			}
		}

		seenTools[toolStr] = true
	}

	return nil
}

func (v *DefaultSkillValidator) ValidateVersion(version string) error {
	if version == "" {
		return nil
	}

	if !v.versionRegex.MatchString(version) {
		return &ValidationError{
			Field:   "version",
			Message: "version must follow semantic versioning (e.g., 1.0.0)",
			Rule:    RuleVersionFormat,
			Details: map[string]string{
				"pattern": v.versionRegex.String(),
			},
		}
	}

	return nil
}

func (v *DefaultSkillValidator) ValidateMetadata(metadata domain.SkillMetadata) error {
	if metadata.Complexity != "" {
		validComplexities := map[string]bool{
			"simple":  true,
			"medium":  true,
			"complex": true,
		}

		if !validComplexities[metadata.Complexity] {
			return &ValidationError{
				Field:   "metadata.complexity",
				Message: fmt.Sprintf("complexity must be one of: simple, medium, complex, got %s", metadata.Complexity),
				Rule:    RuleMetadataInvalid,
				Details: map[string]string{
					"actual": metadata.Complexity,
					"valid":  "simple, medium, complex",
				},
			}
		}
	}

	if len(metadata.Permissions) > 0 {
		for i, perm := range metadata.Permissions {
			if perm == "" {
				return &ValidationError{
					Field:   "metadata.permissions",
					Message: fmt.Sprintf("permission at index %d cannot be empty", i),
					Rule:    RuleMetadataInvalid,
					Details: map[string]string{
						"index": fmt.Sprintf("%d", i),
					},
				}
			}
		}
	}

	return nil
}
