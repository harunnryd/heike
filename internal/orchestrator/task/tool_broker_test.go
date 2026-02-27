package task

import (
	"testing"

	"github.com/harunnryd/heike/internal/model/contract"
	"github.com/harunnryd/heike/internal/tool"
	_ "github.com/harunnryd/heike/internal/tool/builtin"

	"github.com/stretchr/testify/assert"
)

func TestBuildCapabilityCatalog_DeterministicAndUnique(t *testing.T) {
	descriptors := []tool.ToolDescriptor{
		{Definition: contract.ToolDef{Name: "search_query", Description: "Search the web"}},
		{Definition: contract.ToolDef{Name: "SEARCH_QUERY", Description: "duplicate"}},
		{Definition: contract.ToolDef{Name: "open", Description: "Open URL"}},
	}

	catalog := BuildCapabilityCatalog(descriptors)
	caps := catalog.Capabilities()

	if assert.Len(t, caps, 2) {
		assert.Equal(t, "open", caps[0].Name)
		assert.Equal(t, "search_query", caps[1].Name)
	}
}

func TestBuildCapabilityCatalog_InferBuiltinSource(t *testing.T) {
	descriptors := []tool.ToolDescriptor{
		{Definition: contract.ToolDef{Name: "exec_command", Description: "Run a command"}},
	}

	catalog := BuildCapabilityCatalog(descriptors)
	capability, ok := catalog.Find("exec_command")
	if assert.True(t, ok) {
		assert.Equal(t, "builtin", capability.Source)
	}
}

func TestDefaultToolBroker_SelectsRelevantWebTools(t *testing.T) {
	tools := []tool.ToolDescriptor{
		{
			Definition: contract.ToolDef{Name: "open", Description: "Open web pages"},
			Metadata: tool.ToolMetadata{
				Capabilities: []string{"web.fetch", "research.web"},
				Risk:         tool.RiskMedium,
			},
		},
		{
			Definition: contract.ToolDef{Name: "search_query", Description: "Search the web"},
			Metadata: tool.ToolMetadata{
				Capabilities: []string{"web.search", "research.web"},
				Risk:         tool.RiskMedium,
			},
		},
		{
			Definition: contract.ToolDef{Name: "exec_command", Description: "Execute shell command"},
			Metadata: tool.ToolMetadata{
				Capabilities: []string{"exec.command"},
				Risk:         tool.RiskHigh,
			},
		},
		{
			Definition: contract.ToolDef{Name: "time", Description: "Get current time"},
			Metadata: tool.ToolMetadata{
				Capabilities: []string{"time.query"},
				Risk:         tool.RiskLow,
			},
		},
	}

	broker := NewDefaultToolBroker(2)
	selected := broker.Select("Research AI trends on the web and list sources", tools)

	if assert.Len(t, selected, 2) {
		names := []string{selected[0].Definition.Name, selected[1].Definition.Name}
		assert.Contains(t, names, "search_query")
		assert.Contains(t, names, "open")
	}
}

func TestDefaultToolBroker_SelectionMetadataContainsReasons(t *testing.T) {
	tools := []tool.ToolDescriptor{
		{
			Definition: contract.ToolDef{Name: "search_query", Description: "Search web content"},
			Metadata: tool.ToolMetadata{
				Capabilities: []string{"web.search", "research.web"},
				Risk:         tool.RiskMedium,
			},
		},
		{
			Definition: contract.ToolDef{Name: "open", Description: "Open web pages"},
			Metadata: tool.ToolMetadata{
				Capabilities: []string{"web.fetch"},
				Risk:         tool.RiskMedium,
			},
		},
		{
			Definition: contract.ToolDef{Name: "time", Description: "Get current time"},
			Metadata: tool.ToolMetadata{
				Capabilities: []string{"time.query"},
				Risk:         tool.RiskLow,
			},
		},
	}

	broker := NewDefaultToolBroker(2)
	result := broker.SelectWithMetadata("Use search_query to find API docs", tools)

	if assert.Len(t, result.Tools, 2) {
		assert.Equal(t, "search_query", result.Tools[0].Definition.Name)
	}
	if assert.Len(t, result.Details, 2) {
		assert.Equal(t, "search_query", result.Details[0].Name)
		assert.NotEmpty(t, result.Details[0].Reasons)
	}
}

func TestDefaultToolBroker_RespectsUnlimitedBudget(t *testing.T) {
	tools := []tool.ToolDescriptor{
		{Definition: contract.ToolDef{Name: "search_query"}},
		{Definition: contract.ToolDef{Name: "open"}},
		{Definition: contract.ToolDef{Name: "time"}},
	}

	broker := NewDefaultToolBroker(0)
	selected := broker.Select("any", tools)

	assert.Len(t, selected, len(tools))
}
