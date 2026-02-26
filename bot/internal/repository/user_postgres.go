package repository

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/mediaharvester/tg-downloader/bot/internal/domain"
	"github.com/mediaharvester/tg-downloader/shared/models"
)

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) domain.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) GetByTelegramID(ctx context.Context, telegramID int64) (*models.User, error) {
	query := `
		SELECT id, telegram_id, username, can_youtube, can_instagram, can_tiktok, daily_limit, monthly_limit, auto_best_download, created_at, updated_at
		FROM users
		WHERE telegram_id = $1
	`

	var user models.User
	var canYoutube, canInstagram, canTiktok, autoBest sql.NullBool
	var dailyLimit, monthlyLimit sql.NullInt32

	err := r.db.QueryRowContext(ctx, query, telegramID).Scan(
		&user.ID,
		&user.TelegramID,
		&user.Username,
		&canYoutube,
		&canInstagram,
		&canTiktok,
		&dailyLimit,
		&monthlyLimit,
		&autoBest,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil && strings.Contains(err.Error(), "can_instagram") {
		legacyQuery := `
			SELECT id, telegram_id, username, can_youtube, can_tiktok, daily_limit, monthly_limit, auto_best_download, created_at, updated_at
			FROM users
			WHERE telegram_id = $1
		`
		err = r.db.QueryRowContext(ctx, legacyQuery, telegramID).Scan(
			&user.ID,
			&user.TelegramID,
			&user.Username,
			&canYoutube,
			&canTiktok,
			&dailyLimit,
			&monthlyLimit,
			&autoBest,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
	}

	if err == sql.ErrNoRows {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	if canYoutube.Valid {
		user.CanYoutube = &canYoutube.Bool
	}
	if canInstagram.Valid {
		user.CanInstagram = &canInstagram.Bool
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
	if autoBest.Valid {
		user.AutoBestDownload = &autoBest.Bool
	}

	return &user, nil
}

func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (telegram_id, username, auto_best_download, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	now := time.Now()
	var autoBest interface{}
	if user.AutoBestDownload != nil {
		autoBest = *user.AutoBestDownload
	}

	err := r.db.QueryRowContext(ctx, query, user.TelegramID, user.Username, autoBest, now, now).Scan(&user.ID)
	return err
}

func (r *userRepository) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users
		SET username = $2, can_youtube = $3, can_instagram = $4, can_tiktok = $5, daily_limit = $6, monthly_limit = $7, auto_best_download = $8, updated_at = $9
		WHERE id = $1
	`

	now := time.Now()

	var canYoutube, canInstagram, canTiktok interface{}
	var dailyLimit, monthlyLimit interface{}
	var autoBest interface{}

	if user.CanYoutube != nil {
		canYoutube = *user.CanYoutube
	}
	if user.CanInstagram != nil {
		canInstagram = *user.CanInstagram
	}
	if user.CanTiktok != nil {
		canTiktok = *user.CanTiktok
	}
	if user.DailyLimit != nil {
		dailyLimit = *user.DailyLimit
	}
	if user.MonthlyLimit != nil {
		monthlyLimit = *user.MonthlyLimit
	}
	if user.AutoBestDownload != nil {
		autoBest = *user.AutoBestDownload
	}

	_, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Username,
		canYoutube,
		canInstagram,
		canTiktok,
		dailyLimit,
		monthlyLimit,
		autoBest,
		now,
	)
	if err != nil && strings.Contains(err.Error(), "can_instagram") {
		legacyQuery := `
			UPDATE users
			SET username = $2, can_youtube = $3, can_tiktok = $4, daily_limit = $5, monthly_limit = $6, auto_best_download = $7, updated_at = $8
			WHERE id = $1
		`
		_, err = r.db.ExecContext(ctx, legacyQuery,
			user.ID,
			user.Username,
			canYoutube,
			canTiktok,
			dailyLimit,
			monthlyLimit,
			autoBest,
			now,
		)
	}

	return err
}

func (r *userRepository) UpdateAutoBestDownload(ctx context.Context, userID int64, autoBest *bool) error {
	query := `UPDATE users SET auto_best_download = $2, updated_at = $3 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, userID, autoBest, time.Now())
	return err
}

func (r *userRepository) GetOrCreate(ctx context.Context, telegramID int64, username string) (*models.User, error) {
	user, err := r.GetByTelegramID(ctx, telegramID)
	if err == nil {
		return user, nil
	}

	if err != domain.ErrUserNotFound {
		return nil, err
	}

	user = &models.User{TelegramID: telegramID, Username: username}
	if err := r.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}
