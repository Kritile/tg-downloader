package worker

import (
	"testing"
)

func TestBuildYtDlpArgs(t *testing.T) {
	wp := &WorkerPool{proxyAddr: "xray-client:10808"}

	t.Run("YouTube with proxy and format", func(t *testing.T) {
		args := wp.buildYtDlpArgs(
			"https://youtube.com/watch?v=test",
			"/tmp/downloads/test/%(id)s.%(ext)s",
			"22",
			false,
			false,
		)

		hasProxy, hasFormat, hasNoPlaylist, hasOutput, hasURL := false, false, false, false, false
		for i, arg := range args {
			if arg == "--proxy" && i+1 < len(args) && args[i+1] == "socks5://xray-client:10808" {
				hasProxy = true
			}
			if arg == "--format" && i+1 < len(args) && args[i+1] == "22" {
				hasFormat = true
			}
			if arg == "--no-playlist" {
				hasNoPlaylist = true
			}
			if arg == "--output" {
				hasOutput = true
			}
			if arg == "https://youtube.com/watch?v=test" {
				hasURL = true
			}
		}

		if !hasProxy || !hasFormat || !hasNoPlaylist || !hasOutput || !hasURL {
			t.Errorf("unexpected args: %v", args)
		}
	})

	t.Run("TikTok with proxy and impersonate", func(t *testing.T) {
		args := wp.buildYtDlpArgs(
			"https://tiktok.com/@user/video/test",
			"/tmp/downloads/test/%(id)s.%(ext)s",
			"",
			true,
			false,
		)

		hasProxy, hasImpersonate, hasBestFormat := false, false, false
		for i, arg := range args {
			if arg == "--proxy" && i+1 < len(args) && args[i+1] == "socks5://xray-client:10808" {
				hasProxy = true
			}
			if arg == "--impersonate" && i+1 < len(args) && args[i+1] == "chrome:120" {
				hasImpersonate = true
			}
			if arg == "--format" && i+1 < len(args) && args[i+1] == "best" {
				hasBestFormat = true
			}
		}

		if !hasProxy || !hasImpersonate || !hasBestFormat {
			t.Errorf("unexpected args: %v", args)
		}
	})

	t.Run("Reels with proxy and impersonate", func(t *testing.T) {
		args := wp.buildYtDlpArgs(
			"https://instagram.com/reel/abc",
			"/tmp/downloads/test/%(id)s.%(ext)s",
			"",
			false,
			true,
		)
		hasImpersonate := false
		for i, arg := range args {
			if arg == "--impersonate" && i+1 < len(args) && args[i+1] == "chrome:120" {
				hasImpersonate = true
			}
		}
		if !hasImpersonate {
			t.Errorf("expected impersonate for reels: %v", args)
		}
	})

	t.Run("without proxy configured", func(t *testing.T) {
		wpNoProxy := &WorkerPool{proxyAddr: ""}
		args := wpNoProxy.buildYtDlpArgs(
			"https://youtube.com/watch?v=test",
			"/tmp/downloads/test/%(id)s.%(ext)s",
			"",
			false,
			false,
		)

		for _, arg := range args {
			if arg == "--proxy" {
				t.Fatalf("did not expect proxy argument when proxyAddr is empty: %v", args)
			}
		}
	})
}

func TestWorkerPool_ProxyAddr(t *testing.T) {
	t.Run("proxy address is set", func(t *testing.T) {
		wp := NewWorkerPool(nil, nil, 3, nil, "xray-client:10808")
		if wp.proxyAddr != "xray-client:10808" {
			t.Errorf("expected proxyAddr xray-client:10808, got %s", wp.proxyAddr)
		}
	})

	t.Run("proxy address is empty", func(t *testing.T) {
		wp := NewWorkerPool(nil, nil, 3, nil, "")
		if wp.proxyAddr != "" {
			t.Errorf("expected empty proxyAddr, got %s", wp.proxyAddr)
		}
	})
}
