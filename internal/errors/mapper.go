package errors

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ErrorMapper maps external errors to Heike error taxonomy
type ErrorMapper interface {
	MapError(err error) error
	IsRetryable(err error) bool
	Category(err error) string
}

// DefaultErrorMapper implements Heike error taxonomy mapping
type DefaultErrorMapper struct {
	mapping map[string]map[error]string
}

// NewDefaultErrorMapper creates a new error mapper
func NewDefaultErrorMapper() *DefaultErrorMapper {
	return &DefaultErrorMapper{
		mapping: make(map[string]map[error]string),
	}
}

// MapError maps external errors to Heike error categories
func (m *DefaultErrorMapper) MapError(err error) error {
	if err == nil {
		return nil
	}

	// Propagate context errors as-is
	if errors.Is(err, context.Canceled) {
		return err
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("request timeout: %w", ErrTransient)
	}

	// Map based on error message content
	errStr := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errStr, "not found"), strings.Contains(errStr, "does not exist"):
		return fmt.Errorf("resource not found: %w", ErrNotFound)

	case strings.Contains(errStr, "permission denied"), strings.Contains(errStr, "unauthorized"), strings.Contains(errStr, "forbidden"):
		return fmt.Errorf("access denied: %w", ErrPermissionDenied)

	case strings.Contains(errStr, "rate limit"), strings.Contains(errStr, "quota"), strings.Contains(errStr, "too many requests"):
		return fmt.Errorf("rate limited: %w", ErrTransient)

	case strings.Contains(errStr, "invalid input"), strings.Contains(errStr, "invalid request"), strings.Contains(errStr, "bad request"):
		return fmt.Errorf("invalid request: %w", ErrInvalidInput)
	case strings.Contains(errStr, "invalid model output"), strings.Contains(errStr, "malformed json"), strings.Contains(errStr, "invalid json"):
		return fmt.Errorf("invalid model output: %w", ErrInvalidModelOutput)

	case strings.Contains(errStr, "timeout"), strings.Contains(errStr, "deadline exceeded"):
		return fmt.Errorf("request timeout: %w", ErrTransient)

	case strings.Contains(errStr, "network"), strings.Contains(errStr, "connection"), strings.Contains(errStr, "unreachable"):
		return fmt.Errorf("network error: %w", ErrTransient)

	case strings.Contains(errStr, "conflict"), strings.Contains(errStr, "already exists"):
		return fmt.Errorf("conflict: %w", ErrConflict)

	case strings.Contains(errStr, "duplicate"):
		return fmt.Errorf("duplicate event: %w", ErrDuplicateEvent)

	default:
		return fmt.Errorf("internal error: %w", ErrInternal)
	}
}

// IsRetryable determines if an error should trigger a retry
func (m *DefaultErrorMapper) IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.Canceled) {
		return false
	}

	if errors.Is(err, ErrTransient) {
		return true
	}

	if errors.Is(err, ErrConflict) {
		return true
	}

	return false
}

// Category returns Heike error category for an error
func (m *DefaultErrorMapper) Category(err error) string {
	if err == nil {
		return ""
	}

	switch {
	case errors.Is(err, ErrDuplicateEvent):
		return "ErrDuplicateEvent"
	case errors.Is(err, ErrApprovalRequired):
		return "ErrApprovalRequired"
	case errors.Is(err, ErrPermissionDenied):
		return "ErrPermissionDenied"
	case errors.Is(err, ErrInvalidInput):
		return "ErrInvalidInput"
	case errors.Is(err, ErrNotFound):
		return "ErrNotFound"
	case errors.Is(err, ErrConflict):
		return "ErrConflict"
	case errors.Is(err, ErrTransient):
		return "ErrTransient"
	case errors.Is(err, ErrInvalidModelOutput):
		return "ErrInvalidModelOutput"
	case errors.Is(err, ErrInternal):
		return "ErrInternal"
	default:
		return "Unknown"
	}
}

// Wrap wraps an error with context using Heike error categories
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s: %w", message, err)
}

// WrapWithCategory wraps an error with specific Heike error category
func WrapWithCategory(err error, message string, category error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s: %w", message, category)
}

// IsCategory checks if error belongs to specific category
func IsCategory(err error, category error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, category)
}

// NotFound wraps error as not found
func NotFound(message string) error {
	return fmt.Errorf("%s: %w", message, ErrNotFound)
}

// PermissionDenied wraps error as permission denied
func PermissionDenied(message string) error {
	return fmt.Errorf("%s: %w", message, ErrPermissionDenied)
}

// InvalidInput wraps error as invalid input
func InvalidInput(message string) error {
	return fmt.Errorf("%s: %w", message, ErrInvalidInput)
}

// Transient wraps error as transient
func Transient(message string) error {
	return fmt.Errorf("%s: %w", message, ErrTransient)
}

// Internal wraps error as internal
func Internal(message string) error {
	return fmt.Errorf("%s: %w", message, ErrInternal)
}

// InvalidModelOutput wraps error as invalid model output
func InvalidModelOutput(message string) error {
	return fmt.Errorf("%s: %w", message, ErrInvalidModelOutput)
}

// IsRetryable checks if an error is transient or conflict related, indicating it can be retried
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	return errors.Is(err, ErrTransient) || errors.Is(err, ErrConflict)
}
