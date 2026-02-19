package usecase

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mediaharvester/tg-downloader/bot/internal/domain"
)

const (
	maxFileSize = 50 * 1024 * 1024 // 50MB
)

type downloaderService struct {
	downloadPath string
}

func NewDownloaderService(downloadPath string) domain.DownloaderService {
	return &downloaderService{
		downloadPath: downloadPath,
	}
}

func (s *downloaderService) Download(ctx context.Context, url string, destPath string) (string, int64, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, domain.DownloadTimeout)
	defer cancel()

	// Create download directory if it doesn't exist
	if err := os.MkdirAll(s.downloadPath, 0755); err != nil {
		return "", 0, fmt.Errorf("failed to create download directory: %w", err)
	}

	// Generate output template
	outputTemplate := filepath.Join(s.downloadPath, destPath, "%(id)s.%(ext)s")

	// Build yt-dlp command with TikTok-specific options
	args := []string{
		"--format", "best[height<=720]/best", // Allow fallback to any best format
		"--output", outputTemplate,
		"--no-playlist", // Don't download playlists
		"--user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	}
	
	// Add TikTok-specific impersonation if it's a TikTok URL
	if strings.Contains(url, "tiktok.com") {
		args = append(args, "--extractor-args", "tiktok:api=api")
	}

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)

	// Execute command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", 0, fmt.Errorf("yt-dlp failed: %w, output: %s", err, string(output))
	}

	// Find the downloaded file
	filePath, err := s.findDownloadedFile(destPath)
	if err != nil {
		return "", 0, err
	}

	// Check file size
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to get file info: %w", err)
	}

	if fileInfo.Size() > maxFileSize {
		// Delete the file if too large
		os.Remove(filePath)
		return "", 0, domain.ErrFileTooLarge
	}

	return filePath, fileInfo.Size(), nil
}

func (s *downloaderService) findDownloadedFile(subDir string) (string, error) {
	dir := filepath.Join(s.downloadPath, subDir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("failed to read download directory: %w", err)
	}

	// Find video files (common extensions)
	videoExtensions := []string{".mp4", ".webm", ".mkv", ".avi", ".mov"}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := strings.ToLower(entry.Name())
		for _, ext := range videoExtensions {
			if strings.HasSuffix(name, ext) {
				return filepath.Join(dir, entry.Name()), nil
			}
		}
	}

	return "", fmt.Errorf("no video file found in download directory")
}

// CleanupFile deletes a file from the filesystem
func CleanupFile(filePath string) error {
	if filePath == "" {
		return nil
	}
	return os.Remove(filePath)
}
