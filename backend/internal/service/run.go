package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"devops-platform/internal/compiler"
	"devops-platform/internal/executor"
	"devops-platform/internal/repository"
)

// RunService coordinates triggering, tracking and logs streaming for pipeline runs
type RunService struct {
	runRepo      repository.PipelineRunRepository
	pipelineRepo repository.PipelineRepository
	auditRepo    repository.AuditRepository
	exec         executor.Executor
	logger       *zap.Logger
}

// NewRunService creates a new RunService
func NewRunService(runRepo repository.PipelineRunRepository, pipelineRepo repository.PipelineRepository, auditRepo repository.AuditRepository, exec executor.Executor, logger *zap.Logger) *RunService {
	return &RunService{
		runRepo:      runRepo,
		pipelineRepo: pipelineRepo,
		auditRepo:    auditRepo,
		exec:         exec,
		logger:       logger,
	}
}

// TriggerRun triggers a new pipeline execution
func (s *RunService) TriggerRun(ctx context.Context, userID, ip, pipelineID, triggeredBy string) (*repository.PipelineRun, error) {
	pipeline, err := s.pipelineRepo.GetByID(ctx, pipelineID)
	if err != nil {
		return nil, err
	}

	spec, err := compiler.Parse(pipeline.DSLContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pipeline DSL: %w", err)
	}

	run := &repository.PipelineRun{
		ID:          uuid.New().String(),
		PipelineID:  pipelineID,
		Namespace:   "default",
		Status:      "Pending",
		TriggeredBy: triggeredBy,
		CreatedAt:   time.Now().UTC(),
	}

	// Submit to executor
	k8sName, err := s.exec.Submit(ctx, run, spec)
	if err != nil {
		return nil, fmt.Errorf("failed to submit run to executor: %w", err)
	}
	run.K8sPipelineRunName = k8sName

	if err := s.runRepo.Create(ctx, run); err != nil {
		return nil, fmt.Errorf("failed to persist pipeline run: %w", err)
	}

	_ = s.auditRepo.Record(ctx, &repository.AuditLog{
		UserID:       userID,
		Action:       "TRIGGER_PIPELINE",
		ResourceType: "pipeline_run",
		ResourceID:   run.ID,
		Detail:       fmt.Sprintf("Triggered run %s for pipeline '%s'", run.ID[:8], pipeline.Name),
		IP:           ip,
	})

	return run, nil
}

// GetRun retrieves a run by ID
func (s *RunService) GetRun(ctx context.Context, id string) (*repository.PipelineRun, error) {
	return s.runRepo.GetByID(ctx, id)
}

// ListRunsByPipeline lists runs for a pipeline
func (s *RunService) ListRunsByPipeline(ctx context.Context, pipelineID string, page, pageSize int) ([]repository.PipelineRun, int, error) {
	return s.runRepo.ListByPipeline(ctx, pipelineID, page, pageSize)
}

// ListRunsByProject lists runs across all pipelines in a project
func (s *RunService) ListRunsByProject(ctx context.Context, projectID string, page, pageSize int) ([]repository.PipelineRun, int, error) {
	return s.runRepo.ListByProject(ctx, projectID, page, pageSize)
}

// CancelRun cancels a running pipeline run
func (s *RunService) CancelRun(ctx context.Context, userID, ip, id string) error {
	run, err := s.runRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if run.Status == "Succeeded" || run.Status == "Failed" || run.Status == "Cancelled" {
		return fmt.Errorf("cannot cancel run in status %s", run.Status)
	}

	if err := s.exec.Cancel(ctx, run.Namespace, run.K8sPipelineRunName); err != nil {
		return fmt.Errorf("failed to cancel run in executor: %w", err)
	}

	finished := time.Now().UTC()
	if err := s.runRepo.UpdateStatus(ctx, id, "Cancelled", nil, &finished); err != nil {
		return fmt.Errorf("failed to update run status: %w", err)
	}

	_ = s.auditRepo.Record(ctx, &repository.AuditLog{
		UserID:       userID,
		Action:       "CANCEL_PIPELINE",
		ResourceType: "pipeline_run",
		ResourceID:   run.ID,
		Detail:       fmt.Sprintf("Cancelled run %s", run.ID[:8]),
		IP:           ip,
	})

	return nil
}

// StreamLogs returns an event channel of logs for SSE streaming
func (s *RunService) StreamLogs(ctx context.Context, id string) (<-chan string, error) {
	run, err := s.runRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.exec.GetLogsStream(ctx, run.Namespace, run.K8sPipelineRunName)
}
