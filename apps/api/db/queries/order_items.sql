-- name: InsertOrderItem :exec
INSERT INTO order_items (id, order_id, product_id, product_name, product_kind, unit_price, qty, line_total)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: ListOrderItems :many
SELECT * FROM order_items WHERE order_id = ANY($1::uuid[]) ORDER BY order_id, product_name;
