package config

import (
	"fmt"
	"strings"
	"time"
)

// DurationOrDefault parses a duration string and falls back to defaultValue when empty.
func DurationOrDefault(value string, defaultValue string) (time.Duration, error) {
	candidate := strings.TrimSpace(value)
	if candidate == "" {
		candidate = strings.TrimSpace(defaultValue)
	}
	if candidate == "" {
		return 0, fmt.Errorf("duration value is empty")
	}

	d, err := time.ParseDuration(candidate)
	if err != nil {
		return 0, fmt.Errorf("parse duration %q: %w", candidate, err)
	}
	return d, nil
}
