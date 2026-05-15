package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const defaultAPIURL = "https://api.telegram.org"

// Bot is a minimal Telegram Bot API client for sending messages.
type Bot struct {
	token  string
	apiURL string
	client *http.Client
}

// NewBot creates a new Bot with the given token.
func NewBot(token string) *Bot {
	return &Bot{
		token:  token,
		apiURL: defaultAPIURL,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// sendMessageRequest is the JSON payload for the sendMessage API call.
type sendMessageRequest struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

// apiResponse represents the Telegram Bot API response envelope.
type apiResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
}

// SendMessage sends an HTML-formatted text message to the given chat.
func (b *Bot) SendMessage(chatID, text string) error {
	payload := sendMessageRequest{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "HTML",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("telegram: marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", b.apiURL, b.token)

	resp, err := b.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var apiResp apiResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			return fmt.Errorf("telegram: unexpected status %d", resp.StatusCode)
		}
		return fmt.Errorf("telegram: API error %d: %s", resp.StatusCode, apiResp.Description)
	}

	return nil
}
