package domain

import (
	"testing"
)

func TestSkillID_String(t *testing.T) {
	id := SkillID("test-skill")
	if id.String() != "test-skill" {
		t.Errorf("String() = %v, want test-skill", id.String())
	}
}

func TestSkillID_IsValid(t *testing.T) {
	tests := []struct {
		id    SkillID
		valid bool
	}{
		{"test", true},
		{"", false},
		{"skill-123", true},
	}

	for _, tt := range tests {
		if tt.id.IsValid() != tt.valid {
			t.Errorf("SkillID(%q).IsValid() = %v, want %v", tt.id, tt.id.IsValid(), tt.valid)
		}
	}
}

func TestTag_String(t *testing.T) {
	tag := Tag("web_scrape")
	if tag.String() != "web_scrape" {
		t.Errorf("String() = %v, want web_scrape", tag.String())
	}
}

func TestTag_IsValid(t *testing.T) {
	tests := []struct {
		tag   Tag
		valid bool
	}{
		{"web_scrape", true},
		{"", false},
		{"data_analysis", true},
	}

	for _, tt := range tests {
		if tt.tag.IsValid() != tt.valid {
			t.Errorf("Tag(%q).IsValid() = %v, want %v", tt.tag, tt.tag.IsValid(), tt.valid)
		}
	}
}

func TestToolRef_String(t *testing.T) {
	tool := ToolRef("http_client")
	if tool.String() != "http_client" {
		t.Errorf("String() = %v, want http_client", tool.String())
	}
}

func TestToolRef_IsValid(t *testing.T) {
	tests := []struct {
		tool  ToolRef
		valid bool
	}{
		{"http_client", true},
		{"", false},
		{"file_io", true},
	}

	for _, tt := range tests {
		if tt.tool.IsValid() != tt.valid {
			t.Errorf("ToolRef(%q).IsValid() = %v, want %v", tt.tool, tt.tool.IsValid(), tt.valid)
		}
	}
}
