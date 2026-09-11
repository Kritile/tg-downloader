package main

import "testing"

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
