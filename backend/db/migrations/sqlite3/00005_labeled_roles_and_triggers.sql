-- +goose Up
CREATE TABLE IF NOT EXISTS k8s_labeled_roles (
    name TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    role_type TEXT NOT NULL DEFAULT 'worker', -- admin, worker, readonly
    description TEXT NOT NULL DEFAULT '',
    label_key TEXT NOT NULL DEFAULT 'tekton-devops-dashboard',
    label_value TEXT NOT NULL DEFAULT 'true',
    permissions_json TEXT NOT NULL DEFAULT '{}',
    is_builtin INTEGER NOT NULL DEFAULT 0,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS k8s_triggers_resources (
    id TEXT PRIMARY KEY,
    kind TEXT NOT NULL, -- eventlisteners, triggers, triggerbindings, clustertriggerbindings, triggertemplates, interceptors, clusterinterceptors
    api_version TEXT NOT NULL DEFAULT 'triggers.tekton.dev/v1beta1',
    name TEXT NOT NULL,
    namespace TEXT NOT NULL DEFAULT 'default',
    spec_json TEXT NOT NULL DEFAULT '{}',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(kind, namespace, name)
);

CREATE INDEX IF NOT EXISTS idx_triggers_kind_ns ON k8s_triggers_resources(kind, namespace);

-- +goose Down
DROP TABLE IF EXISTS k8s_triggers_resources;
DROP TABLE IF EXISTS k8s_labeled_roles;
