-- name: InsertOrderEvent :exec
INSERT INTO order_events (id, depot_id, order_id, type, actor_type, actor_id, idempotency_key, meta)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: ListOrderEvents :many
SELECT * FROM order_events WHERE order_id = $1 AND depot_id = $2 ORDER BY created_at, id;

-- name: OrderEventExists :one
SELECT EXISTS (SELECT 1 FROM order_events WHERE order_id = $1 AND idempotency_key = $2);
