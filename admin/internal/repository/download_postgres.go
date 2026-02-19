package repository

import (
	"context"
	"database/sql"

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
