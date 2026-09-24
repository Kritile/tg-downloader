package models

import "time"

type User struct {
	ID               int64      `json:"id"`
	TelegramID       int64      `json:"telegram_id"`
	Username         string     `json:"username"`
	CanYoutube       *bool      `json:"can_youtube"`
	CanInstagram     *bool      `json:"can_instagram"`
	CanTiktok        *bool      `json:"can_tiktok"`
	DailyLimit       *int       `json:"daily_limit"`
	MonthlyLimit     *int       `json:"monthly_limit"`
	AutoBestDownload *bool      `json:"auto_best_download"`
	IsBlocked        bool       `json:"is_blocked"`
	LastSeenAt       *time.Time `json:"last_seen_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type DownloadJobRecord struct {
	ID              int64      `json:"id"`
	Platform        string     `json:"platform"`
	UserID          int64      `json:"user_id"`
	ChatID          int64      `json:"chat_id"`
	URL             string     `json:"url"`
	Source          string     `json:"source"`
	Format          string     `json:"format"`
	Status          string     `json:"status"`
	RetryCount      int        `json:"retry_count"`
	ErrorMessage    string     `json:"error_message"`
	StatusMessageID string     `json:"status_message_id"`
	QueuedAt        time.Time  `json:"queued_at"`
	StartedAt       *time.Time `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type Download struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Source    string    `json:"source"`
	VideoURL  string    `json:"video_url"`
	Format    string    `json:"format"`
	FileSize  *int64    `json:"file_size"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type Settings struct {
	ID                      int64 `json:"id"`
	DefaultDailyLimit       int   `json:"default_daily_limit"`
	DefaultMonthlyLimit     int   `json:"default_monthly_limit"`
	DefaultYoutubeAllowed   bool  `json:"default_youtube_allowed"`
	DefaultInstagramAllowed bool  `json:"default_instagram_allowed"`
	DefaultTiktokAllowed    bool  `json:"default_tiktok_allowed"`
}

type Admin struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}
