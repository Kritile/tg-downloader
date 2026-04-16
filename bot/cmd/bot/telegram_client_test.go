package main

import (
	"net/http"
	"testing"
)

func TestNewTelegramHTTPClient(t *testing.T) {
	t.Run("without proxy uses default client transport", func(t *testing.T) {
		client, err := newTelegramHTTPClient("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client.Transport != nil {
			t.Fatalf("expected nil transport for direct client, got %T", client.Transport)
		}
	})

	t.Run("with proxy configures custom transport", func(t *testing.T) {
		client, err := newTelegramHTTPClient("127.0.0.1:1080")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		transport, ok := client.Transport.(*http.Transport)
		if !ok {
			t.Fatalf("expected *http.Transport, got %T", client.Transport)
		}
		if transport.DialContext == nil {
			t.Fatal("expected proxy-aware DialContext")
		}
	})

	t.Run("invalid proxy fails fast", func(t *testing.T) {
		_, err := newTelegramHTTPClient("://bad proxy")
		if err == nil {
			t.Fatal("expected error for malformed proxy")
		}
	})
}
