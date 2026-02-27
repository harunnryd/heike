package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/harunnryd/heike/internal/model/contract"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type Provider struct {
	client anthropic.Client
}

func New(apiKey string) *Provider {
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &Provider{client: client}
}

func (p *Provider) Name() string {
	return "anthropic"
}

func (p *Provider) Generate(ctx context.Context, req contract.CompletionRequest) (*contract.CompletionResponse, error) {
	var messages []anthropic.MessageParam
	for _, m := range req.Messages {
		switch m.Role {
		case "user":
			messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(m.Content)))
		case "assistant":
			messages = append(messages, anthropic.NewAssistantMessage(anthropic.NewTextBlock(m.Content)))
		case "tool":
			messages = append(messages, anthropic.NewUserMessage(anthropic.NewToolResultBlock(m.ToolCallID, m.Content, false)))
		default:
			messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(m.Content)))
		}
	}

	var tools []anthropic.ToolUnionParam
	for _, t := range req.Tools {
		tool := anthropic.ToolParam{
			Name:        t.Name,
			Description: anthropic.String(t.Description),
			InputSchema: anthropic.ToolInputSchemaParam{Properties: map[string]interface{}{}},
		}
		if t.Parameters != nil {
			if props, ok := t.Parameters["properties"].(map[string]interface{}); ok {
				tool.InputSchema = anthropic.ToolInputSchemaParam{Properties: props}
			}
		}
		tools = append(tools, anthropic.ToolUnionParam{OfTool: &tool})
	}

	modelName := req.Model
	if modelName == "" {
		modelName = string(anthropic.ModelClaude3_7SonnetLatest)
	}

	msg, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(modelName),
		MaxTokens: 1024,
		Messages:  messages,
		Tools:     tools,
	})
	if err != nil {
		return nil, fmt.Errorf("anthropic request failed: %w", err)
	}

	resp := &contract.CompletionResponse{}
	for _, block := range msg.Content {
		switch b := block.AsAny().(type) {
		case anthropic.TextBlock:
			resp.Content += b.Text
		case anthropic.ToolUseBlock:
			inputJSON, _ := json.Marshal(b.Input)
			resp.ToolCalls = append(resp.ToolCalls, &contract.ToolCall{
				ID:    b.ID,
				Name:  b.Name,
				Input: string(inputJSON),
			})
		}
	}

	return resp, nil
}

func (p *Provider) Embed(ctx context.Context, text string) ([]float32, error) {
	return nil, fmt.Errorf("embedding not supported by anthropic provider")
}
