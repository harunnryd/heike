package parser

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/harunnryd/heike/internal/skill/domain"
)

type YAMLFrontmatterParser struct {
	validator func(*domain.Skill) error
}

func NewYAMLFrontmatterParser() SkillParser {
	return &YAMLFrontmatterParser{
		validator: domain.ValidateSkill,
	}
}

func (p *YAMLFrontmatterParser) Parse(content string) (*domain.Skill, error) {
	metadata, body, err := p.extractFrontmatter(content)
	if err != nil {
		return nil, err
	}

	skill := p.mapToSkill(metadata)
	skill.Content = body

	if skill.ID == "" {
		skill.ID = domain.SkillID(skill.Name)
	}

	now := time.Now()
	if skill.CreatedAt.IsZero() {
		skill.CreatedAt = now
	}
	if skill.UpdatedAt.IsZero() {
		skill.UpdatedAt = now
	}

	return skill, nil
}

func (p *YAMLFrontmatterParser) Validate(content string) error {
	skill, err := p.Parse(content)
	if err != nil {
		return err
	}

	return p.validator(skill)
}

func (p *YAMLFrontmatterParser) extractFrontmatter(content string) (map[string]interface{}, string, error) {
	if !strings.HasPrefix(content, "---\n") {
		return nil, "", &ParseError{
			Code:    CodeMissingFrontmatter,
			Message: "frontmatter must start with ---",
		}
	}

	endIndex := strings.Index(content, "\n---\n")
	if endIndex == -1 {
		return nil, "", &ParseError{
			Code:    CodeMissingFrontmatter,
			Message: "frontmatter must end with ---",
		}
	}

	frontmatter := content[4 : endIndex+1]
	body := strings.TrimLeft(content[endIndex+5:], "\n")

	var metadata map[string]interface{}
	if err := yaml.Unmarshal([]byte(frontmatter), &metadata); err != nil {
		return nil, "", &ParseError{
			Code:     CodeInvalidYAML,
			Message:  "invalid YAML frontmatter",
			Original: err,
		}
	}

	return metadata, body, nil
}

func (p *YAMLFrontmatterParser) mapToSkill(metadata map[string]interface{}) *domain.Skill {
	skill := &domain.Skill{
		Tags:  []domain.Tag{},
		Tools: []domain.ToolRef{},
	}

	if name, ok := metadata["name"].(string); ok {
		skill.Name = name
	}

	if desc, ok := metadata["description"].(string); ok {
		skill.Description = desc
	}

	if id, ok := metadata["id"].(string); ok {
		skill.ID = domain.SkillID(id)
	}

	if version, ok := metadata["version"].(string); ok {
		skill.Version = version
	}

	if author, ok := metadata["author"].(string); ok {
		skill.Author = author
	}

	if tags, ok := metadata["tags"].([]interface{}); ok {
		for _, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				skill.Tags = append(skill.Tags, domain.Tag(tagStr))
			}
		}
	}

	if tools, ok := metadata["tools"].([]interface{}); ok {
		for _, tool := range tools {
			if toolStr, ok := tool.(string); ok {
				skill.Tools = append(skill.Tools, domain.ToolRef(toolStr))
			}
		}
	}

	if meta, ok := metadata["metadata"].(map[string]interface{}); ok {
		skill.Metadata = p.mapMetadata(meta)
	}

	return skill
}

func (p *YAMLFrontmatterParser) mapMetadata(meta map[string]interface{}) domain.SkillMetadata {
	metadata := domain.SkillMetadata{}

	if category, ok := meta["category"].(string); ok {
		metadata.Category = category
	}

	if complexity, ok := meta["complexity"].(string); ok {
		metadata.Complexity = complexity
	}

	if permissions, ok := meta["permissions"].([]interface{}); ok {
		for _, perm := range permissions {
			if permStr, ok := perm.(string); ok {
				metadata.Permissions = append(metadata.Permissions, permStr)
			}
		}
	}

	return metadata
}

func (p *YAMLFrontmatterParser) ParseFromBytes(data []byte) (*domain.Skill, error) {
	content := string(data)
	return p.Parse(content)
}

func (p *YAMLFrontmatterParser) ParseFromReader(reader *bytes.Reader) (*domain.Skill, error) {
	data := new(bytes.Buffer)
	data.ReadFrom(reader)
	return p.Parse(data.String())
}

func (p *YAMLFrontmatterParser) ParseErrorWithLine(err error, line int) *ParseError {
	if parseErr, ok := err.(*ParseError); ok {
		parseErr.Line = line
		return parseErr
	}

	return &ParseError{
		Line:     line,
		Message:  err.Error(),
		Code:     CodeInvalidYAML,
		Original: err,
	}
}

func (p *YAMLFrontmatterParser) FormatError(err error) string {
	if parseErr, ok := err.(*ParseError); ok {
		return parseErr.Error()
	}

	if err != nil {
		return fmt.Sprintf("parse error: %v", err)
	}

	return "unknown parse error"
}
