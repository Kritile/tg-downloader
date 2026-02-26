package repository

import (
	"context"
	"database/sql"
	"strings"

	"github.com/mediaharvester/tg-downloader/bot/internal/domain"
	"github.com/mediaharvester/tg-downloader/shared/models"
)

type settingsRepository struct {
	db *sql.DB
}

func NewSettingsRepository(db *sql.DB) domain.SettingsRepository {
	return &settingsRepository{db: db}
}

func (r *settingsRepository) Get(ctx context.Context) (*models.Settings, error) {
	query := `
		SELECT id, default_daily_limit, default_monthly_limit, default_youtube_allowed, default_instagram_allowed, default_tiktok_allowed
		FROM settings
		WHERE id = 1
	`

	var settings models.Settings
	err := r.db.QueryRowContext(ctx, query).Scan(
		&settings.ID,
		&settings.DefaultDailyLimit,
		&settings.DefaultMonthlyLimit,
		&settings.DefaultYoutubeAllowed,
		&settings.DefaultInstagramAllowed,
		&settings.DefaultTiktokAllowed,
	)
	if err != nil && strings.Contains(err.Error(), "default_instagram_allowed") {
		legacyQuery := `
			SELECT id, default_daily_limit, default_monthly_limit, default_youtube_allowed, default_tiktok_allowed
			FROM settings
			WHERE id = 1
		`
		err = r.db.QueryRowContext(ctx, legacyQuery).Scan(
			&settings.ID,
			&settings.DefaultDailyLimit,
			&settings.DefaultMonthlyLimit,
			&settings.DefaultYoutubeAllowed,
			&settings.DefaultTiktokAllowed,
		)
		settings.DefaultInstagramAllowed = settings.DefaultYoutubeAllowed
	}

	if err == sql.ErrNoRows {
		return nil, domain.ErrSettingsNotFound
	}
	if err != nil {
		return nil, err
	}

	return &settings, nil
}

func (r *settingsRepository) Update(ctx context.Context, settings *models.Settings) error {
	query := `
		UPDATE settings
		SET default_daily_limit = $2, default_monthly_limit = $3, default_youtube_allowed = $4, default_instagram_allowed = $5, default_tiktok_allowed = $6
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query,
		settings.ID,
		settings.DefaultDailyLimit,
		settings.DefaultMonthlyLimit,
		settings.DefaultYoutubeAllowed,
		settings.DefaultInstagramAllowed,
		settings.DefaultTiktokAllowed,
	)
	if err != nil && strings.Contains(err.Error(), "default_instagram_allowed") {
		legacyQuery := `
			UPDATE settings
			SET default_daily_limit = $2, default_monthly_limit = $3, default_youtube_allowed = $4, default_tiktok_allowed = $5
			WHERE id = $1
		`
		_, err = r.db.ExecContext(ctx, legacyQuery,
			settings.ID,
			settings.DefaultDailyLimit,
			settings.DefaultMonthlyLimit,
			settings.DefaultYoutubeAllowed,
			settings.DefaultTiktokAllowed,
		)
	}

	return err
}
