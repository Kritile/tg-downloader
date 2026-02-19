package models

// DownloadJob represents a job in the Redis queue
type DownloadJob struct {
	UserID   int64  `json:"user_id"`
	ChatID   int64  `json:"chat_id"`
	URL      string `json:"url"`
	Source   string `json:"source"`
	Username string `json:"username"`
}

// VideoSource represents the source of a video
type VideoSource string

const (
	SourceYoutube VideoSource = "youtube"
	SourceTiktok  VideoSource = "tiktok"
	SourceUnknown VideoSource = "unknown"
)

// DownloadStatus represents the status of a download
type DownloadStatus string

const (
	StatusPending    DownloadStatus = "pending"
	StatusDownloading DownloadStatus = "downloading"
	StatusCompleted  DownloadStatus = "completed"
	StatusFailed     DownloadStatus = "failed"
	StatusTooLarge   DownloadStatus = "too_large"
)

// Telegram max file size: 50MB
const MaxTelegramFileSize = 50 * 1024 * 1024
