package main

import (
	"net/http"
	"testing"
)

func TestTelegramAPIEndpoint(t *testing.T) {
	tests := map[string]struct {
		input string
		want  string
	}{
		"base URL":                 {"http://telegram-bot-api:8081/", "http://telegram-bot-api:8081/bot%s/%s"},
		"formatted endpoint":       {"http://telegram-bot-api:8081/bot%s/%s", "http://telegram-bot-api:8081/bot%s/%s"},
		"empty uses local default": {"", "http://telegram-bot-api:8081/bot%s/%s"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := telegramAPIEndpoint(test.input); got != test.want {
				t.Fatalf("telegramAPIEndpoint(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestNewTelegramHTTPClientUsesConfiguredProxy(t *testing.T) {
	client, err := newTelegramHTTPClient("127.0.0.1:1080")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok || transport.DialContext == nil {
		t.Fatalf("expected proxy-aware HTTP transport, got %#v", client.Transport)
	}
}
