-- name: CountOrdersByStatusForDate :many
SELECT status, count(*)::bigint AS total
FROM orders
WHERE depot_id = $1 AND scheduled_date = sqlc.arg('day')::date
GROUP BY status;

-- name: DeliveredTotals :one
SELECT coalesce(sum(total), 0)::bigint AS revenue,
       coalesce(sum(refill_qty), 0)::bigint AS gallons,
       count(*)::bigint AS orders,
       count(DISTINCT customer_id)::bigint AS active_customers
FROM orders
WHERE depot_id = $1 AND status = 'delivered'
  AND delivered_at >= sqlc.arg('from_at')::timestamptz AND delivered_at < sqlc.arg('to_at')::timestamptz;

-- name: ExpectedDemandToday :one
SELECT coalesce(sum(c.usual_qty), 0)::bigint
FROM customers c
WHERE c.depot_id = $1 AND c.is_active AND c.is_verified
  AND c.predicted_empty_at IS NOT NULL AND c.predicted_empty_at < sqlc.arg('before')::timestamptz
  AND NOT EXISTS (SELECT 1 FROM orders o WHERE o.customer_id = c.id AND o.status IN ('pending', 'confirmed', 'on_delivery'));

-- name: CountOrdersBySource :many
SELECT source, count(*)::bigint AS total
FROM orders
WHERE depot_id = $1 AND created_at >= sqlc.arg('from_at')::timestamptz AND created_at < sqlc.arg('to_at')::timestamptz
GROUP BY source;

-- name: CountNewCustomers :one
SELECT count(*)::bigint FROM customers
WHERE depot_id = $1 AND created_at >= sqlc.arg('from_at')::timestamptz AND created_at < sqlc.arg('to_at')::timestamptz;

-- name: ListDeliveredOrdersForExport :many
SELECT o.code, o.delivered_at, o.delivery_name, o.delivery_phone, o.refill_qty, o.free_qty, o.total, o.payment_method, o.payment_status, o.source, u.name AS courier_name
FROM orders o
LEFT JOIN users u ON u.id = o.courier_id
WHERE o.depot_id = $1 AND o.status = 'delivered'
  AND o.delivered_at >= sqlc.arg('from_at')::timestamptz AND o.delivered_at < sqlc.arg('to_at')::timestamptz
ORDER BY o.delivered_at;

-- name: ListDailyRevenue :many
SELECT (delivered_at AT TIME ZONE sqlc.arg('tz')::text)::date AS day,
       coalesce(sum(total), 0)::bigint AS revenue,
       coalesce(sum(refill_qty), 0)::bigint AS gallons
FROM orders
WHERE depot_id = $1 AND status = 'delivered'
  AND delivered_at >= sqlc.arg('from_at')::timestamptz AND delivered_at < sqlc.arg('to_at')::timestamptz
GROUP BY 1
ORDER BY 1;
