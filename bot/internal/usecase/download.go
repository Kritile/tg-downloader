package usecase

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mediaharvester/tg-downloader/bot/internal/domain"
	"github.com/mediaharvester/tg-downloader/shared/models"
)

type downloadService struct {
	downloader     domain.DownloaderService
	downloadRepo   domain.DownloadRepository
	userRepo       domain.UserRepository
	permissionSvc  domain.PermissionService
	limitSvc       domain.LimitService
}

func NewDownloadService(
	downloader domain.DownloaderService,
	downloadRepo domain.DownloadRepository,
	userRepo domain.UserRepository,
	permissionSvc domain.PermissionService,
	limitSvc domain.LimitService,
) domain.DownloadService {
	return &downloadService{
		downloader:     downloader,
		downloadRepo:   downloadRepo,
		userRepo:       userRepo,
		permissionSvc:  permissionSvc,
		limitSvc:       limitSvc,
	}
}

func (s *downloadService) ProcessDownload(ctx context.Context, job *models.DownloadJob) error {
	// Get user
	user, err := s.userRepo.GetByTelegramID(ctx, job.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Check permission
	hasPermission, err := s.permissionSvc.CheckPermission(ctx, user, models.VideoSource(job.Source))
	if err != nil {
		return fmt.Errorf("failed to check permission: %w", err)
	}
	if !hasPermission {
		return domain.ErrPermissionDenied
	}

	// Check daily limit
	dailyOk, _, _, err := s.limitSvc.CheckDailyLimit(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to check daily limit: %w", err)
	}
	if !dailyOk {
		return domain.ErrDailyLimitExceeded
	}

	// Check monthly limit
	monthlyOk, _, _, err := s.limitSvc.CheckMonthlyLimit(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to check monthly limit: %w", err)
	}
	if !monthlyOk {
		return domain.ErrMonthlyLimitExceeded
	}

	// Create unique subdirectory for this download
	subDir := fmt.Sprintf("user_%d_%s", user.ID, job.URL)

	// Download the video
	filePath, fileSize, err := s.downloader.Download(ctx, job.URL, subDir)
	if err != nil {
		// Log the download attempt as failed
		s.logDownload(user.ID, models.VideoSource(job.Source), job.URL, nil, string(models.StatusFailed))
		return fmt.Errorf("download failed: %w", err)
	}

	// Log successful download
	s.logDownload(user.ID, models.VideoSource(job.Source), job.URL, &fileSize, string(models.StatusCompleted))

	// The file will be sent by the bot handler and deleted immediately after
	// Store the file path in context for the caller to use
	ctx = context.WithValue(ctx, "filePath", filePath)

	return nil
}

func (s *downloadService) logDownload(userID int64, source models.VideoSource, videoURL string, fileSize *int64, status string) {
	download := &models.Download{
		UserID:   userID,
		Source:   string(source),
		VideoURL: videoURL,
		FileSize: fileSize,
		Status:   status,
	}

	// Use background context for logging
	ctx := context.Background()
	s.downloadRepo.Create(ctx, download)
}

// GetFilePath retrieves the file path from context
func GetFilePath(ctx context.Context) string {
	if filePath, ok := ctx.Value("filePath").(string); ok {
		return filePath
	}
	return ""
}

// CleanupDownloadedFile deletes the downloaded file
func CleanupDownloadedFile(filePath string) error {
	if filePath == "" {
		return nil
	}

	// Remove the file
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Try to remove the parent directory if empty
	dir := filepath.Dir(filePath)
	os.Remove(dir) // Ignore errors - directory might not be empty

	return nil
}
