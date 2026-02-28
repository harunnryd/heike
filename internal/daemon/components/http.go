package components

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/daemon"
	heikeErrors "github.com/harunnryd/heike/internal/errors"
)

type HTTPServerComponent struct {
	daemon      *daemon.Daemon
	runtime     daemon.RuntimeAPI
	cfg         *config.ServerConfig
	deps        []string
	server      *http.Server
	shutdownTTL time.Duration
	initialized bool
	started     bool
	mu          sync.RWMutex
	startTime   time.Time
}

func NewHTTPServerComponent(d *daemon.Daemon, cfg *config.ServerConfig) *HTTPServerComponent {
	return NewHTTPServerComponentWithDependencies(d, cfg, []string{
		"Runtime",
	})
}

func NewHTTPServerComponentWithDependencies(d *daemon.Daemon, cfg *config.ServerConfig, deps []string) *HTTPServerComponent {
	depList := make([]string, len(deps))
	copy(depList, deps)
	return &HTTPServerComponent{
		daemon:      d,
		cfg:         cfg,
		deps:        depList,
		initialized: false,
		started:     false,
	}
}

func (h *HTTPServerComponent) Name() string {
	return "HTTPServer"
}

func (h *HTTPServerComponent) Dependencies() []string {
	deps := make([]string, len(h.deps))
	copy(deps, h.deps)
	return deps
}

func (h *HTTPServerComponent) Init(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.daemon == nil {
		return fmt.Errorf("daemon manager not configured")
	}
	runtimeComp := h.daemon.Component("Runtime")
	if runtimeComp == nil {
		return fmt.Errorf("runtime component not registered")
	}
	runtimeAPI, ok := runtimeComp.(daemon.RuntimeAPI)
	if !ok {
		return fmt.Errorf("runtime component does not implement daemon runtime api")
	}
	h.runtime = runtimeAPI

	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.handleHealth)
	mux.HandleFunc("/api/v1/events", h.handleEvents)
	mux.HandleFunc("/api/v1/sessions", h.handleSessions)
	mux.HandleFunc("/api/v1/sessions/", h.handleSessions)
	mux.HandleFunc("/api/v1/approvals", h.handleApprovals)
	mux.HandleFunc("/api/v1/approvals/", h.handleApprovals)
	mux.HandleFunc("/api/v1/zanshin/status", h.handleZanshinStatus)

	readTimeout, err := config.DurationOrDefault(h.cfg.ReadTimeout, config.DefaultServerReadTimeout)
	if err != nil {
		return fmt.Errorf("parse server read timeout: %w", err)
	}
	writeTimeout, err := config.DurationOrDefault(h.cfg.WriteTimeout, config.DefaultServerWriteTimeout)
	if err != nil {
		return fmt.Errorf("parse server write timeout: %w", err)
	}
	idleTimeout, err := config.DurationOrDefault(h.cfg.IdleTimeout, config.DefaultServerIdleTimeout)
	if err != nil {
		return fmt.Errorf("parse server idle timeout: %w", err)
	}
	shutdownTimeout, err := config.DurationOrDefault(h.cfg.ShutdownTimeout, config.DefaultServerShutdownTimeout)
	if err != nil {
		return fmt.Errorf("parse server shutdown timeout: %w", err)
	}

	h.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", h.cfg.Port),
		Handler:      mux,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}
	h.shutdownTTL = shutdownTimeout

	h.initialized = true
	slog.Info("HTTPServer initialized", "component", h.Name(), "port", h.cfg.Port)
	return nil
}

func (h *HTTPServerComponent) Start(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.initialized {
		return fmt.Errorf("HTTPServer not initialized")
	}

	go func() {
		slog.Info("HTTP server listening", "component", h.Name(), "addr", h.server.Addr)
		if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server failed", "component", h.Name(), "error", err)
		}
	}()

	h.started = true
	h.startTime = time.Now()
	slog.Info("HTTPServer started", "component", h.Name())
	return nil
}

func (h *HTTPServerComponent) Stop(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.started {
		slog.Info("HTTPServer not started, skipping stop", "component", h.Name())
		return nil
	}

	slog.Info("Stopping HTTPServer...", "component", h.Name())
	shutdownCtx, cancel := context.WithTimeout(ctx, h.shutdownTTL)
	defer cancel()

	if err := h.server.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTPServer shutdown error", "component", h.Name(), "error", err)
		return err
	}

	h.started = false
	slog.Info("HTTPServer stopped", "component", h.Name())
	return nil
}

func (h *HTTPServerComponent) Health(ctx context.Context) (*daemon.ComponentHealth, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if !h.initialized {
		return &daemon.ComponentHealth{
			Name:    h.Name(),
			Healthy: false,
			Error:   fmt.Errorf("not initialized"),
		}, nil
	}

	if !h.started {
		return &daemon.ComponentHealth{
			Name:    h.Name(),
			Healthy: false,
			Error:   fmt.Errorf("not started"),
		}, nil
	}

	return &daemon.ComponentHealth{
		Name:    h.Name(),
		Healthy: true,
		Error:   nil,
	}, nil
}

