-- +goose Up
CREATE TABLE IF NOT EXISTS k8s_labeled_roles (
    name VARCHAR(128) PRIMARY KEY,
    display_name VARCHAR(255) NOT NULL,
    role_type VARCHAR(32) NOT NULL DEFAULT 'worker',
    description TEXT NOT NULL DEFAULT '',
    label_key VARCHAR(128) NOT NULL DEFAULT 'tekton-devops-dashboard',
    label_value VARCHAR(64) NOT NULL DEFAULT 'true',
    permissions_json TEXT NOT NULL DEFAULT '{}',
    is_builtin INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS k8s_triggers_resources (
    id VARCHAR(64) PRIMARY KEY,
    kind VARCHAR(64) NOT NULL,
    api_version VARCHAR(64) NOT NULL DEFAULT 'triggers.tekton.dev/v1beta1',
    name VARCHAR(128) NOT NULL,
    namespace VARCHAR(128) NOT NULL DEFAULT 'default',
    spec_json TEXT NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (kind, namespace, name)
);

-- +goose Down
DROP TABLE IF EXISTS k8s_triggers_resources;
DROP TABLE IF EXISTS k8s_labeled_roles;
