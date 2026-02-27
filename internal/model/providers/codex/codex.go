package codex

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/harunnryd/heike/internal/auth"
	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/model/contract"

	"github.com/sashabaranov/go-openai"
)

const (
	defaultCodexBaseURL      = config.DefaultCodexBaseURL
	defaultCodexModel        = "gpt-5.2"
	defaultCodexInstructions = config.DefaultThinkerSystemPrompt
	codexOAuthOriginator     = "codex_cli_rs"
)

type RuntimeConfig struct {
	RequestTimeout         time.Duration
	EmbeddingInputMaxChars int
}

type Provider struct {
	baseURL     string
	token       string
	tokenPath   string
	runtimeConf RuntimeConfig
}

func New(token, baseURL, tokenPath string, runtimeConf RuntimeConfig) *Provider {
	if baseURL == "" {
		baseURL = defaultCodexBaseURL
	}
	if runtimeConf.RequestTimeout <= 0 {
		timeout, err := config.DurationOrDefault("", config.DefaultCodexRequestTimeout)
		if err == nil {
			runtimeConf.RequestTimeout = timeout
		}
	}
	if runtimeConf.EmbeddingInputMaxChars <= 0 {
		runtimeConf.EmbeddingInputMaxChars = config.DefaultCodexEmbeddingInputMaxChars
	}

	return &Provider{
		baseURL:     baseURL,
		token:       token,
		tokenPath:   tokenPath,
		runtimeConf: runtimeConf,
	}
}

func (p *Provider) Name() string {
	return "openai-codex"
}

func (p *Provider) Generate(ctx context.Context, req contract.CompletionRequest) (*contract.CompletionResponse, error) {
	// Get Token (Refresh if needed)
	// If token provided via constructor (e.g. from config), use it.
	// Otherwise, load from auth file.
	var accessToken string
	var accountID string

	if p.token != "" {
		// Token from config/env
		accessToken = p.token
		// AccountID not available in static token config unless parsed, but auth package handles it.
		// For static token, we assume it's valid.
	} else {
		tok, err := loadToken(p.tokenPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load codex token: %w", err)
		}
		accessToken = tok.AccessToken
		accountID = tok.AccountID
	}

	// Prepare Request
	systemPrompt, inputItems := toCodexInput(req.Messages)
	if strings.TrimSpace(systemPrompt) == "" {
		systemPrompt = defaultCodexInstructions
	}

	model := resolveCodexModel(req.Model)
	reqBody := codexRequest{
		Model:        model,
		Store:        false,
		Stream:       true,
		Instructions: systemPrompt,
		Input:        inputItems,
		Text: codexTextConfig{
			Verbosity: "medium",
		},
		Include:           []string{"reasoning.encrypted_content"},
		PromptCacheKey:    codexPromptCacheKey(req.Messages),
		ToolChoice:        "auto",
		ParallelToolCalls: true,
	}

	if len(req.Tools) > 0 {
		reqBody.Tools = toCodexTools(req.Tools)
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	endpoint := codexResponsesEndpoint(p.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+accessToken)
	// We might need AccountID if present in token logic, but auth package handles it opaquely for now.
	// Assuming token is sufficient or we add logic to extract accountID.
	// For now, let's assume token is enough or we fetch user info separately if needed.
	if accountID != "" {
		httpReq.Header.Set("chatgpt-account-id", accountID)
	}

	httpReq.Header.Set("OpenAI-Beta", "responses=experimental")
	httpReq.Header.Set("originator", codexOAuthOriginator)
	httpReq.Header.Set("User-Agent", "heike (go)")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Content-Type", "application/json")

	client := newCodexStreamingHTTPClient(p.runtimeConf.RequestTimeout)
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		return nil, fmt.Errorf("codex http %d: %s", resp.StatusCode, string(raw))
	}

	// Process SSE Stream
	return consumeCodexSSE(resp.Body)
}

