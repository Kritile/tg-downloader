package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mediaharvester/tg-downloader/shared/config"
	xproxy "golang.org/x/net/proxy"
)

func initTelegramBotAPI(token string, proxyAddr string) (*tgbotapi.BotAPI, error) {
	client, err := newTelegramHTTPClient(proxyAddr)
	if err != nil {
		return nil, err
	}

	return tgbotapi.NewBotAPIWithClient(token, tgbotapi.APIEndpoint, client)
}

// initTelegramBotAPILocal connects to a Telegram Local Bot API server directly.
// It deliberately does not accept a proxy: Xray is reserved for YouTube yt-dlp traffic.
func initTelegramBotAPILocal(token, endpoint string) (*tgbotapi.BotAPI, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("TELEGRAM_API_URL is required")
	}
	return tgbotapi.NewBotAPIWithAPIEndpoint(token, endpoint)
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

	baseDialer := &net.Dialer{}
	dialer, err := xproxy.FromURL(proxyURL, baseDialer)
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
