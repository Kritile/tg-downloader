package worker

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mediaharvester/tg-downloader/bot/internal/domain"
	"github.com/mediaharvester/tg-downloader/shared/models"
)

const (
	maxRetries   = 2
	downloadPath = "/tmp/downloads"
)

type WorkerPool struct {
	queueService   *QueueService
	bot            domain.Notifier
	workerCount    int
	downloadRepo   domain.DownloadRepository
	proxyAddr      string
}

func NewWorkerPool(
	queueService *QueueService,
	bot domain.Notifier,
	workerCount int,
	downloadRepo domain.DownloadRepository,
	proxyAddr string,
) *WorkerPool {
	return &WorkerPool{
		queueService:   queueService,
		bot:            bot,
		workerCount:    workerCount,
		downloadRepo:   downloadRepo,
		proxyAddr:      proxyAddr,
	}
}

func (wp *WorkerPool) Start(ctx context.Context) error {
	log.Printf("Starting worker pool with %d workers", wp.workerCount)

	// Start multiple workers
	for i := 0; i < wp.workerCount; i++ {
		go wp.worker(ctx, i)
	}

	// Wait for context cancellation
	<-ctx.Done()
	log.Println("Worker pool shutting down...")
	return ctx.Err()
}

func (wp *WorkerPool) worker(ctx context.Context, workerID int) {
	log.Printf("Worker %d started", workerID)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d stopped", workerID)
			return
		default:
			wp.processJob(ctx, workerID)
		}
	}
}

func (wp *WorkerPool) processJob(ctx context.Context, workerID int) {
	// Pop job from queue
	job, err := wp.queueService.PopJob(ctx)
	if err != nil {
		// Context cancelled or queue closed
		if ctx.Err() != nil {
			return
		}
		log.Printf("Worker %d failed to pop job: %v", workerID, err)
		time.Sleep(time.Second) // Prevent tight loop on errors
		return
	}

	log.Printf("Worker %d processing job for user %d, URL: %s", workerID, job.UserID, job.URL)

	// Process with retries
	err = wp.processWithRetry(ctx, job, workerID)
	if err != nil {
		log.Printf("Worker %d failed to process job after retries: %v", workerID, err)
		wp.logFailedDownload(job.UserID, models.VideoSource(job.Source), job.URL)
		wp.bot.SendDownloadFailed(job.ChatID)
	}
}

func (wp *WorkerPool) processWithRetry(ctx context.Context, job *models.DownloadJob, workerID int) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("Worker %d retrying job (attempt %d/%d)", workerID, attempt, maxRetries)
			time.Sleep(time.Second * time.Duration(attempt))
		}

		err := wp.executeDownload(ctx, job)
		if err == nil {
			return nil
		}

		lastErr = err

		// Don't retry on certain errors
		if err == domain.ErrPermissionDenied || 
		   err == domain.ErrDailyLimitExceeded || 
		   err == domain.ErrMonthlyLimitExceeded {
			return err
		}
	}

	return lastErr
}

func (wp *WorkerPool) executeDownload(ctx context.Context, job *models.DownloadJob) error {
	// Create unique subdirectory for this download
	subDir := fmt.Sprintf("user_%d_%s", job.UserID, sanitizeFilename(job.URL))

	// Create download directory if it doesn't exist
	fullPath := filepath.Join(downloadPath, subDir)
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return fmt.Errorf("failed to create download directory: %w", err)
	}

	// Generate output template
	outputTemplate := filepath.Join(fullPath, "%(id)s.%(ext)s")

	// Determine if we need to use proxy (YouTube and TikTok)
	isYouTube := job.Source == string(models.SourceYoutube)
	isTikTok := job.Source == string(models.SourceTiktok)
	useProxy := isYouTube || isTikTok

	// Build yt-dlp command
	args := wp.buildYtDlpArgs(job.URL, outputTemplate, job.Format, useProxy, isTikTok)

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)

	// Execute command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("yt-dlp failed: %w, output: %s", err, string(output))
	}

	// Find the downloaded file
	filePath, err := wp.findDownloadedFile(fullPath)
	if err != nil {
		return err
	}

	// Check file size
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	if fileInfo.Size() > models.MaxTelegramFileSize {
		// Delete the file if too large
		os.Remove(filePath)
		wp.bot.SendFileTooLarge(job.ChatID)
		return domain.ErrFileTooLarge
	}

	// Log successful download
	wp.logSuccessfulDownload(job.UserID, models.VideoSource(job.Source), job.URL, job.Format, fileInfo.Size())

	// Send video to user
	err = wp.bot.SendVideo(job.ChatID, filePath)
	if err != nil {
		// Try to delete file even if send fails
		CleanupDownloadedFile(filePath)
		return fmt.Errorf("failed to send video: %w", err)
	}

	// File is already deleted by SendVideo
	return nil
}

// buildYtDlpArgs builds yt-dlp command arguments
func (wp *WorkerPool) buildYtDlpArgs(url, outputTemplate, format string, useProxy, isTikTok bool) []string {
	args := []string{
		"--no-playlist", // Don't download playlists
		"--output", outputTemplate,
	}

	// Add proxy for YouTube and TikTok
	if useProxy && wp.proxyAddr != "" {
		args = append(args, "--proxy", "socks5h://"+wp.proxyAddr)
	}

	// Add format selection
	if format != "" {
		args = append(args, "--format", format)
	} else {
		// Default format selection
		if isTikTok {
			// TikTok needs more flexible format selection
			// Use available impersonation target
			args = append(args, "--format", "best")
			args = append(args, "--impersonate", "chrome:119")
		} else {
			// YouTube and others - prefer 720p or lower to stay under Telegram limit
			args = append(args, "--format", "best[height<=720]/best")
		}
	}

	args = append(args, url)
	return args
}

func (wp *WorkerPool) findDownloadedFile(dir string) (string, error) {
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

func (wp *WorkerPool) logSuccessfulDownload(userID int64, source models.VideoSource, videoURL string, format string, fileSize int64) {
	download := &models.Download{
		UserID:   userID,
		Source:   string(source),
		VideoURL: videoURL,
		Format:   format,
		FileSize: &fileSize,
		Status:   string(models.StatusCompleted),
	}

	ctx := context.Background()
	wp.downloadRepo.Create(ctx, download)
}

func (wp *WorkerPool) logFailedDownload(userID int64, source models.VideoSource, videoURL string) {
	download := &models.Download{
		UserID:   userID,
		Source:   string(source),
		VideoURL: videoURL,
		Status:   string(models.StatusFailed),
	}

	ctx := context.Background()
	wp.downloadRepo.Create(ctx, download)
}

// CleanupDownloadedFile deletes the downloaded file
func CleanupDownloadedFile(filePath string) {
	if filePath == "" {
		return
	}
	os.Remove(filePath)
	
	// Try to remove the parent directory if empty
	dir := filepath.Dir(filePath)
	os.Remove(dir) // Ignore errors
}

// sanitizeFilename creates a safe filename from a URL
func sanitizeFilename(url string) string {
	// Replace problematic characters
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(url)
}
