-- +goose Up
DELETE FROM users WHERE username IN ('developer', 'viewer');
DELETE FROM k8s_role_bindings WHERE username IN ('developer', 'viewer');

UPDATE k8s_labeled_roles SET description = '全局最高管理权限' WHERE name = 'tekton-devops-admin';
UPDATE k8s_labeled_roles SET description = '命名空间内资源读写与执行权限' WHERE name = 'tekton-devops-worker';
UPDATE k8s_labeled_roles SET description = '命名空间内只读查看权限' WHERE name = 'tekton-devops-readonly';

-- +goose Down
SELECT 1;
