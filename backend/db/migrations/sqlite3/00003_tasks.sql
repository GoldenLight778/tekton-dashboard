-- +goose Up
CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    namespace TEXT NOT NULL DEFAULT 'default',
    description TEXT NOT NULL DEFAULT '',
    image TEXT NOT NULL DEFAULT 'alpine:latest',
    script TEXT NOT NULL DEFAULT '',
    spec_json TEXT NOT NULL DEFAULT '{}',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(namespace, name)
);

CREATE INDEX IF NOT EXISTS idx_tasks_namespace ON tasks(namespace);

-- +goose Down
DROP TABLE IF EXISTS tasks;
