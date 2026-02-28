package errors

import (
	"context"
	"errors"
	"fmt"
)

// Sentinel errors for different categories
var (
	// ErrDuplicateEvent - duplicate event detected (ignore silently in interactive, ignore silently in background)
	ErrDuplicateEvent = errors.New("duplicate event")

	// ErrApprovalRequired - approval required (show approval request in interactive, persist pending approval in background)
	ErrApprovalRequired = errors.New("approval required")

	// ErrPermissionDenied - permission denied (show message in interactive, fail job in background)
	ErrPermissionDenied = errors.New("permission denied")

	// ErrInvalidInput - invalid input (show validation error in interactive, fail job in background)
	ErrInvalidInput = errors.New("invalid input")

	// ErrNotFound - resource not found (show error in interactive, fail job in background)
	ErrNotFound = errors.New("not found")

	// ErrConflict - conflict (queue/retry deterministically in interactive, retry with backoff in background)
	ErrConflict = errors.New("conflict")

	// ErrTransient - transient error (show retry hint in interactive, retry with backoff in background)
	ErrTransient = errors.New("transient error")

	// ErrInvalidModelOutput - model returned malformed structured output
	ErrInvalidModelOutput = errors.New("invalid model output")

	// ErrInternal - internal error (generic message + trace id in interactive, retry once then fail in background)
	ErrInternal = errors.New("internal error")
)

func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

func WrapWithCategory(err error, message string, category error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, category)
}

func IsCategory(err error, category error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, category)
}

func NotFound(message string) error {
	return fmt.Errorf("%s: %w", message, ErrNotFound)
}

func PermissionDenied(message string) error {
	return fmt.Errorf("%s: %w", message, ErrPermissionDenied)
}

func InvalidInput(message string) error {
	return fmt.Errorf("%s: %w", message, ErrInvalidInput)
}

func Transient(message string) error {
	return fmt.Errorf("%s: %w", message, ErrTransient)
}

func Internal(message string) error {
	return fmt.Errorf("%s: %w", message, ErrInternal)
}

func InvalidModelOutput(message string) error {
	return fmt.Errorf("%s: %w", message, ErrInvalidModelOutput)
}

func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	return errors.Is(err, ErrTransient) || errors.Is(err, ErrConflict)
}
