package repository

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql" // MySQL driver
	_ "github.com/lib/pq"              // PostgreSQL driver
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
	_ "modernc.org/sqlite" // pure Go SQLite driver

	platformdb "devops-platform/db"
)

// DBConfig holds database connection configuration
type DBConfig struct {
	Driver          string        // sqlite, mysql, postgres
	Datasource      string        // connection string or file path
	DSN             string        // alias for Datasource
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// InitDBFromConfig initializes the database (SQLite, MySQL, or PostgreSQL), configures connection pools,
// and runs dialect-appropriate goose migrations.
func InitDBFromConfig(cfg DBConfig, logger *zap.Logger) (*sql.DB, error) {
	driver := strings.ToLower(cfg.Driver)
	if driver == "" || driver == "sqlite3" {
		driver = "sqlite"
	}

	dsn := cfg.Datasource
	if dsn == "" {
		dsn = cfg.DSN
	}

	var sqlDriver string
	var dialect string
	var migrationDir string

	switch driver {
	case "sqlite":
		sqlDriver = "sqlite"
		dialect = "sqlite3"
		migrationDir = "migrations/sqlite3"
		if dsn != ":memory:" && dsn != "" {
			dir := filepath.Dir(dsn)
			if dir != "" && dir != "." {
				_ = os.MkdirAll(dir, 0755)
			}
		}
	case "mysql":
		sqlDriver = "mysql"
		dialect = "mysql"
		migrationDir = "migrations/mysql"
	case "postgres", "postgresql":
		sqlDriver = "postgres"
		dialect = "postgres"
		migrationDir = "migrations/postgres"
	default:
		return nil, fmt.Errorf("unsupported database driver: %s (supported: sqlite, mysql, postgres)", driver)
	}

	logger.Info("opening database",
		zap.String("driver", sqlDriver),
		zap.String("datasource", maskDSN(dsn)),
	)

	db, err := sql.Open(sqlDriver, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database (%s): %w", sqlDriver, err)
	}

	// Configure connection pool
	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	} else if driver == "sqlite" {
		db.SetMaxOpenConns(1) // Single writer for SQLite
	} else {
		db.SetMaxOpenConns(25)
	}

	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	} else {
		db.SetMaxIdleConns(5)
	}

	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	} else {
		db.SetConnMaxLifetime(30 * time.Minute)
	}

	// Ping to verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database (%s): %w", sqlDriver, err)
	}

	// Dialect-specific initial pragmas
	if driver == "sqlite" {
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
	}

	// Run embedded migrations via goose
	logger.Info("running database migrations...", zap.String("dialect", dialect), zap.String("dir", migrationDir))
	goose.SetBaseFS(platformdb.MigrationsFS)
	if err := goose.SetDialect(dialect); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set goose dialect %s: %w", dialect, err)
	}

	goose.SetLogger(goose.NopLogger())

	if err := goose.Up(db, migrationDir); err != nil {
		// Fallback to root "migrations" if directory structure is standard
		if errFallback := goose.Up(db, "migrations"); errFallback != nil {
			db.Close()
			return nil, fmt.Errorf("failed to run database migrations (%s): %w", migrationDir, err)
		}
	}

	logger.Info("database migrations applied successfully", zap.String("dialect", dialect))
	return db, nil
}

// InitDB preserves backwards compatibility, initializing SQLite
func InitDB(dbPath string, logger *zap.Logger) (*sql.DB, error) {
	return InitDBFromConfig(DBConfig{
		Driver:     "sqlite",
		Datasource: dbPath,
		DSN:        dbPath,
	}, logger)
}

func maskDSN(dsn string) string {
	if strings.Contains(dsn, "@") {
		parts := strings.Split(dsn, "@")
		return "***@" + parts[len(parts)-1]
	}
	return dsn
}
