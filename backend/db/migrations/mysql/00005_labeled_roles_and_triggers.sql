-- +goose Up
CREATE TABLE IF NOT EXISTS k8s_labeled_roles (
    name VARCHAR(128) PRIMARY KEY,
    display_name VARCHAR(255) NOT NULL,
    role_type VARCHAR(32) NOT NULL DEFAULT 'worker',
    description TEXT,
    label_key VARCHAR(128) NOT NULL DEFAULT 'tekton-devops-dashboard',
    label_value VARCHAR(64) NOT NULL DEFAULT 'true',
    permissions_json TEXT,
    is_builtin TINYINT(1) NOT NULL DEFAULT 0,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS k8s_triggers_resources (
    id VARCHAR(64) PRIMARY KEY,
    kind VARCHAR(64) NOT NULL,
    api_version VARCHAR(64) NOT NULL DEFAULT 'triggers.tekton.dev/v1beta1',
    name VARCHAR(128) NOT NULL,
    namespace VARCHAR(128) NOT NULL DEFAULT 'default',
    spec_json TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_triggers_kind_ns_name (kind, namespace, name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS k8s_triggers_resources;
DROP TABLE IF EXISTS k8s_labeled_roles;
