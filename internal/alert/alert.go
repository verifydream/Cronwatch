package alert

import (
	"bytes"
	"encoding/json"
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

	payload, err := json.Marshal(map[string]string{
		"chat_id":    t.chatID,
		"text":       message,
		"parse_mode": "Markdown",
	})
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.token)
	resp, err := http.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		t.log.Error().Err(err).Msg("Failed to send Telegram message")
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("telegram API returned %s", resp.Status)
		t.log.Error().Err(err).Msg("Failed to send Telegram message")
		return err
	}
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

	payload, err := json.Marshal(map[string]string{"text": message})
	if err != nil {
		return err
	}
	resp, err := http.Post(w.url, "application/json", bytes.NewReader(payload))
	if err != nil {
		w.log.Error().Err(err).Msg("Failed to send webhook")
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 300 {
		err := fmt.Errorf("webhook returned %s", resp.Status)
		w.log.Error().Err(err).Msg("Failed to send webhook")
		return err
	}
	return nil
}
