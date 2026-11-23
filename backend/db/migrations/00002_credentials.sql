-- +goose Up
CREATE TABLE IF NOT EXISTS credentials (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    type TEXT NOT NULL, -- git_token, ssh_key, kubeconfig, secret_text
    encrypted_data TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id, name)
);

CREATE INDEX IF NOT EXISTS idx_credentials_project ON credentials(project_id);

-- +goose Down
DROP TABLE IF EXISTS credentials;
