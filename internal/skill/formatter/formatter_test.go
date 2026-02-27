package formatter

import (
	"testing"

	"github.com/harunnryd/heike/internal/skill/domain"
)

func TestFormatterFactory_Create(t *testing.T) {
	factory := NewFormatterFactory()

	tests := []struct {
		name    string
		format  OutputFormat
		wantErr bool
	}{
		{
			name:    "table format",
			format:  OutputFormatTable,
			wantErr: false,
		},
		{
			name:    "json format",
			format:  OutputFormatJSON,
			wantErr: false,
		},
		{
			name:    "yaml format",
			format:  OutputFormatYAML,
			wantErr: false,
		},
		{
			name:    "invalid format",
			format:  OutputFormat("invalid"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter, err := factory.Create(tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && formatter == nil {
				t.Error("Create() returned nil formatter for valid format")
			}
		})
	}
}

func TestParseOutputFormat(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    OutputFormat
		wantErr bool
	}{
		{
			name:    "table uppercase",
			input:   "TABLE",
			want:    OutputFormatTable,
			wantErr: false,
		},
		{
			name:    "table lowercase",
			input:   "table",
			want:    OutputFormatTable,
			wantErr: false,
		},
		{
			name:    "json uppercase",
			input:   "JSON",
			want:    OutputFormatJSON,
			wantErr: false,
		},
		{
			name:    "json lowercase",
			input:   "json",
			want:    OutputFormatJSON,
			wantErr: false,
		},
		{
			name:    "yaml uppercase",
			input:   "YAML",
			want:    OutputFormatYAML,
			wantErr: false,
		},
		{
			name:    "yaml lowercase",
			input:   "yaml",
			want:    OutputFormatYAML,
			wantErr: false,
		},
		{
			name:    "invalid format",
			input:   "invalid",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseOutputFormat(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseOutputFormat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseOutputFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTableFormatter_FormatSkills(t *testing.T) {
	formatter := NewTableFormatter()

	skills := []*domain.Skill{
		{
			ID:          "skill-1",
			Name:        "Skill One",
			Description: "Test skill one",
			Tags:        []domain.Tag{"tag1", "tag2"},
			Tools:       []domain.ToolRef{"tool1", "tool2"},
		},
		{
			ID:          "skill-2",
			Name:        "Skill Two",
			Description: "Test skill two",
			Tags:        []domain.Tag{"tag3"},
			Tools:       []domain.ToolRef{"tool3"},
		},
	}

	output, err := formatter.FormatSkills(skills)
	if err != nil {
		t.Fatalf("FormatSkills() error = %v", err)
	}

	if output == "" {
		t.Error("FormatSkills() returned empty output")
	}

	if !containsString(output, "skill-1") || !containsString(output, "skill-2") {
		t.Error("FormatSkills() output missing skill IDs")
	}

	if !containsString(output, "Skill One") || !containsString(output, "Skill Two") {
		t.Error("FormatSkills() output missing skill names")
	}
}

func TestTableFormatter_FormatSkills_Empty(t *testing.T) {
	formatter := NewTableFormatter()

	output, err := formatter.FormatSkills([]*domain.Skill{})
	if err != nil {
		t.Fatalf("FormatSkills() error = %v", err)
	}

	if output != "No skills found" {
		t.Errorf("FormatSkills() = %v, want 'No skills found'", output)
	}
}

func TestTableFormatter_FormatSkill(t *testing.T) {
	formatter := NewTableFormatter()

	skill := &domain.Skill{
		ID:          "skill-1",
		Name:        "Skill One",
		Description: "Test skill one",
		Tags:        []domain.Tag{"tag1", "tag2"},
		Tools:       []domain.ToolRef{"tool1", "tool2"},
	}

	output, err := formatter.FormatSkill(skill)
	if err != nil {
		t.Fatalf("FormatSkill() error = %v", err)
	}

	if output == "" {
		t.Error("FormatSkill() returned empty output")
	}

	if !containsString(output, "skill-1") {
		t.Error("FormatSkill() output missing skill ID")
	}

	if !containsString(output, "Skill One") {
		t.Error("FormatSkill() output missing skill name")
	}
}

func TestTableFormatter_FormatSkill_Nil(t *testing.T) {
	formatter := NewTableFormatter()

	output, err := formatter.FormatSkill(nil)
	if err != nil {
		t.Fatalf("FormatSkill() error = %v", err)
	}

	if output != "No skill found" {
		t.Errorf("FormatSkill() = %v, want 'No skill found'", output)
	}
}

func TestJSONFormatter_FormatSkills(t *testing.T) {
	formatter := NewJSONFormatter()

	skills := []*domain.Skill{
		{
			ID:          "skill-1",
			Name:        "Skill One",
			Description: "Test skill one",
			Tags:        []domain.Tag{"tag1"},
			Tools:       []domain.ToolRef{"tool1"},
		},
	}

	output, err := formatter.FormatSkills(skills)
	if err != nil {
		t.Fatalf("FormatSkills() error = %v", err)
	}

	if output == "" {
		t.Error("FormatSkills() returned empty output")
	}

	if !containsString(output, "skill-1") {
		t.Error("FormatSkills() output missing skill ID")
	}
}

func TestJSONFormatter_FormatSkill(t *testing.T) {
	formatter := NewJSONFormatter()

	skill := &domain.Skill{
		ID:          "skill-1",
		Name:        "Skill One",
		Description: "Test skill one",
		Tags:        []domain.Tag{"tag1"},
		Tools:       []domain.ToolRef{"tool1"},
	}

	output, err := formatter.FormatSkill(skill)
	if err != nil {
		t.Fatalf("FormatSkill() error = %v", err)
	}

	if output == "" {
		t.Error("FormatSkill() returned empty output")
	}
}

func TestJSONFormatter_FormatSkill_Nil(t *testing.T) {
	formatter := NewJSONFormatter()

	output, err := formatter.FormatSkill(nil)
	if err != nil {
		t.Fatalf("FormatSkill() error = %v", err)
	}

	if output != "null" {
		t.Errorf("FormatSkill() = %v, want 'null'", output)
	}
}

func TestYAMLFormatter_FormatSkills(t *testing.T) {
	formatter := NewYAMLFormatter()

	skills := []*domain.Skill{
		{
			ID:          "skill-1",
			Name:        "Skill One",
			Description: "Test skill one",
			Tags:        []domain.Tag{"tag1"},
			Tools:       []domain.ToolRef{"tool1"},
		},
	}

	output, err := formatter.FormatSkills(skills)
	if err != nil {
		t.Fatalf("FormatSkills() error = %v", err)
	}

	if output == "" {
		t.Error("FormatSkills() returned empty output")
	}

	if !containsString(output, "skill-1") {
		t.Error("FormatSkills() output missing skill ID")
	}
}

func TestYAMLFormatter_FormatSkill(t *testing.T) {
	formatter := NewYAMLFormatter()

	skill := &domain.Skill{
		ID:          "skill-1",
		Name:        "Skill One",
		Description: "Test skill one",
		Tags:        []domain.Tag{"tag1"},
		Tools:       []domain.ToolRef{"tool1"},
	}

	output, err := formatter.FormatSkill(skill)
	if err != nil {
		t.Fatalf("FormatSkill() error = %v", err)
	}

	if output == "" {
		t.Error("FormatSkill() returned empty output")
	}
}

func TestYAMLFormatter_FormatSkill_Nil(t *testing.T) {
	formatter := NewYAMLFormatter()

	output, err := formatter.FormatSkill(nil)
	if err != nil {
		t.Fatalf("FormatSkill() error = %v", err)
	}

	if output != "null" {
		t.Errorf("FormatSkill() = %v, want 'null'", output)
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short string",
			input:    "hello",
			maxLen:   20,
			expected: "hello",
		},
		{
			name:     "exact length",
			input:    "hello world",
			maxLen:   11,
			expected: "hello world",
		},
		{
			name:     "too long",
			input:    "hello world test",
			maxLen:   10,
			expected: "hello w...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			len(s) > len(substr) && containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
