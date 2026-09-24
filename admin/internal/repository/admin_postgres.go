package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/mediaharvester/tg-downloader/admin/internal/domain"
	"github.com/mediaharvester/tg-downloader/shared/models"
)

type adminUserRepository struct {
	db *sql.DB
}

func NewAdminUserRepository(db *sql.DB) domain.AdminUserRepository {
	return &adminUserRepository{db: db}
}

func (r *adminUserRepository) GetByUsername(ctx context.Context, username string) (*models.Admin, error) {
	query := `
		SELECT id, username, password_hash, created_at
		FROM admins
		WHERE username = $1
	`

	var admin models.Admin
	err := r.db.QueryRowContext(ctx, query, username).Scan(
		&admin.ID,
		&admin.Username,
		&admin.PasswordHash,
		&admin.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domain.ErrAdminNotFound
	}
	if err != nil {
		return nil, err
	}

	return &admin, nil
}

func (r *adminUserRepository) GetByID(ctx context.Context, id int64) (*models.Admin, error) {
	query := `
		SELECT id, username, password_hash, created_at
		FROM admins
		WHERE id = $1
	`

	var admin models.Admin
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&admin.ID,
		&admin.Username,
		&admin.PasswordHash,
		&admin.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domain.ErrAdminNotFound
	}
	if err != nil {
		return nil, err
	}

	return &admin, nil
}

func (r *adminUserRepository) UpdatePasswordHash(ctx context.Context, id int64, passwordHash string) error {
	query := `UPDATE admins SET password_hash = $2 WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id, passwordHash)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrAdminNotFound
	}

	return nil
}

func (r *adminUserRepository) Create(ctx context.Context, admin *models.Admin) error {
	query := `
		INSERT INTO admins (username, password_hash, created_at)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	now := time.Now()
	err := r.db.QueryRowContext(ctx, query, admin.Username, admin.PasswordHash, now).Scan(&admin.ID)
	return err
}

func (r *adminUserRepository) GetAll(ctx context.Context) ([]*models.Admin, error) {
	query := `
		SELECT id, username, password_hash, created_at
		FROM admins
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var admins []*models.Admin
	for rows.Next() {
		var admin models.Admin
		err := rows.Scan(
			&admin.ID,
			&admin.Username,
			&admin.PasswordHash,
			&admin.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		admins = append(admins, &admin)
	}

	return admins, rows.Err()
}

func (r *adminUserRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM admins WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// Settings repository for admin
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

	return err
}

// User repository implementation for admin
type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) domain.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) GetByID(ctx context.Context, id int64) (*models.User, error) {
	query := `
		SELECT id, telegram_id, username, can_youtube, can_instagram, can_tiktok, daily_limit, monthly_limit, is_blocked, last_seen_at, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user models.User
	var canYoutube, canInstagram, canTiktok sql.NullBool
	var dailyLimit, monthlyLimit sql.NullInt32

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.TelegramID,
		&user.Username,
		&canYoutube,
		&canInstagram,
		&canTiktok,
		&dailyLimit,
		&monthlyLimit,
		&user.IsBlocked,
		&user.LastSeenAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

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

	return &user, nil
}

func (r *userRepository) GetAll(ctx context.Context, limit, offset int) ([]*models.User, int, error) {
	countQuery := `SELECT COUNT(*) FROM users`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, telegram_id, username, can_youtube, can_instagram, can_tiktok, daily_limit, monthly_limit, is_blocked, last_seen_at, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	users := make([]*models.User, 0)
	for rows.Next() {
		var user models.User
		var canYoutube, canInstagram, canTiktok sql.NullBool
		var dailyLimit, monthlyLimit sql.NullInt32
		if err := rows.Scan(&user.ID, &user.TelegramID, &user.Username, &canYoutube, &canInstagram, &canTiktok, &dailyLimit, &monthlyLimit, &user.IsBlocked, &user.LastSeenAt, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, 0, err
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
			v := int(dailyLimit.Int32)
			user.DailyLimit = &v
		}
		if monthlyLimit.Valid {
			v := int(monthlyLimit.Int32)
			user.MonthlyLimit = &v
		}
		users = append(users, &user)
	}

	return users, total, rows.Err()
}

