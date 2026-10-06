-- name: CreateProduct :one
INSERT INTO products (id, depot_id, name, kind, price, sort_order)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListProducts :many
SELECT * FROM products WHERE depot_id = $1 ORDER BY sort_order, created_at;

-- name: ListActiveProducts :many
SELECT * FROM products WHERE depot_id = $1 AND is_active ORDER BY sort_order, created_at;

-- name: GetProduct :one
SELECT * FROM products WHERE id = $1 AND depot_id = $2;

-- name: GetRefillProduct :one
SELECT * FROM products WHERE depot_id = $1 AND kind = 'refill' AND is_active ORDER BY sort_order, created_at LIMIT 1;

-- name: UpdateProduct :one
UPDATE products
SET name = COALESCE(sqlc.narg('name'), name),
    price = COALESCE(sqlc.narg('price'), price),
    kind = COALESCE(sqlc.narg('kind'), kind),
    is_active = COALESCE(sqlc.narg('is_active'), is_active),
    sort_order = COALESCE(sqlc.narg('sort_order'), sort_order),
    updated_at = now()
WHERE id = $1 AND depot_id = $2
RETURNING *;
