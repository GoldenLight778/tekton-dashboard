-- +goose Up
DELETE FROM users WHERE username IN ('developer', 'viewer');
DELETE FROM k8s_role_bindings WHERE username IN ('developer', 'viewer');

-- +goose Down
SELECT 1;