func (h *HTTPServerComponent) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"error": "method not allowed"})
		return
	}

	healthResponse := map[string]interface{}{
		"status":  "ok",
		"version": "1.0.0",
	}

	componentHealths := h.daemon.ComponentHealth()
	componentHealthMap := make(map[string]interface{})
	for name, ch := range componentHealths {
		componentHealthMap[name] = map[string]interface{}{
			"healthy": ch.Healthy,
		}
		if ch.Error != nil {
			componentHealthMap[name].(map[string]interface{})["error"] = ch.Error.Error()
		}
	}

	healthResponse["components"] = componentHealthMap
	writeJSON(w, http.StatusOK, healthResponse)
}

type eventRequest struct {
	Source    string            `json:"source"`
	Type      string            `json:"type"`
	SessionID string            `json:"session_id"`
	Content   string            `json:"content"`
	Metadata  map[string]string `json:"metadata"`
}

func (h *HTTPServerComponent) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"error": "method not allowed"})
		return
	}
	var req eventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid request body"})
		return
	}
	id, err := h.runtime.SubmitEvent(r.Context(), daemon.RuntimeEvent{
		Source:    strings.TrimSpace(req.Source),
		Type:      strings.TrimSpace(req.Type),
		SessionID: strings.TrimSpace(req.SessionID),
		Content:   req.Content,
		Metadata:  req.Metadata,
	})
	if err != nil {
		switch {
		case errors.Is(err, heikeErrors.ErrDuplicateEvent):
			writeJSON(w, http.StatusOK, map[string]interface{}{"status": "duplicate", "id": id})
		case errors.Is(err, heikeErrors.ErrTransient):
			writeJSON(w, http.StatusTooManyRequests, map[string]interface{}{"error": "queue full"})
		default:
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
		}
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]interface{}{"status": "accepted", "id": id})
}

func (h *HTTPServerComponent) handleSessions(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/v1/sessions" {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"error": "method not allowed"})
			return
		}
		sessions, err := h.runtime.ListSessions(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"sessions": sessions})
		return
	}

	// /api/v1/sessions/{id}/stream
	if !strings.HasPrefix(r.URL.Path, "/api/v1/sessions/") || !strings.HasSuffix(r.URL.Path, "/stream") {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"error": "not found"})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"error": "method not allowed"})
		return
	}
	raw := strings.TrimPrefix(r.URL.Path, "/api/v1/sessions/")
	sessionID := strings.TrimSuffix(raw, "/stream")
	sessionID = strings.Trim(sessionID, "/")
	if sessionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "session id is required"})
		return
	}
	h.streamSession(w, r, sessionID)
}

func (h *HTTPServerComponent) streamSession(w http.ResponseWriter, r *http.Request, sessionID string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": "streaming is not supported"})
		return
	}

	from := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("from")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid from query"})
			return
		}
		from = n
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	writeSSE(w, "connected")
	flusher.Flush()

	ticker := time.NewTicker(400 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			lines, err := h.runtime.ReadTranscript(r.Context(), sessionID, 0)
			if err != nil {
				writeSSE(w, fmt.Sprintf("{\"error\":%q}", err.Error()))
				flusher.Flush()
				return
			}
			if from > len(lines) {
				from = len(lines)
			}
			for ; from < len(lines); from++ {
				writeSSE(w, lines[from])
			}
			flusher.Flush()
		}
	}
}

func (h *HTTPServerComponent) handleApprovals(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/v1/approvals" {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"error": "method not allowed"})
			return
		}
		approvals, err := h.runtime.ListPendingApprovals(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"approvals": approvals})
		return
	}

	if !strings.HasPrefix(r.URL.Path, "/api/v1/approvals/") || !strings.HasSuffix(r.URL.Path, "/resolve") {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"error": "not found"})
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"error": "method not allowed"})
		return
	}
	raw := strings.TrimPrefix(r.URL.Path, "/api/v1/approvals/")
	approvalID := strings.TrimSuffix(raw, "/resolve")
	approvalID = strings.Trim(approvalID, "/")
	if approvalID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "approval id is required"})
		return
	}

	var req struct {
		Approve bool `json:"approve"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid request body"})
		return
	}
	if err := h.runtime.ResolveApproval(r.Context(), approvalID, req.Approve); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "resolved", "id": approvalID, "approve": req.Approve})
}

func (h *HTTPServerComponent) handleZanshinStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"error": "method not allowed"})
		return
	}
	writeJSON(w, http.StatusOK, h.runtime.ZanshinStatus(r.Context()))
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeSSE(w http.ResponseWriter, data string) {
	_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
}