func (p *Provider) Embed(ctx context.Context, text string) ([]float32, error) {
	// Get Token
	var accessToken string
	if p.token != "" {
		accessToken = p.token
	} else {
		tok, err := loadToken(p.tokenPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load codex token for embedding: %w", err)
		}
		accessToken = tok.AccessToken
	}

	// Create OpenAI Client with OAuth Token
	// We use the standard OpenAI API for embeddings, assuming the OAuth token is valid.
	config := openai.DefaultConfig(accessToken)
	client := openai.NewClientWithConfig(config)

	// Truncate input before API call to avoid embedding request rejection.
	truncatedText := text
	if len(truncatedText) > p.runtimeConf.EmbeddingInputMaxChars {
		truncatedText = truncatedText[:p.runtimeConf.EmbeddingInputMaxChars]
	}

	resp, err := client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{truncatedText},
		Model: openai.SmallEmbedding3,
	})
	if err != nil {
		return nil, fmt.Errorf("codex embedding failed: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data returned")
	}

	return resp.Data[0].Embedding, nil
}

func loadToken(tokenPath string) (*auth.CodexToken, error) {
	path, err := auth.ResolveTokenPath(tokenPath)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("auth file not found, run 'heike provider login openai-codex'")
	}
	defer f.Close()

	var tok auth.CodexToken
	if err := json.NewDecoder(f).Decode(&tok); err != nil {
		return nil, err
	}

	return &tok, nil
}

type codexRequest struct {
	Model             string           `json:"model"`
	Store             bool             `json:"store"`
	Stream            bool             `json:"stream"`
	Instructions      string           `json:"instructions"`
	Input             []codexInputItem `json:"input"`
	Text              codexTextConfig  `json:"text"`
	Include           []string         `json:"include,omitempty"`
	PromptCacheKey    string           `json:"prompt_cache_key,omitempty"`
	ToolChoice        string           `json:"tool_choice,omitempty"`
	ParallelToolCalls bool             `json:"parallel_tool_calls,omitempty"`
	Tools             []codexTool      `json:"tools,omitempty"`
}

type codexTextConfig struct {
	Verbosity string `json:"verbosity,omitempty"`
}

