package components

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/daemon"
)

type HTTPServerComponent struct {
	daemon      *daemon.Daemon
	cfg         *config.ServerConfig
	server      *http.Server
	shutdownTTL time.Duration
	initialized bool
	started     bool
	mu          sync.RWMutex
	startTime   time.Time
}

func NewHTTPServerComponent(d *daemon.Daemon, cfg *config.ServerConfig) *HTTPServerComponent {
	return &HTTPServerComponent{
		daemon:      d,
		cfg:         cfg,
		initialized: false,
		started:     false,
	}
}

func (h *HTTPServerComponent) Name() string {
	return "HTTPServer"
}

func (h *HTTPServerComponent) Dependencies() []string {
	return []string{"StoreWorker", "PolicyEngine", "Orchestrator", "Ingress", "Workers", "Scheduler"}
}

func (h *HTTPServerComponent) Init(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.handleHealth)

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
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(healthResponse)
}
