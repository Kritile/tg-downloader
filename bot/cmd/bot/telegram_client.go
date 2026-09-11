package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mediaharvester/tg-downloader/shared/config"
	xproxy "golang.org/x/net/proxy"
)

// initTelegramBotAPILocal connects Telegram to the configured endpoint through
// the shared SOCKS5 Xray transport. MAX has its own direct client and never
// calls this function.
func initTelegramBotAPILocal(token, endpoint, proxyAddr string) (*tgbotapi.BotAPI, error) {
	client, err := newTelegramHTTPClient(proxyAddr)
	if err != nil {
		return nil, err
	}
	return tgbotapi.NewBotAPIWithClient(token, telegramAPIEndpoint(endpoint), client)
}

func newTelegramHTTPClient(proxyAddr string) (*http.Client, error) {
	normalized, err := config.NormalizeSocks5Proxy(proxyAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid XRAY_SOCKS5_PROXY: %w", err)
	}
	if normalized == "" {
		return &http.Client{}, nil
	}
	proxyURL, err := url.Parse(normalized)
	if err != nil {
		return nil, fmt.Errorf("parse normalized proxy URL: %w", err)
	}
	dialer, err := xproxy.FromURL(proxyURL, &net.Dialer{})
	if err != nil {
		return nil, fmt.Errorf("create SOCKS5 dialer: %w", err)
	}
	transport := &http.Transport{}
	if contextDialer, ok := dialer.(xproxy.ContextDialer); ok {
		transport.DialContext = contextDialer.DialContext
	} else {
		transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
			return xproxy.Dial(ctx, network, address)
		}
	}
	return &http.Client{Transport: transport}, nil
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
