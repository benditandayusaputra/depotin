-- name: CreateSession :one
INSERT INTO sessions (id, user_id, family_id, token_hash, expires_at, user_agent, ip)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetSessionByTokenHash :one
SELECT * FROM sessions WHERE token_hash = $1;

-- name: MarkSessionRotated :execrows
UPDATE sessions SET rotated_at = $2 WHERE id = $1 AND rotated_at IS NULL AND revoked_at IS NULL;

-- name: RevokeSession :exec
UPDATE sessions SET revoked_at = $2 WHERE id = $1 AND revoked_at IS NULL;

-- name: RevokeSessionFamily :exec
UPDATE sessions SET revoked_at = $2 WHERE family_id = $1 AND revoked_at IS NULL;

-- name: RevokeUserSessions :exec
UPDATE sessions SET revoked_at = $2 WHERE user_id = $1 AND revoked_at IS NULL;

-- name: RevokeUserSessionsExceptFamily :exec
UPDATE sessions SET revoked_at = $3 WHERE user_id = $1 AND family_id <> $2 AND revoked_at IS NULL;

-- name: CountActiveSessions :one
SELECT count(*) FROM sessions WHERE user_id = $1 AND revoked_at IS NULL AND rotated_at IS NULL AND expires_at > $2;
