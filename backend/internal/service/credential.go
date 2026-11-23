package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"devops-platform/internal/crypto"
	"devops-platform/internal/repository"
)

var (
	ErrInvalidCredentialName = errors.New("credential name must be between 2 and 50 characters")
	ErrEmptySecret           = errors.New("credential secret cannot be empty")
	ErrInvalidCredentialType = errors.New("unsupported credential type")
)

var validCredTypes = map[string]bool{
	"git_token":   true,
	"ssh_key":     true,
	"kubeconfig":  true,
	"secret_text": true,
}

// CredentialService manages encrypted project credentials
type CredentialService struct {
	credRepo    repository.CredentialRepository
	projectRepo repository.ProjectRepository
	auditRepo   repository.AuditRepository
	secretKey   []byte
	logger      *zap.Logger
}

// NewCredentialService creates a new CredentialService
func NewCredentialService(
	credRepo repository.CredentialRepository,
	projectRepo repository.ProjectRepository,
	auditRepo repository.AuditRepository,
	secretKey string,
	logger *zap.Logger,
) *CredentialService {
	return &CredentialService{
		credRepo:    credRepo,
		projectRepo: projectRepo,
		auditRepo:   auditRepo,
		secretKey:   []byte(secretKey),
		logger:      logger,
	}
}

// CreateCredential encrypts and stores a new credential
func (s *CredentialService) CreateCredential(ctx context.Context, userID, ip, projectID, name, credType, rawSecret string) (*repository.Credential, error) {
	name = strings.TrimSpace(name)
	if len(name) < 2 || len(name) > 50 {
		return nil, ErrInvalidCredentialName
	}

	credType = strings.TrimSpace(credType)
	if !validCredTypes[credType] {
		return nil, ErrInvalidCredentialType
	}

	rawSecret = strings.TrimSpace(rawSecret)
	if rawSecret == "" {
		return nil, ErrEmptySecret
	}

	// Verify project exists
	if _, err := s.projectRepo.GetByID(ctx, projectID); err != nil {
		return nil, err
	}

	// Check duplicate name
	if _, err := s.credRepo.GetByProjectAndName(ctx, projectID, name); err == nil {
		return nil, repository.ErrCredentialAlreadyExists
	}

	// Encrypt secret with AES-GCM
	encrypted, err := crypto.EncryptAESGCM(rawSecret, s.secretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt credential: %w", err)
	}

	cred := &repository.Credential{
		ID:            uuid.New().String(),
		ProjectID:     projectID,
		Name:          name,
		Type:          credType,
		EncryptedData: encrypted,
		MaskedValue:   "******",
	}

	if err := s.credRepo.Create(ctx, cred); err != nil {
		return nil, fmt.Errorf("failed to store credential: %w", err)
	}

	// Audit log
	_ = s.auditRepo.Record(ctx, &repository.AuditLog{
		UserID:       userID,
		Action:       "create_credential",
		ResourceType: "credential",
		ResourceID:   cred.ID,
		Detail:       fmt.Sprintf(`{"project_id":"%s","name":"%s","type":"%s"}`, projectID, name, credType),
		IP:           ip,
	})

	s.logger.Info("credential created",
		zap.String("id", cred.ID),
		zap.String("project_id", projectID),
		zap.String("name", name),
		zap.String("type", credType),
	)

	return cred, nil
}

// ListByProject returns all credentials for a project with secrets masked
func (s *CredentialService) ListByProject(ctx context.Context, projectID string) ([]repository.Credential, error) {
	if _, err := s.projectRepo.GetByID(ctx, projectID); err != nil {
		return nil, err
	}

	creds, err := s.credRepo.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	for i := range creds {
		creds[i].EncryptedData = ""
		creds[i].MaskedValue = "******"
	}

	return creds, nil
}

// DecryptCredential retrieves and decrypts the secret value (internal platform use)
func (s *CredentialService) DecryptCredential(ctx context.Context, id string) (string, error) {
	cred, err := s.credRepo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}

	return crypto.DecryptAESGCM(cred.EncryptedData, s.secretKey)
}

// DeleteCredential deletes a credential by ID
func (s *CredentialService) DeleteCredential(ctx context.Context, userID, ip, id string) error {
	cred, err := s.credRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.credRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete credential: %w", err)
	}

	_ = s.auditRepo.Record(ctx, &repository.AuditLog{
		UserID:       userID,
		Action:       "delete_credential",
		ResourceType: "credential",
		ResourceID:   id,
		Detail:       fmt.Sprintf(`{"project_id":"%s","name":"%s"}`, cred.ProjectID, cred.Name),
		IP:           ip,
	})

	s.logger.Info("credential deleted",
		zap.String("id", id),
		zap.String("name", cred.Name),
	)

	return nil
}
