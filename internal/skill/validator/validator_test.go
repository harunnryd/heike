package validator

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/harunnryd/heike/internal/skill/domain"
)

func TestNewSkillValidator(t *testing.T) {
	validator := NewSkillValidator()

	if validator == nil {
		t.Error("NewSkillValidator() returned nil")
	}
}

func TestDefaultSkillValidator_ValidSkill(t *testing.T) {
	validator := NewSkillValidator()

	skill := &domain.Skill{
		Name:        "test_skill",
		Description: "Test skill description",
		Tags:        []domain.Tag{"web_scrape", "data_analysis"},
		Tools:       []domain.ToolRef{"search_query"},
		Version:     "1.0.0",
	}

	err := validator.Validate(skill)
	if err != nil {
		t.Errorf("Validate() failed: %v", err)
	}
}

func TestDefaultSkillValidator_NilSkill(t *testing.T) {
	validator := NewSkillValidator()

	err := validator.Validate(nil)
	if err == nil {
		t.Error("Validate() should error for nil skill")
	}

	var validErr *ValidationError
	if !errors.As(err, &validErr) {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if validErr.Rule != RuleRequiredName {
		t.Errorf("validErr.Rule = %v, want %v", validErr.Rule, RuleRequiredName)
	}
}

func TestDefaultSkillValidator_ValidateName_Empty(t *testing.T) {
	validator := NewSkillValidator()

	err := validator.ValidateName("")
	if err == nil {
		t.Error("ValidateName() should error for empty name")
	}

	var validErr *ValidationError
	if !errors.As(err, &validErr) {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if validErr.Rule != RuleRequiredName {
		t.Errorf("validErr.Rule = %v, want %v", validErr.Rule, RuleRequiredName)
	}
}

func TestDefaultSkillValidator_ValidateName_TooShort(t *testing.T) {
	validator := NewSkillValidator()

	err := validator.ValidateName("ab")
	if err == nil {
		t.Error("ValidateName() should error for too short name")
	}

	var validErr *ValidationError
	if !errors.As(err, &validErr) {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if validErr.Rule != RuleNameLength {
		t.Errorf("validErr.Rule = %v, want %v", validErr.Rule, RuleNameLength)
	}
}

func TestDefaultSkillValidator_ValidateName_TooLong(t *testing.T) {
	validator := NewSkillValidator()

	longName := string(make([]byte, 101))
	for i := range longName {
		longName = longName[:i] + "a" + longName[i+1:]
	}

	err := validator.ValidateName(longName)
	if err == nil {
		t.Error("ValidateName() should error for too long name")
	}

	var validErr *ValidationError
	if !errors.As(err, &validErr) {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if validErr.Rule != RuleNameLength {
		t.Errorf("validErr.Rule = %v, want %v", validErr.Rule, RuleNameLength)
	}
}

func TestDefaultSkillValidator_ValidateName_InvalidFormat(t *testing.T) {
	validator := NewSkillValidator()

	testCases := []struct {
		name string
	}{
		{"InvalidName"},
		{"invalid-name"},
		{"invalid name"},
		{"123_invalid"},
		{"_invalid"},
		{"invalid!"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidateName(tc.name)
			if err == nil {
				t.Errorf("ValidateName() should error for invalid name: %s", tc.name)
			}

			var validErr *ValidationError
			if !errors.As(err, &validErr) {
				t.Errorf("expected ValidationError, got %T", err)
			}

			if validErr.Rule != RuleNameFormat {
				t.Errorf("validErr.Rule = %v, want %v", validErr.Rule, RuleNameFormat)
			}
		})
	}
}

func TestDefaultSkillValidator_ValidateName_Valid(t *testing.T) {
	validator := NewSkillValidator()

	testCases := []string{
		"test_skill",
		"web_scrape",
		"data_analysis",
		"skill123",
		"my_skill_1",
	}

	for _, name := range testCases {
		t.Run(name, func(t *testing.T) {
			err := validator.ValidateName(name)
			if err != nil {
				t.Errorf("ValidateName() failed for valid name %s: %v", name, err)
			}
		})
	}
}

func TestDefaultSkillValidator_ValidateDescription_Empty(t *testing.T) {
	validator := NewSkillValidator()

	err := validator.ValidateDescription("")
	if err == nil {
		t.Error("ValidateDescription() should error for empty description")
	}

	var validErr *ValidationError
	if !errors.As(err, &validErr) {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if validErr.Rule != RuleRequiredDesc {
		t.Errorf("validErr.Rule = %v, want %v", validErr.Rule, RuleRequiredDesc)
	}
}

func TestDefaultSkillValidator_ValidateDescription_TooLong(t *testing.T) {
	validator := NewSkillValidator()

	longDesc := string(make([]byte, 501))
	for i := range longDesc {
		longDesc = longDesc[:i] + "a" + longDesc[i+1:]
	}

	err := validator.ValidateDescription(longDesc)
	if err == nil {
		t.Error("ValidateDescription() should error for too long description")
	}

	var validErr *ValidationError
	if !errors.As(err, &validErr) {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if validErr.Rule != RuleDescLength {
		t.Errorf("validErr.Rule = %v, want %v", validErr.Rule, RuleDescLength)
	}
}

func TestDefaultSkillValidator_ValidateTags_Empty(t *testing.T) {
	validator := NewSkillValidator()

	err := validator.ValidateTags([]domain.Tag{})
	if err == nil {
		t.Error("ValidateTags() should error for empty tags")
	}

	var validErr *ValidationError
	if !errors.As(err, &validErr) {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if validErr.Rule != RuleRequiredTags {
		t.Errorf("validErr.Rule = %v, want %v", validErr.Rule, RuleRequiredTags)
	}
}

func TestDefaultSkillValidator_ValidateTags_TooMany(t *testing.T) {
	validator := NewSkillValidator()

	tags := make([]domain.Tag, 11)
	for i := range tags {
		tags[i] = domain.Tag("tag" + string(rune('0'+i)))
	}

	err := validator.ValidateTags(tags)
	if err == nil {
		t.Error("ValidateTags() should error for too many tags")
	}

	var validErr *ValidationError
	if !errors.As(err, &validErr) {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if validErr.Rule != RuleTagCount {
		t.Errorf("validErr.Rule = %v, want %v", validErr.Rule, RuleTagCount)
	}
}

func TestDefaultSkillValidator_ValidateTags_EmptyTag(t *testing.T) {
	validator := NewSkillValidator()

	tags := []domain.Tag{"valid_tag", ""}

	err := validator.ValidateTags(tags)
	if err == nil {
		t.Error("ValidateTags() should error for empty tag")
	}

	var validErr *ValidationError
	if !errors.As(err, &validErr) {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if validErr.Rule != RuleTagFormat {
		t.Errorf("validErr.Rule = %v, want %v", validErr.Rule, RuleTagFormat)
	}
}

func TestDefaultSkillValidator_ValidateTags_TooLong(t *testing.T) {
	validator := NewSkillValidator()

	longTag := string(make([]byte, 51))
	for i := range longTag {
		longTag = longTag[:i] + "a" + longTag[i+1:]
	}

	tags := []domain.Tag{domain.Tag(longTag)}

	err := validator.ValidateTags(tags)
	if err == nil {
		t.Error("ValidateTags() should error for too long tag")
	}

	var validErr *ValidationError
	if !errors.As(err, &validErr) {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if validErr.Rule != RuleTagLength {
		t.Errorf("validErr.Rule = %v, want %v", validErr.Rule, RuleTagLength)
	}
}

func TestDefaultSkillValidator_ValidateTags_InvalidFormat(t *testing.T) {
	validator := NewSkillValidator()

	testCases := [][]domain.Tag{
		{"InvalidTag"},
		{"invalid-tag"},
		{"invalid tag"},
		{"123_invalid"},
		{"_invalid"},
		{"invalid!"},
	}

	for _, tags := range testCases {
		t.Run(string(tags[0]), func(t *testing.T) {
			err := validator.ValidateTags(tags)
			if err == nil {
				t.Errorf("ValidateTags() should error for invalid tag: %s", tags[0])
			}

			var validErr *ValidationError
			if !errors.As(err, &validErr) {
				t.Errorf("expected ValidationError, got %T", err)
			}

			if validErr.Rule != RuleTagFormat {
				t.Errorf("validErr.Rule = %v, want %v", validErr.Rule, RuleTagFormat)
			}
		})
	}
}

func TestDefaultSkillValidator_ValidateTags_Duplicate(t *testing.T) {
	validator := NewSkillValidator()

	tags := []domain.Tag{"tag1", "tag2", "tag1"}

	err := validator.ValidateTags(tags)
	if err == nil {
		t.Error("ValidateTags() should error for duplicate tags")
	}

	var validErr *ValidationError
	if !errors.As(err, &validErr) {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if validErr.Rule != RuleTagFormat {
		t.Errorf("validErr.Rule = %v, want %v", validErr.Rule, RuleTagFormat)
	}
}

func TestDefaultSkillValidator_ValidateTags_Valid(t *testing.T) {
	validator := NewSkillValidator()

	testCases := [][]domain.Tag{
		{"web_scrape"},
		{"web_scrape", "data_analysis"},
		{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"},
	}

	for i, tags := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			err := validator.ValidateTags(tags)
			if err != nil {
				t.Errorf("ValidateTags() failed for valid tags %v: %v", tags, err)
			}
		})
	}
}

func TestDefaultSkillValidator_ValidateTools_TooMany(t *testing.T) {
	validator := NewSkillValidator()

	tools := make([]domain.ToolRef, 21)
	for i := range tools {
		tools[i] = domain.ToolRef("tool" + string(rune('0'+i)))
	}

	err := validator.ValidateTools(tools)
	if err == nil {
		t.Error("ValidateTools() should error for too many tools")
	}

	var validErr *ValidationError
	if !errors.As(err, &validErr) {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if validErr.Rule != RuleToolCount {
		t.Errorf("validErr.Rule = %v, want %v", validErr.Rule, RuleToolCount)
	}
}

func TestDefaultSkillValidator_ValidateTools_EmptyTool(t *testing.T) {
	validator := NewSkillValidator()

	tools := []domain.ToolRef{"valid_tool", ""}

	err := validator.ValidateTools(tools)
	if err == nil {
		t.Error("ValidateTools() should error for empty tool")
	}

	var validErr *ValidationError
	if !errors.As(err, &validErr) {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if validErr.Rule != RuleToolFormat {
		t.Errorf("validErr.Rule = %v, want %v", validErr.Rule, RuleToolFormat)
	}
}

func TestDefaultSkillValidator_ValidateTools_TooLong(t *testing.T) {
	validator := NewSkillValidator()

	longTool := string(make([]byte, 101))
	for i := range longTool {
		longTool = longTool[:i] + "a" + longTool[i+1:]
	}

	tools := []domain.ToolRef{domain.ToolRef(longTool)}

	err := validator.ValidateTools(tools)
	if err == nil {
		t.Error("ValidateTools() should error for too long tool")
	}

	var validErr *ValidationError
	if !errors.As(err, &validErr) {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if validErr.Rule != RuleToolLength {
		t.Errorf("validErr.Rule = %v, want %v", validErr.Rule, RuleToolLength)
	}
}

func TestDefaultSkillValidator_ValidateTools_InvalidFormat(t *testing.T) {
	validator := NewSkillValidator()

	testCases := [][]domain.ToolRef{
		{"InvalidTool"},
		{"invalid-tool"},
		{"invalid tool"},
		{"123_invalid"},
		{"_invalid"},
		{"invalid!"},
	}

	for _, tools := range testCases {
		t.Run(string(tools[0]), func(t *testing.T) {
			err := validator.ValidateTools(tools)
			if err == nil {
				t.Errorf("ValidateTools() should error for invalid tool: %s", tools[0])
			}

			var validErr *ValidationError
			if !errors.As(err, &validErr) {
				t.Errorf("expected ValidationError, got %T", err)
			}

			if validErr.Rule != RuleToolFormat {
				t.Errorf("validErr.Rule = %v, want %v", validErr.Rule, RuleToolFormat)
			}
		})
	}
}

func TestDefaultSkillValidator_ValidateTools_Duplicate(t *testing.T) {
	validator := NewSkillValidator()

	tools := []domain.ToolRef{"tool1", "tool2", "tool1"}

	err := validator.ValidateTools(tools)
	if err == nil {
		t.Error("ValidateTools() should error for duplicate tools")
	}

	var validErr *ValidationError
	if !errors.As(err, &validErr) {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if validErr.Rule != RuleToolFormat {
		t.Errorf("validErr.Rule = %v, want %v", validErr.Rule, RuleToolFormat)
	}
}

func TestDefaultSkillValidator_ValidateTools_Valid(t *testing.T) {
	validator := NewSkillValidator()

	testCases := [][]domain.ToolRef{
		{},
		{"search_query"},
		{"search_query", "content_extractor"},
	}

	for i, tools := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			err := validator.ValidateTools(tools)
			if err != nil {
				t.Errorf("ValidateTools() failed for valid tools %v: %v", tools, err)
			}
		})
	}
}

func TestDefaultSkillValidator_ValidateVersion_InvalidFormat(t *testing.T) {
	validator := NewSkillValidator()

	testCases := []string{
		"1",
		"1.0",
		"v1.0.0",
		"1.0.0-beta",
		"invalid",
		"a.b.c",
	}

	for _, version := range testCases {
		t.Run(version, func(t *testing.T) {
			err := validator.ValidateVersion(version)
			if err == nil {
				t.Errorf("ValidateVersion() should error for invalid version: %s", version)
			}

			var validErr *ValidationError
			if !errors.As(err, &validErr) {
				t.Errorf("expected ValidationError, got %T", err)
			}

			if validErr.Rule != RuleVersionFormat {
				t.Errorf("validErr.Rule = %v, want %v", validErr.Rule, RuleVersionFormat)
			}
		})
	}
}

func TestDefaultSkillValidator_ValidateVersion_Empty(t *testing.T) {
	validator := NewSkillValidator()

	err := validator.ValidateVersion("")
	if err != nil {
		t.Errorf("ValidateVersion() failed for empty version: %v", err)
	}
}

func TestDefaultSkillValidator_ValidateVersion_Valid(t *testing.T) {
	validator := NewSkillValidator()

	testCases := []string{
		"1.0.0",
		"0.0.1",
		"10.20.30",
		"999.999.999",
	}

	for _, version := range testCases {
		t.Run(version, func(t *testing.T) {
			err := validator.ValidateVersion(version)
			if err != nil {
				t.Errorf("ValidateVersion() failed for valid version %s: %v", version, err)
			}
		})
	}
}

func TestDefaultSkillValidator_ValidateMetadata_InvalidComplexity(t *testing.T) {
	validator := NewSkillValidator()

	metadata := domain.SkillMetadata{
		Complexity: "invalid",
	}

	err := validator.ValidateMetadata(metadata)
	if err == nil {
		t.Error("ValidateMetadata() should error for invalid complexity")
	}

	var validErr *ValidationError
	if !errors.As(err, &validErr) {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if validErr.Rule != RuleMetadataInvalid {
		t.Errorf("validErr.Rule = %v, want %v", validErr.Rule, RuleMetadataInvalid)
	}
}

func TestDefaultSkillValidator_ValidateMetadata_EmptyPermission(t *testing.T) {
	validator := NewSkillValidator()

	metadata := domain.SkillMetadata{
		Permissions: []string{"read", "", "write"},
	}

	err := validator.ValidateMetadata(metadata)
	if err == nil {
		t.Error("ValidateMetadata() should error for empty permission")
	}

	var validErr *ValidationError
	if !errors.As(err, &validErr) {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if validErr.Rule != RuleMetadataInvalid {
		t.Errorf("validErr.Rule = %v, want %v", validErr.Rule, RuleMetadataInvalid)
	}
}

func TestDefaultSkillValidator_ValidateMetadata_Valid(t *testing.T) {
	validator := NewSkillValidator()

	testCases := []domain.SkillMetadata{
		{},
		{
			Category:   "testing",
			Complexity: "simple",
		},
		{
			Category:    "testing",
			Complexity:  "medium",
			Permissions: []string{"read", "write"},
		},
		{
			Category:    "testing",
			Complexity:  "complex",
			Permissions: []string{"read", "write", "execute"},
		},
	}

	for i, metadata := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			err := validator.ValidateMetadata(metadata)
			if err != nil {
				t.Errorf("ValidateMetadata() failed for valid metadata %v: %v", metadata, err)
			}
		})
	}
}

func TestValidationError_Error_WithDetails(t *testing.T) {
	err := &ValidationError{
		Field:   "name",
		Message: "name is too short",
		Rule:    RuleNameLength,
		Details: map[string]string{
			"min":    "3",
			"actual": "2",
		},
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "NAME_LENGTH: name - name is too short") {
		t.Errorf("Error() = %v, should contain NAME_LENGTH: name - name is too short", errStr)
	}
	if !strings.Contains(errStr, "actual=2") {
		t.Errorf("Error() = %v, should contain actual=2", errStr)
	}
	if !strings.Contains(errStr, "min=3") {
		t.Errorf("Error() = %v, should contain min=3", errStr)
	}
}

func TestValidationError_Error_NoDetails(t *testing.T) {
	err := &ValidationError{
		Field:   "name",
		Message: "name is required",
		Rule:    RuleRequiredName,
	}

	want := "REQUIRED_NAME: name - name is required"
	if err.Error() != want {
		t.Errorf("Error() = %v, want %v", err.Error(), want)
	}
}
