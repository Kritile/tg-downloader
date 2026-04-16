package worker

import (
	"testing"
)

func TestBuildYtDlpArgs(t *testing.T) {
	wp := &WorkerPool{
		proxyAddr: "socks5://xray-client:10808",
	}

	t.Run("YouTube with proxy and format", func(t *testing.T) {
		args := wp.buildYtDlpArgs(
			"https://youtube.com/watch?v=test",
			"/tmp/downloads/test/%(id)s.%(ext)s",
			"22",
			"socks5://xray-client:10808",
			false, // isTikTok
		)

		// Check for proxy
		hasProxy := false
		hasFormat := false
		hasNoPlaylist := false
		hasOutput := false
		hasURL := false

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

		if !hasProxy {
			t.Error("expected proxy argument for YouTube")
		}
		if !hasFormat {
			t.Error("expected format argument")
		}
		if !hasNoPlaylist {
			t.Error("expected --no-playlist")
		}
		if !hasOutput {
			t.Error("expected --output")
		}
		if !hasURL {
			t.Error("expected URL in args")
		}
	})

	t.Run("TikTok with proxy", func(t *testing.T) {
		args := wp.buildYtDlpArgs(
			"https://tiktok.com/@user/video/test",
			"/tmp/downloads/test/%(id)s.%(ext)s",
			"",
			"socks5://xray-client:10808",
			true, // isTikTok
		)

		hasProxy := false

		for i, arg := range args {
			if arg == "--proxy" && i+1 < len(args) && args[i+1] == "socks5://xray-client:10808" {
				hasProxy = true
			}
		}

		if !hasProxy {
			t.Error("expected proxy argument for TikTok")
		}
	})

	t.Run("YouTube without proxy configured", func(t *testing.T) {
		wpNoProxy := &WorkerPool{proxyAddr: ""}
		args := wpNoProxy.buildYtDlpArgs(
			"https://youtube.com/watch?v=test",
			"/tmp/downloads/test/%(id)s.%(ext)s",
			"",
			"",
			false, // isTikTok
		)

		hasProxy := false
		for _, arg := range args {
			if arg == "--proxy" {
				hasProxy = true
				break
			}
		}

		if hasProxy {
			t.Error("did not expect proxy argument when proxyAddr is empty")
		}
	})

	t.Run("TikTok default format", func(t *testing.T) {
		args := wp.buildYtDlpArgs(
			"https://tiktok.com/@user/video/test",
			"/tmp/downloads/test/%(id)s.%(ext)s",
			"",
			"socks5://xray-client:10808",
			true, // isTikTok
		)

		hasBestFormat := false

		for i, arg := range args {
			if arg == "--format" && i+1 < len(args) && args[i+1] == "best" {
				hasBestFormat = true
			}
		}

		if !hasBestFormat {
			t.Error("expected best format for TikTok")
		}
	})

	t.Run("YouTube default format", func(t *testing.T) {
		args := wp.buildYtDlpArgs(
			"https://youtube.com/watch?v=test",
			"/tmp/downloads/test/%(id)s.%(ext)s",
			"",
			"socks5://xray-client:10808",
			false, // isTikTok
		)

		hasDefaultFormat := false
		for i, arg := range args {
			if arg == "--format" && i+1 < len(args) && args[i+1] == "best[height<=720]/best" {
				hasDefaultFormat = true
				break
			}
		}

		if !hasDefaultFormat {
			t.Error("expected default 720p format for YouTube")
		}
	})

	t.Run("custom format selection", func(t *testing.T) {
		args := wp.buildYtDlpArgs(
			"https://youtube.com/watch?v=test",
			"/tmp/downloads/test/%(id)s.%(ext)s",
			"137+140",
			"socks5://xray-client:10808",
			false, // isTikTok
		)

		hasCustomFormat := false
		for i, arg := range args {
			if arg == "--format" && i+1 < len(args) && args[i+1] == "137+140" {
				hasCustomFormat = true
				break
			}
		}

		if !hasCustomFormat {
			t.Error("expected custom format 137+140")
		}
	})
}

func TestWorkerPool_ProxyAddr(t *testing.T) {
	t.Run("proxy address is set", func(t *testing.T) {
		wp := NewWorkerPool(nil, nil, 3, nil, "socks5://xray-client:10808")
		if wp.proxyAddr != "socks5://xray-client:10808" {
			t.Errorf("expected proxyAddr socks5://xray-client:10808, got %s", wp.proxyAddr)
		}
	})

	t.Run("proxy address is empty", func(t *testing.T) {
		wp := NewWorkerPool(nil, nil, 3, nil, "")
		if wp.proxyAddr != "" {
			t.Errorf("expected empty proxyAddr, got %s", wp.proxyAddr)
		}
	})
}
