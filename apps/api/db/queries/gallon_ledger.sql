-- name: InsertLedger :one
INSERT INTO gallon_ledger (id, depot_id, customer_id, order_id, kind, delta, balance_after, note, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: ListLedgerByCustomer :many
SELECT * FROM gallon_ledger
WHERE customer_id = $1 AND depot_id = $2
  AND (sqlc.narg('before_at')::timestamptz IS NULL OR (created_at, id) < (sqlc.narg('before_at')::timestamptz, sqlc.narg('before_id')::uuid))
ORDER BY created_at DESC, id DESC
LIMIT $3;

-- name: SumLedgerByCustomer :one
SELECT coalesce(sum(delta), 0)::bigint FROM gallon_ledger WHERE customer_id = $1;

-- name: SetCustomerLoanBalance :exec
UPDATE customers SET loan_balance = $3, updated_at = now() WHERE id = $1 AND depot_id = $2;

-- name: ApplyDeliveryToCustomer :exec
UPDATE customers
SET loan_balance = $3,
    stamp_count = $4,
    days_per_gallon = $5,
    prediction_samples = $6,
    prediction_confidence = $7,
    predicted_empty_at = $8,
    last_delivered_at = $9,
    last_delivered_qty = $10,
    is_verified = true,
    updated_at = now()
WHERE id = $1 AND depot_id = $2;

-- name: ListCourierQueue :many
SELECT sqlc.embed(orders), customers.lat, customers.lng, customers.area, customers.loan_balance
FROM orders
JOIN customers ON customers.id = orders.customer_id
WHERE orders.courier_id = $1 AND orders.depot_id = $2
  AND orders.status IN ('confirmed', 'on_delivery')
  AND orders.scheduled_date <= sqlc.arg('until_date')::date
ORDER BY orders.created_at;
