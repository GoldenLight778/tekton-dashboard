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
	ErrInvalidProjectName = errors.New("project name must be between 2 and 50 characters")
)

// ProjectService manages projects and project-related operations
type ProjectService struct {
	projectRepo repository.ProjectRepository
	auditRepo   repository.AuditRepository
	logger      *zap.Logger
}

// NewProjectService creates a new ProjectService
func NewProjectService(projectRepo repository.ProjectRepository, auditRepo repository.AuditRepository, logger *zap.Logger) *ProjectService {
	return &ProjectService{
		projectRepo: projectRepo,
		auditRepo:   auditRepo,
		logger:      logger,
	}
}

// CreateProject creates a new project and records an audit log
func (s *ProjectService) CreateProject(ctx context.Context, userID, ip, name, description string) (*repository.Project, error) {
	name = strings.TrimSpace(name)
	if len(name) < 2 || len(name) > 50 {
		return nil, ErrInvalidProjectName
	}

	// Check if already exists
	if _, err := s.projectRepo.GetByName(ctx, name); err == nil {
		return nil, repository.ErrProjectAlreadyExists
	}

	project := &repository.Project{
		ID:          uuid.New().String(),
		Name:        name,
		Description: strings.TrimSpace(description),
		CreatedBy:   userID,
	}

	if err := s.projectRepo.Create(ctx, project); err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	// Record audit log
	_ = s.auditRepo.Record(ctx, &repository.AuditLog{
		UserID:       userID,
		Action:       "CREATE_PROJECT",
		ResourceType: "project",
		ResourceID:   project.ID,
		Detail:       fmt.Sprintf("Created project '%s'", project.Name),
		IP:           ip,
	})

	return project, nil
}

// GetProject retrieves a project by ID
func (s *ProjectService) GetProject(ctx context.Context, id string) (*repository.Project, error) {
	return s.projectRepo.GetByID(ctx, id)
}

// ListProjects lists projects with pagination
func (s *ProjectService) ListProjects(ctx context.Context, page, pageSize int) ([]repository.Project, int, error) {
	return s.projectRepo.List(ctx, page, pageSize)
}

// UpdateProject updates an existing project
func (s *ProjectService) UpdateProject(ctx context.Context, userID, ip, id, name, description string) (*repository.Project, error) {
	name = strings.TrimSpace(name)
	if len(name) < 2 || len(name) > 50 {
		return nil, ErrInvalidProjectName
	}

	existing, err := s.projectRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	existing.Name = name
	existing.Description = strings.TrimSpace(description)

	if err := s.projectRepo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	_ = s.auditRepo.Record(ctx, &repository.AuditLog{
		UserID:       userID,
		Action:       "UPDATE_PROJECT",
		ResourceType: "project",
		ResourceID:   id,
		Detail:       fmt.Sprintf("Updated project '%s'", existing.Name),
		IP:           ip,
	})

	return existing, nil
}

// DeleteProject deletes a project by ID
func (s *ProjectService) DeleteProject(ctx context.Context, userID, ip, id string) error {
	existing, err := s.projectRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.projectRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	_ = s.auditRepo.Record(ctx, &repository.AuditLog{
		UserID:       userID,
		Action:       "DELETE_PROJECT",
		ResourceType: "project",
		ResourceID:   id,
		Detail:       fmt.Sprintf("Deleted project '%s'", existing.Name),
		IP:           ip,
	})

	return nil
}
