package repository

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
	_ "modernc.org/sqlite" // pure Go SQLite driver

	platformdb "devops-platform/db"
)

// InitDB initializes the SQLite database, configures pragmas and runs migrations.
func InitDB(dbPath string, logger *zap.Logger) (*sql.DB, error) {
	// Ensure parent directory exists (unless in-memory)
	if dbPath != ":memory:" {
		dir := filepath.Dir(dbPath)
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create db directory %s: %w", dir, err)
			}
		}
	}

	logger.Info("opening sqlite database", zap.String("path", dbPath))

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Ping database to establish connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping sqlite database: %w", err)
	}

	// Configure PRAGMAs for concurrency & reliability
	pragmas := []string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA foreign_keys = ON;",
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA synchronous = NORMAL;",
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			logger.Warn("failed to set pragma", zap.String("pragma", pragma), zap.Error(err))
		}
	}

	// Single writer connection pool for SQLite to prevent locking contention
	db.SetMaxOpenConns(1)

	// Run embedded migrations via goose
	logger.Info("running database migrations...")
	goose.SetBaseFS(platformdb.MigrationsFS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set goose dialect: %w", err)
	}

	// Mute standard goose logs unless in debug, or adapt to zap
	goose.SetLogger(goose.NopLogger())

	if err := goose.Up(db, "migrations"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run database migrations: %w", err)
	}

	logger.Info("database migrations applied successfully")
	return db, nil
}
