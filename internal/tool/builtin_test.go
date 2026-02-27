package tool_test

import (
	"testing"

	"github.com/harunnryd/heike/internal/tool"
	_ "github.com/harunnryd/heike/internal/tool/builtin"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuiltinNames_DeterministicAndComplete(t *testing.T) {
	names := tool.BuiltinNames()
	require.NotEmpty(t, names)

	assert.Equal(t, []string{
		"apply_patch",
		"click",
		"exec_command",
		"finance",
		"find",
		"image_query",
		"open",
		"screenshot",
		"search_query",
		"sports",
		"time",
		"view_image",
		"weather",
		"write_stdin",
	}, names)
}

func TestInstantiateBuiltins_UsesRegisteredFactories(t *testing.T) {
	builtins, err := tool.InstantiateBuiltins(tool.BuiltinOptions{})
	require.NoError(t, err)
	require.Len(t, builtins, 14)

	names := make([]string, 0, len(builtins))
	for _, builtin := range builtins {
		names = append(names, tool.NormalizeToolName(builtin.Name()))
	}

	assert.Equal(t, []string{
		"apply_patch",
		"click",
		"exec_command",
		"finance",
		"find",
		"image_query",
		"open",
		"screenshot",
		"search_query",
		"sports",
		"time",
		"view_image",
		"weather",
		"write_stdin",
	}, names)
}

func TestInstantiateBuiltins_DoesNotRequireSandbox(t *testing.T) {
	_, err := tool.InstantiateBuiltins(tool.BuiltinOptions{})
	require.NoError(t, err)
}

func TestIsBuiltinName_StrictToolName(t *testing.T) {
	assert.False(t, tool.IsBuiltinName("search.query"))
	assert.True(t, tool.IsBuiltinName("search_query"))
	assert.True(t, tool.IsBuiltinName("exec_command"))
	assert.True(t, tool.IsBuiltinName("write_stdin"))
	assert.True(t, tool.IsBuiltinName("apply_patch"))
	assert.True(t, tool.IsBuiltinName("view_image"))
	assert.False(t, tool.IsBuiltinName("custom.echo"))
}

func TestRegistryDescriptors_IncludeBuiltinMetadata(t *testing.T) {
	registry := tool.NewRegistry()
	builtins, err := tool.InstantiateBuiltins(tool.BuiltinOptions{})
	require.NoError(t, err)
	for _, builtin := range builtins {
		registry.Register(builtin)
	}

	descriptors := registry.GetDescriptors()
	require.Len(t, descriptors, 14)

	var openDescriptor *tool.ToolDescriptor
	for i := range descriptors {
		if descriptors[i].Definition.Name == "open" {
			openDescriptor = &descriptors[i]
			break
		}
	}
	require.NotNil(t, openDescriptor)
	assert.Equal(t, "builtin", openDescriptor.Metadata.Source)
	assert.Equal(t, tool.RiskMedium, openDescriptor.Metadata.Risk)
	assert.Contains(t, openDescriptor.Metadata.Capabilities, "web.fetch")
}
