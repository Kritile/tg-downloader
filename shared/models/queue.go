package models

// DownloadJob represents a job in the Redis queue
type DownloadJob struct {
	ID              int64  `json:"id,omitempty"`
	Platform        string `json:"platform"`
	UserID          int64  `json:"user_id"`
	ChatID          int64  `json:"chat_id"`
	URL             string `json:"url"`
	Source          string `json:"source"`
	Username        string `json:"username"`
	Format          string `json:"format,omitempty"`
	StatusMessageID string `json:"status_message_id,omitempty"`
}

// VideoSource represents the source of a video
type VideoSource string

const (
	SourceYoutube VideoSource = "youtube"
	SourceTiktok  VideoSource = "tiktok"
	SourceReels   VideoSource = "reels"
	SourceUnknown VideoSource = "unknown"
)

// DownloadStatus represents the status of a download
type DownloadStatus string

const (
	StatusPending     DownloadStatus = "pending"
	StatusDownloading DownloadStatus = "downloading"
	StatusCompleted   DownloadStatus = "completed"
	StatusFailed      DownloadStatus = "failed"
	StatusTooLarge    DownloadStatus = "too_large"
)

const (
	// MaxTelegramFileSize is the Local Bot API outgoing file limit.
	MaxTelegramFileSize int64 = 2000 * 1024 * 1024
	// MaxMaxFileSize preserves the existing MAX upload guard.
	MaxMaxFileSize int64 = 50 * 1024 * 1024
)

// VideoFormat represents a video format option
type VideoFormat struct {
	ID          string  `json:"id"`
	FormatID    string  `json:"format_id"`
	Extension   string  `json:"extension"`
	Resolution  string  `json:"resolution"`
	FileSize    int64   `json:"file_size,omitempty"`
	TBR         float64 `json:"tbr,omitempty"` // Total bitrate in kbps
	FormatNote  string  `json:"format_note,omitempty"`
	DisplayName string  `json:"display_name"`
}
