package cognitive

import (
	"context"
	stdErrors "errors"
	"testing"

	heikeErrors "github.com/harunnryd/heike/internal/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUnifiedReflector_InvalidStructuredOutputReturnsTypedError(t *testing.T) {
	mockLLM := new(MockLLMClient)
	reflector := NewReflector(mockLLM, ReflectorPromptConfig{}, 1)

	ctx := context.Background()
	action := &Action{Type: ActionTypeAnswer, Content: "done"}
	result := &ExecutionResult{Success: true, Output: "done"}

	mockLLM.On("Complete", ctx, mock.Anything).Return("NOT_JSON", nil).Twice()

	_, err := reflector.Reflect(ctx, "Finish task", action, result)
	assert.Error(t, err)
	assert.True(t, stdErrors.Is(err, heikeErrors.ErrInvalidModelOutput))
	mockLLM.AssertExpectations(t)
}
