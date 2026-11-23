package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrRunNotFound = errors.New("pipeline run not found")
)

// PipelineRun represents an execution instance of a pipeline
type PipelineRun struct {
	ID                 string     `json:"id"`
	PipelineID         string     `json:"pipeline_id"`
	K8sPipelineRunName string     `json:"k8s_pipelinerun_name"`
	Namespace          string     `json:"namespace"`
	Status             string     `json:"status"` // Pending, Running, Succeeded, Failed, Cancelled
	TriggeredBy        string     `json:"triggered_by"`
	StartedAt          *time.Time `json:"started_at,omitempty"`
	FinishedAt         *time.Time `json:"finished_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
}

// PipelineRunRepository defines data access for pipeline runs
type PipelineRunRepository interface {
	Create(ctx context.Context, r *PipelineRun) error
	GetByID(ctx context.Context, id string) (*PipelineRun, error)
	ListByPipeline(ctx context.Context, pipelineID string, page, pageSize int) ([]PipelineRun, int, error)
	ListByProject(ctx context.Context, projectID string, page, pageSize int) ([]PipelineRun, int, error)
	UpdateStatus(ctx context.Context, id, status string, startedAt, finishedAt *time.Time) error
}

// SQLitePipelineRunRepository implements PipelineRunRepository
type SQLitePipelineRunRepository struct {
	db *sql.DB
}

// NewPipelineRunRepository creates a new SQLitePipelineRunRepository
func NewPipelineRunRepository(db *sql.DB) *SQLitePipelineRunRepository {
	return &SQLitePipelineRunRepository{db: db}
}

func (r *SQLitePipelineRunRepository) Create(ctx context.Context, run *PipelineRun) error {
	query := `INSERT INTO pipeline_runs (id, pipeline_id, k8s_pipelinerun_name, namespace, status, triggered_by, started_at, finished_at, created_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	now := time.Now().UTC()
	if run.CreatedAt.IsZero() {
		run.CreatedAt = now
	}

	_, err := r.db.ExecContext(ctx, query, run.ID, run.PipelineID, run.K8sPipelineRunName, run.Namespace, run.Status, run.TriggeredBy, run.StartedAt, run.FinishedAt, run.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create pipeline run: %w", err)
	}
	return nil
}

func (r *SQLitePipelineRunRepository) GetByID(ctx context.Context, id string) (*PipelineRun, error) {
	query := `SELECT id, pipeline_id, k8s_pipelinerun_name, namespace, status, triggered_by, started_at, finished_at, created_at
	          FROM pipeline_runs WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)

	var run PipelineRun
	var k8sName sql.NullString
	var started, finished sql.NullTime
	err := row.Scan(&run.ID, &run.PipelineID, &k8sName, &run.Namespace, &run.Status, &run.TriggeredBy, &started, &finished, &run.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRunNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline run: %w", err)
	}
	run.K8sPipelineRunName = k8sName.String
	if started.Valid {
		run.StartedAt = &started.Time
	}
	if finished.Valid {
		run.FinishedAt = &finished.Time
	}
	return &run, nil
}

func (r *SQLitePipelineRunRepository) ListByPipeline(ctx context.Context, pipelineID string, page, pageSize int) ([]PipelineRun, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int
	err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM pipeline_runs WHERE pipeline_id = ?`, pipelineID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count pipeline runs: %w", err)
	}

	query := `SELECT id, pipeline_id, k8s_pipelinerun_name, namespace, status, triggered_by, started_at, finished_at, created_at
	          FROM pipeline_runs WHERE pipeline_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, query, pipelineID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list pipeline runs: %w", err)
	}
	defer rows.Close()

	var runs []PipelineRun
	for rows.Next() {
		var run PipelineRun
		var k8sName sql.NullString
		var started, finished sql.NullTime
		if err := rows.Scan(&run.ID, &run.PipelineID, &k8sName, &run.Namespace, &run.Status, &run.TriggeredBy, &started, &finished, &run.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan pipeline run: %w", err)
		}
		run.K8sPipelineRunName = k8sName.String
		if started.Valid {
			run.StartedAt = &started.Time
		}
		if finished.Valid {
			run.FinishedAt = &finished.Time
		}
		runs = append(runs, run)
	}

	return runs, total, nil
}

func (r *SQLitePipelineRunRepository) ListByProject(ctx context.Context, projectID string, page, pageSize int) ([]PipelineRun, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	countQuery := `SELECT count(*) FROM pipeline_runs pr JOIN pipelines p ON pr.pipeline_id = p.id WHERE p.project_id = ?`
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, projectID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count project pipeline runs: %w", err)
	}

	query := `SELECT pr.id, pr.pipeline_id, pr.k8s_pipelinerun_name, pr.namespace, pr.status, pr.triggered_by, pr.started_at, pr.finished_at, pr.created_at
	          FROM pipeline_runs pr 
	          JOIN pipelines p ON pr.pipeline_id = p.id 
	          WHERE p.project_id = ? 
	          ORDER BY pr.created_at DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, query, projectID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list project pipeline runs: %w", err)
	}
	defer rows.Close()

	var runs []PipelineRun
	for rows.Next() {
		var run PipelineRun
		var k8sName sql.NullString
		var started, finished sql.NullTime
		if err := rows.Scan(&run.ID, &run.PipelineID, &k8sName, &run.Namespace, &run.Status, &run.TriggeredBy, &started, &finished, &run.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan pipeline run: %w", err)
		}
		run.K8sPipelineRunName = k8sName.String
		if started.Valid {
			run.StartedAt = &started.Time
		}
		if finished.Valid {
			run.FinishedAt = &finished.Time
		}
		runs = append(runs, run)
	}

	return runs, total, nil
}

func (r *SQLitePipelineRunRepository) UpdateStatus(ctx context.Context, id, status string, startedAt, finishedAt *time.Time) error {
	query := `UPDATE pipeline_runs SET status = ?, started_at = COALESCE(?, started_at), finished_at = COALESCE(?, finished_at) WHERE id = ?`
	res, err := r.db.ExecContext(ctx, query, status, startedAt, finishedAt, id)
	if err != nil {
		return fmt.Errorf("failed to update pipeline run status: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrRunNotFound
	}
	return nil
}
