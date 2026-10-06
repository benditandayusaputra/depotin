-- name: CreateUser :one
INSERT INTO users (id, depot_id, role, name, phone, password_hash)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetUserByPhone :one
SELECT * FROM users WHERE phone = $1;

-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserInDepot :one
SELECT * FROM users WHERE id = $1 AND depot_id = $2;

-- name: ListUsersByDepot :many
SELECT * FROM users WHERE depot_id = $1 ORDER BY role, created_at;

-- name: ListActiveCouriers :many
SELECT id, name, phone FROM users WHERE depot_id = $1 AND role = 'courier' AND is_active ORDER BY name;

-- name: RecordLoginFailure :one
UPDATE users
SET failed_login_count = failed_login_count + 1,
    locked_until = CASE WHEN failed_login_count + 1 >= sqlc.arg('max_failures')::int THEN sqlc.arg('lock_until')::timestamptz ELSE locked_until END,
    updated_at = now()
WHERE id = $1
RETURNING failed_login_count, locked_until;

-- name: RecordLoginSuccess :exec
UPDATE users
SET failed_login_count = 0, locked_until = NULL, last_login_at = $2, updated_at = now()
WHERE id = $1;

-- name: UpdateUserPassword :exec
UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1;

-- name: UpdateUserProfile :one
UPDATE users
SET name = COALESCE(sqlc.narg('name'), name),
    is_active = COALESCE(sqlc.narg('is_active'), is_active),
    updated_at = now()
WHERE id = $1 AND depot_id = $2
RETURNING *;
