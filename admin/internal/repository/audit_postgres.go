package repository

import (
	"context"
	"database/sql"

	"github.com/mediaharvester/tg-downloader/admin/internal/domain"
)

type auditRepository struct{ db *sql.DB }

func NewAuditRepository(db *sql.DB) domain.AuditRepository { return &auditRepository{db: db} }

func (r *auditRepository) Create(ctx context.Context, adminID int64, action, targetType, targetID, details, ip, userAgent string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO admin_audit_log (admin_id, action, target_type, target_id, details, ip_address, user_agent) VALUES ($1,$2,$3,$4,$5::jsonb, NULLIF($6,'')::inet, $7)`, adminID, action, targetType, targetID, details, ip, userAgent)
	return err
}
