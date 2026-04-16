package usecase

import (
	"net/url"
	"strings"

	"github.com/mediaharvester/tg-downloader/bot/internal/domain"
	"github.com/mediaharvester/tg-downloader/shared/models"
)

type urlValidator struct{}

func NewURLValidator() domain.URLValidator {
	return &urlValidator{}
}

func (v *urlValidator) ValidateAndDetectSource(inputURL string) (models.VideoSource, error) {
	// Parse URL
	parsedURL, err := url.Parse(inputURL)
	if err != nil {
		return models.SourceUnknown, domain.ErrInvalidURL
	}

	host := strings.ToLower(parsedURL.Hostname())

	// Check for YouTube
	if isYouTube(host, parsedURL.Path) {
		return models.SourceYoutube, nil
	}

	// Check for TikTok
	if isTikTok(host) {
		return models.SourceTiktok, nil
	}

	// Check for Instagram Reels
	if isInstagramReels(host, parsedURL.Path) {
		return models.SourceReels, nil
	}

	return models.SourceUnknown, domain.ErrUnsupportedSource
}

func isYouTube(host, path string) bool {
	youtubeDomains := []string{
		"youtube.com",
		"www.youtube.com",
		"youtu.be",
		"www.youtu.be",
	}

	for _, domain := range youtubeDomains {
		if host == domain {
			return true
		}
	}

	// Check for youtu.be short URLs
	if strings.Contains(host, "youtu.be") {
		return true
	}

	return false
}

func isTikTok(host string) bool {
	tiktokDomains := []string{
		"tiktok.com",
		"www.tiktok.com",
		"vm.tiktok.com",
		"vt.tiktok.com",
		"m.tiktok.com",
	}

	for _, domain := range tiktokDomains {
		if host == domain {
			return true
		}
	}

	return false
}

func isInstagramReels(host, path string) bool {
	instagramDomains := []string{
		"instagram.com",
		"www.instagram.com",
		"m.instagram.com",
	}

	isInstagramHost := false
	for _, domain := range instagramDomains {
		if host == domain {
			isInstagramHost = true
			break
		}
	}
	if !isInstagramHost {
		return false
	}

	path = strings.ToLower(path)
	return strings.HasPrefix(path, "/reel/") || strings.HasPrefix(path, "/reels/")
}

// ExtractVideoID extracts video ID from URL if possible
func ExtractVideoID(inputURL string) string {
	parsedURL, err := url.Parse(inputURL)
	if err != nil {
		return ""
	}

	host := strings.ToLower(parsedURL.Hostname())

	// YouTube short URL (youtu.be/VIDEO_ID)
	if strings.Contains(host, "youtu.be") {
		return strings.TrimPrefix(parsedURL.Path, "/")
	}

	// YouTube standard URL (youtube.com/watch?v=VIDEO_ID)
	if strings.Contains(host, "youtube.com") {
		query := parsedURL.Query()
		return query.Get("v")
	}

	// TikTok
	if strings.Contains(host, "tiktok.com") {
		parts := strings.Split(parsedURL.Path, "/")
		for i, part := range parts {
			if part == "video" && i+1 < len(parts) {
				return parts[i+1]
			}
		}
	}

	// Instagram Reels
	if strings.Contains(host, "instagram.com") {
		parts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
		if len(parts) >= 2 && (parts[0] == "reel" || parts[0] == "reels") {
			return parts[1]
		}
	}

	return ""
}

// Compile-time check
var _ domain.URLValidator = (*urlValidator)(nil)
