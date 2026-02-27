package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCallbackPathFromRedirectURI(t *testing.T) {
	tests := []struct {
		name        string
		redirectURI string
		wantPath    string
		wantErr     bool
	}{
		{
			name:        "default callback path",
			redirectURI: "http://localhost:1455/auth/callback",
			wantPath:    "/auth/callback",
		},
		{
			name:        "custom callback path",
			redirectURI: "http://127.0.0.1:9999/oauth/complete",
			wantPath:    "/oauth/complete",
		},
		{
			name:        "missing path",
			redirectURI: "http://localhost:1455",
			wantErr:     true,
		},
		{
			name:        "invalid URI",
			redirectURI: "://bad-uri",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := callbackPathFromRedirectURI(tt.redirectURI)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.wantPath {
				t.Fatalf("path mismatch: got %q want %q", got, tt.wantPath)
			}
		})
	}
}

func TestResolveTokenPath_ExpandsHomeShortcut(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("user home dir: %v", err)
	}

	got, err := ResolveTokenPath("~/.heike/auth/codex.json")
	if err != nil {
		t.Fatalf("resolve token path: %v", err)
	}

	want := filepath.Join(home, ".heike", "auth", "codex.json")
	if got != want {
		t.Fatalf("path mismatch: got %q want %q", got, want)
	}
}
