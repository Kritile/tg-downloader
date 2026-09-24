package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/mediaharvester/tg-downloader/bot/internal/domain"
	"github.com/mediaharvester/tg-downloader/shared/models"
)

type jobRepository struct{ db *sql.DB }

func NewJobRepository(db *sql.DB) domain.JobRepository { return &jobRepository{db: db} }

func (r *jobRepository) Create(ctx context.Context, job *models.DownloadJobRecord) error {
	// DownloadJob.UserID is the external Telegram/MAX user ID because workers
	// use it to load the user. The database relation must use users.id.
	return r.db.QueryRowContext(ctx, `
		INSERT INTO download_jobs (platform, user_id, chat_id, url, source, format, status, status_message_id, queued_at, updated_at)
		VALUES ($1, (SELECT id FROM users WHERE telegram_id = $2), $3, $4, $5, $6, $7, $8, $9, $9)
		RETURNING id
	`, job.Platform, job.UserID, job.ChatID, job.URL, job.Source, job.Format, job.Status, job.StatusMessageID, job.QueuedAt).Scan(&job.ID)
}

func (r *jobRepository) MarkStarted(ctx context.Context, id int64) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `UPDATE download_jobs SET status = 'downloading', started_at = $2, updated_at = $2 WHERE id = $1`, id, now)
	return err
}

func (r *jobRepository) MarkCompleted(ctx context.Context, id int64) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `UPDATE download_jobs SET status = 'completed', completed_at = $2, updated_at = $2 WHERE id = $1`, id, now)
	return err
}

func (r *jobRepository) MarkFailed(ctx context.Context, id int64, message string, retries int) error {
	_, err := r.db.ExecContext(ctx, `UPDATE download_jobs SET status = 'failed', error_message = $2, retry_count = $3, updated_at = NOW() WHERE id = $1`, id, message, retries)
	return err
}
