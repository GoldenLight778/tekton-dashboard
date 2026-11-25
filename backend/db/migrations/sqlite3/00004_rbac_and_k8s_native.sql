-- +goose Up
CREATE TABLE IF NOT EXISTS k8s_namespaces (
    name TEXT PRIMARY KEY,
    alias TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'Active',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT OR IGNORE INTO k8s_namespaces (name, alias, description, status)
VALUES 
    ('default', '默认空间 (default)', 'Kubernetes 默认工作命名空间，存放日常构建任务与流水线', 'Active'),
    ('tekton-pipelines', 'Tekton 核心系统', 'Tekton Pipelines 控制器与系统级公共 Task 资源空间', 'Active');

CREATE TABLE IF NOT EXISTS k8s_role_bindings (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL,
    namespace TEXT NOT NULL DEFAULT '*',
    role_name TEXT NOT NULL DEFAULT 'tekton-dashboard-editor',
    k8s_group TEXT NOT NULL DEFAULT 'tekton-editors',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(username, namespace)
);

CREATE TABLE IF NOT EXISTS k8s_secrets (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    namespace TEXT NOT NULL DEFAULT 'default',
    secret_type TEXT NOT NULL DEFAULT 'kubernetes.io/basic-auth',
    annotation_type TEXT NOT NULL DEFAULT 'tekton.dev/git-0',
    target_url TEXT NOT NULL DEFAULT 'https://github.com',
    username TEXT NOT NULL DEFAULT '',
    service_account TEXT NOT NULL DEFAULT 'default',
    auto_bind_sa INTEGER NOT NULL DEFAULT 1,
    masked_value TEXT NOT NULL DEFAULT '******',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(namespace, name)
);

CREATE INDEX IF NOT EXISTS idx_k8s_secrets_ns ON k8s_secrets(namespace);
CREATE INDEX IF NOT EXISTS idx_k8s_bindings_user ON k8s_role_bindings(username);

-- +goose Down
DROP TABLE IF EXISTS k8s_secrets;
DROP TABLE IF EXISTS k8s_role_bindings;
DROP TABLE IF EXISTS k8s_namespaces;
