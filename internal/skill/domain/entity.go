package domain

import "time"

type Skill struct {
	ID          SkillID       `yaml:"id"`
	Name        string        `yaml:"name"`
	Description string        `yaml:"description"`
	Tags        []Tag         `yaml:"tags"`
	Tools       []ToolRef     `yaml:"tools"`
	Content     string        `yaml:"-"`
	Version     string        `yaml:"version"`
	Author      string        `yaml:"author"`
	Metadata    SkillMetadata `yaml:"metadata"`
	CreatedAt   time.Time     `yaml:"-"`
	UpdatedAt   time.Time     `yaml:"-"`
}

type SkillID string

func (id SkillID) String() string {
	return string(id)
}

func (id SkillID) IsValid() bool {
	if id == "" {
		return false
	}
	return true
}

type Tag string

func (t Tag) String() string {
	return string(t)
}

func (t Tag) IsValid() bool {
	if t == "" {
		return false
	}
	return true
}

type ToolRef string

func (tr ToolRef) String() string {
	return string(tr)
}

func (tr ToolRef) IsValid() bool {
	if tr == "" {
		return false
	}
	return true
}
