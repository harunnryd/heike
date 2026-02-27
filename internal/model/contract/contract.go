package contract

type Message struct {
	Role       string      `json:"role"`
	Content    string      `json:"content"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
	ToolCalls  []*ToolCall `json:"tool_calls,omitempty"`
}

type CompletionRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Tools    []ToolDef `json:"tools,omitempty"`
}

type ToolDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type CompletionResponse struct {
	Content   string      `json:"content"`
	ToolCalls []*ToolCall `json:"tool_calls,omitempty"`
}

type ToolCall struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name"`
	Input string `json:"input"`
}
