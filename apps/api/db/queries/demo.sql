-- name: DeleteDepotAuditLogs :exec
DELETE FROM audit_logs WHERE depot_id = $1;

-- name: DeleteDepotLedger :exec
DELETE FROM gallon_ledger WHERE depot_id = $1;

-- name: DeleteDepotOrderEvents :exec
DELETE FROM order_events WHERE depot_id = $1;

-- name: DeleteDepotOrderItems :exec
DELETE FROM order_items WHERE order_id IN (SELECT id FROM orders WHERE depot_id = $1);

-- name: ClearDepotReminderOrders :exec
UPDATE reminders SET order_id = NULL WHERE depot_id = $1;

-- name: DeleteDepotOrders :exec
DELETE FROM orders WHERE depot_id = $1;

-- name: DeleteDepotReminders :exec
DELETE FROM reminders WHERE depot_id = $1;

-- name: DeleteDepotOrderCounters :exec
DELETE FROM order_counters WHERE depot_id = $1;

-- name: DeleteDepotCustomers :exec
DELETE FROM customers WHERE depot_id = $1;

-- name: DeleteDepotProducts :exec
DELETE FROM products WHERE depot_id = $1;

-- name: DeleteDepotSessions :exec
DELETE FROM sessions WHERE user_id IN (SELECT id FROM users WHERE depot_id = $1);

-- name: DeleteDepotUsers :exec
DELETE FROM users WHERE depot_id = $1;

-- name: DeleteDepot :exec
DELETE FROM depots WHERE id = $1;

-- name: SetDepotDemo :exec
UPDATE depots SET is_demo = true, lat = $2, lng = $3, address = $4, loyalty_every = $5, updated_at = now() WHERE id = $1;
