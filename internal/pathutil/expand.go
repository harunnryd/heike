package pathutil

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// Expand resolves environment variables and "~/" home shortcuts.
func Expand(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", nil
	}

	expanded := os.ExpandEnv(trimmed)
	if expanded == "~" || strings.HasPrefix(expanded, "~/") {
		home, err := resolveHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home dir: %w", err)
		}
		if expanded == "~" {
			expanded = home
		} else {
			expanded = filepath.Join(home, strings.TrimPrefix(expanded, "~/"))
		}
	}

	return filepath.Clean(expanded), nil
}

func resolveHomeDir() (string, error) {
	if home, err := os.UserHomeDir(); err == nil {
		trimmed := strings.TrimSpace(home)
		if trimmed != "" && trimmed != "~" && !strings.HasPrefix(trimmed, "~/") {
			return trimmed, nil
		}
	}

	if current, err := user.Current(); err == nil {
		trimmed := strings.TrimSpace(current.HomeDir)
		if trimmed != "" && trimmed != "~" && !strings.HasPrefix(trimmed, "~/") {
			return trimmed, nil
		}
	}

	envHome := strings.TrimSpace(os.Getenv("HOME"))
	if envHome == "" {
		return "", fmt.Errorf("HOME is not set")
	}
	if envHome == "~" || strings.HasPrefix(envHome, "~/") {
		return "", fmt.Errorf("HOME is not fully resolved: %s", envHome)
	}
	return envHome, nil
}
