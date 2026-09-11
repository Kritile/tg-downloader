package main

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// initTelegramBotAPILocal connects to a Telegram Local Bot API server directly.
// It deliberately does not accept a proxy: Xray is reserved for yt-dlp YouTube traffic.
func initTelegramBotAPILocal(token, endpoint string) (*tgbotapi.BotAPI, error) {
	return tgbotapi.NewBotAPIWithAPIEndpoint(token, telegramAPIEndpoint(endpoint))
}

// telegramAPIEndpoint accepts either a Local Bot API base URL or the complete
// endpoint format expected by go-telegram-bot-api.
func telegramAPIEndpoint(endpoint string) string {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if endpoint == "" {
		endpoint = "http://telegram-bot-api:8081"
	}
	if strings.Contains(endpoint, "%s") {
		return endpoint
	}
	return endpoint + "/bot%s/%s"
}
