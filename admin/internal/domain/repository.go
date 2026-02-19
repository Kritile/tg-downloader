package domain

import (
	"context"
	"errors"

	"github.com/mediaharvester/tg-downloader/shared/models"
)

var (
	ErrAdminNotFound     = errors.New("admin not found")
	ErrSettingsNotFound  = errors.New("settings not found")
	ErrUserNotFound      = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// Repository interfaces
type AdminUserRepository interface {
	GetByUsername(ctx context.Context, username string) (*models.Admin, error)
	Create(ctx context.Context, admin *models.Admin) error
	GetAll(ctx context.Context) ([]*models.Admin, error)
	Delete(ctx context.Context, id int64) error
}

type UserRepository interface {
	GetByTelegramID(ctx context.Context, telegramID int64) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
}

type SettingsRepository interface {
	Get(ctx context.Context) (*models.Settings, error)
	Update(ctx context.Context, settings *models.Settings) error
}

type DownloadRepository interface {
	GetDailyCount(ctx context.Context, userID int64) (int, error)
	GetMonthlyCount(ctx context.Context, userID int64) (int, error)
	GetDownloadsToday(ctx context.Context) (int, error)
	GetDownloadsThisMonth(ctx context.Context) (int, error)
	GetTopUsers(ctx context.Context, limit int) ([]*models.User, []int, error)
	GetTotalUsers(ctx context.Context) (int, error)
}

// Service interfaces
type AdminUserService interface {
	Authenticate(ctx context.Context, username, password string) (*models.Admin, error)
	CreateAdmin(ctx context.Context, username, password string) error
}

type AdminSettingsService interface {
	GetSettings(ctx context.Context) (*models.Settings, error)
	UpdateSettings(ctx context.Context, settings *models.Settings) error
}

type AdminUserManagementService interface {
	GetUserByTelegramID(ctx context.Context, telegramID int64) (*models.User, error)
	UpdateUserPermissions(ctx context.Context, userID int64, canYoutube, canTiktok *bool) error
	UpdateUserLimits(ctx context.Context, userID int64, dailyLimit, monthlyLimit *int) error
	GetAllUsers(ctx context.Context, limit, offset int) ([]*models.User, int, error)
	SearchUsers(ctx context.Context, query string, limit int) ([]*models.User, error)
}

type AdminStatsService interface {
	GetDownloadsToday(ctx context.Context) (int, error)
	GetDownloadsThisMonth(ctx context.Context) (int, error)
	GetTopUsers(ctx context.Context, limit int) ([]*models.User, []int, error)
	GetTotalUsers(ctx context.Context) (int, error)
}
