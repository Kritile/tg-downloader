package domain

import (
	"context"
	"github.com/mediaharvester/tg-downloader/shared/models"
)

// FormatService handles video format listing and selection
type FormatService interface {
	// ListFormats retrieves available formats for a video URL
	ListFormats(ctx context.Context, url string, source models.VideoSource) ([]models.VideoFormat, error)
	// SelectFormat validates and returns the selected format
	SelectFormat(formats []models.VideoFormat, formatID string) (*models.VideoFormat, error)
	// GetBestFormat returns the best format for automatic download
	GetBestFormat(formats []models.VideoFormat) *models.VideoFormat
}

// FormatDownloader defines interface for downloading with format selection
type FormatDownloader interface {
	// DownloadWithFormat downloads video using specified format
	DownloadWithFormat(ctx context.Context, url string, formatID string, destPath string, useProxy bool) (string, int64, error)
	// ListFormatsWithProxy lists formats with optional proxy support
	ListFormatsWithProxy(ctx context.Context, url string, useProxy bool) ([]models.VideoFormat, error)
}
