package codex

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/harunnryd/heike/internal/model/contract"

	"github.com/stretchr/testify/assert"
)

func TestToCodexTools_NormalizesFunctionNames(t *testing.T) {
	tools := []contract.ToolDef{
		{
			Name:        "search.query",
			Description: "Search the web",
			Parameters:  map[string]interface{}{"type": "object"},
		},
	}

	got := toCodexTools(tools)
	if assert.Len(t, got, 1) {
		assert.Equal(t, "search_query", got[0].Name)
		assert.Equal(t, "function", got[0].Type)
	}
}

func TestToCodexInput_NormalizesAssistantToolCallNames(t *testing.T) {
	messages := []contract.Message{
		{
			Role: "assistant",
			ToolCalls: []*contract.ToolCall{
				{ID: "call_1", Name: "search.query", Input: `{"q":"heike"}`},
			},
		},
	}

	_, items := toCodexInput(messages)
	if assert.Len(t, items, 1) {
		assert.Equal(t, "function_call", items[0].Type)
		assert.Equal(t, "search_query", items[0].Name)
		assert.Equal(t, "", items[0].ID)
		assert.Equal(t, "call_1", items[0].CallID)
		assert.Equal(t, `{"q":"heike"}`, items[0].Arguments)
	}
}

func TestToCodexInput_NormalizesAssistantToolCallInput(t *testing.T) {
	messages := []contract.Message{
		{
			Role: "assistant",
			ToolCalls: []*contract.ToolCall{
				{ID: "call_1", Name: "search.query", Input: "{"},
			},
		},
	}

	_, items := toCodexInput(messages)
	if assert.Len(t, items, 1) {
		assert.Equal(t, "{}", items[0].Arguments)
	}
}

func TestCodexToolName(t *testing.T) {
	assert.Equal(t, "search_query", codexToolName("search.query"))
	assert.Equal(t, "tool_name_", codexToolName("tool name?"))
	assert.Equal(t, "tool", codexToolName(""))
}

func TestNormalizeCodexToolInput(t *testing.T) {
	assert.Equal(t, "{}", normalizeCodexToolInput(""))
	assert.Equal(t, "{}", normalizeCodexToolInput(`""`))
	assert.Equal(t, `{"path":"a.go"}`, normalizeCodexToolInput(`"{\"path\":\"a.go\"}"`))
	assert.Equal(t, `{"path":"a.go"}`, normalizeCodexToolInput(`{"path":"a.go"}`))
	assert.Equal(t, "{}", normalizeCodexToolInput("{"))
}

func TestConsumeCodexSSE_ParsesFunctionCallByItemID(t *testing.T) {
	stream := strings.Join([]string{
		`event: response.output_text.delta`,
		`data: {"type":"response.output_text.delta","delta":"Hi"}`,
		``,
		`event: response.output_item.added`,
		`data: {"type":"response.output_item.added","item":{"id":"fc_1","type":"function_call","call_id":"call_1","name":"open"}}`,
		``,
		`event: response.function_call_arguments.delta`,
		`data: {"type":"response.function_call_arguments.delta","item_id":"fc_1","delta":"{\"url\":\"https://example.com\"}"}`,
		``,
		`event: response.output_item.done`,
		`data: {"type":"response.output_item.done","item":{"id":"fc_1","type":"function_call","call_id":"call_1","name":"open"}}`,
		``,
		`data: [DONE]`,
		``,
	}, "\n")

	got, err := consumeCodexSSE(strings.NewReader(stream))
	assert.NoError(t, err)
	if assert.NotNil(t, got) {
		assert.Equal(t, "Hi", got.Content)
		if assert.Len(t, got.ToolCalls, 1) {
			assert.Equal(t, "call_1", got.ToolCalls[0].ID)
			assert.Equal(t, "open", got.ToolCalls[0].Name)
			assert.Equal(t, `{"url":"https://example.com"}`, got.ToolCalls[0].Input)
		}
	}
}

func TestConsumeCodexSSE_FallbackFromResponseCompleted(t *testing.T) {
	stream := strings.Join([]string{
		`event: response.completed`,
		`data: {"type":"response.completed","response":{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"done"}]},{"type":"function_call","id":"fc_2","call_id":"call_2","name":"search_query","arguments":"{\"q\":\"heike\"}"}]}}`,
		``,
		`data: [DONE]`,
		``,
	}, "\n")

	got, err := consumeCodexSSE(strings.NewReader(stream))
	assert.NoError(t, err)
	if assert.NotNil(t, got) {
		assert.Equal(t, "done", got.Content)
		if assert.Len(t, got.ToolCalls, 1) {
			assert.Equal(t, "call_2", got.ToolCalls[0].ID)
			assert.Equal(t, "search_query", got.ToolCalls[0].Name)
			assert.Equal(t, `{"q":"heike"}`, got.ToolCalls[0].Input)
		}
	}
}

func TestConsumeCodexSSE_AllowsLargePayloadLine(t *testing.T) {
	longText := strings.Repeat("a", 70_000)
	payload, err := json.Marshal(map[string]string{
		"type":  "response.output_text.delta",
		"delta": longText,
	})
	assert.NoError(t, err)

	stream := "data: " + string(payload) + "\n\n" + "data: [DONE]\n\n"
	got, err := consumeCodexSSE(strings.NewReader(stream))
	assert.NoError(t, err)
	if assert.NotNil(t, got) {
		assert.Len(t, got.Content, len(longText))
	}
}

func TestConsumeCodexSSE_PropagatesErrorEvent(t *testing.T) {
	stream := strings.Join([]string{
		`event: error`,
		`data: {"error":{"message":"boom"}}`,
		``,
	}, "\n")

	_, err := consumeCodexSSE(strings.NewReader(stream))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
}

func TestCodexResponseHeaderTimeout(t *testing.T) {
	assert.Equal(t, 30*time.Second, codexResponseHeaderTimeout(0))
	assert.Equal(t, 10*time.Second, codexResponseHeaderTimeout(10*time.Second))
	assert.Equal(t, 30*time.Second, codexResponseHeaderTimeout(30*time.Second))
	assert.Equal(t, 45*time.Second, codexResponseHeaderTimeout(2*time.Minute))
}

func TestNewCodexStreamingHTTPClient_HasNoGlobalTimeout(t *testing.T) {
	client := newCodexStreamingHTTPClient(2 * time.Minute)
	assert.NotNil(t, client)
	assert.Equal(t, time.Duration(0), client.Timeout)

	transport, ok := client.Transport.(*http.Transport)
	if assert.True(t, ok) {
		assert.Equal(t, 45*time.Second, transport.ResponseHeaderTimeout)
	}
}
