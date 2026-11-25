package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrTaskNotFound      = errors.New("task not found")
	ErrTaskAlreadyExists = errors.New("task with this name already exists in namespace")
)

// Task represents a basic reusable Tekton Task definition stored in the platform
type Task struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Namespace   string    `json:"namespace"`
	Description string    `json:"description"`
	Image       string    `json:"image"`
	Script      string    `json:"script"`
	SpecJSON    string    `json:"spec_json"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TaskRepository defines data access for reusable tasks
type TaskRepository interface {
	Create(ctx context.Context, t *Task) error
	GetByNamespaceAndName(ctx context.Context, namespace, name string) (*Task, error)
	ListAll(ctx context.Context) ([]Task, error)
	DeleteByNamespaceAndName(ctx context.Context, namespace, name string) error
}

// SQLiteTaskRepository implements TaskRepository across SQLite, MySQL, and Postgres
type SQLiteTaskRepository struct {
	db     *sql.DB
	driver string
}

// NewTaskRepository creates a new SQLiteTaskRepository
func NewTaskRepository(db *sql.DB, driver ...string) *SQLiteTaskRepository {
	d := "sqlite"
	if len(driver) > 0 && driver[0] != "" {
		d = driver[0]
	}
	return &SQLiteTaskRepository{db: db, driver: d}
}

func (r *SQLiteTaskRepository) Create(ctx context.Context, t *Task) error {
	now := time.Now().UTC()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	t.UpdatedAt = now
	if t.Namespace == "" {
		t.Namespace = "default"
	}
	if t.Image == "" {
		t.Image = "alpine:latest"
	}
	if t.SpecJSON == "" {
		t.SpecJSON = "{}"
	}

	query := `INSERT INTO tasks (id, name, namespace, description, image, script, spec_json, created_at, updated_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, Rebind(r.driver, query),
		t.ID, t.Name, t.Namespace, t.Description, t.Image, t.Script, t.SpecJSON, t.CreatedAt, t.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}
	return nil
}

func (r *SQLiteTaskRepository) GetByNamespaceAndName(ctx context.Context, namespace, name string) (*Task, error) {
	if namespace == "" {
		namespace = "default"
	}
	query := `SELECT id, name, namespace, description, image, script, spec_json, created_at, updated_at
	          FROM tasks WHERE namespace = ? AND name = ?`
	row := r.db.QueryRowContext(ctx, Rebind(r.driver, query), namespace, name)

	var t Task
	err := row.Scan(&t.ID, &t.Name, &t.Namespace, &t.Description, &t.Image, &t.Script, &t.SpecJSON, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTaskNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	return &t, nil
}

func (r *SQLiteTaskRepository) ListAll(ctx context.Context) ([]Task, error) {
	query := `SELECT id, name, namespace, description, image, script, spec_json, created_at, updated_at
	          FROM tasks ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	var list []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.Name, &t.Namespace, &t.Description, &t.Image, &t.Script, &t.SpecJSON, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		list = append(list, t)
	}
	return list, nil
}

func (r *SQLiteTaskRepository) DeleteByNamespaceAndName(ctx context.Context, namespace, name string) error {
	if namespace == "" {
		namespace = "default"
	}
	query := `DELETE FROM tasks WHERE namespace = ? AND name = ?`
	res, err := r.db.ExecContext(ctx, Rebind(r.driver, query), namespace, name)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ErrTaskNotFound
	}
	return nil
}
