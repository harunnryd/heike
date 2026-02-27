package parser

import (
	"errors"
	"testing"

	"github.com/harunnryd/heike/internal/skill/domain"
)

func TestYAMLFrontmatterParser_Parse_Valid(t *testing.T) {
	parser := NewYAMLFrontmatterParser()

	content := `---
name: test_skill
description: Test skill for parsing
tags:
  - web_scrape
  - data_analysis
tools:
  - search_query
  - content_extractor
version: 1.0.0
author: test_author
metadata:
  category: testing
  complexity: simple
  permissions:
    - read
    - write
---

This is the skill content body.`

	skill, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	if skill.Name != "test_skill" {
		t.Errorf("skill.Name = %v, want test_skill", skill.Name)
	}

	if skill.Description != "Test skill for parsing" {
		t.Errorf("skill.Description = %v, want Test skill for parsing", skill.Description)
	}

	if len(skill.Tags) != 2 {
		t.Errorf("len(skill.Tags) = %v, want 2", len(skill.Tags))
	}

	if len(skill.Tools) != 2 {
		t.Errorf("len(skill.Tools) = %v, want 2", len(skill.Tools))
	}

	if skill.Version != "1.0.0" {
		t.Errorf("skill.Version = %v, want 1.0.0", skill.Version)
	}

	if skill.Author != "test_author" {
		t.Errorf("skill.Author = %v, want test_author", skill.Author)
	}

	if skill.Content != "This is the skill content body." {
		t.Errorf("skill.Content = %v, want This is the skill content body.", skill.Content)
	}

	if skill.Metadata.Category != "testing" {
		t.Errorf("skill.Metadata.Category = %v, want testing", skill.Metadata.Category)
	}

	if skill.Metadata.Complexity != "simple" {
		t.Errorf("skill.Metadata.Complexity = %v, want simple", skill.Metadata.Complexity)
	}

	if len(skill.Metadata.Permissions) != 2 {
		t.Errorf("len(skill.Metadata.Permissions) = %v, want 2", len(skill.Metadata.Permissions))
	}
}

func TestYAMLFrontmatterParser_Parse_Minimal(t *testing.T) {
	parser := NewYAMLFrontmatterParser()

	content := `---
name: minimal_skill
description: Minimal skill
tags:
  - basic
---

Content here.`

	skill, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	if skill.Name != "minimal_skill" {
		t.Errorf("skill.Name = %v, want minimal_skill", skill.Name)
	}

	if skill.Content != "Content here." {
		t.Errorf("skill.Content = %v, want Content here.", skill.Content)
	}

	if skill.ID == "" {
		t.Error("skill.ID should be auto-generated from name")
	}

	if skill.ID != domain.SkillID("minimal_skill") {
		t.Errorf("skill.ID = %v, want minimal_skill", skill.ID)
	}
}

func TestYAMLFrontmatterParser_Parse_NoFrontmatter(t *testing.T) {
	parser := NewYAMLFrontmatterParser()

	content := "No frontmatter here"

	_, err := parser.Parse(content)
	if err == nil {
		t.Error("Parse() should error for missing frontmatter")
	}

	var parseErr *ParseError
	if !isParseError(err, &parseErr) {
		t.Errorf("expected ParseError, got %T", err)
	}

	if parseErr.Code != CodeMissingFrontmatter {
		t.Errorf("parseErr.Code = %v, want %v", parseErr.Code, CodeMissingFrontmatter)
	}
}

func TestYAMLFrontmatterParser_Parse_MissingEndDelimiter(t *testing.T) {
	parser := NewYAMLFrontmatterParser()

	content := `---
name: test_skill
description: Test
tags:
  - test
Content here without end delimiter`

	_, err := parser.Parse(content)
	if err == nil {
		t.Error("Parse() should error for missing end delimiter")
	}

	var parseErr *ParseError
	if !isParseError(err, &parseErr) {
		t.Errorf("expected ParseError, got %T", err)
	}

	if parseErr.Code != CodeMissingFrontmatter {
		t.Errorf("parseErr.Code = %v, want %v", parseErr.Code, CodeMissingFrontmatter)
	}
}

