package db

import "embed"

// MigrationsFS embeds all SQL migration files for auto-migration
//
//go:embed migrations/*
var MigrationsFS embed.FS
