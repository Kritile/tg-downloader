package repository

import (
	"context"
	"database/sql"
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
		SELECT id, telegram_id, username, can_youtube, can_tiktok, daily_limit, monthly_limit, created_at, updated_at
		FROM users
		WHERE telegram_id = $1
	`

	var user models.User
	var canYoutube, canTiktok sql.NullBool
	var dailyLimit, monthlyLimit sql.NullInt32

	err := r.db.QueryRowContext(ctx, query, telegramID).Scan(
		&user.ID,
		&user.TelegramID,
		&user.Username,
		&canYoutube,
		&canTiktok,
		&dailyLimit,
		&monthlyLimit,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	// Handle nullable fields
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

	return &user, nil
}

func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (telegram_id, username, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	now := time.Now()
	err := r.db.QueryRowContext(ctx, query, user.TelegramID, user.Username, now, now).Scan(&user.ID)
	return err
}

func (r *userRepository) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users
		SET username = $2, can_youtube = $3, can_tiktok = $4, daily_limit = $5, monthly_limit = $6, updated_at = $7
		WHERE id = $1
	`

	now := time.Now()

	var canYoutube, canTiktok interface{}
	var dailyLimit, monthlyLimit interface{}

	if user.CanYoutube != nil {
		canYoutube = *user.CanYoutube
	} else {
		canYoutube = nil
	}
	if user.CanTiktok != nil {
		canTiktok = *user.CanTiktok
	} else {
		canTiktok = nil
	}
	if user.DailyLimit != nil {
		dailyLimit = *user.DailyLimit
	} else {
		dailyLimit = nil
	}
	if user.MonthlyLimit != nil {
		monthlyLimit = *user.MonthlyLimit
	} else {
		monthlyLimit = nil
	}

	_, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Username,
		canYoutube,
		canTiktok,
		dailyLimit,
		monthlyLimit,
		now,
	)

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

	// Create new user
	user = &models.User{
		TelegramID: telegramID,
		Username:   username,
	}

	err = r.Create(ctx, user)
	if err != nil {
		return nil, err
	}

	return user, nil
}
