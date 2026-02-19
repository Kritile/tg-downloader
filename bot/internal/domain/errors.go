package domain

import (
	"errors"
	"time"
)

const (
	DownloadTimeout = 10 * time.Minute
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrDownloadNotFound   = errors.New("download not found")
	ErrSettingsNotFound   = errors.New("settings not found")
	ErrAdminNotFound      = errors.New("admin not found")
	ErrInvalidURL         = errors.New("invalid URL")
	ErrUnsupportedSource  = errors.New("unsupported source")
	ErrPermissionDenied   = errors.New("permission denied")
	ErrDailyLimitExceeded = errors.New("daily limit exceeded")
	ErrMonthlyLimitExceeded = errors.New("monthly limit exceeded")
	ErrFileTooLarge       = errors.New("file too large")
	ErrDownloadFailed     = errors.New("download failed")
	ErrInvalidCredentials = errors.New("invalid credentials")
)
