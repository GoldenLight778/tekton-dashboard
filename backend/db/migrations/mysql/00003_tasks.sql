-- +goose Up
CREATE TABLE IF NOT EXISTS tasks (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    namespace VARCHAR(128) NOT NULL DEFAULT 'default',
    description TEXT,
    image VARCHAR(255) NOT NULL DEFAULT 'alpine:latest',
    script TEXT,
    spec_json TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_namespace_task (namespace, name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_tasks_namespace ON tasks(namespace);

-- +goose Down
DROP TABLE IF EXISTS tasks;
