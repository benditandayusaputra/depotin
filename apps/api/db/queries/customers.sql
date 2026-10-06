-- name: CreateCustomer :one
INSERT INTO customers (id, depot_id, name, phone, address, address_note, area, lat, lng, source, is_verified, usual_qty)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: GetCustomer :one
SELECT * FROM customers WHERE id = $1 AND depot_id = $2;

-- name: GetCustomerForUpdate :one
SELECT * FROM customers WHERE id = $1 AND depot_id = $2 FOR UPDATE;

-- name: GetCustomerByPhone :one
SELECT * FROM customers WHERE depot_id = $1 AND phone = $2;

-- name: GetCustomerByTokenHash :one
SELECT * FROM customers WHERE token_hash = $1 AND is_active;

-- name: UpdateCustomer :one
UPDATE customers
SET name = COALESCE(sqlc.narg('name'), name),
    phone = COALESCE(sqlc.narg('phone'), phone),
    address = COALESCE(sqlc.narg('address'), address),
    address_note = COALESCE(sqlc.narg('address_note'), address_note),
    area = COALESCE(sqlc.narg('area'), area),
    lat = COALESCE(sqlc.narg('lat'), lat),
    lng = COALESCE(sqlc.narg('lng'), lng),
    usual_qty = COALESCE(sqlc.narg('usual_qty'), usual_qty),
    is_active = COALESCE(sqlc.narg('is_active'), is_active),
    updated_at = now()
WHERE id = $1 AND depot_id = $2
RETURNING *;

-- name: SetCustomerLocation :execrows
UPDATE customers SET lat = $3, lng = $4, updated_at = now() WHERE id = $1 AND depot_id = $2;

-- name: SetCustomerToken :execrows
UPDATE customers SET token_hash = $3, token_enc = $4, token_rotated_at = $5, updated_at = now()
WHERE id = $1 AND depot_id = $2;

-- name: SnoozeCustomer :execrows
UPDATE customers SET reminder_snoozed_until = $3, updated_at = now() WHERE id = $1 AND depot_id = $2;

-- name: SearchCustomers :many
SELECT * FROM customers
WHERE depot_id = $1
  AND (name ILIKE '%' || sqlc.arg('q')::text || '%'
       OR phone ILIKE '%' || sqlc.arg('q')::text || '%'
       OR address ILIKE '%' || sqlc.arg('q')::text || '%')
ORDER BY is_active DESC, name
LIMIT $2;

-- name: ListCustomers :many
SELECT * FROM customers
WHERE depot_id = $1
  AND (sqlc.narg('before_at')::timestamptz IS NULL OR (created_at, id) < (sqlc.narg('before_at')::timestamptz, sqlc.narg('before_id')::uuid))
ORDER BY created_at DESC, id DESC
LIMIT $2;

-- name: ListDueCustomers :many
SELECT customers.* FROM customers
WHERE customers.depot_id = $1 AND is_active AND is_verified
  AND predicted_empty_at IS NOT NULL AND predicted_empty_at <= sqlc.arg('due_before')::timestamptz
  AND NOT EXISTS (SELECT 1 FROM orders o WHERE o.customer_id = customers.id AND o.status IN ('pending', 'confirmed', 'on_delivery'))
ORDER BY predicted_empty_at, id
LIMIT $2;

-- name: ListAtRiskCustomers :many
SELECT customers.* FROM customers
WHERE customers.depot_id = $1 AND is_active AND is_verified
  AND last_delivered_at IS NOT NULL AND days_per_gallon IS NOT NULL AND last_delivered_qty IS NOT NULL
  AND sqlc.arg('now')::timestamptz > last_delivered_at + (2 * last_delivered_qty * days_per_gallon) * interval '1 day'
  AND NOT EXISTS (SELECT 1 FROM orders o WHERE o.customer_id = customers.id AND o.status IN ('pending', 'confirmed', 'on_delivery'))
ORDER BY last_delivered_at, id
LIMIT $2;

-- name: ListLoanCustomers :many
SELECT * FROM customers
WHERE depot_id = $1 AND loan_balance > 0
ORDER BY loan_balance DESC, last_delivered_at NULLS FIRST, id
LIMIT $2;

-- name: ListIdleLoanCustomers :many
SELECT * FROM customers
WHERE depot_id = $1 AND loan_balance > 0
  AND (last_delivered_at IS NULL OR last_delivered_at < sqlc.arg('idle_before')::timestamptz)
ORDER BY loan_balance DESC, last_delivered_at NULLS FIRST, id
LIMIT $2;

-- name: SumLoanBalance :one
SELECT coalesce(sum(loan_balance), 0)::bigint FROM customers WHERE depot_id = $1;

-- name: CountCustomers :one
SELECT count(*) FROM customers WHERE depot_id = $1 AND is_active;
