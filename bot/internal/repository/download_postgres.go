package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/mediaharvester/tg-downloader/bot/internal/domain"
	"github.com/mediaharvester/tg-downloader/shared/models"
)

type downloadRepository struct {
	db *sql.DB
}

func NewDownloadRepository(db *sql.DB) domain.DownloadRepository {
	return &downloadRepository{db: db}
}

func (r *downloadRepository) Create(ctx context.Context, download *models.Download) error {
	query := `
		INSERT INTO downloads (user_id, source, video_url, file_size, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	now := time.Now()
	if download.Status == "" {
		download.Status = string(models.StatusCompleted)
	}

	err := r.db.QueryRowContext(ctx, query,
		download.UserID,
		download.Source,
		download.VideoURL,
		download.FileSize,
		download.Status,
		now,
	).Scan(&download.ID)

	return err
}

func (r *downloadRepository) GetDailyCount(ctx context.Context, userID int64) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM downloads
		WHERE user_id = $1 AND created_at >= date_trunc('day', NOW())
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&count)
	return count, err
}

func (r *downloadRepository) GetMonthlyCount(ctx context.Context, userID int64) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM downloads
		WHERE user_id = $1 AND created_at >= date_trunc('month', NOW())
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&count)
	return count, err
}

func (r *downloadRepository) GetUserDownloads(ctx context.Context, userID int64, limit, offset int) ([]*models.Download, error) {
	query := `
		SELECT id, user_id, source, video_url, file_size, status, created_at
		FROM downloads
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var downloads []*models.Download
	for rows.Next() {
		var d models.Download
		err := rows.Scan(
			&d.ID,
			&d.UserID,
			&d.Source,
			&d.VideoURL,
			&d.FileSize,
			&d.Status,
			&d.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		downloads = append(downloads, &d)
	}

	return downloads, rows.Err()
}
