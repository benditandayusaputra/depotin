-- name: InsertAuditLog :exec
INSERT INTO audit_logs (id, depot_id, user_id, action, entity_type, entity_id, meta, ip)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: ListAuditLogs :many
SELECT * FROM audit_logs
WHERE depot_id = $1
  AND (sqlc.narg('before_at')::timestamptz IS NULL OR (created_at, id) < (sqlc.narg('before_at')::timestamptz, sqlc.narg('before_id')::uuid))
ORDER BY created_at DESC, id DESC
LIMIT $2;
