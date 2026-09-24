package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/mediaharvester/tg-downloader/admin/internal/domain"
	"github.com/mediaharvester/tg-downloader/shared/models"
)

type downloadRepository struct {
	db *sql.DB
}

func NewDownloadRepository(db *sql.DB) domain.DownloadRepository {
	return &downloadRepository{db: db}
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

func (r *downloadRepository) GetDownloadsToday(ctx context.Context) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM downloads
		WHERE created_at >= date_trunc('day', NOW())
	`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}

func (r *downloadRepository) GetDownloadsThisMonth(ctx context.Context) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM downloads
		WHERE created_at >= date_trunc('month', NOW())
	`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}

func (r *downloadRepository) GetTopUsers(ctx context.Context, limit int) ([]*models.User, []int, error) {
	query := `
		SELECT u.id, u.telegram_id, u.username, u.can_youtube, u.can_tiktok, u.daily_limit, u.monthly_limit, u.created_at, u.updated_at, COUNT(d.id) as download_count
		FROM users u
		LEFT JOIN downloads d ON u.id = d.user_id
		WHERE d.created_at >= date_trunc('month', NOW())
		GROUP BY u.id
		ORDER BY download_count DESC
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var users []*models.User
	var counts []int

	for rows.Next() {
		var user models.User
		var canYoutube, canTiktok sql.NullBool
		var dailyLimit, monthlyLimit sql.NullInt32
		var count int

		err := rows.Scan(
			&user.ID,
			&user.TelegramID,
			&user.Username,
			&canYoutube,
			&canTiktok,
			&dailyLimit,
			&monthlyLimit,
			&user.CreatedAt,
			&user.UpdatedAt,
			&count,
		)
		if err != nil {
			return nil, nil, err
		}

		if canYoutube.Valid {
			user.CanYoutube = &canYoutube.Bool
		}
		if canTiktok.Valid {
			user.CanTiktok = &canTiktok.Bool
		}
		if dailyLimit.Valid {
			val := int(dailyLimit.Int32)
			user.DailyLimit = &val
		}
		if monthlyLimit.Valid {
			val := int(monthlyLimit.Int32)
			user.MonthlyLimit = &val
		}

		users = append(users, &user)
		counts = append(counts, count)
	}

	return users, counts, rows.Err()
}

func (r *downloadRepository) GetTotalUsers(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM users`
	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}

func (r *downloadRepository) GetDownloadsBySource(ctx context.Context) (map[string]int, error) {
	query := `
		SELECT source, COUNT(*)
		FROM downloads
		GROUP BY source
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := map[string]int{
		"youtube": 0,
		"reels":   0,
		"tiktok":  0,
	}

	for rows.Next() {
		var source string
		var count int
		if err := rows.Scan(&source, &count); err != nil {
			return nil, err
		}
		stats[source] = count
	}

	return stats, rows.Err()
}

func (r *downloadRepository) List(ctx context.Context, status, source, platform string, limit, offset int) ([]*models.DownloadJobRecord, int, error) {
	conditions := []string{"1=1"}
	args := make([]interface{}, 0, 5)
	arg := 1
	for _, filter := range []struct{ value, column string }{{status, "status"}, {source, "source"}, {platform, "platform"}} {
		if filter.value != "" {
			conditions = append(conditions, fmt.Sprintf("%s = $%d", filter.column, arg))
			args = append(args, filter.value)
			arg++
		}
	}
	where := strings.Join(conditions, " AND ")
	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM download_jobs WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	query := "SELECT id, platform, user_id, chat_id, url, source, format, status, retry_count, error_message, status_message_id, queued_at, started_at, completed_at, updated_at FROM download_jobs WHERE " + where + fmt.Sprintf(" ORDER BY queued_at DESC LIMIT $%d OFFSET $%d", arg, arg+1)
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	jobs := make([]*models.DownloadJobRecord, 0)
	for rows.Next() {
		var j models.DownloadJobRecord
		if err := rows.Scan(&j.ID, &j.Platform, &j.UserID, &j.ChatID, &j.URL, &j.Source, &j.Format, &j.Status, &j.RetryCount, &j.ErrorMessage, &j.StatusMessageID, &j.QueuedAt, &j.StartedAt, &j.CompletedAt, &j.UpdatedAt); err != nil {
			return nil, 0, err
		}
		jobs = append(jobs, &j)
	}
	return jobs, total, rows.Err()
}
