package domain

import (
	"testing"
)

func TestValidateSkillName_Empty(t *testing.T) {
	err := ValidateSkillName("")
	if err == nil {
		t.Error("ValidateSkillName('') should error")
	}
	ve, ok := err.(*SkillValidationError)
	if !ok {
		t.Fatal("error should be *SkillValidationError")
	}
	if ve.Code != CodeMissingField {
		t.Errorf("error code = %v, want %v", ve.Code, CodeMissingField)
	}
}

func TestValidateSkillName_TooShort(t *testing.T) {
	err := ValidateSkillName("ab")
	if err == nil {
		t.Error("ValidateSkillName('ab') should error")
	}
	ve, ok := err.(*SkillValidationError)
	if !ok {
		t.Fatal("error should be *SkillValidationError")
	}
	if ve.Code != CodeInvalidLength {
		t.Errorf("error code = %v, want %v", ve.Code, CodeInvalidLength)
	}
}

func TestValidateSkillName_TooLong(t *testing.T) {
	longName := string(make([]byte, 101))
	err := ValidateSkillName(longName)
	if err == nil {
		t.Error("ValidateSkillName(101 chars) should error")
	}
	ve, ok := err.(*SkillValidationError)
	if !ok {
		t.Fatal("error should be *SkillValidationError")
	}
	if ve.Code != CodeInvalidLength {
		t.Errorf("error code = %v, want %v", ve.Code, CodeInvalidLength)
	}
}

func TestValidateSkillName_InvalidFormat(t *testing.T) {
	err := ValidateSkillName("invalid name!")
	if err == nil {
		t.Error("ValidateSkillName('invalid name!') should error")
	}
	ve, ok := err.(*SkillValidationError)
	if !ok {
		t.Fatal("error should be *SkillValidationError")
	}
	if ve.Code != CodeInvalidFormat {
		t.Errorf("error code = %v, want %v", ve.Code, CodeInvalidFormat)
	}
}

func TestValidateSkillName_Valid(t *testing.T) {
	tests := []string{
		"valid_name",
		"valid-name",
		"valid_name123",
		"VALID-NAME",
	}

	for _, name := range tests {
		err := ValidateSkillName(name)
		if err != nil {
			t.Errorf("ValidateSkillName(%q) should not error, got: %v", name, err)
		}
	}
}

func TestValidateSkillDescription_Empty(t *testing.T) {
	err := ValidateSkillDescription("")
	if err == nil {
		t.Error("ValidateSkillDescription('') should error")
	}
	ve, ok := err.(*SkillValidationError)
	if !ok {
		t.Fatal("error should be *SkillValidationError")
	}
	if ve.Code != CodeMissingField {
		t.Errorf("error code = %v, want %v", ve.Code, CodeMissingField)
	}
}

func TestValidateSkillDescription_TooLong(t *testing.T) {
	longDesc := string(make([]byte, 1001))
	err := ValidateSkillDescription(longDesc)
	if err == nil {
		t.Error("ValidateSkillDescription(1001 chars) should error")
	}
	ve, ok := err.(*SkillValidationError)
	if !ok {
		t.Fatal("error should be *SkillValidationError")
	}
	if ve.Code != CodeInvalidLength {
		t.Errorf("error code = %v, want %v", ve.Code, CodeInvalidLength)
	}
}

func TestValidateSkillDescription_Valid(t *testing.T) {
	err := ValidateSkillDescription("Valid description")
	if err != nil {
		t.Errorf("ValidateSkillDescription('Valid description') should not error, got: %v", err)
	}
}

func TestValidateSkillTags_Empty(t *testing.T) {
	err := ValidateSkillTags([]Tag{})
	if err == nil {
		t.Error("ValidateSkillTags([]) should error")
	}
	ve, ok := err.(*SkillValidationError)
	if !ok {
		t.Fatal("error should be *SkillValidationError")
	}
	if ve.Code != CodeMissingField {
		t.Errorf("error code = %v, want %v", ve.Code, CodeMissingField)
	}
}

func TestValidateSkillTags_EmptyTag(t *testing.T) {
	err := ValidateSkillTags([]Tag{"", "tag2"})
	if err == nil {
		t.Error("ValidateSkillTags(['', 'tag2']) should error")
	}
	ve, ok := err.(*SkillValidationError)
	if !ok {
		t.Fatal("error should be *SkillValidationError")
	}
	if ve.Code != CodeInvalidValue {
		t.Errorf("error code = %v, want %v", ve.Code, CodeInvalidValue)
	}
}

func TestValidateSkillTags_TooLong(t *testing.T) {
	longTag := string(make([]byte, 51))
	err := ValidateSkillTags([]Tag{Tag(longTag)})
	if err == nil {
		t.Error("ValidateSkillTags([51 chars]) should error")
	}
	ve, ok := err.(*SkillValidationError)
	if !ok {
		t.Fatal("error should be *SkillValidationError")
	}
	if ve.Code != CodeInvalidLength {
		t.Errorf("error code = %v, want %v", ve.Code, CodeInvalidLength)
	}
}

