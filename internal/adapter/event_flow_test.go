package adapter

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type capturedEvent struct {
	source    string
	eventType string
	sessionID string
	content   string
	metadata  map[string]string
}

func TestTelegramAdapter_EventFlow(t *testing.T) {
	var got capturedEvent

	adapter := NewTelegramAdapter("test-token", func(ctx context.Context, source string, eventType string, sessionID string, content string, metadata map[string]string) error {
		got = capturedEvent{
			source:    source,
			eventType: eventType,
			sessionID: sessionID,
			content:   content,
			metadata:  metadata,
		}
		return nil
	}, 1)

	adapter.handleUpdate(context.Background(), tgbotapi.Update{
		UpdateID: 99,
		Message: &tgbotapi.Message{
			MessageID: 123,
			Text:      "hello from telegram",
			Chat:      &tgbotapi.Chat{ID: 456},
			From:      &tgbotapi.User{ID: 789, UserName: "alice"},
		},
	})

	if got.source != "telegram" {
		t.Fatalf("source = %q, want %q", got.source, "telegram")
	}
	if got.eventType != "user_message" {
		t.Fatalf("eventType = %q, want %q", got.eventType, "user_message")
	}
	if got.sessionID != "456" {
		t.Fatalf("sessionID = %q, want %q", got.sessionID, "456")
	}
	if got.content != "hello from telegram" {
		t.Fatalf("content = %q, want %q", got.content, "hello from telegram")
	}
	if got.metadata["user_id"] != "789" {
		t.Fatalf("metadata user_id = %q, want %q", got.metadata["user_id"], "789")
	}
	if got.metadata["user_name"] != "alice" {
		t.Fatalf("metadata user_name = %q, want %q", got.metadata["user_name"], "alice")
	}
	if got.metadata["msg_id"] != "123" {
		t.Fatalf("metadata msg_id = %q, want %q", got.metadata["msg_id"], "123")
	}
}

func TestSlackAdapter_EventFlow(t *testing.T) {
	secret := "test-signing-secret"

	var got capturedEvent
	adapter := NewSlackAdapter(0, secret, "xoxb-test", func(ctx context.Context, source string, eventType string, sessionID string, content string, metadata map[string]string) error {
		got = capturedEvent{
			source:    source,
			eventType: eventType,
			sessionID: sessionID,
			content:   content,
			metadata:  metadata,
		}
		return nil
	})

	body := []byte(`{"type":"event_callback","event":{"type":"message","user":"U123","text":"hello from slack","channel":"C123","ts":"1710000000.000100"}}`)
	req := httptest.NewRequest(http.MethodPost, "/slack/events", bytes.NewReader(body))

	ts := strconv.FormatInt(time.Now().Unix(), 10)
	base := "v0:" + ts + ":" + string(body)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(base))
	sig := "v0=" + hex.EncodeToString(mac.Sum(nil))

	req.Header.Set("X-Slack-Request-Timestamp", ts)
	req.Header.Set("X-Slack-Signature", sig)

	rr := httptest.NewRecorder()
	adapter.handleEvents(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	if got.source != "slack" {
		t.Fatalf("source = %q, want %q", got.source, "slack")
	}
	if got.eventType != "user_message" {
		t.Fatalf("eventType = %q, want %q", got.eventType, "user_message")
	}
	if got.sessionID != "C123" {
		t.Fatalf("sessionID = %q, want %q", got.sessionID, "C123")
	}
	if got.content != "hello from slack" {
		t.Fatalf("content = %q, want %q", got.content, "hello from slack")
	}
	if got.metadata["user_id"] != "U123" {
		t.Fatalf("metadata user_id = %q, want %q", got.metadata["user_id"], "U123")
	}
	if got.metadata["ts"] != "1710000000.000100" {
		t.Fatalf("metadata ts = %q, want %q", got.metadata["ts"], "1710000000.000100")
	}
}
