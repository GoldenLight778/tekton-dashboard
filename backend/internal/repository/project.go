package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrProjectNotFound      = errors.New("project not found")
	ErrProjectAlreadyExists = errors.New("project name already exists")
)

// Project represents a workspace or repository project in the platform
type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ProjectRepository defines data access for projects
type ProjectRepository interface {
	Create(ctx context.Context, p *Project) error
	GetByID(ctx context.Context, id string) (*Project, error)
	GetByName(ctx context.Context, name string) (*Project, error)
	List(ctx context.Context, page, pageSize int) ([]Project, int, error)
	Update(ctx context.Context, p *Project) error
	Delete(ctx context.Context, id string) error
}

// SQLiteProjectRepository implements ProjectRepository
type SQLiteProjectRepository struct {
	db *sql.DB
}

// NewProjectRepository creates a new SQLiteProjectRepository
func NewProjectRepository(db *sql.DB) *SQLiteProjectRepository {
	return &SQLiteProjectRepository{db: db}
}

func (r *SQLiteProjectRepository) Create(ctx context.Context, p *Project) error {
	query := `INSERT INTO projects (id, name, description, created_by, created_at, updated_at)
	          VALUES (?, ?, ?, ?, ?, ?)`
	now := time.Now().UTC()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	p.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, query, p.ID, p.Name, p.Description, p.CreatedBy, p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}
	return nil
}

func (r *SQLiteProjectRepository) GetByID(ctx context.Context, id string) (*Project, error) {
	query := `SELECT id, name, description, created_by, created_at, updated_at FROM projects WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)

	var p Project
	var desc sql.NullString
	err := row.Scan(&p.ID, &p.Name, &desc, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProjectNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	p.Description = desc.String
	return &p, nil
}

func (r *SQLiteProjectRepository) GetByName(ctx context.Context, name string) (*Project, error) {
	query := `SELECT id, name, description, created_by, created_at, updated_at FROM projects WHERE name = ?`
	row := r.db.QueryRowContext(ctx, query, name)

	var p Project
	var desc sql.NullString
	err := row.Scan(&p.ID, &p.Name, &desc, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProjectNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get project by name: %w", err)
	}
	p.Description = desc.String
	return &p, nil
}

func (r *SQLiteProjectRepository) List(ctx context.Context, page, pageSize int) ([]Project, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int
	err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM projects`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count projects: %w", err)
	}

	query := `SELECT id, name, description, created_by, created_at, updated_at 
	          FROM projects ORDER BY created_at DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, query, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list projects: %w", err)
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		var desc sql.NullString
		if err := rows.Scan(&p.ID, &p.Name, &desc, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan project: %w", err)
		}
		p.Description = desc.String
		projects = append(projects, p)
	}

	return projects, total, nil
}

func (r *SQLiteProjectRepository) Update(ctx context.Context, p *Project) error {
	query := `UPDATE projects SET name = ?, description = ?, updated_at = ? WHERE id = ?`
	p.UpdatedAt = time.Now().UTC()
	res, err := r.db.ExecContext(ctx, query, p.Name, p.Description, p.UpdatedAt, p.ID)
	if err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrProjectNotFound
	}
	return nil
}

func (r *SQLiteProjectRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM projects WHERE id = ?`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrProjectNotFound
	}
	return nil
}
