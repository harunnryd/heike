package tool

import (
	"encoding/json"
	"testing"
)

func TestValidateInput(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type": "string",
			},
			"age": map[string]interface{}{
				"type": "number",
			},
			"tags": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "string",
				},
			},
		},
		"required": []string{"name"},
	}

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "Valid input",
			input:   `{"name": "Alice", "age": 30, "tags": ["admin"]}`,
			wantErr: false,
		},
		{
			name:    "Missing required field",
			input:   `{"age": 30}`,
			wantErr: true,
		},
		{
			name:    "Invalid type (string vs number)",
			input:   `{"name": "Alice", "age": "thirty"}`,
			wantErr: true,
		},
		{
			name:    "Invalid array item type",
			input:   `{"name": "Alice", "tags": [123]}`,
			wantErr: true,
		},
		{
			name:    "Extra fields (allowed)",
			input:   `{"name": "Alice", "extra": "field"}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInput(schema, json.RawMessage(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateInput() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
