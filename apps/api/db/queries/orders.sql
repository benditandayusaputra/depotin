-- name: NextOrderNumber :one
INSERT INTO order_counters (depot_id, day, last_no) VALUES ($1, $2, 1)
ON CONFLICT (depot_id, day) DO UPDATE SET last_no = order_counters.last_no + 1
RETURNING last_no;

-- name: CreateOrder :one
INSERT INTO orders (
  id, depot_id, customer_id, code, source, status, fulfilment, scheduled_date,
  delivery_name, delivery_phone, delivery_address, delivery_note, note,
  refill_qty, free_qty, subtotal, delivery_fee, discount, total,
  courier_id, reminder_id, track_token_hash, idempotency_key, created_by, created_at, confirmed_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8,
  $9, $10, $11, $12, $13,
  $14, $15, $16, $17, $18, $19,
  $20, $21, $22, $23, $24, $25, $26
)
RETURNING *;

-- name: GetOrder :one
SELECT * FROM orders WHERE id = $1 AND depot_id = $2;

-- name: GetOrderForUpdate :one
SELECT * FROM orders WHERE id = $1 AND depot_id = $2 FOR UPDATE;

-- name: GetOrderByIdempotencyKey :one
SELECT * FROM orders WHERE depot_id = $1 AND idempotency_key = $2;

-- name: GetOrderByTrackTokenHash :one
SELECT * FROM orders WHERE track_token_hash = $1;

-- name: ListOrders :many
SELECT * FROM orders
WHERE depot_id = $1
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
  AND (sqlc.narg('scheduled_date')::date IS NULL OR scheduled_date = sqlc.narg('scheduled_date')::date)
  AND (sqlc.narg('courier_id')::uuid IS NULL OR courier_id = sqlc.narg('courier_id')::uuid)
  AND (sqlc.narg('before_at')::timestamptz IS NULL OR (created_at, id) < (sqlc.narg('before_at')::timestamptz, sqlc.narg('before_id')::uuid))
ORDER BY created_at DESC, id DESC
LIMIT $2;

-- name: ListActiveOrders :many
SELECT * FROM orders
WHERE depot_id = $1 AND status IN ('pending', 'confirmed', 'on_delivery')
ORDER BY created_at DESC, id DESC
LIMIT $2;

-- name: ListOrdersByCustomer :many
SELECT * FROM orders
WHERE customer_id = $1 AND depot_id = $2
  AND (sqlc.narg('before_at')::timestamptz IS NULL OR (created_at, id) < (sqlc.narg('before_at')::timestamptz, sqlc.narg('before_id')::uuid))
ORDER BY created_at DESC, id DESC
LIMIT $3;

-- name: ListActiveOrdersByCustomer :many
SELECT * FROM orders
WHERE customer_id = $1 AND status IN ('pending', 'confirmed', 'on_delivery')
ORDER BY created_at DESC;

-- name: CountActiveOrdersByCustomer :one
SELECT count(*) FROM orders WHERE customer_id = $1 AND status IN ('pending', 'confirmed', 'on_delivery');

-- name: CountPendingOrdersByPhone :one
SELECT count(*) FROM orders WHERE depot_id = $1 AND delivery_phone = $2 AND status = 'pending';

-- name: HasActiveFreeOrder :one
SELECT EXISTS (
  SELECT 1 FROM orders WHERE customer_id = $1 AND free_qty > 0 AND status IN ('pending', 'confirmed', 'on_delivery')
);

-- name: ListRecentDeliveries :many
SELECT delivered_at, refill_qty FROM orders
WHERE customer_id = $1 AND status = 'delivered' AND refill_qty > 0 AND delivered_at IS NOT NULL
ORDER BY delivered_at DESC
LIMIT $2;

-- name: ConfirmOrder :one
UPDATE orders SET status = 'confirmed', confirmed_at = $3
WHERE id = $1 AND depot_id = $2 AND status = 'pending'
RETURNING *;

-- name: AssignCourier :one
UPDATE orders SET courier_id = $3
WHERE id = $1 AND depot_id = $2 AND status IN ('confirmed', 'on_delivery')
RETURNING *;

-- name: DispatchOrder :one
UPDATE orders SET status = 'on_delivery', dispatched_at = $3, courier_id = COALESCE(courier_id, sqlc.narg('courier_id')::uuid)
WHERE id = $1 AND depot_id = $2 AND status = 'confirmed'
RETURNING *;

-- name: DeliverOrder :one
UPDATE orders
SET status = 'delivered', delivered_at = $3, gallons_returned = $4,
    payment_method = $5, payment_status = $6, paid_at = CASE WHEN $6 = 'paid' THEN $3 ELSE paid_at END
WHERE id = $1 AND depot_id = $2 AND status = sqlc.arg('from_status')::text
RETURNING *;

-- name: CancelOrder :one
UPDATE orders SET status = 'cancelled', cancelled_at = $3, cancel_reason = $4
WHERE id = $1 AND depot_id = $2 AND status IN ('pending', 'confirmed', 'on_delivery')
RETURNING *;

-- name: CancelPendingOrderByCustomer :one
UPDATE orders SET status = 'cancelled', cancelled_at = $2, cancel_reason = 'Dibatalkan pelanggan'
WHERE id = $1 AND status = 'pending'
RETURNING *;

-- name: MarkOrderPaid :one
UPDATE orders SET payment_status = 'paid', paid_at = $3, payment_method = COALESCE(sqlc.narg('payment_method'), payment_method, 'cash')
WHERE id = $1 AND depot_id = $2 AND payment_status = 'unpaid'
RETURNING *;
