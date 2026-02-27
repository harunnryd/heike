package zai

import (
	"context"
	"fmt"

	"github.com/harunnryd/heike/internal/model/contract"

	"github.com/sashabaranov/go-openai"
)

const (
	DefaultBaseURL = "https://api.z.ai/api/paas/v4/"
	CodingBaseURL  = "https://api.z.ai/api/coding/paas/v4/"
)

type Provider struct {
	client *openai.Client
	model  string
}

func New(apiKey string, model string) (*Provider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("api key is required")
	}
	if model == "" {
		model = "glm-5"
	}

	config := openai.DefaultConfig(apiKey)
	config.BaseURL = CodingBaseURL

	return &Provider{
		client: openai.NewClientWithConfig(config),
		model:  model,
	}, nil
}

func (p *Provider) Name() string {
	return "zai"
}

func (p *Provider) Generate(ctx context.Context, req contract.CompletionRequest) (*contract.CompletionResponse, error) {
	messages := make([]openai.ChatCompletionMessage, len(req.Messages))
	for i, m := range req.Messages {
		msg := openai.ChatCompletionMessage{
			Role:       m.Role,
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
		}
		if len(m.ToolCalls) > 0 {
			toolCalls := make([]openai.ToolCall, len(m.ToolCalls))
			for j, tc := range m.ToolCalls {
				toolCalls[j] = openai.ToolCall{
					ID:   tc.ID,
					Type: openai.ToolTypeFunction,
					Function: openai.FunctionCall{
						Name:      tc.Name,
						Arguments: tc.Input,
					},
				}
			}
			msg.ToolCalls = toolCalls
		}
		messages[i] = msg
	}

	var tools []openai.Tool
	if len(req.Tools) > 0 {
		tools = make([]openai.Tool, len(req.Tools))
		for i, t := range req.Tools {
			params := t.Parameters
			if params == nil {
				params = map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				}
			}
			tools[i] = openai.Tool{
				Type: openai.ToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name:        t.Name,
					Description: t.Description,
					Parameters:  params,
				},
			}
		}
	}

	chatReq := openai.ChatCompletionRequest{
		Model:    p.model,
		Messages: messages,
		Tools:    tools,
	}

	resp, err := p.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("zai request failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return &contract.CompletionResponse{
			Content:   "",
			ToolCalls: nil,
		}, nil
	}

	choice := resp.Choices[0]
	result := &contract.CompletionResponse{
		Content:   choice.Message.Content,
		ToolCalls: nil,
	}

	if len(choice.Message.ToolCalls) > 0 {
		result.ToolCalls = make([]*contract.ToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			id := tc.ID
			if id == "" {
				id = fmt.Sprintf("call_%d", i+1)
			}
			result.ToolCalls[i] = &contract.ToolCall{
				ID:    id,
				Name:  tc.Function.Name,
				Input: tc.Function.Arguments,
			}
		}
	}

	return result, nil
}

func (p *Provider) Embed(ctx context.Context, text string) ([]float32, error) {
	return nil, fmt.Errorf("embedding not supported by zai provider")
}
