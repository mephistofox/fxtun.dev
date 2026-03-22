package telegram

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestBot_SendMessage(t *testing.T) {
	const (
		token  = "123456:ABC-DEF"
		chatID = "99887766"
		text   = "<b>Hello</b> world"
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify method.
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		// Verify path includes token.
		expectedPath := "/bot" + token + "/sendMessage"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
		}

		// Verify content type.
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}

		// Verify payload.
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}

		var req sendMessageRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}

		if req.ChatID != chatID {
			t.Errorf("expected chat_id %s, got %s", chatID, req.ChatID)
		}
		if req.Text != text {
			t.Errorf("expected text %q, got %q", text, req.Text)
		}
		if req.ParseMode != "HTML" {
			t.Errorf("expected parse_mode HTML, got %s", req.ParseMode)
		}

		// Respond with success.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(apiResponse{OK: true})
	}))
	defer srv.Close()

	bot := NewBot(token)
	bot.apiURL = srv.URL

	if err := bot.SendMessage(chatID, text); err != nil {
		t.Fatalf("SendMessage returned error: %v", err)
	}
}

func TestBot_SendMessage_Error(t *testing.T) {
	const (
		token       = "123456:ABC-DEF"
		chatID      = "invalid"
		text        = "test"
		description = "Bad Request: chat not found"
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(apiResponse{
			OK:          false,
			Description: description,
		})
	}))
	defer srv.Close()

	bot := NewBot(token)
	bot.apiURL = srv.URL

	err := bot.SendMessage(chatID, text)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	expected := "telegram: API error 400: " + description
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

// capturingSrv returns an httptest.Server that captures the last sent message text
// and a function to retrieve it. Thread-safe for use with goroutine-based send.
func capturingSrv(t *testing.T) (*httptest.Server, func() string) {
	t.Helper()

	var (
		mu       sync.Mutex
		captured string
		done     = make(chan struct{}, 1)
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}

		var req sendMessageRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("unmarshal body: %v", err)
		}

		mu.Lock()
		captured = req.Text
		mu.Unlock()

		select {
		case done <- struct{}{}:
		default:
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(apiResponse{OK: true})
	}))

	getText := func() string {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for message to be sent")
		}
		mu.Lock()
		defer mu.Unlock()
		return captured
	}

	return srv, getText
}

func TestAdminNotifier_NotifyNewUser(t *testing.T) {
	srv, getText := capturingSrv(t)
	defer srv.Close()

	bot := NewBot("test-token")
	bot.apiURL = srv.URL

	notifier := NewAdminNotifier(bot, "12345")
	notifier.NotifyNewUser(42, "John Doe", "john@example.com")

	text := getText()

	checks := []string{
		"Новый пользователь",
		"John Doe",
		"john@example.com",
		"ID: 42",
	}
	for _, c := range checks {
		if !strings.Contains(text, c) {
			t.Errorf("expected message to contain %q, got:\n%s", c, text)
		}
	}
}

func TestAdminNotifier_NotifyNewUser_EscapesHTML(t *testing.T) {
	srv, getText := capturingSrv(t)
	defer srv.Close()

	bot := NewBot("test-token")
	bot.apiURL = srv.URL

	notifier := NewAdminNotifier(bot, "12345")
	notifier.NotifyNewUser(1, "<script>alert(1)</script>", "a&b@test.com")

	text := getText()

	if strings.Contains(text, "<script>") {
		t.Error("message contains unescaped <script> tag")
	}
	if !strings.Contains(text, "&lt;script&gt;") {
		t.Error("expected escaped <script> tag")
	}
	if !strings.Contains(text, "a&amp;b@test.com") {
		t.Error("expected escaped ampersand in email")
	}
}

func TestAdminNotifier_NotifyNewSubscription(t *testing.T) {
	srv, getText := capturingSrv(t)
	defer srv.Close()

	bot := NewBot("test-token")
	bot.apiURL = srv.URL

	notifier := NewAdminNotifier(bot, "12345")
	notifier.NotifyNewSubscription(7, "Alice", "Pro", 9.99, "stripe")

	text := getText()

	checks := []string{
		"Новая подписка",
		"Alice",
		"ID: 7",
		"Pro",
		"9.99",
		"stripe",
	}
	for _, c := range checks {
		if !strings.Contains(text, c) {
			t.Errorf("expected message to contain %q, got:\n%s", c, text)
		}
	}
}

func TestAdminNotifier_NotifyFirstTunnel(t *testing.T) {
	srv, getText := capturingSrv(t)
	defer srv.Close()

	bot := NewBot("test-token")
	bot.apiURL = srv.URL

	notifier := NewAdminNotifier(bot, "12345")

	registeredAt := time.Now().Add(-2 * time.Hour)
	notifier.NotifyFirstTunnel(99, "Bob", "http", "app.example.com:8080", registeredAt)

	text := getText()

	checks := []string{
		"Первый туннель",
		"Bob",
		"ID: 99",
		"http",
		"app.example.com:8080",
		"назад",
	}
	for _, c := range checks {
		if !strings.Contains(text, c) {
			t.Errorf("expected message to contain %q, got:\n%s", c, text)
		}
	}
}

func TestEscapeHTML(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"hello", "hello"},
		{"<b>bold</b>", "&lt;b&gt;bold&lt;/b&gt;"},
		{"a & b", "a &amp; b"},
		{"<>&", "&lt;&gt;&amp;"},
		{"", ""},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := escapeHTML(tc.input)
			if got != tc.expected {
				t.Errorf("escapeHTML(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d        time.Duration
		expected string
	}{
		{30 * time.Second, "менее минуты"},
		{0, "менее минуты"},
		{5 * time.Minute, "5 мин"},
		{59 * time.Minute, "59 мин"},
		{1 * time.Hour, "1 ч"},
		{23 * time.Hour, "23 ч"},
		{24 * time.Hour, "1 д"},
		{72 * time.Hour, "3 д"},
		{365 * 24 * time.Hour, "365 д"},
	}
	for _, tc := range tests {
		t.Run(fmt.Sprintf("%v", tc.d), func(t *testing.T) {
			got := formatDuration(tc.d)
			if got != tc.expected {
				t.Errorf("formatDuration(%v) = %q, want %q", tc.d, got, tc.expected)
			}
		})
	}
}
