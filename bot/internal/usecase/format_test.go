package usecase

import (
	"testing"

	"github.com/mediaharvester/tg-downloader/shared/models"
)

func TestFormatService_SelectFormat(t *testing.T) {
	fs := NewFormatService("")

	formats := []models.VideoFormat{
		{FormatID: "18", DisplayName: "360p mp4"},
		{FormatID: "22", DisplayName: "720p mp4"},
		{FormatID: "137", DisplayName: "1080p mp4"},
	}

	t.Run("found format", func(t *testing.T) {
		result, err := fs.SelectFormat(formats, "22")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.FormatID != "22" {
			t.Errorf("expected format ID 22, got %s", result.FormatID)
		}
	})

	t.Run("format not found", func(t *testing.T) {
		_, err := fs.SelectFormat(formats, "999")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("empty formats list", func(t *testing.T) {
		_, err := fs.SelectFormat([]models.VideoFormat{}, "22")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestFormatService_GetBestFormat(t *testing.T) {
	fs := NewFormatService("")

	t.Run("prefers 720p", func(t *testing.T) {
		formats := []models.VideoFormat{
			{FormatID: "137", DisplayName: "1080p mp4", Resolution: "1080p", TBR: 5000},
			{FormatID: "22", DisplayName: "720p mp4", Resolution: "720p", TBR: 2500},
			{FormatID: "18", DisplayName: "360p mp4", Resolution: "360p", TBR: 500},
		}
		result := fs.GetBestFormat(formats)
		if result.FormatID != "22" {
			t.Errorf("expected format ID 22 (720p), got %s", result.FormatID)
		}
	})

	t.Run("prefers 480p when no 720p", func(t *testing.T) {
		formats := []models.VideoFormat{
			{FormatID: "137", DisplayName: "1080p mp4", Resolution: "1080p", TBR: 5000},
			{FormatID: "18", DisplayName: "480p mp4", Resolution: "480p", TBR: 1000},
			{FormatID: "17", DisplayName: "144p mp4", Resolution: "144p", TBR: 100},
		}
		result := fs.GetBestFormat(formats)
		if result.FormatID != "18" {
			t.Errorf("expected format ID 18 (480p), got %s", result.FormatID)
		}
	})

	t.Run("returns first when no preferred resolution", func(t *testing.T) {
		formats := []models.VideoFormat{
			{FormatID: "137", DisplayName: "1080p mp4", Resolution: "1080p", TBR: 5000},
			{FormatID: "247", DisplayName: "HD webm", Resolution: "HD", TBR: 2500},
		}
		result := fs.GetBestFormat(formats)
		// When no 720p/480p/360p in resolution string, returns first (highest bitrate)
		if result.FormatID != "137" {
			t.Errorf("expected format ID 137 (first, highest bitrate), got %s", result.FormatID)
		}
	})

	t.Run("nil for empty list", func(t *testing.T) {
		result := fs.GetBestFormat([]models.VideoFormat{})
		if result != nil {
			t.Errorf("expected nil, got %+v", result)
		}
	})
}

func TestFormatService_BuildDisplayName(t *testing.T) {
	fs := &formatService{}

	tests := []struct {
		name       string
		formatID   string
		resolution string
		ext        string
		note       string
		expected   string
	}{
		{
			name:       "full info",
			formatID:   "22",
			resolution: "720p",
			ext:        "mp4",
			note:       "avc1.64001F, mp4a.40.2",
			expected:   "720p mp4 avc1.64001F, mp4a.40.2",
		},
		{
			name:       "no note",
			formatID:   "18",
			resolution: "360p",
			ext:        "mp4",
			note:       "",
			expected:   "360p mp4",
		},
		{
			name:       "only format ID",
			formatID:   "999",
			resolution: "",
			ext:        "",
			note:       "",
			expected:   "999",
		},
		{
			name:       "skip unknown resolution",
			formatID:   "22",
			resolution: "unknown",
			ext:        "mp4",
			note:       "",
			expected:   "mp4",
		},
		{
			name:       "skip dash note",
			formatID:   "22",
			resolution: "720p",
			ext:        "mp4",
			note:       "-",
			expected:   "720p mp4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fs.buildDisplayName(tt.formatID, tt.resolution, tt.ext, tt.note)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFormatService_IsVideoFormat(t *testing.T) {
	fs := &formatService{}

	tests := []struct {
		name     string
		format   models.VideoFormat
		expected bool
	}{
		{
			name:     "valid video format",
			format:   models.VideoFormat{Resolution: "720p", Extension: "mp4"},
			expected: true,
		},
		{
			name:     "audio only",
			format:   models.VideoFormat{Resolution: "audio only", Extension: "m4a"},
			expected: false,
		},
		{
			name:     "empty resolution",
			format:   models.VideoFormat{Resolution: "", Extension: "mp4"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fs.isVideoFormat(&tt.format)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestFormatService_ParseBitrate(t *testing.T) {
	fs := &formatService{}

	tests := []struct {
		name     string
		note     string
		expected float64
	}{
		{"kilobit note", "250k", 250},
		{"megabit note", "1.5M", 1500},
		{"with text", "medium 500k", 500},
		{"no bitrate", "avc1.64001F", 0},
		{"empty", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fs.parseBitrate(tt.note)
			if result != tt.expected {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}
