package ingress

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	heikeErrors "github.com/harunnryd/heike/internal/errors"
)

// HTTPServer exposes an HTTP endpoint for ingesting events.
type HTTPServer struct {
	ingress *Ingress
	server  *http.Server
}

// NewHTTPServer creates a new HTTP server.
func NewHTTPServer(port int, ingress *Ingress) *HTTPServer {
	mux := http.NewServeMux()
	s := &HTTPServer{
		ingress: ingress,
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
	}

	mux.HandleFunc("/api/v1/events", s.handleEvents)
	return s
}

// Start starts the HTTP server in a goroutine.
func (s *HTTPServer) Start() {
	go func() {
		slog.Info("Starting HTTP Ingress server", "addr", s.server.Addr)
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server failed", "error", err)
		}
	}()
}

// Stop stops the HTTP server gracefully.
func (s *HTTPServer) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

type eventRequest struct {
	Source    string            `json:"source"`
	Type      string            `json:"type"`
	SessionID string            `json:"session_id"`
	Content   string            `json:"content"`
	Metadata  map[string]string `json:"metadata"`
}

func (s *HTTPServer) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req eventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Source == "" || req.Content == "" {
		http.Error(w, "Missing required fields: source, content", http.StatusBadRequest)
		return
	}

	eventType := EventType(req.Type)
	if eventType == "" {
		eventType = TypeUserMessage // Default to user
	}

	// Create event
	evt := NewEvent(req.Source, eventType, req.SessionID, req.Content, req.Metadata)

	// Submit to ingress
	if err := s.ingress.Submit(r.Context(), &evt); err != nil {
		if errors.Is(err, heikeErrors.ErrDuplicateEvent) {
			// Idempotency: Return 200 OK for duplicates, but log it
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"duplicate","id":"` + evt.ID + `"}`))
			return
		}
		if errors.Is(err, heikeErrors.ErrTransient) {
			// Queue full or any transient error
			http.Error(w, "Queue full", http.StatusTooManyRequests)
			return
		}
		slog.Error("Failed to submit event", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status":"accepted","id":"` + evt.ID + `"}`))
}
