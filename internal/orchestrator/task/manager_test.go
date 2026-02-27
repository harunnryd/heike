package task

import (
	"context"
	"testing"
	"time"

	"github.com/harunnryd/heike/internal/cognitive"
	"github.com/harunnryd/heike/internal/model/contract"
	"github.com/harunnryd/heike/internal/skill"
	"github.com/harunnryd/heike/internal/tool"

	"github.com/stretchr/testify/assert"
)

type stubEngine struct {
	capturedContext *cognitive.CognitiveContext
}

func (s *stubEngine) Run(ctx context.Context, goal string, opts ...cognitive.ExecutionOption) (*cognitive.Result, error) {
	cCtx := &cognitive.CognitiveContext{}
	for _, opt := range opts {
		opt(cCtx)
	}
	s.capturedContext = cCtx
	return &cognitive.Result{Content: "ok"}, nil
}

type stubDecomposer struct{}

func (d *stubDecomposer) ShouldDecompose(task string) bool {
	return false
}

func (d *stubDecomposer) Decompose(ctx context.Context, task string) ([]*SubTask, error) {
	return nil, nil
}

type stubSessionManager struct {
	context *cognitive.CognitiveContext
}

func (s *stubSessionManager) GetContext(ctx context.Context, sessionID string) (*cognitive.CognitiveContext, error) {
	return s.context, nil
}

func (s *stubSessionManager) AppendInteraction(ctx context.Context, sessionID string, role, content string) error {
	return nil
}

func (s *stubSessionManager) PersistTool(ctx context.Context, sessionID, toolCallID, content string) error {
	return nil
}

type stubResponseSink struct {
	lastSessionID string
	lastContent   string
}

func (s *stubResponseSink) Send(ctx context.Context, sessionID string, content string) error {
	s.lastSessionID = sessionID
	s.lastContent = content
	return nil
}

func TestTaskManager_InjectsToolDefinitionsIntoSimpleTaskContext(t *testing.T) {
	engine := &stubEngine{}
	sessionManager := &stubSessionManager{
		context: &cognitive.CognitiveContext{SessionID: "session-1"},
	}

	tools := []tool.ToolDescriptor{
		{Definition: contract.ToolDef{Name: "search_query", Description: "Search the web"}},
		{Definition: contract.ToolDef{Name: "open", Description: "Open a web page"}},
	}

	manager := NewManager(
		engine,
		&stubDecomposer{},
		sessionManager,
		tools,
		NewDefaultToolBroker(10),
		nil,
		3,
		time.Second,
		10,
		4,
		&stubResponseSink{},
	)

	err := manager.HandleRequest(context.Background(), "session-1", "Research release notes")
	assert.NoError(t, err)
	if assert.NotNil(t, engine.capturedContext) {
		assert.Len(t, engine.capturedContext.AvailableTools, 2)
		assert.Equal(t, "search_query", engine.capturedContext.AvailableTools[0].Name)
		assert.Equal(t, "open", engine.capturedContext.AvailableTools[1].Name)
	}
}

func TestTaskManager_AppliesToolBrokerBudget(t *testing.T) {
	engine := &stubEngine{}
	sessionManager := &stubSessionManager{
		context: &cognitive.CognitiveContext{SessionID: "session-2"},
	}

	tools := []tool.ToolDescriptor{
		{Definition: contract.ToolDef{Name: "search_query", Description: "Search the web"}},
		{Definition: contract.ToolDef{Name: "open", Description: "Open web pages"}},
		{Definition: contract.ToolDef{Name: "exec_command", Description: "Execute shell commands"}},
	}

	manager := NewManager(
		engine,
		&stubDecomposer{},
		sessionManager,
		tools,
		NewDefaultToolBroker(1),
		nil,
		3,
		time.Second,
		10,
		4,
		&stubResponseSink{},
	)
	err := manager.HandleRequest(context.Background(), "session-2", "Research AI updates on the web")
	assert.NoError(t, err)
	if assert.NotNil(t, engine.capturedContext) {
		assert.Len(t, engine.capturedContext.AvailableTools, 1)
	}
}

func TestTaskManager_InjectsRelevantSkillsIntoContext(t *testing.T) {
	engine := &stubEngine{}
	sessionManager := &stubSessionManager{
		context: &cognitive.CognitiveContext{SessionID: "session-3"},
	}

	registry := skill.NewRegistry()
	registry.Register(&skill.Skill{
		Name:        "web_research",
		Description: "Find and verify web sources",
		Tags:        []string{"research", "web"},
		Tools:       []string{"search_query", "open", "find"},
		Content:     "Use search_query, then open and find to validate key facts.",
	})

	manager := NewManager(
		engine,
		&stubDecomposer{},
		sessionManager,
		[]tool.ToolDescriptor{},
		NewDefaultToolBroker(10),
		registry,
		3,
		time.Second,
		10,
		4,
		&stubResponseSink{},
	)

	err := manager.HandleRequest(context.Background(), "session-3", "Use $web_research to gather evidence")
	assert.NoError(t, err)
	if assert.NotNil(t, engine.capturedContext) {
		assert.Contains(t, engine.capturedContext.AvailableSkills, "web_research")
		assert.Contains(t, engine.capturedContext.Metadata["skills_context"], "web_research")
		assert.Contains(t, engine.capturedContext.Metadata["skills_context"], "search_query")
	}
}

func TestTaskManager_SendsFinalResponse(t *testing.T) {
	engine := &stubEngine{}
	sessionManager := &stubSessionManager{
		context: &cognitive.CognitiveContext{SessionID: "session-send"},
	}
	sink := &stubResponseSink{}

	manager := NewManager(
		engine,
		&stubDecomposer{},
		sessionManager,
		[]tool.ToolDescriptor{},
		NewDefaultToolBroker(10),
		nil,
		3,
		time.Second,
		10,
		4,
		sink,
	)

	err := manager.HandleRequest(context.Background(), "session-send", "answer this")
	assert.NoError(t, err)
	assert.Equal(t, "session-send", sink.lastSessionID)
	assert.Equal(t, "ok", sink.lastContent)
}
