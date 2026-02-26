package domain

import (
	"context"

	"github.com/mediaharvester/tg-downloader/shared/models"
)

type QueueService interface {
	PushJob(ctx context.Context, job *models.DownloadJob) error
}

type DownloaderService interface {
	Download(ctx context.Context, url string, destPath string) (string, int64, error)
}

type PermissionService interface {
	CheckPermission(ctx context.Context, user *models.User, source models.VideoSource) (bool, error)
}

type LimitService interface {
	CheckDailyLimit(ctx context.Context, user *models.User) (bool, int, int, error)
	CheckMonthlyLimit(ctx context.Context, user *models.User) (bool, int, int, error)
}

type UserService interface {
	GetOrCreate(ctx context.Context, telegramID int64, username string) (*models.User, error)
	SetAutoBestDownload(ctx context.Context, userID int64, autoBest bool) error
}

type DownloadService interface {
	ProcessDownload(ctx context.Context, job *models.DownloadJob) error
}

type URLValidator interface {
	ValidateAndDetectSource(inputURL string) (models.VideoSource, error)
}

// Notifier defines the interface for sending messages to users
type Notifier interface {
	SendVideo(chatID int64, filePath string) error
	SendFileTooLarge(chatID int64)
	SendDownloadFailed(chatID int64)
	SendFormatSelection(chatID int64, formats []models.VideoFormat) error
}
