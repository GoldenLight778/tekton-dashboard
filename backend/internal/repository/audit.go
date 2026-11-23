package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// AuditLog represents an action record in the system
type AuditLog struct {
	ID           int64     `json:"id"`
	UserID       string    `json:"user_id"`
	Action       string    `json:"action"`        // CREATE_PROJECT, TRIGGER_PIPELINE, etc.
	ResourceType string    `json:"resource_type"` // project, pipeline, etc.
	ResourceID   string    `json:"resource_id"`
	Detail       string    `json:"detail"`
	IP           string    `json:"ip"`
	CreatedAt    time.Time `json:"created_at"`
}

// AuditRepository handles persisting audit logs
type AuditRepository interface {
	Record(ctx context.Context, log *AuditLog) error
	List(ctx context.Context, page, pageSize int) ([]AuditLog, int, error)
}

// SQLiteAuditRepository implements AuditRepository
type SQLiteAuditRepository struct {
	db     *sql.DB
	driver string
}

// NewAuditRepository creates a new SQLiteAuditRepository
func NewAuditRepository(db *sql.DB, driver ...string) *SQLiteAuditRepository {
	d := "sqlite"
	if len(driver) > 0 && driver[0] != "" {
		d = driver[0]
	}
	return &SQLiteAuditRepository{db: db, driver: d}
}

func (r *SQLiteAuditRepository) Record(ctx context.Context, log *AuditLog) error {
	now := time.Now().UTC()
	if log.CreatedAt.IsZero() {
		log.CreatedAt = now
	}

	if r.driver == "postgres" || r.driver == "pgx" {
		query := `INSERT INTO audit_logs (user_id, action, resource_type, resource_id, detail, ip, created_at)
		          VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`
		err := r.db.QueryRowContext(ctx, query, log.UserID, log.Action, log.ResourceType, log.ResourceID, log.Detail, log.IP, log.CreatedAt).Scan(&log.ID)
		if err != nil {
			return fmt.Errorf("failed to record audit log: %w", err)
		}
		return nil
	}

	query := `INSERT INTO audit_logs (user_id, action, resource_type, resource_id, detail, ip, created_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?)`
	res, err := r.db.ExecContext(ctx, Rebind(r.driver, query), log.UserID, log.Action, log.ResourceType, log.ResourceID, log.Detail, log.IP, log.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to record audit log: %w", err)
	}
	id, _ := res.LastInsertId()
	log.ID = id
	return nil
}

func (r *SQLiteAuditRepository) List(ctx context.Context, page, pageSize int) ([]AuditLog, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int
	err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM audit_logs`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	query := `SELECT id, user_id, action, resource_type, resource_id, detail, ip, created_at
	          FROM audit_logs ORDER BY created_at DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, Rebind(r.driver, query), pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list audit logs: %w", err)
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var l AuditLog
		var uid, detail, ip sql.NullString
		if err := rows.Scan(&l.ID, &uid, &l.Action, &l.ResourceType, &l.ResourceID, &detail, &ip, &l.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit log: %w", err)
		}
		l.UserID = uid.String
		l.Detail = detail.String
		l.IP = ip.String
		logs = append(logs, l)
	}

	return logs, total, nil
}
