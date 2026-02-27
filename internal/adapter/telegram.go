package adapter

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/errors"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramAdapter struct {
	token         string
	updateTimeout int
	eventHandler  EventHandler
	bot           *tgbotapi.BotAPI
	updates       tgbotapi.UpdatesChannel
}

func NewTelegramAdapter(token string, eventHandler EventHandler, updateTimeout int) *TelegramAdapter {
	if updateTimeout <= 0 {
		updateTimeout = config.DefaultTelegramUpdateTimeout
	}
	return &TelegramAdapter{
		token:         token,
		updateTimeout: updateTimeout,
		eventHandler:  eventHandler,
	}
}

func (t *TelegramAdapter) Name() string {
	return "telegram"
}

func (t *TelegramAdapter) Start(ctx context.Context) error {
	var err error
	t.bot, err = tgbotapi.NewBotAPI(t.token)
	if err != nil {
		return errors.Wrap(err, "failed to init telegram bot")
	}

	slog.Info("Telegram Adapter started", "user", t.bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = t.updateTimeout

	t.updates = t.bot.GetUpdatesChan(u)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case update := <-t.updates:
				t.handleUpdate(ctx, update)
			}
		}
	}()

	return nil
}

func (t *TelegramAdapter) Stop(ctx context.Context) error {
	return nil
}

func (t *TelegramAdapter) handleUpdate(ctx context.Context, update tgbotapi.Update) {
	if update.Message == nil {
		return
	}

	// Telegram UpdateID is unique and sequential
	// MessageID is unique per chat
	// We need globally unique ID for idempotency.
	// "telegram:<UpdateID>" is good.

	msg := update.Message

	// Convert Chat ID to string
	sessionID := fmt.Sprintf("%d", msg.Chat.ID)

	metadata := map[string]string{
		"user_id":   fmt.Sprintf("%d", msg.From.ID),
		"user_name": msg.From.UserName,
		"msg_id":    fmt.Sprintf("%d", msg.MessageID),
	}

	// Call event handler instead of submitting directly to ingress
	// This fixes circular dependency
	if t.eventHandler != nil {
		if err := t.eventHandler(ctx, "telegram", "user_message", sessionID, msg.Text, metadata); err != nil {
			slog.Error("Failed to handle Telegram event", "error", err)
		}
	}
}

// Send sends a reply back to Telegram
func (t *TelegramAdapter) Send(ctx context.Context, sessionID string, content string) error {
	chatID, err := strconv.ParseInt(sessionID, 10, 64)
	if err != nil {
		return errors.InvalidInput("invalid telegram session ID: " + err.Error())
	}

	msg := tgbotapi.NewMessage(chatID, content)
	_, err = t.bot.Send(msg)
	if err != nil {
		return errors.Wrap(err, "failed to send telegram message")
	}

	slog.Debug("Telegram message sent", "chat_id", sessionID)
	return nil
}

func (t *TelegramAdapter) Health(ctx context.Context) error {
	if t.bot == nil {
		return errors.Transient("Telegram bot not initialized")
	}

	// Check bot info
	_, err := t.bot.GetMe()
	if err != nil {
		return errors.Transient("Telegram connection failed: " + err.Error())
	}

	return nil
}
