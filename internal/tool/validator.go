package tool

import (
	"encoding/json"
	"fmt"
)

// ValidateInput checks if the JSON input matches the tool's parameter schema.
// This is a lightweight implementation of JSON Schema validation.
func ValidateInput(schema map[string]interface{}, input json.RawMessage) error {
	var inputMap map[string]interface{}
	if err := json.Unmarshal(input, &inputMap); err != nil {
		return fmt.Errorf("invalid JSON input: %w", err)
	}

	return validateObject(schema, inputMap)
}

func validateObject(schema map[string]interface{}, input map[string]interface{}) error {
	// Check Required Fields
	if required, ok := schema["required"].([]interface{}); ok {
		for _, field := range required {
			fieldName, ok := field.(string)
			if !ok {
				continue // Malformed schema
			}
			if _, exists := input[fieldName]; !exists {
				return fmt.Errorf("missing required field: %s", fieldName)
			}
		}
	} else if required, ok := schema["required"].([]string); ok {
		// Handle []string definition as well
		for _, fieldName := range required {
			if _, exists := input[fieldName]; !exists {
				return fmt.Errorf("missing required field: %s", fieldName)
			}
		}
	}

	// Check Properties
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		return nil // No properties defined
	}

	for key, value := range input {
		propSchema, defined := properties[key]
		if !defined {
			// Strict mode: disallow unknown fields?
			// For now, let's allow extra fields but maybe warn (or ignore).
			// To be strict: return fmt.Errorf("unknown field: %s", key)
			continue
		}

		propSchemaMap, ok := propSchema.(map[string]interface{})
		if !ok {
			continue
		}

		if err := validateType(key, propSchemaMap, value); err != nil {
			return err
		}
	}

	return nil
}

func validateType(fieldName string, schema map[string]interface{}, value interface{}) error {
	expectedType, ok := schema["type"].(string)
	if !ok {
		return nil // Type not specified
	}

	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("field '%s' expected string, got %T", fieldName, value)
		}
	case "number", "integer":
		// JSON unmarshals numbers to float64
		if _, ok := value.(float64); !ok {
			return fmt.Errorf("field '%s' expected number, got %T", fieldName, value)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("field '%s' expected boolean, got %T", fieldName, value)
		}
	case "array":
		arr, ok := value.([]interface{})
		if !ok {
			return fmt.Errorf("field '%s' expected array, got %T", fieldName, value)
		}
		// Validate items if specified
		if itemsSchema, ok := schema["items"].(map[string]interface{}); ok {
			for i, item := range arr {
				if err := validateType(fmt.Sprintf("%s[%d]", fieldName, i), itemsSchema, item); err != nil {
					return err
				}
			}
		}
	case "object":
		obj, ok := value.(map[string]interface{})
		if !ok {
			return fmt.Errorf("field '%s' expected object, got %T", fieldName, value)
		}
		return validateObject(schema, obj)
	}

	return nil
}
