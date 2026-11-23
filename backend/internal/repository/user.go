package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("username already exists")
)

// User represents a system user
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"` // admin, dev, viewer
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UserRepository defines data access methods for users
type UserRepository interface {
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	Count(ctx context.Context) (int, error)
}

// SQLiteUserRepository implements UserRepository on SQLite/MySQL/Postgres
type SQLiteUserRepository struct {
	db     *sql.DB
	driver string
}

// NewUserRepository creates a new SQLiteUserRepository
func NewUserRepository(db *sql.DB, driver ...string) *SQLiteUserRepository {
	d := "sqlite"
	if len(driver) > 0 && driver[0] != "" {
		d = driver[0]
	}
	return &SQLiteUserRepository{db: db, driver: d}
}

func (r *SQLiteUserRepository) Create(ctx context.Context, u *User) error {
	query := `INSERT INTO users (id, username, password_hash, role, created_at, updated_at) 
	          VALUES (?, ?, ?, ?, ?, ?)`
	now := time.Now().UTC()
	if u.CreatedAt.IsZero() {
		u.CreatedAt = now
	}
	u.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, Rebind(r.driver, query), u.ID, u.Username, u.PasswordHash, u.Role, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (r *SQLiteUserRepository) GetByID(ctx context.Context, id string) (*User, error) {
	query := `SELECT id, username, password_hash, role, created_at, updated_at FROM users WHERE id = ?`
	row := r.db.QueryRowContext(ctx, Rebind(r.driver, query), id)

	var u User
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}
	return &u, nil
}

func (r *SQLiteUserRepository) GetByUsername(ctx context.Context, username string) (*User, error) {
	query := `SELECT id, username, password_hash, role, created_at, updated_at FROM users WHERE username = ?`
	row := r.db.QueryRowContext(ctx, Rebind(r.driver, query), username)

	var u User
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}
	return &u, nil
}

func (r *SQLiteUserRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM users`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}
	return count, nil
}
