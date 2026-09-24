package main

import (
	"net/http"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// initTelegramBotAPILocal connects Go to the Local Bot API over the internal
// Docker network. The Bot API container is responsible for routing its
// outbound TDLib traffic through the configured SOCKS5 proxy. MAX has its own
// direct client and never calls this function.
func initTelegramBotAPILocal(token, endpoint string) (*tgbotapi.BotAPI, error) {
	transport := &http.Transport{
		DisableKeepAlives:  true,
		DisableCompression: true,
		ForceAttemptHTTP2:  false,
	}
	return tgbotapi.NewBotAPIWithClient(token, telegramAPIEndpoint(endpoint), &http.Client{Transport: transport})
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
