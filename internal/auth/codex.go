package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/pathutil"
)

const (
	codexOAuthClientID   = "app_EMoamEEZ73f0CkXaXp7hrann"
	codexOAuthIssuer     = "https://auth.openai.com"
	codexOAuthAuthorize  = "https://auth.openai.com/oauth/authorize"
	codexOAuthTokenURL   = "https://auth.openai.com/oauth/token"
	codexOAuthScope      = "openid profile email offline_access"
	codexOAuthOriginator = "codex_cli_rs"
	defaultCallbackAddr  = config.DefaultCodexAuthCallbackAddr
	defaultRedirectURI   = config.DefaultCodexAuthRedirectURI
	defaultOAuthTimeout  = config.DefaultCodexAuthOAuthTimeout
)

const codexOAuthSuccessHTML = "<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\" /><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\" /><title>Authentication successful</title></head><body><p>Authentication successful. Return to your terminal to continue.</p></body></html>"

type CodexToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	IDToken      string `json:"id_token"`
	AccountID    string `json:"account_id,omitempty"`
}

type CodexOAuthConfig struct {
	CallbackAddr string
	RedirectURI  string
	OAuthTimeout string
	TokenPath    string
}

type resolvedOAuthConfig struct {
	CallbackAddr string
	RedirectURI  string
	Timeout      time.Duration
	TokenPath    string
}

// LoginCodexOAuthInteractive performs the PKCE OAuth flow for OpenAI Codex
func LoginCodexOAuthInteractive(ctx context.Context, cfg CodexOAuthConfig) (*CodexToken, error) {
	resolvedCfg, err := resolveOAuthConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("resolve oauth config: %w", err)
	}

	// Generate PKCE Verifier & Challenge
	verifier, challenge, err := generatePKCE()
	if err != nil {
		return nil, fmt.Errorf("pkce generation failed: %w", err)
	}

	// Generate State
	state, err := createState()
	if err != nil {
		return nil, fmt.Errorf("state generation failed: %w", err)
	}

	// Start Local Callback Server
	codeCh := make(chan string, 1)
	server, err := startLocalServer(state, codeCh, resolvedCfg.CallbackAddr, resolvedCfg.RedirectURI)
	if err != nil {
		return nil, fmt.Errorf("failed to start local server: %w", err)
	}
	defer server.Close()

	// Start Browser
	authURL := buildAuthorizeURL(state, challenge, resolvedCfg.RedirectURI)
	fmt.Printf("Opening browser to: %s\n", authURL)
	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Failed to open browser automatically. Please visit the URL above manually.\n")
	}

	// Wait for Code
	fmt.Println("Waiting for authentication callback...")
	var code string
	select {
	case code = <-codeCh:
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(resolvedCfg.Timeout):
		return nil, fmt.Errorf("authentication timed out")
	}

	if code == "" {
		return nil, fmt.Errorf("received empty authorization code")
	}

	// Exchange Code for Token
	fmt.Println("Exchanging code for token...")
	return exchangeCode(ctx, code, verifier, resolvedCfg.RedirectURI)
}

func generatePKCE() (verifier, challenge string, err error) {
	rnd := make([]byte, 32)
	if _, err := rand.Read(rnd); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(rnd)
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

func createState() (string, error) {
	rnd := make([]byte, 16)
	if _, err := rand.Read(rnd); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(rnd), nil
}

func buildAuthorizeURL(state, challenge string, redirectURI string) string {
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", codexOAuthClientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("scope", codexOAuthScope)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	q.Set("state", state)
	q.Set("id_token_add_organizations", "true")
	q.Set("codex_cli_simplified_flow", "true")
	q.Set("originator", codexOAuthOriginator)
	return codexOAuthAuthorize + "?" + q.Encode()
}

func startLocalServer(expectedState string, codeCh chan<- string, callbackAddr string, redirectURI string) (io.Closer, error) {
	callbackPath, err := callbackPathFromRedirectURI(redirectURI)
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		state := r.URL.Query().Get("state")
		if state != expectedState {
			http.Error(w, "State mismatch", http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Missing code", http.StatusBadRequest)
			return
		}

		select {
		case codeCh <- code:
		default:
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(codexOAuthSuccessHTML))
	})

	ln, err := net.Listen("tcp", callbackAddr)
	if err != nil {
		return nil, err
	}
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()

	return srv, nil
}

func callbackPathFromRedirectURI(redirectURI string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(redirectURI))
	if err != nil {
		return "", fmt.Errorf("parse redirect URI: %w", err)
	}
	if u.Path == "" {
		return "", fmt.Errorf("redirect URI path is empty")
	}
	return u.EscapedPath(), nil
}

func exchangeCode(ctx context.Context, code, verifier string, redirectURI string) (*CodexToken, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", codexOAuthClientID)
	form.Set("code", code)
	form.Set("code_verifier", verifier)
	form.Set("redirect_uri", redirectURI)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, codexOAuthTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed: %s", string(body))
	}

	var token CodexToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}

	// Decode Account ID from Access Token (JWT)
	// Simplified: just take it if present in claims, otherwise ignore
	// Real implementation needs JWT parsing, but for now we just return the token
	// Assuming the caller will use AccessToken as Bearer token.

	return &token, nil
}

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

func SaveToken(token *CodexToken, tokenPath string) error {
	path, err := ResolveTokenPath(tokenPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(token)
}

func ResolveTokenPath(tokenPath string) (string, error) {
	path := strings.TrimSpace(tokenPath)
	if path != "" {
		return pathutil.Expand(path)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".heike", "auth", "codex.json"), nil
}

func resolveOAuthConfig(cfg CodexOAuthConfig) (resolvedOAuthConfig, error) {
	callbackAddr := strings.TrimSpace(cfg.CallbackAddr)
	if callbackAddr == "" {
		callbackAddr = defaultCallbackAddr
	}

	redirectURI := strings.TrimSpace(cfg.RedirectURI)
	if redirectURI == "" {
		redirectURI = defaultRedirectURI
	}

	timeoutValue := strings.TrimSpace(cfg.OAuthTimeout)
	if timeoutValue == "" {
		timeoutValue = defaultOAuthTimeout
	}
	timeout, err := time.ParseDuration(timeoutValue)
	if err != nil {
		return resolvedOAuthConfig{}, fmt.Errorf("parse oauth timeout %q: %w", timeoutValue, err)
	}

	return resolvedOAuthConfig{
		CallbackAddr: callbackAddr,
		RedirectURI:  redirectURI,
		Timeout:      timeout,
		TokenPath:    strings.TrimSpace(cfg.TokenPath),
	}, nil
}