func TestValidateSkillTags_Valid(t *testing.T) {
	err := ValidateSkillTags([]Tag{"tag1", "tag2"})
	if err != nil {
		t.Errorf("ValidateSkillTags(['tag1', 'tag2']) should not error, got: %v", err)
	}
}

func TestValidateSkillTools_Empty(t *testing.T) {
	err := ValidateSkillTools([]ToolRef{})
	if err != nil {
		t.Errorf("ValidateSkillTools([]) should not error, got: %v", err)
	}
}

func TestValidateSkillTools_EmptyTool(t *testing.T) {
	err := ValidateSkillTools([]ToolRef{"", "tool2"})
	if err == nil {
		t.Error("ValidateSkillTools(['', 'tool2']) should error")
	}
	ve, ok := err.(*SkillValidationError)
	if !ok {
		t.Fatal("error should be *SkillValidationError")
	}
	if ve.Code != CodeInvalidValue {
		t.Errorf("error code = %v, want %v", ve.Code, CodeInvalidValue)
	}
}

func TestValidateSkillTools_Duplicate(t *testing.T) {
	err := ValidateSkillTools([]ToolRef{"tool1", "tool1"})
	if err == nil {
		t.Error("ValidateSkillTools(['tool1', 'tool1']) should error")
	}
	ve, ok := err.(*SkillValidationError)
	if !ok {
		t.Fatal("error should be *SkillValidationError")
	}
	if ve.Code != CodeInvalidValue {
		t.Errorf("error code = %v, want %v", ve.Code, CodeInvalidValue)
	}
}

func TestValidateSkillTools_Valid(t *testing.T) {
	err := ValidateSkillTools([]ToolRef{"tool1", "tool2"})
	if err != nil {
		t.Errorf("ValidateSkillTools(['tool1', 'tool2']) should not error, got: %v", err)
	}
}

func TestValidateSkillVersion_InvalidFormat(t *testing.T) {
	tests := []string{
		"invalid",
		"1.0",
		"v1.0.0",
	}

	for _, version := range tests {
		err := ValidateSkillVersion(version)
		if err == nil {
			t.Errorf("ValidateSkillVersion(%q) should error", version)
		}
		ve, ok := err.(*SkillValidationError)
		if !ok {
			t.Fatal("error should be *SkillValidationError")
		}
		if ve.Code != CodeInvalidFormat {
			t.Errorf("error code = %v, want %v", ve.Code, CodeInvalidFormat)
		}
	}
}

func TestValidateSkillVersion_Empty(t *testing.T) {
	err := ValidateSkillVersion("")
	if err != nil {
		t.Errorf("ValidateSkillVersion('') should not error, got: %v", err)
	}
}

func TestValidateSkillVersion_Valid(t *testing.T) {
	tests := []string{
		"1.0.0",
		"0.1.0",
		"10.20.30",
	}

	for _, version := range tests {
		err := ValidateSkillVersion(version)
		if err != nil {
			t.Errorf("ValidateSkillVersion(%q) should not error, got: %v", version, err)
		}
	}
}

func TestValidateSkill_Valid(t *testing.T) {
	skill := &Skill{
		Name:        "valid_skill",
		Description: "A valid skill description",
		Tags:        []Tag{"tag1", "tag2"},
		Tools:       []ToolRef{"tool1"},
		Version:     "1.0.0",
	}

	err := ValidateSkill(skill)
	if err != nil {
		t.Errorf("ValidateSkill() should not error for valid skill, got: %v", err)
	}
}

func TestValidateSkill_InvalidName(t *testing.T) {
	skill := &Skill{
		Name:        "invalid name!",
		Description: "A skill",
		Tags:        []Tag{"tag1"},
	}

	err := ValidateSkill(skill)
	if err == nil {
		t.Error("ValidateSkill() should error for invalid name")
	}
	ve, ok := err.(*SkillValidationError)
	if !ok {
		t.Fatal("error should be *SkillValidationError")
	}
	if ve.Field != "name" {
		t.Errorf("error field = %v, want name", ve.Field)
	}
}

func TestValidateSkill_InvalidMetadata(t *testing.T) {
	skill := &Skill{
		Name:        "valid_skill",
		Description: "A skill",
		Tags:        []Tag{"tag1"},
		Metadata: SkillMetadata{
			Complexity: "invalid",
		},
	}

	err := ValidateSkill(skill)
	if err == nil {
		t.Error("ValidateSkill() should error for invalid metadata")
	}
	ve, ok := err.(*SkillValidationError)
	if !ok {
		t.Fatal("error should be *SkillValidationError")
	}
	if ve.Field != "metadata" {
		t.Errorf("error field = %v, want metadata", ve.Field)
	}
}
