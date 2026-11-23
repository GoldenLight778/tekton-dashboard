package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrCredentialNotFound      = errors.New("credential not found")
	ErrCredentialAlreadyExists = errors.New("credential with this name already exists in project")
)

// Credential represents an encrypted secret belonging to a project
type Credential struct {
	ID            string    `json:"id"`
	ProjectID     string    `json:"project_id"`
	Name          string    `json:"name"`
	Type          string    `json:"type"` // git_token, ssh_key, kubeconfig, secret_text
	EncryptedData string    `json:"-"`    // raw secret never serialized
	MaskedValue   string    `json:"masked_value,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// CredentialRepository defines data access for credentials
type CredentialRepository interface {
	Create(ctx context.Context, c *Credential) error
	GetByID(ctx context.Context, id string) (*Credential, error)
	GetByProjectAndName(ctx context.Context, projectID, name string) (*Credential, error)
	ListByProject(ctx context.Context, projectID string) ([]Credential, error)
	Delete(ctx context.Context, id string) error
}

// SQLiteCredentialRepository implements CredentialRepository
type SQLiteCredentialRepository struct {
	db     *sql.DB
	driver string
}

// NewCredentialRepository creates a new SQLiteCredentialRepository
func NewCredentialRepository(db *sql.DB, driver ...string) *SQLiteCredentialRepository {
	d := "sqlite"
	if len(driver) > 0 && driver[0] != "" {
		d = driver[0]
	}
	return &SQLiteCredentialRepository{db: db, driver: d}
}

func (r *SQLiteCredentialRepository) Create(ctx context.Context, c *Credential) error {
	query := `INSERT INTO credentials (id, project_id, name, type, encrypted_data, created_at, updated_at)
	          VALUES (?, ?, ?, ?, ?, ? ,?)`
	now := time.Now().UTC()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	c.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, Rebind(r.driver, query), c.ID, c.ProjectID, c.Name, c.Type, c.EncryptedData, c.CreatedAt, c.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create credential: %w", err)
	}
	return nil
}

func (r *SQLiteCredentialRepository) GetByID(ctx context.Context, id string) (*Credential, error) {
	query := `SELECT id, project_id, name, type, encrypted_data, created_at, updated_at FROM credentials WHERE id = ?`
	row := r.db.QueryRowContext(ctx, Rebind(r.driver, query), id)

	var c Credential
	err := row.Scan(&c.ID, &c.ProjectID, &c.Name, &c.Type, &c.EncryptedData, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCredentialNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get credential: %w", err)
	}
	return &c, nil
}

func (r *SQLiteCredentialRepository) GetByProjectAndName(ctx context.Context, projectID, name string) (*Credential, error) {
	query := `SELECT id, project_id, name, type, encrypted_data, created_at, updated_at FROM credentials WHERE project_id = ? AND name = ?`
	row := r.db.QueryRowContext(ctx, Rebind(r.driver, query), projectID, name)

	var c Credential
	err := row.Scan(&c.ID, &c.ProjectID, &c.Name, &c.Type, &c.EncryptedData, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCredentialNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get credential: %w", err)
	}
	return &c, nil
}

func (r *SQLiteCredentialRepository) ListByProject(ctx context.Context, projectID string) ([]Credential, error) {
	query := `SELECT id, project_id, name, type, encrypted_data, created_at, updated_at 
	          FROM credentials WHERE project_id = ? ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, Rebind(r.driver, query), projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list credentials: %w", err)
	}
	defer rows.Close()

	var list []Credential
	for rows.Next() {
		var c Credential
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.Name, &c.Type, &c.EncryptedData, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan credential: %w", err)
		}
		list = append(list, c)
	}
	return list, nil
}

func (r *SQLiteCredentialRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM credentials WHERE id = ?`
	res, err := r.db.ExecContext(ctx, Rebind(r.driver, query), id)
	if err != nil {
		return fmt.Errorf("failed to delete credential: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrCredentialNotFound
	}
	return nil
}
