package worker

import "testing"

func TestBuildYtDlpArgs(t *testing.T) {
	wp := &WorkerPool{proxyAddr: "xray-client:10808"}

	t.Run("YouTube with proxy and custom format", func(t *testing.T) {
		args := wp.buildYtDlpArgs(
			"https://youtube.com/watch?v=test",
			"/tmp/downloads/test/%(id)s.%(ext)s",
			"22",
			"socks5://xray-client:10808",
			false,
			false,
		)

		hasProxy, hasFormat, hasURL := false, false, false
		for i, arg := range args {
			if arg == "--proxy" && i+1 < len(args) && args[i+1] == "socks5://xray-client:10808" {
				hasProxy = true
			}
			if arg == "--format" && i+1 < len(args) && args[i+1] == "22" {
				hasFormat = true
			}
			if arg == "https://youtube.com/watch?v=test" {
				hasURL = true
			}
		}
		if !hasProxy || !hasFormat || !hasURL {
			t.Errorf("unexpected args: %v", args)
		}
	})

	t.Run("TikTok default format is b and uses proxy", func(t *testing.T) {
		args := wp.buildYtDlpArgs(
			"https://tiktok.com/@user/video/test",
			"/tmp/downloads/test/%(id)s.%(ext)s",
			"",
			"socks5://xray-client:10808",
			true,
			false,
		)

		hasProxy, hasBest := false, false
		for i, arg := range args {
			if arg == "--proxy" && i+1 < len(args) && args[i+1] == "socks5://xray-client:10808" {
				hasProxy = true
			}
			if arg == "--format" && i+1 < len(args) && args[i+1] == "b" {
				hasBest = true
			}
		}
		if !hasProxy || !hasBest {
			t.Errorf("unexpected args: %v", args)
		}
	})

	t.Run("Reels uses proxy and b format", func(t *testing.T) {
		args := wp.buildYtDlpArgs(
			"https://instagram.com/reel/abc",
			"/tmp/downloads/test/%(id)s.%(ext)s",
			"",
			"socks5://xray-client:10808",
			false,
			true,
		)

		hasProxy, hasBest := false, false
		for i, arg := range args {
			if arg == "--proxy" && i+1 < len(args) && args[i+1] == "socks5://xray-client:10808" {
				hasProxy = true
			}
			if arg == "--format" && i+1 < len(args) && args[i+1] == "b" {
				hasBest = true
			}
		}
		if !hasProxy || !hasBest {
			t.Errorf("unexpected args: %v", args)
		}
	})

	t.Run("YouTube default format uses 720p cap", func(t *testing.T) {
		args := wp.buildYtDlpArgs(
			"https://youtube.com/watch?v=test",
			"/tmp/downloads/test/%(id)s.%(ext)s",
			"",
			"socks5://xray-client:10808",
			false,
			false,
		)

		hasDefaultFormat := false
		for i, arg := range args {
			if arg == "--format" && i+1 < len(args) && args[i+1] == "best[height<=720]/best" {
				hasDefaultFormat = true
				break
			}
		}
		if !hasDefaultFormat {
			t.Errorf("unexpected args: %v", args)
		}
	})

	t.Run("TikTok and Reels add impersonation", func(t *testing.T) {
		tiktokArgs := wp.buildYtDlpArgs(
			"https://tiktok.com/@user/video/test",
			"/tmp/downloads/test/%(id)s.%(ext)s",
			"",
			"socks5://xray-client:10808",
			true,
			false,
		)
		reelsArgs := wp.buildYtDlpArgs(
			"https://instagram.com/reel/abc",
			"/tmp/downloads/test/%(id)s.%(ext)s",
			"",
			"socks5://xray-client:10808",
			false,
			true,
		)

		for _, args := range [][]string{tiktokArgs, reelsArgs} {
			hasImpersonate := false
			for i, arg := range args {
				if arg == "--impersonate" && i+1 < len(args) && args[i+1] == "chrome" {
					hasImpersonate = true
					break
				}
			}
			if !hasImpersonate {
				t.Errorf("expected impersonation args in %v", args)
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
