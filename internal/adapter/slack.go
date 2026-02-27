package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/harunnryd/heike/internal/errors"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

type SlackAdapter struct {
	signingSecret string
	botToken      string
	eventHandler  EventHandler
	server        *http.Server
	port          int
	client        *slack.Client
}

func NewSlackAdapter(port int, signingSecret, botToken string, eventHandler EventHandler) *SlackAdapter {
	if signingSecret == "" {
		signingSecret = os.Getenv("SLACK_SIGNING_SECRET")
	}
	if botToken == "" {
		botToken = os.Getenv("SLACK_BOT_TOKEN")
	}
	return &SlackAdapter{
		signingSecret: signingSecret,
		botToken:      botToken,
		eventHandler:  eventHandler,
		port:          port,
		client:        slack.New(botToken),
	}
}

func (s *SlackAdapter) Name() string {
	return "slack"
}

func (s *SlackAdapter) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/slack/events", s.handleEvents)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	go func() {
		slog.Info("Slack Adapter listening", "port", s.port)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Slack server failed", "error", err)
		}
	}()

	<-ctx.Done()
	return s.server.Shutdown(context.Background())
}

func (s *SlackAdapter) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

func (s *SlackAdapter) Send(ctx context.Context, sessionID string, content string) error {
	// sessionID maps to channel ID for Slack
	_, _, err := s.client.PostMessageContext(ctx, sessionID, slack.MsgOptionText(content, false))
	if err != nil {
		return errors.Wrap(err, "failed to send Slack message")
	}
	slog.Debug("Slack message sent", "channel", sessionID)
	return nil
}

func (s *SlackAdapter) Health(ctx context.Context) error {
	if s.server == nil {
		return errors.Transient("Slack server not started")
	}

	if s.client == nil {
		return errors.Transient("Slack client not initialized")
	}

	// Check if client can connect
	_, err := s.client.AuthTestContext(ctx)
	if err != nil {
		return errors.Transient("Slack connection failed")
	}

	return nil
}

func (s *SlackAdapter) handleEvents(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sv, err := slack.NewSecretsVerifier(r.Header, s.signingSecret)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if _, err := sv.Write(body); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := sv.Ensure(); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if eventsAPIEvent.Type == slackevents.URLVerification {
		var r *slackevents.ChallengeResponse
		err := json.Unmarshal([]byte(body), &r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(r.Challenge))
		return
	}

	if eventsAPIEvent.Type == slackevents.CallbackEvent {
		innerEvent := eventsAPIEvent.InnerEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.MessageEvent:
			// Ignore bot messages
			if ev.BotID != "" {
				return
			}

			metadata := map[string]string{
				"user_id": ev.User,
				"ts":      ev.TimeStamp,
			}

			// Call event handler instead of submitting directly to ingress
			// This fixes circular dependency
			if s.eventHandler != nil {
				if err := s.eventHandler(r.Context(), "slack", "user_message", ev.Channel, ev.Text, metadata); err != nil {
					slog.Error("Failed to handle Slack event", "error", err)
				}
			}
		}
	}

	w.WriteHeader(http.StatusOK)
}