type codexTool struct {
	Type        string                 `json:"type"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type codexInputItem struct {
	Type      string              `json:"type,omitempty"`
	Role      string              `json:"role,omitempty"`
	Status    string              `json:"status,omitempty"`
	ID        string              `json:"id,omitempty"`
	Content   []codexInputContent `json:"content,omitempty"`
	CallID    string              `json:"call_id,omitempty"`
	Name      string              `json:"name,omitempty"`
	Arguments string              `json:"arguments,omitempty"`
	Output    string              `json:"output,omitempty"`
}

type codexInputContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

func toCodexTools(tools []contract.ToolDef) []codexTool {
	out := make([]codexTool, 0, len(tools))
	for _, t := range tools {
		out = append(out, codexTool{
			Type:        "function",
			Name:        codexToolName(t.Name),
			Description: t.Description,
			Parameters:  t.Parameters,
		})
	}
	return out
}

func toCodexInput(messages []contract.Message) (string, []codexInputItem) {
	systemPrompt := ""
	input := make([]codexInputItem, 0, len(messages))

	for i, m := range messages {
		switch m.Role {
		case "system":
			systemPrompt = m.Content
		case "user":
			input = append(input, codexInputItem{
				Role: "user",
				Content: []codexInputContent{
					{Type: "input_text", Text: m.Content},
				},
			})
		case "assistant":
			if m.Content != "" {
				input = append(input, codexInputItem{
					Type:   "message",
					Role:   "assistant",
					Status: "completed",
					ID:     fmt.Sprintf("msg_%d", i),
					Content: []codexInputContent{
						{Type: "output_text", Text: m.Content},
					},
				})
			}
			for _, tc := range m.ToolCalls {
				input = append(input, codexInputItem{
					Type: "function_call",
					// Let Codex assign/track internal function_call IDs ("fc_*").
					// We only replay the provider call_id so tool outputs can be linked.
					CallID:    tc.ID,
					Name:      codexToolName(tc.Name),
					Arguments: normalizeCodexToolInput(tc.Input),
				})
			}
		case "tool":
			input = append(input, codexInputItem{
				Type:   "function_call_output",
				CallID: m.ToolCallID,
				Output: m.Content,
			})
		}
	}
	return systemPrompt, input
}

func codexToolName(name string) string {
	normalized := strings.TrimSpace(strings.ReplaceAll(name, ".", "_"))
	if normalized == "" {
		return "tool"
	}

	var sb strings.Builder
	for _, r := range normalized {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
			sb.WriteRune(r)
		} else {
			sb.WriteRune('_')
		}
	}

	cleaned := sb.String()
	if cleaned == "" {
		return "tool"
	}
	return cleaned
}

func resolveCodexModel(model string) string {
	if model == "" {
		return defaultCodexModel
	}
	// Strip prefix if present
	if strings.HasPrefix(model, "openai-codex/") {
		return strings.TrimPrefix(model, "openai-codex/")
	}
	return model
}

func codexResponsesEndpoint(baseURL string) string {
	base := strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(base, "/codex/responses") {
		return base
	}
	if strings.HasSuffix(base, "/codex") {
		return base + "/responses"
	}
	return base + "/codex/responses"
}

func codexPromptCacheKey(messages []contract.Message) string {
	b, _ := json.Marshal(messages)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// --- SSE Processing ---

type codexSSEEvent struct {
	Type      string                `json:"type"`
	Delta     string                `json:"delta"`
	Text      string                `json:"text"`
	Name      string                `json:"name"`
	CallID    string                `json:"call_id"`
	ItemID    string                `json:"item_id"`
	Arguments json.RawMessage       `json:"arguments"`
	Item      codexSSEOutputItem    `json:"item"`
	Response  codexSSEResponse      `json:"response"`
	Error     *codexSSEErrorPayload `json:"error"`
}

type codexSSEOutputItem struct {
	Type      string               `json:"type"`
	Status    string               `json:"status"`
	ID        string               `json:"id"`
	CallID    string               `json:"call_id"`
	Name      string               `json:"name"`
	Arguments json.RawMessage      `json:"arguments"`
	Content   []codexSSEOutputText `json:"content"`
}

type codexSSEResponse struct {
	Status string                `json:"status"`
	Output []codexSSEOutputItem  `json:"output"`
	Error  *codexSSEErrorPayload `json:"error"`
}

type codexSSEOutputText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type codexSSEErrorPayload struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

func consumeCodexSSE(r io.Reader) (*contract.CompletionResponse, error) {
	out := &contract.CompletionResponse{}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 8<<20)

	toolByItemID := make(map[string]*contract.ToolCall)
	toolByCallID := make(map[string]*contract.ToolCall)
	toolOrder := make([]*contract.ToolCall, 0, 4)

	var eventName string
	dataLines := make([]string, 0, 1)

	flushEvent := func() (bool, error) {
		if len(dataLines) == 0 {
			eventName = ""
			return false, nil
		}

		data := strings.Join(dataLines, "\n")
		dataLines = dataLines[:0]
		done, err := applyCodexSSEPayload(out, toolByItemID, toolByCallID, &toolOrder, eventName, data)
		eventName = ""
		return done, err
	}

	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if line == "" {
			done, err := flushEvent()
			if err != nil {
				return nil, err
			}
			if done {
				break
			}
			continue
		}

		if strings.HasPrefix(line, ":") {
			continue
		}
		if strings.HasPrefix(line, "event:") {
			eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if strings.HasPrefix(line, "data:") {
			payload := strings.TrimPrefix(line, "data:")
			if strings.HasPrefix(payload, " ") {
				payload = payload[1:]
			}
			dataLines = append(dataLines, payload)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("codex stream read failed: %w", err)
	}

	if len(dataLines) > 0 {
		if _, err := flushEvent(); err != nil {
			return nil, err
		}
	}

	for _, tc := range toolOrder {
		appendCodexToolCall(out, tc)
	}

	return out, nil
}

func applyCodexSSEPayload(
	out *contract.CompletionResponse,
	toolByItemID map[string]*contract.ToolCall,
	toolByCallID map[string]*contract.ToolCall,
	toolOrder *[]*contract.ToolCall,
	eventName, data string,
) (bool, error) {
	payload := strings.TrimSpace(data)
	if payload == "" {
		return false, nil
	}
	if payload == "[DONE]" {
		return true, nil
	}

	var evt codexSSEEvent
	if err := json.Unmarshal([]byte(payload), &evt); err != nil {
		return false, fmt.Errorf("codex stream event decode failed: %w", err)
	}
	if evt.Type == "" && eventName != "" {
		evt.Type = eventName
	}

	switch evt.Type {
	case "response.output_text.delta":
		out.Content += evt.Delta
	case "response.output_text.done":
		if out.Content == "" && evt.Text != "" {
			out.Content = evt.Text
		}
	case "response.output_item.added":
		if evt.Item.Type == "function_call" {
			tc := ensureCodexToolBuffer(toolByItemID, toolByCallID, toolOrder, evt.Item.ID, evt.Item.CallID, evt.Item.Name)
			if tc != nil && len(evt.Item.Arguments) > 0 {
				tc.Input = string(evt.Item.Arguments)
			}
		}
	case "response.function_call_arguments.delta":
		tc := ensureCodexToolBuffer(toolByItemID, toolByCallID, toolOrder, evt.ItemID, evt.CallID, evt.Name)
		if tc != nil {
			tc.Input += evt.Delta
		}
	case "response.function_call_arguments.done":
		tc := ensureCodexToolBuffer(toolByItemID, toolByCallID, toolOrder, evt.ItemID, evt.CallID, evt.Name)
		if tc != nil {
			if len(evt.Arguments) > 0 {
				tc.Input = string(evt.Arguments)
			}
			if tc.Name == "" && evt.Name != "" {
				tc.Name = evt.Name
			}
		}
	case "response.output_item.done":
		switch evt.Item.Type {
		case "function_call":
			tc := ensureCodexToolBuffer(toolByItemID, toolByCallID, toolOrder, evt.Item.ID, evt.Item.CallID, evt.Item.Name)
			if tc == nil {
				return false, nil
			}
			if len(evt.Item.Arguments) > 0 {
				tc.Input = string(evt.Item.Arguments)
			}
			appendCodexToolCall(out, tc)
			removeCodexToolBuffer(toolByItemID, toolByCallID, tc, evt.Item.ID, evt.Item.CallID)
		case "message":
			if out.Content == "" {
				out.Content = codexOutputText(evt.Item.Content)
			}
		}
	case "response.completed":
		if err := applyCodexCompletedFallback(out, toolByItemID, toolByCallID, toolOrder, evt.Response); err != nil {
			return false, err
		}
	case "response.failed", "error":
		return false, codexStreamError(evt, payload)
	}

	return false, nil
}

func ensureCodexToolBuffer(
	toolByItemID map[string]*contract.ToolCall,
	toolByCallID map[string]*contract.ToolCall,
	toolOrder *[]*contract.ToolCall,
	itemID, callID, name string,
) *contract.ToolCall {
	var tc *contract.ToolCall
	if callID != "" {
		tc = toolByCallID[callID]
	}
	if tc == nil && itemID != "" {
		tc = toolByItemID[itemID]
	}
	if tc == nil {
		tc = &contract.ToolCall{}
		*toolOrder = append(*toolOrder, tc)
	}
	if tc.Name == "" && name != "" {
		tc.Name = name
	}
	if callID != "" {
		tc.ID = callID
		toolByCallID[callID] = tc
	}
	if itemID != "" {
		toolByItemID[itemID] = tc
	}
	return tc
}

func removeCodexToolBuffer(
	toolByItemID map[string]*contract.ToolCall,
	toolByCallID map[string]*contract.ToolCall,
	tc *contract.ToolCall,
	itemID, callID string,
) {
	if callID != "" {
		delete(toolByCallID, callID)
	}
	if itemID != "" {
		delete(toolByItemID, itemID)
	}
	for id, buf := range toolByItemID {
		if buf == tc {
			delete(toolByItemID, id)
		}
	}
	for id, buf := range toolByCallID {
		if buf == tc {
			delete(toolByCallID, id)
		}
	}
}

func appendCodexToolCall(out *contract.CompletionResponse, tc *contract.ToolCall) {
	if tc == nil || strings.TrimSpace(tc.Name) == "" {
		return
	}

	tc.Input = normalizeCodexToolInput(tc.Input)
	if tc.ID == "" {
		tc.ID = fmt.Sprintf("call_%d", len(out.ToolCalls)+1)
	}

	for _, existing := range out.ToolCalls {
		if existing.ID == tc.ID {
			existing.Name = tc.Name
			existing.Input = tc.Input
			return
		}
	}

	out.ToolCalls = append(out.ToolCalls, tc)
}

func applyCodexCompletedFallback(
	out *contract.CompletionResponse,
	toolByItemID map[string]*contract.ToolCall,
	toolByCallID map[string]*contract.ToolCall,
	toolOrder *[]*contract.ToolCall,
	resp codexSSEResponse,
) error {
	if resp.Status == "failed" {
		if resp.Error != nil && strings.TrimSpace(resp.Error.Message) != "" {
			return fmt.Errorf("codex stream failed: %s", resp.Error.Message)
		}
		return fmt.Errorf("codex stream failed")
	}

	for _, item := range resp.Output {
		switch item.Type {
		case "message":
			if out.Content == "" {
				out.Content = codexOutputText(item.Content)
			}
		case "function_call":
			tc := ensureCodexToolBuffer(toolByItemID, toolByCallID, toolOrder, item.ID, item.CallID, item.Name)
			if tc == nil {
				continue
			}
			if len(item.Arguments) > 0 {
				tc.Input = string(item.Arguments)
			}
			appendCodexToolCall(out, tc)
			removeCodexToolBuffer(toolByItemID, toolByCallID, tc, item.ID, item.CallID)
		}
	}

	return nil
}

func codexOutputText(content []codexSSEOutputText) string {
	var sb strings.Builder
	for _, c := range content {
		switch c.Type {
		case "output_text", "text", "input_text":
			sb.WriteString(c.Text)
		}
	}
	return sb.String()
}

func codexStreamError(evt codexSSEEvent, raw string) error {
	if evt.Error != nil && strings.TrimSpace(evt.Error.Message) != "" {
		return fmt.Errorf("codex stream error: %s", evt.Error.Message)
	}
	if evt.Response.Error != nil && strings.TrimSpace(evt.Response.Error.Message) != "" {
		return fmt.Errorf("codex stream error: %s", evt.Response.Error.Message)
	}
	return fmt.Errorf("codex stream error: %s", raw)
}

func normalizeCodexToolInput(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "{}"
	}

	// Some events encode arguments as a JSON string; decode once if needed.
	var embedded string
	if json.Unmarshal([]byte(s), &embedded) == nil {
		s = strings.TrimSpace(embedded)
		if s == "" {
			return "{}"
		}
	}

	if !json.Valid([]byte(s)) {
		return "{}"
	}

	return s
}

func newCodexStreamingHTTPClient(requestTimeout time.Duration) *http.Client {
	dialer := &net.Dialer{
		Timeout:   15 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: codexResponseHeaderTimeout(requestTimeout),
	}

	// Do not use http.Client.Timeout for SSE because it caps total stream duration.
	return &http.Client{Transport: transport}
}

func codexResponseHeaderTimeout(requestTimeout time.Duration) time.Duration {
	const (
		defaultTimeout = 30 * time.Second
		maxTimeout     = 45 * time.Second
	)

	if requestTimeout <= 0 {
		return defaultTimeout
	}
	if requestTimeout < defaultTimeout {
		return requestTimeout
	}
	if requestTimeout > maxTimeout {
		return maxTimeout
	}
	return requestTimeout
}
