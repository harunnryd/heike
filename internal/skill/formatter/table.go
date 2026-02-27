package formatter

import (
	"strings"

	"github.com/harunnryd/heike/internal/skill/domain"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

type TableFormatter struct {
	headerStyle  lipgloss.Style
	cellStyle    lipgloss.Style
	oddRowStyle  lipgloss.Style
	evenRowStyle lipgloss.Style
	borderStyle  lipgloss.Style
}

func NewTableFormatter() *TableFormatter {
	purple := lipgloss.Color("99")
	gray := lipgloss.Color("245")
	lightGray := lipgloss.Color("241")

	return &TableFormatter{
		headerStyle: lipgloss.NewStyle().
			Foreground(purple).
			Bold(true).
			Align(lipgloss.Center).
			Padding(0, 1),
		cellStyle: lipgloss.NewStyle().
			Padding(0, 1).
			Width(20),
		oddRowStyle: lipgloss.NewStyle().
			Foreground(gray).
			Padding(0, 1).
			Width(20),
		evenRowStyle: lipgloss.NewStyle().
			Foreground(lightGray).
			Padding(0, 1).
			Width(20),
		borderStyle: lipgloss.NewStyle().
			Foreground(purple),
	}
}

func (f *TableFormatter) FormatSkills(skills []*domain.Skill) (string, error) {
	if len(skills) == 0 {
		return "No skills found", nil
	}

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(f.borderStyle).
		StyleFunc(func(row, col int) lipgloss.Style {
			switch {
			case row == table.HeaderRow:
				return f.headerStyle
			case row%2 == 0:
				return f.evenRowStyle
			default:
				return f.oddRowStyle
			}
		}).
		Headers("ID", "Name", "Tags", "Tools")

	for _, skill := range skills {
		tags := make([]string, len(skill.Tags))
		for i, tag := range skill.Tags {
			tags[i] = tag.String()
		}
		tools := make([]string, len(skill.Tools))
		for i, tool := range skill.Tools {
			tools[i] = tool.String()
		}

		t.Row(
			skill.ID.String(),
			truncateString(skill.Name, 20),
			truncateString(strings.Join(tags, ", "), 25),
			truncateString(strings.Join(tools, ", "), 25),
		)
	}

	return t.String(), nil
}

func (f *TableFormatter) FormatSkill(skill *domain.Skill) (string, error) {
	if skill == nil {
		return "No skill found", nil
	}

	tags := make([]string, len(skill.Tags))
	for i, tag := range skill.Tags {
		tags[i] = tag.String()
	}
	tools := make([]string, len(skill.Tools))
	for i, tool := range skill.Tools {
		tools[i] = tool.String()
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(f.borderStyle).
		StyleFunc(func(row, col int) lipgloss.Style {
			if col == 0 {
				return f.headerStyle
			}
			return f.cellStyle
		})

	t.Row("ID", skill.ID.String())
	t.Row("Name", skill.Name)
	t.Row("Description", truncateString(skill.Description, 60))
	t.Row("Tags", strings.Join(tags, ", "))
	t.Row("Tools", strings.Join(tools, ", "))

	return t.String(), nil
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
