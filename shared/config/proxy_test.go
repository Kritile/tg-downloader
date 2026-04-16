package config

import "testing"

func TestNormalizeSocks5Proxy(t *testing.T) {
	t.Run("empty proxy", func(t *testing.T) {
		normalized, err := NormalizeSocks5Proxy("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if normalized != "" {
			t.Fatalf("expected empty proxy, got %q", normalized)
		}
	})

	t.Run("host port", func(t *testing.T) {
		normalized, err := NormalizeSocks5Proxy("127.0.0.1:1080")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if normalized != "socks5://127.0.0.1:1080" {
			t.Fatalf("unexpected normalized proxy: %q", normalized)
		}
	})

	t.Run("credentials", func(t *testing.T) {
		normalized, err := NormalizeSocks5Proxy("user:pass@127.0.0.1:1080")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if normalized != "socks5://user:pass@127.0.0.1:1080" {
			t.Fatalf("unexpected normalized proxy: %q", normalized)
		}
	})

	t.Run("already prefixed", func(t *testing.T) {
		normalized, err := NormalizeSocks5Proxy("socks5://user:pass@127.0.0.1:1080")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if normalized != "socks5://user:pass@127.0.0.1:1080" {
			t.Fatalf("unexpected normalized proxy: %q", normalized)
		}
	})

	t.Run("invalid proxy", func(t *testing.T) {
		_, err := NormalizeSocks5Proxy("://bad proxy")
		if err == nil {
			t.Fatal("expected error for malformed proxy")
		}
	})
}
