package conformance_test

import (
	"context"
	"testing"

	"github.com/harunnryd/heike/internal/model/contract"
)

type mockProvider struct {
	calls []contract.CompletionRequest
}

func (p *mockProvider) Name() string { return "mock" }

func (p *mockProvider) Generate(ctx context.Context, req contract.CompletionRequest) (*contract.CompletionResponse, error) {
	p.calls = append(p.calls, req)

	if len(p.calls) == 1 {
		return &contract.CompletionResponse{
			ToolCalls: []*contract.ToolCall{{
				ID:    "call_1",
				Name:  "exec_command",
				Input: `{"cmd":"echo hello"}`,
			}},
		}, nil
	}

	return &contract.CompletionResponse{Content: "done"}, nil
}

func (p *mockProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	return []float32{0.1, 0.2}, nil
}

func TestToolCallMustBeFollowedByToolResultMessage(t *testing.T) {
	p := &mockProvider{}

	messages := []contract.Message{{Role: "user", Content: "run a command"}}
	tools := []contract.ToolDef{{Name: "exec_command", Description: "run command", Parameters: map[string]interface{}{"type": "object"}}}

	resp, err := p.Generate(context.Background(), contract.CompletionRequest{Model: "x", Messages: messages, Tools: tools})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call")
	}

	messages = append(messages, contract.Message{Role: "assistant", ToolCalls: resp.ToolCalls})
	messages = append(messages, contract.Message{Role: "tool", ToolCallID: resp.ToolCalls[0].ID, Content: "ok"})

	_, err = p.Generate(context.Background(), contract.CompletionRequest{Model: "x", Messages: messages, Tools: tools})
	if err != nil {
		t.Fatalf("generate 2: %v", err)
	}

	if len(p.calls) != 2 {
		t.Fatalf("expected 2 calls")
	}

	second := p.calls[1]
	foundTool := false
	for _, m := range second.Messages {
		if m.Role == "tool" && m.ToolCallID == "call_1" {
			foundTool = true
			break
		}
	}
	if !foundTool {
		t.Fatalf("expected tool result message with tool_call_id call_1")
	}
}
