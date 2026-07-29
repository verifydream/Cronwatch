package alert

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/rs/zerolog"
)

type TelegramSender struct {
	token  string
	chatID string
	log    zerolog.Logger
}

func NewTelegramSender(token, chatID string, log zerolog.Logger) *TelegramSender {
	return &TelegramSender{token: token, chatID: chatID, log: log}
}

func (t *TelegramSender) Send(message string) error {
	if t.token == "" || t.chatID == "" {
		t.log.Warn().Msg("Telegram not configured, skipping alert")
		return nil
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.token)
	payload := fmt.Sprintf(`{"chat_id":"%s","text":"%s","parse_mode":"Markdown"}`, t.chatID, message)

	resp, err := http.Post(url, "application/json", bytes.NewBufferString(payload))
	if err != nil {
		t.log.Error().Err(err).Msg("Failed to send Telegram message")
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return nil
}

type WebhookSender struct {
	url string
	log zerolog.Logger
}

func NewWebhookSender(url string, log zerolog.Logger) *WebhookSender {
	return &WebhookSender{url: url, log: log}
}

func (w *WebhookSender) Send(message string) error {
	if w.url == "" {
		return nil
	}

	payload := fmt.Sprintf(`{"text":"%s"}`, message)
	resp, err := http.Post(w.url, "application/json", bytes.NewBufferString(payload))
	if err != nil {
		w.log.Error().Err(err).Msg("Failed to send webhook")
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return nil
}
