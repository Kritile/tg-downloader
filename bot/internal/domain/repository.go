package domain

import (
	"context"

	"github.com/mediaharvester/tg-downloader/shared/models"
)

type UserRepository interface {
	GetByTelegramID(ctx context.Context, telegramID int64) (*models.User, error)
	Create(ctx context.Context, user *models.User) error
	Update(ctx context.Context, user *models.User) error
	GetOrCreate(ctx context.Context, telegramID int64, username string) (*models.User, error)
	UpdateAutoBestDownload(ctx context.Context, userID int64, autoBest *bool) error
}

type DownloadRepository interface {
	Create(ctx context.Context, download *models.Download) error
	GetDailyCount(ctx context.Context, userID int64) (int, error)
	GetMonthlyCount(ctx context.Context, userID int64) (int, error)
	GetUserDownloads(ctx context.Context, userID int64, limit, offset int) ([]*models.Download, error)
}

type SettingsRepository interface {
	Get(ctx context.Context) (*models.Settings, error)
	Update(ctx context.Context, settings *models.Settings) error
}

type AdminRepository interface {
	GetByUsername(ctx context.Context, username string) (*models.Admin, error)
	Create(ctx context.Context, admin *models.Admin) error
}