func (r *userRepository) SearchByTelegramID(ctx context.Context, telegramID int64, limit int) ([]*models.User, error) {
	query := `
		SELECT id, telegram_id, username, can_youtube, can_instagram, can_tiktok, daily_limit, monthly_limit, is_blocked, last_seen_at, created_at, updated_at
		FROM users
		WHERE telegram_id = $1
		LIMIT $2
	`
	rows, err := r.db.QueryContext(ctx, query, telegramID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]*models.User, 0)
	for rows.Next() {
		var user models.User
		var canYoutube, canInstagram, canTiktok sql.NullBool
		var dailyLimit, monthlyLimit sql.NullInt32
		if err := rows.Scan(&user.ID, &user.TelegramID, &user.Username, &canYoutube, &canInstagram, &canTiktok, &dailyLimit, &monthlyLimit, &user.IsBlocked, &user.LastSeenAt, &user.CreatedAt, &user.UpdatedAt); err != nil {
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
			v := int(dailyLimit.Int32)
			user.DailyLimit = &v
		}
		if monthlyLimit.Valid {
			v := int(monthlyLimit.Int32)
			user.MonthlyLimit = &v
		}
		users = append(users, &user)
	}

	return users, rows.Err()
}

func (r *userRepository) Search(ctx context.Context, query string, limit, offset int) ([]*models.User, int, error) {
	pattern := "%" + query + "%"
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE CAST(telegram_id AS TEXT) ILIKE $1 OR COALESCE(username, '') ILIKE $1`, pattern).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, telegram_id, username, can_youtube, can_instagram, can_tiktok, daily_limit, monthly_limit, is_blocked, last_seen_at, created_at, updated_at
		FROM users
		WHERE CAST(telegram_id AS TEXT) ILIKE $1 OR COALESCE(username, '') ILIKE $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`, pattern, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	users := make([]*models.User, 0)
	for rows.Next() {
		var u models.User
		var yt, ig, tt sql.NullBool
		var dl, ml sql.NullInt32
		if err := rows.Scan(&u.ID, &u.TelegramID, &u.Username, &yt, &ig, &tt, &dl, &ml, &u.IsBlocked, &u.LastSeenAt, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, err
		}
		if yt.Valid {
			u.CanYoutube = &yt.Bool
		}
		if ig.Valid {
			u.CanInstagram = &ig.Bool
		}
		if tt.Valid {
			u.CanTiktok = &tt.Bool
		}
		if dl.Valid {
			v := int(dl.Int32)
			u.DailyLimit = &v
		}
		if ml.Valid {
			v := int(ml.Int32)
			u.MonthlyLimit = &v
		}
		users = append(users, &u)
	}
	return users, total, rows.Err()
}

func (r *userRepository) SetBlocked(ctx context.Context, userID int64, blocked bool) error {
	result, err := r.db.ExecContext(ctx, `UPDATE users SET is_blocked = $2, updated_at = NOW() WHERE id = $1`, userID, blocked)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (r *userRepository) GetByTelegramID(ctx context.Context, telegramID int64) (*models.User, error) {
	query := `
		SELECT id, telegram_id, username, can_youtube, can_instagram, can_tiktok, daily_limit, monthly_limit, created_at, updated_at
		FROM users
		WHERE telegram_id = $1
	`

	var user models.User
	var canYoutube, canInstagram, canTiktok sql.NullBool
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
		&user.CreatedAt,
		&user.UpdatedAt,
	)

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

	return &user, nil
}

func (r *userRepository) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users
		SET username = $2, can_youtube = $3, can_instagram = $4, can_tiktok = $5, daily_limit = $6, monthly_limit = $7, updated_at = $8
		WHERE id = $1
	`

	now := time.Now()

	var canYoutube, canInstagram, canTiktok interface{}
	var dailyLimit, monthlyLimit interface{}

	if user.CanYoutube != nil {
		canYoutube = *user.CanYoutube
	} else {
		canYoutube = nil
	}
	if user.CanInstagram != nil {
		canInstagram = *user.CanInstagram
	} else {
		canInstagram = nil
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
		canInstagram,
		canTiktok,
		dailyLimit,
		monthlyLimit,
		now,
	)

	return err
}
