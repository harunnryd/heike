package tool

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/harunnryd/heike/internal/config"
	heikeErrors "github.com/harunnryd/heike/internal/errors"
	"github.com/harunnryd/heike/internal/policy"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubLookupTool struct {
	name string
}

func (t *stubLookupTool) Name() string        { return t.name }
func (t *stubLookupTool) Description() string { return "stub" }
func (t *stubLookupTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}
func (t *stubLookupTool) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	_ = ctx
	_ = input
	return json.Marshal(map[string]string{"status": "ok"})
}

func TestRegistryRegister_UsesSingleName(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&stubLookupTool{name: "search_query"})

	_, ok := registry.Get("search_query")
	require.True(t, ok)

	_, ok = registry.Get("web.search_query")
	require.False(t, ok)

	_, ok = registry.Get("legacy_search_name")
	require.False(t, ok)

	descriptors := registry.GetDescriptors()
	require.Len(t, descriptors, 1)
	assert.Equal(t, "search_query", descriptors[0].Definition.Name)
}

func TestRunnerExecute_UsesResolvedNameForPolicy(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	pol, err := policy.NewEngine(config.GovernanceConfig{
		RequireApproval: []string{"search_query"},
	}, "strict-name-policy-"+t.Name(), "")
	require.NoError(t, err)

	registry := NewRegistry()
	registry.Register(&stubLookupTool{name: "search_query"})
	runner := NewRunner(registry, pol)

	_, err = runner.Execute(context.Background(), "search_query", json.RawMessage(`{"q":"heike"}`), "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, heikeErrors.ErrApprovalRequired))
}

func TestRunnerExecute_SandboxRequireEscalatedRequiresApproval(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	pol, err := policy.NewEngine(config.GovernanceConfig{
		AutoAllow: []string{"exec_command"},
	}, "sandbox-policy-"+t.Name(), "")
	require.NoError(t, err)

	registry := NewRegistry()
	registry.Register(&stubLookupTool{name: "exec_command"})
	runner := NewRunner(registry, pol)

	_, err = runner.Execute(context.Background(), "exec_command", json.RawMessage(`{"cmd":"echo ok","sandbox_permissions":"require_escalated"}`), "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, heikeErrors.ErrApprovalRequired))
}

func TestRunnerExecute_ApprovalPathConsumesQuota(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	pol, err := policy.NewEngine(config.GovernanceConfig{
		RequireApproval: []string{"exec_command"},
		DailyToolLimit:  1,
	}, "approval-quota-"+t.Name(), "")
	require.NoError(t, err)

	_, approvalID, err := pol.Check("exec_command", json.RawMessage(`{}`))
	require.Error(t, err)
	require.True(t, errors.Is(err, heikeErrors.ErrApprovalRequired))
	require.NotEmpty(t, approvalID)
	require.NoError(t, pol.Resolve(approvalID, true))

	registry := NewRegistry()
	registry.Register(&stubLookupTool{name: "exec_command"})
	runner := NewRunner(registry, pol)

	_, err = runner.Execute(context.Background(), "exec_command", json.RawMessage(`{}`), approvalID)
	require.NoError(t, err)

	_, err = runner.Execute(context.Background(), "exec_command", json.RawMessage(`{}`), approvalID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "quota exceeded")
}
