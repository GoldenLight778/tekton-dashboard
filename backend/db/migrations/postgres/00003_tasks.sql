-- +goose Up
CREATE TABLE IF NOT EXISTS tasks (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    namespace VARCHAR(128) NOT NULL DEFAULT 'default',
    description TEXT NOT NULL DEFAULT '',
    image VARCHAR(255) NOT NULL DEFAULT 'alpine:latest',
    script TEXT NOT NULL DEFAULT '',
    spec_json TEXT NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (namespace, name)
);

CREATE INDEX IF NOT EXISTS idx_tasks_namespace ON tasks(namespace);

-- +goose Down
DROP TABLE IF EXISTS tasks;
