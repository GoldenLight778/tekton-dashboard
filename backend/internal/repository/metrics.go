package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// DashboardMetrics represents aggregated statistics for the platform dashboard
type DashboardMetrics struct {
	TotalProjects      int                    `json:"total_projects"`
	TotalPipelines     int                    `json:"total_pipelines"`
	TotalRuns          int                    `json:"total_runs"`
	SucceededRuns      int                    `json:"succeeded_runs"`
	FailedRuns         int                    `json:"failed_runs"`
	RunningRuns        int                    `json:"running_runs"`
	CancelledRuns      int                    `json:"cancelled_runs"`
	SuccessRate        float64                `json:"success_rate"`        // 0.0 to 100.0
	AvgDurationSeconds float64                `json:"avg_duration_seconds"`
	StatusCounts       map[string]int         `json:"status_counts"`
	RecentRuns         []DashboardRecentRun   `json:"recent_runs"`
}

// DashboardRecentRun represents an enriched run item for the dashboard
type DashboardRecentRun struct {
	ID           string     `json:"id"`
	PipelineID   string     `json:"pipeline_id"`
	PipelineName string     `json:"pipeline_name"`
	ProjectName  string     `json:"project_name"`
	Status       string     `json:"status"`
	TriggeredBy  string     `json:"triggered_by"`
	DurationSec  int64      `json:"duration_seconds"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// MetricsRepository defines queries for platform statistics
type MetricsRepository interface {
	GetDashboardMetrics(ctx context.Context) (*DashboardMetrics, error)
}

// SQLiteMetricsRepository implements MetricsRepository
type SQLiteMetricsRepository struct {
	db     *sql.DB
	driver string
}

// NewMetricsRepository creates a new SQLiteMetricsRepository
func NewMetricsRepository(db *sql.DB, driver ...string) *SQLiteMetricsRepository {
	d := "sqlite"
	if len(driver) > 0 && driver[0] != "" {
		d = driver[0]
	}
	return &SQLiteMetricsRepository{db: db, driver: d}
}

func (r *SQLiteMetricsRepository) GetDashboardMetrics(ctx context.Context) (*DashboardMetrics, error) {
	metrics := &DashboardMetrics{
		StatusCounts: make(map[string]int),
		RecentRuns:   make([]DashboardRecentRun, 0),
	}

	// 1. Total projects
	if err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM projects`).Scan(&metrics.TotalProjects); err != nil {
		return nil, fmt.Errorf("failed to count projects: %w", err)
	}

	// 2. Total pipelines
	if err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM pipelines`).Scan(&metrics.TotalPipelines); err != nil {
		return nil, fmt.Errorf("failed to count pipelines: %w", err)
	}

	// 3. Status breakdown from pipeline_runs
	rows, err := r.db.QueryContext(ctx, `SELECT status, count(*) FROM pipeline_runs GROUP BY status`)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate run status: %w", err)
	}
	defer rows.Close()

	totalRuns := 0
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		metrics.StatusCounts[status] = count
		totalRuns += count

		switch status {
		case "Succeeded":
			metrics.SucceededRuns = count
		case "Failed":
			metrics.FailedRuns = count
		case "Running":
			metrics.RunningRuns = count
		case "Cancelled":
			metrics.CancelledRuns = count
		}
	}
	metrics.TotalRuns = totalRuns

	// Calculate success rate
	completedRuns := metrics.SucceededRuns + metrics.FailedRuns + metrics.CancelledRuns
	if completedRuns > 0 {
		metrics.SuccessRate = float64(metrics.SucceededRuns) / float64(completedRuns) * 100.0
	}

	// 4. Average duration in seconds (computed in Go for multi-database cross-compatibility)
	durationRows, err := r.db.QueryContext(ctx, "SELECT started_at, finished_at FROM pipeline_runs WHERE finished_at IS NOT NULL AND started_at IS NOT NULL")
	if err == nil {
		var totalSec float64
		var count int
		for durationRows.Next() {
			var s, f time.Time
			if err := durationRows.Scan(&s, &f); err == nil && !f.Before(s) {
				totalSec += f.Sub(s).Seconds()
				count++
			}
		}
		durationRows.Close()
		if count > 0 {
			metrics.AvgDurationSeconds = totalSec / float64(count)
		}
	}

	// 5. Recent runs with project and pipeline names
	recentQuery := `
		SELECT r.id, r.pipeline_id, coalesce(p.name, ''), coalesce(pr.name, ''), 
		       r.status, r.triggered_by, r.started_at, r.finished_at, r.created_at
		FROM pipeline_runs r
		LEFT JOIN pipelines p ON r.pipeline_id = p.id
		LEFT JOIN projects pr ON p.project_id = pr.id
		ORDER BY r.created_at DESC
		LIMIT 10`
	recentRows, err := r.db.QueryContext(ctx, recentQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent runs: %w", err)
	}
	defer recentRows.Close()

	for recentRows.Next() {
		var item DashboardRecentRun
		var started, finished sql.NullTime
		if err := recentRows.Scan(
			&item.ID, &item.PipelineID, &item.PipelineName, &item.ProjectName,
			&item.Status, &item.TriggeredBy, &started, &finished, &item.CreatedAt,
		); err != nil {
			return nil, err
		}

		if started.Valid {
			item.StartedAt = &started.Time
		}
		if finished.Valid {
			item.FinishedAt = &finished.Time
			if started.Valid {
				item.DurationSec = int64(finished.Time.Sub(started.Time).Seconds())
			}
		}

		metrics.RecentRuns = append(metrics.RecentRuns, item)
	}

	return metrics, nil
}
