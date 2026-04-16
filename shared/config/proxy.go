package config

import (
	"fmt"
	"net/url"
	"strings"
)

const socks5Scheme = "socks5"

// NormalizeSocks5Proxy returns a canonical socks5:// URL for the configured proxy.
// Supported inputs are "host:port", "user:pass@host:port", and already-prefixed
// SOCKS5 URLs.
func NormalizeSocks5Proxy(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}

	candidate := raw
	if !strings.Contains(candidate, "://") {
		candidate = socks5Scheme + "://" + candidate
	}

	parsed, err := url.Parse(candidate)
	if err != nil {
		return "", fmt.Errorf("parse proxy URL: %w", err)
	}

	if parsed.Scheme != socks5Scheme && parsed.Scheme != "socks5h" {
		return "", fmt.Errorf("unsupported proxy scheme %q", parsed.Scheme)
	}
	if parsed.Hostname() == "" {
		return "", fmt.Errorf("proxy host is required")
	}
	if parsed.Port() == "" {
		return "", fmt.Errorf("proxy port is required")
	}

	return parsed.String(), nil
}
