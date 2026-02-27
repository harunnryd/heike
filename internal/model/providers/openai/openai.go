package openai

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/harunnryd/heike/internal/model/contract"

	"github.com/sashabaranov/go-openai"
)

type Provider struct {
	client *openai.Client
	model  string
}

func New(apiKey, baseURL, model string) *Provider {
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	cfg := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		baseURL = strings.TrimSuffix(baseURL, "/")
		cfg.BaseURL = baseURL
	}

	// Support for Azure AD / Microsoft ID OAuth (Custom Token Source)
	// If the apiKey starts with "ey...", it might be a JWT token.
	// However, standard OpenAI config expects "Bearer <token>".
	// The underlying library handles this if we just pass the token as apiKey.
	// But for proper OAuth rotation, we might need a custom HTTP client or transport.
	// For now, we assume static token provided via config or env.

	client := openai.NewClientWithConfig(cfg)
	return &Provider{client: client, model: model}
}

func (p *Provider) Name() string {
	return "openai"
}

func (p *Provider) Generate(ctx context.Context, req contract.CompletionRequest) (*contract.CompletionResponse, error) {
	var messages []openai.ChatCompletionMessage
	for _, m := range req.Messages {
		msg := openai.ChatCompletionMessage{
			Role:       m.Role,
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
		}

		if len(m.ToolCalls) > 0 {
			var tcs []openai.ToolCall
			for _, tc := range m.ToolCalls {
				tcs = append(tcs, openai.ToolCall{
					ID:   tc.ID,
					Type: openai.ToolTypeFunction,
					Function: openai.FunctionCall{
						Name:      tc.Name,
						Arguments: tc.Input,
					},
				})
			}
			msg.ToolCalls = tcs
		}

		messages = append(messages, msg)
	}

	var tools []openai.Tool
	for _, t := range req.Tools {
		params := t.Parameters
		if params == nil {
			params = map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}
		}
		tools = append(tools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			},
		})
	}

	chatReq := openai.ChatCompletionRequest{
		Model:    req.Model,
		Messages: messages,
		Tools:    tools,
	}

	resp, err := p.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("openai request failed: %w", err)
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned")
	}

	choice := resp.Choices[0]
	result := &contract.CompletionResponse{Content: choice.Message.Content}

	if len(choice.Message.ToolCalls) > 0 {
		for _, tc := range choice.Message.ToolCalls {
			id := tc.ID
			if id == "" {
				id = fmt.Sprintf("call_%d", len(result.ToolCalls)+1)
			}
			result.ToolCalls = append(result.ToolCalls, &contract.ToolCall{
				ID:    id,
				Name:  tc.Function.Name,
				Input: tc.Function.Arguments,
			})
		}
	}

	return result, nil
}

func (p *Provider) Embed(ctx context.Context, text string) ([]float32, error) {
	model := p.model
	if model == "" {
		model = string(openai.SmallEmbedding3)
	}

	req := openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.EmbeddingModel(model),
	}

	resp, err := p.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("openai embedding failed: %w", err)
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data returned")
	}

	return resp.Data[0].Embedding, nil
}
