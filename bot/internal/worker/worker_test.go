package worker

import "testing"

func TestBuildYtDlpArgs(t *testing.T) {
	wp := &WorkerPool{proxyAddr: "xray-client:10808"}

	t.Run("YouTube with proxy and custom format", func(t *testing.T) {
		args := wp.buildYtDlpArgs(
			"https://youtube.com/watch?v=test",
			"/tmp/downloads/test/%(id)s.%(ext)s",
			"22",
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
}
