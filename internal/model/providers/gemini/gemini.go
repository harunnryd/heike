package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/harunnryd/heike/internal/model/contract"

	"google.golang.org/genai"
)

type Provider struct {
	client *genai.Client
}

const defaultEmbeddingModel = "text-embedding-004"

func New(apiKey string) (*Provider, error) {
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{APIKey: apiKey, Backend: genai.BackendGeminiAPI})
	if err != nil {
		return nil, err
	}
	return &Provider{client: client}, nil
}

func (p *Provider) Name() string {
	return "gemini"
}

func (p *Provider) Generate(ctx context.Context, req contract.CompletionRequest) (*contract.CompletionResponse, error) {
	var contents []*genai.Content
	for _, m := range req.Messages {
		switch m.Role {
		case "tool":
			var obj map[string]any
			_ = json.Unmarshal([]byte(m.Content), &obj)
			contents = append(contents, &genai.Content{Role: "function", Parts: []*genai.Part{{FunctionResponse: &genai.FunctionResponse{ID: m.ToolCallID, Name: m.ToolCallID, Response: obj}}}})
		case "assistant":
			contents = append(contents, &genai.Content{Role: "model", Parts: []*genai.Part{{Text: m.Content}}})
		default:
			contents = append(contents, &genai.Content{Role: "user", Parts: []*genai.Part{{Text: m.Content}}})
		}
	}

	var tools []*genai.Tool
	if len(req.Tools) > 0 {
		var decls []*genai.FunctionDeclaration
		for _, t := range req.Tools {
			b, _ := json.Marshal(t.Parameters)
			var schema genai.Schema
			_ = json.Unmarshal(b, &schema)
			decls = append(decls, &genai.FunctionDeclaration{Name: t.Name, Description: t.Description, Parameters: &schema})
		}
		tools = append(tools, &genai.Tool{FunctionDeclarations: decls})
	}

	resp, err := p.client.Models.GenerateContent(ctx, req.Model, contents, &genai.GenerateContentConfig{Tools: tools})
	if err != nil {
		return nil, fmt.Errorf("gemini request failed: %w", err)
	}

	out := &contract.CompletionResponse{}
	if resp == nil {
		return out, nil
	}

	for _, fc := range resp.FunctionCalls() {
		argsJSON, _ := json.Marshal(fc.Args)
		id := fc.ID
		if id == "" {
			id = fc.Name
		}
		out.ToolCalls = append(out.ToolCalls, &contract.ToolCall{ID: id, Name: fc.Name, Input: string(argsJSON)})
	}

	if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
		for _, part := range resp.Candidates[0].Content.Parts {
			if part.Text != "" {
				out.Content += part.Text
			}
		}
	}

	return out, nil
}

func (p *Provider) Embed(ctx context.Context, text string) ([]float32, error) {
	resp, err := p.client.Models.EmbedContent(ctx, defaultEmbeddingModel, genai.Text(text), nil)
	if err != nil {
		return nil, fmt.Errorf("gemini embedding failed: %w", err)
	}
	if resp == nil || len(resp.Embeddings) == 0 || len(resp.Embeddings[0].Values) == 0 {
		return nil, fmt.Errorf("gemini embedding returned empty result")
	}

	return resp.Embeddings[0].Values, nil
}
