package usecase_test

import (
	"testing"

	"github.com/mediaharvester/tg-downloader/bot/internal/usecase"
	"github.com/mediaharvester/tg-downloader/shared/models"
)

func TestURLValidator_ValidateAndDetectSource(t *testing.T) {
	validator := usecase.NewURLValidator()

	tests := []struct {
		name           string
		url            string
		expectedSource models.VideoSource
		expectError    bool
	}{
		{
			name:           "YouTube standard URL",
			url:            "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			expectedSource: models.SourceYoutube,
			expectError:    false,
		},
		{
			name:           "YouTube short URL",
			url:            "https://youtu.be/dQw4w9WgXcQ",
			expectedSource: models.SourceYoutube,
			expectError:    false,
		},
		{
			name:           "TikTok standard URL",
			url:            "https://www.tiktok.com/@user/video/1234567890",
			expectedSource: models.SourceTiktok,
			expectError:    false,
		},
		{
			name:           "TikTok mobile URL",
			url:            "https://vm.tiktok.com/abcd123",
			expectedSource: models.SourceTiktok,
			expectError:    false,
		},
		{
			name:           "Invalid URL",
			url:            "not-a-url",
			expectedSource: models.SourceUnknown,
			expectError:    true,
		},
		{
			name:           "Instagram Reels URL",
			url:            "https://www.instagram.com/reel/CxYzAbC123/",
			expectedSource: models.SourceReels,
			expectError:    false,
		},
		{
			name:           "Unsupported source",
			url:            "https://instagram.com/p/abc123",
			expectedSource: models.SourceUnknown,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, err := validator.ValidateAndDetectSource(tt.url)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if source != tt.expectedSource {
				t.Errorf("expected source %v, got %v", tt.expectedSource, source)
			}
		})
	}
}

func TestExtractVideoID(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "YouTube standard URL",
			url:      "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			expected: "dQw4w9WgXcQ",
		},
		{
			name:     "YouTube short URL",
			url:      "https://youtu.be/dQw4w9WgXcQ",
			expected: "dQw4w9WgXcQ",
		},
		{
			name:     "TikTok URL",
			url:      "https://www.tiktok.com/@user/video/1234567890",
			expected: "1234567890",
		},
		{
			name:     "Instagram Reels URL",
			url:      "https://www.instagram.com/reel/CxYzAbC123/",
			expected: "CxYzAbC123",
		},
		{
			name:     "Invalid URL",
			url:      "not-a-url",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			videoID := usecase.ExtractVideoID(tt.url)
			if videoID != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, videoID)
			}
		})
	}
}
