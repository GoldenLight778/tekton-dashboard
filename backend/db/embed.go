package db

import "embed"

// MigrationsFS embeds all SQL migration files for auto-migration
//
//go:embed migrations/*.sql
var MigrationsFS embed.FS
