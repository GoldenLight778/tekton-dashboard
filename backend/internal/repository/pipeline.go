package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrPipelineNotFound      = errors.New("pipeline not found")
	ErrPipelineAlreadyExists = errors.New("pipeline with this name already exists in project")
)

// Pipeline represents a defined CI/CD pipeline in a project
type Pipeline struct {
	ID         string    `json:"id"`
	ProjectID  string    `json:"project_id"`
	Name       string    `json:"name"`
	DSLContent string    `json:"dsl_content"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// PipelineRepository defines data access for pipelines
type PipelineRepository interface {
	Create(ctx context.Context, p *Pipeline) error
	GetByID(ctx context.Context, id string) (*Pipeline, error)
	GetByProjectAndName(ctx context.Context, projectID, name string) (*Pipeline, error)
	ListByProject(ctx context.Context, projectID string, page, pageSize int) ([]Pipeline, int, error)
	ListAll(ctx context.Context) ([]Pipeline, error)
	Update(ctx context.Context, p *Pipeline) error
	Delete(ctx context.Context, id string) error
}

// SQLitePipelineRepository implements PipelineRepository
type SQLitePipelineRepository struct {
	db     *sql.DB
	driver string
}

// NewPipelineRepository creates a new SQLitePipelineRepository
func NewPipelineRepository(db *sql.DB, driver ...string) *SQLitePipelineRepository {
	d := "sqlite"
	if len(driver) > 0 && driver[0] != "" {
		d = driver[0]
	}
	return &SQLitePipelineRepository{db: db, driver: d}
}

func (r *SQLitePipelineRepository) Create(ctx context.Context, p *Pipeline) error {
	query := `INSERT INTO pipelines (id, project_id, name, dsl_content, created_at, updated_at)
	          VALUES (?, ?, ?, ?, ?, ?)`
	now := time.Now().UTC()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	p.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, Rebind(r.driver, query), p.ID, p.ProjectID, p.Name, p.DSLContent, p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create pipeline: %w", err)
	}
	return nil
}

func (r *SQLitePipelineRepository) GetByID(ctx context.Context, id string) (*Pipeline, error) {
	query := `SELECT id, project_id, name, dsl_content, created_at, updated_at FROM pipelines WHERE id = ?`
	row := r.db.QueryRowContext(ctx, Rebind(r.driver, query), id)

	var p Pipeline
	err := row.Scan(&p.ID, &p.ProjectID, &p.Name, &p.DSLContent, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrPipelineNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline: %w", err)
	}
	return &p, nil
}

func (r *SQLitePipelineRepository) GetByProjectAndName(ctx context.Context, projectID, name string) (*Pipeline, error) {
	query := `SELECT id, project_id, name, dsl_content, created_at, updated_at FROM pipelines WHERE project_id = ? AND name = ?`
	row := r.db.QueryRowContext(ctx, Rebind(r.driver, query), projectID, name)

	var p Pipeline
	err := row.Scan(&p.ID, &p.ProjectID, &p.Name, &p.DSLContent, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrPipelineNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline: %w", err)
	}
	return &p, nil
}

func (r *SQLitePipelineRepository) ListByProject(ctx context.Context, projectID string, page, pageSize int) ([]Pipeline, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int
	err := r.db.QueryRowContext(ctx, Rebind(r.driver, `SELECT count(*) FROM pipelines WHERE project_id = ?`), projectID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count pipelines: %w", err)
	}

	query := `SELECT id, project_id, name, dsl_content, created_at, updated_at 
	          FROM pipelines WHERE project_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, Rebind(r.driver, query), projectID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list pipelines: %w", err)
	}
	defer rows.Close()

	var list []Pipeline
	for rows.Next() {
		var p Pipeline
		if err := rows.Scan(&p.ID, &p.ProjectID, &p.Name, &p.DSLContent, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan pipeline: %w", err)
		}
		list = append(list, p)
	}

	return list, total, nil
}

func (r *SQLitePipelineRepository) ListAll(ctx context.Context) ([]Pipeline, error) {
	query := `SELECT id, project_id, name, dsl_content, created_at, updated_at 
	          FROM pipelines ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list all pipelines: %w", err)
	}
	defer rows.Close()

	var list []Pipeline
	for rows.Next() {
		var p Pipeline
		if err := rows.Scan(&p.ID, &p.ProjectID, &p.Name, &p.DSLContent, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan pipeline: %w", err)
		}
		list = append(list, p)
	}
	return list, nil
}


func (r *SQLitePipelineRepository) Update(ctx context.Context, p *Pipeline) error {
	query := `UPDATE pipelines SET name = ?, dsl_content = ?, updated_at = ? WHERE id = ?`
	p.UpdatedAt = time.Now().UTC()
	res, err := r.db.ExecContext(ctx, Rebind(r.driver, query), p.Name, p.DSLContent, p.UpdatedAt, p.ID)
	if err != nil {
		return fmt.Errorf("failed to update pipeline: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrPipelineNotFound
	}
	return nil
}

func (r *SQLitePipelineRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM pipelines WHERE id = ?`
	res, err := r.db.ExecContext(ctx, Rebind(r.driver, query), id)
	if err != nil {
		return fmt.Errorf("failed to delete pipeline: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrPipelineNotFound
	}
	return nil
}
