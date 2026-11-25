package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"devops-platform/internal/crypto"
	"devops-platform/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
)

// AuthService handles authentication logic
type AuthService struct {
	userRepo  repository.UserRepository
	jwtSecret string
	logger    *zap.Logger
}

// NewAuthService creates a new AuthService
func NewAuthService(userRepo repository.UserRepository, jwtSecret string, logger *zap.Logger) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
		logger:    logger,
	}
}

// EnsureDefaultAdmin checks if any users exist, and creates a default admin if empty
func (s *AuthService) EnsureDefaultAdmin(ctx context.Context) error {
	count, err := s.userRepo.Count(ctx)
	if err != nil {
		return fmt.Errorf("failed to check existing users: %w", err)
	}

	if count > 0 {
		return nil
	}

	s.logger.Info("no users found, initializing default admin account (username: admin)")

	hash, err := crypto.HashPassword("admin123")
	if err != nil {
		return fmt.Errorf("failed to hash default admin password: %w", err)
	}

	admin := &repository.User{
		ID:           uuid.New().String(),
		Username:     "admin",
		PasswordHash: hash,
		Role:         "admin",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := s.userRepo.Create(ctx, admin); err != nil {
		return fmt.Errorf("failed to create default admin user: %w", err)
	}

	s.logger.Info("default admin created successfully (username: admin, default password: admin123)")
	return nil
}

// LoginResult contains the JWT token and user info
type LoginResult struct {
	Token string           `json:"token"`
	User  *repository.User `json:"user"`
}

// Login verifies credentials and returns a JWT token
func (s *AuthService) Login(ctx context.Context, username, password string) (*LoginResult, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("login failed: %w", err)
	}

	if !crypto.CheckPassword(user.PasswordHash, password) {
		return nil, ErrInvalidCredentials
	}

	// Generate JWT token (valid for 24 hours)
	token, err := crypto.GenerateToken(user.ID, user.Username, user.Role, s.jwtSecret, 24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &LoginResult{
		Token: token,
		User:  user,
	}, nil
}

// GetUserByID retrieves user profile by ID
func (s *AuthService) GetUserByID(ctx context.Context, id string) (*repository.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

// ListUsers returns all registered users
func (s *AuthService) ListUsers(ctx context.Context) ([]repository.User, error) {
	return s.userRepo.List(ctx)
}

// CreateUser registers a new platform user with hashed password and role
func (s *AuthService) CreateUser(ctx context.Context, username, password, role string) (*repository.User, error) {
	if existing, err := s.userRepo.GetByUsername(ctx, username); err == nil && existing != nil {
		return nil, repository.ErrUserAlreadyExists
	}
	hash, err := crypto.HashPassword(password)
	if err != nil {
		return nil, err
	}
	if role == "" {
		role = "editor"
	}
	u := &repository.User{
		ID:           uuid.New().String(),
		Username:     username,
		PasswordHash: hash,
		Role:         role,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if err := s.userRepo.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// UpdateUserRole updates a user's default cluster role
func (s *AuthService) UpdateUserRole(ctx context.Context, id, role string) error {
	return s.userRepo.UpdateRole(ctx, id, role)
}

// DeleteUser removes a user by ID
func (s *AuthService) DeleteUser(ctx context.Context, id string) error {
	return s.userRepo.Delete(ctx, id)
}
