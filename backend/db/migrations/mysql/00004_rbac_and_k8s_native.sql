-- +goose Up
CREATE TABLE IF NOT EXISTS k8s_namespaces (
    name VARCHAR(128) PRIMARY KEY,
    alias VARCHAR(255) NOT NULL DEFAULT '',
    description TEXT,
    status VARCHAR(32) NOT NULL DEFAULT 'Active',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS k8s_role_bindings (
    id VARCHAR(64) PRIMARY KEY,
    username VARCHAR(128) NOT NULL,
    namespace VARCHAR(128) NOT NULL DEFAULT '*',
    role_name VARCHAR(128) NOT NULL DEFAULT 'tekton-dashboard-editor',
    k8s_group VARCHAR(128) NOT NULL DEFAULT 'tekton-editors',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_user_ns (username, namespace)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS k8s_secrets (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    namespace VARCHAR(128) NOT NULL DEFAULT 'default',
    secret_type VARCHAR(128) NOT NULL DEFAULT 'kubernetes.io/basic-auth',
    annotation_type VARCHAR(128) NOT NULL DEFAULT 'tekton.dev/git-0',
    target_url VARCHAR(255) NOT NULL DEFAULT 'https://github.com',
    username VARCHAR(128) NOT NULL DEFAULT '',
    service_account VARCHAR(128) NOT NULL DEFAULT 'default',
    auto_bind_sa TINYINT(1) NOT NULL DEFAULT 1,
    masked_value VARCHAR(128) NOT NULL DEFAULT '******',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_ns_secret (namespace, name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS k8s_secrets;
DROP TABLE IF EXISTS k8s_role_bindings;
DROP TABLE IF EXISTS k8s_namespaces;
