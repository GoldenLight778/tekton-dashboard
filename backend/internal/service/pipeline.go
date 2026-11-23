package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"devops-platform/internal/repository"
)

var (
	ErrInvalidPipelineName = errors.New("pipeline name must be between 2 and 50 characters")
	ErrEmptyPipelineDSL    = errors.New("pipeline DSL content cannot be empty")
)

// PipelineService manages pipeline definitions
type PipelineService struct {
	pipelineRepo repository.PipelineRepository
	projectRepo  repository.ProjectRepository
	auditRepo    repository.AuditRepository
	logger       *zap.Logger
}

// NewPipelineService creates a new PipelineService
func NewPipelineService(pipelineRepo repository.PipelineRepository, projectRepo repository.ProjectRepository, auditRepo repository.AuditRepository, logger *zap.Logger) *PipelineService {
	return &PipelineService{
		pipelineRepo: pipelineRepo,
		projectRepo:  projectRepo,
		auditRepo:    auditRepo,
		logger:       logger,
	}
}

// CreatePipeline creates a new pipeline inside a project
func (s *PipelineService) CreatePipeline(ctx context.Context, userID, ip, projectID, name, dslContent string) (*repository.Pipeline, error) {
	name = strings.TrimSpace(name)
	if len(name) < 2 || len(name) > 50 {
		return nil, ErrInvalidPipelineName
	}

	// Verify project exists
	if _, err := s.projectRepo.GetByID(ctx, projectID); err != nil {
		return nil, err
	}

	// Check duplicate name within project
	if _, err := s.pipelineRepo.GetByProjectAndName(ctx, projectID, name); err == nil {
		return nil, repository.ErrPipelineAlreadyExists
	}

	dslContent = strings.TrimSpace(dslContent)
	if dslContent == "" {
		// Provide a default minimal pipeline DSL
		dslContent = fmt.Sprintf("name: %s\nstages:\n  - name: test\n    tasks:\n      - name: echo\n        image: alpine:latest\n        script: echo 'Hello Pipeline'", name)
	}

	pipeline := &repository.Pipeline{
		ID:         uuid.New().String(),
		ProjectID:  projectID,
		Name:       name,
		DSLContent: dslContent,
	}

	if err := s.pipelineRepo.Create(ctx, pipeline); err != nil {
		return nil, fmt.Errorf("failed to create pipeline: %w", err)
	}

	_ = s.auditRepo.Record(ctx, &repository.AuditLog{
		UserID:       userID,
		Action:       "CREATE_PIPELINE",
		ResourceType: "pipeline",
		ResourceID:   pipeline.ID,
		Detail:       fmt.Sprintf("Created pipeline '%s' in project %s", pipeline.Name, projectID),
		IP:           ip,
	})

	return pipeline, nil
}

// GetPipeline retrieves a pipeline by ID
func (s *PipelineService) GetPipeline(ctx context.Context, id string) (*repository.Pipeline, error) {
	return s.pipelineRepo.GetByID(ctx, id)
}

// ListPipelinesByProject lists all pipelines belonging to a project
func (s *PipelineService) ListPipelinesByProject(ctx context.Context, projectID string, page, pageSize int) ([]repository.Pipeline, int, error) {
	// Verify project exists
	if _, err := s.projectRepo.GetByID(ctx, projectID); err != nil {
		return nil, 0, err
	}
	return s.pipelineRepo.ListByProject(ctx, projectID, page, pageSize)
}

// UpdatePipeline updates an existing pipeline's name or DSL
func (s *PipelineService) UpdatePipeline(ctx context.Context, userID, ip, id, name, dslContent string) (*repository.Pipeline, error) {
	existing, err := s.pipelineRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if name != "" {
		name = strings.TrimSpace(name)
		if len(name) < 2 || len(name) > 50 {
			return nil, ErrInvalidPipelineName
		}
		existing.Name = name
	}

	if dslContent != "" {
		existing.DSLContent = strings.TrimSpace(dslContent)
	}

	if err := s.pipelineRepo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("failed to update pipeline: %w", err)
	}

	_ = s.auditRepo.Record(ctx, &repository.AuditLog{
		UserID:       userID,
		Action:       "UPDATE_PIPELINE",
		ResourceType: "pipeline",
		ResourceID:   id,
		Detail:       fmt.Sprintf("Updated pipeline '%s'", existing.Name),
		IP:           ip,
	})

	return existing, nil
}

// DeletePipeline deletes a pipeline by ID
func (s *PipelineService) DeletePipeline(ctx context.Context, userID, ip, id string) error {
	existing, err := s.pipelineRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.pipelineRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete pipeline: %w", err)
	}

	_ = s.auditRepo.Record(ctx, &repository.AuditLog{
		UserID:       userID,
		Action:       "DELETE_PIPELINE",
		ResourceType: "pipeline",
		ResourceID:   id,
		Detail:       fmt.Sprintf("Deleted pipeline '%s'", existing.Name),
		IP:           ip,
	})

	return nil
}