func TestYAMLFrontmatterParser_Parse_InvalidYAML(t *testing.T) {
	parser := NewYAMLFrontmatterParser()

	content := `---
name: test_skill
description: Test
tags:
  - test
invalid yaml: [unclosed bracket
---

Content here.`

	_, err := parser.Parse(content)
	if err == nil {
		t.Error("Parse() should error for invalid YAML")
	}

	var parseErr *ParseError
	if !isParseError(err, &parseErr) {
		t.Errorf("expected ParseError, got %T", err)
	}

	if parseErr.Code != CodeInvalidYAML {
		t.Errorf("parseErr.Code = %v, want %v", parseErr.Code, CodeInvalidYAML)
	}
}

func TestYAMLFrontmatterParser_Parse_WithID(t *testing.T) {
	parser := NewYAMLFrontmatterParser()

	content := `---
id: custom_id
name: test_skill
description: Test
tags:
  - test
---

Content here.`

	skill, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	if skill.ID != domain.SkillID("custom_id") {
		t.Errorf("skill.ID = %v, want custom_id", skill.ID)
	}
}

func TestYAMLFrontmatterParser_Validate_Valid(t *testing.T) {
	parser := NewYAMLFrontmatterParser()

	content := `---
name: valid_skill
description: Valid skill description
tags:
  - web_scrape
  - data_analysis
tools:
  - search_query
version: 1.0.0
author: test_author
---

Content here.`

	err := parser.Validate(content)
	if err != nil {
		t.Errorf("Validate() failed: %v", err)
	}
}

func TestYAMLFrontmatterParser_Validate_InvalidName(t *testing.T) {
	parser := NewYAMLFrontmatterParser()

	content := `---
name: "invalid name!"
description: Test
tags:
  - test
---

Content here.`

	err := parser.Validate(content)
	if err == nil {
		t.Error("Validate() should error for invalid name")
	}

	var validErr *domain.SkillValidationError
	if !isValidationError(err, &validErr) {
		t.Errorf("expected SkillValidationError, got %T", err)
	}
}

func TestYAMLFrontmatterParser_Validate_EmptyTags(t *testing.T) {
	parser := NewYAMLFrontmatterParser()

	content := `---
name: test_skill
description: Test
---

Content here.`

	err := parser.Validate(content)
	if err == nil {
		t.Error("Validate() should error for empty tags")
	}

	var validErr *domain.SkillValidationError
	if !isValidationError(err, &validErr) {
		t.Errorf("expected SkillValidationError, got %T", err)
	}
}

func TestYAMLFrontmatterParser_Parse_EmptyBody(t *testing.T) {
	parser := NewYAMLFrontmatterParser()

	content := `---
name: test_skill
description: Test
tags:
  - test
---
`

	skill, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	if skill.Content != "" {
		t.Errorf("skill.Content = %v, want empty string", skill.Content)
	}
}

func TestYAMLFrontmatterParser_Parse_WithTrailingWhitespace(t *testing.T) {
	parser := NewYAMLFrontmatterParser()

	content := `---
name: test_skill
description: Test
tags:
  - test
---

Content here.
  `

	skill, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	expectedContent := "Content here.\n  "
	if skill.Content != expectedContent {
		t.Errorf("skill.Content = %q, want %q", skill.Content, expectedContent)
	}
}

func TestParseError_Error(t *testing.T) {
	err := &ParseError{
		Line:    10,
		Message: "test error",
		Code:    CodeInvalidYAML,
	}

	want := "INVALID_YAML at line 10: test error"
	if err.Error() != want {
		t.Errorf("Error() = %v, want %v", err.Error(), want)
	}
}

func TestParseError_Error_NoLine(t *testing.T) {
	err := &ParseError{
		Message: "test error",
		Code:    CodeMissingFrontmatter,
	}

	want := "MISSING_FRONTMATTER: test error"
	if err.Error() != want {
		t.Errorf("Error() = %v, want %v", err.Error(), want)
	}
}

func isParseError(err error, parseErr **ParseError) bool {
	return errors.As(err, parseErr)
}

func isValidationError(err error, validErr **domain.SkillValidationError) bool {
	return errors.As(err, validErr)
}
