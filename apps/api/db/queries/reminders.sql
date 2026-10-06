-- name: ReopenReminderForCancelledOrder :exec
UPDATE reminders SET status = 'sent', order_id = NULL WHERE order_id = $1 AND status = 'ordered';

-- name: GetReminder :one
SELECT * FROM reminders WHERE id = $1 AND depot_id = $2;

-- name: MarkReminderOrdered :execrows
UPDATE reminders SET status = 'ordered', order_id = $2
WHERE id = $1 AND customer_id = $3 AND status = 'sent' AND sent_at >= sqlc.arg('sent_after')::timestamptz;

-- name: ExpireStaleReminders :execrows
UPDATE reminders SET status = 'expired'
WHERE depot_id = $1 AND status = 'queued' AND due_date < sqlc.arg('before_date')::date;

-- name: QueueReminders :execrows
INSERT INTO reminders (id, depot_id, customer_id, due_date, predicted_empty_at, status)
SELECT uuidv7(), c.depot_id, c.id, sqlc.arg('due_date')::date, c.predicted_empty_at, 'queued'
FROM customers c
WHERE c.depot_id = $1 AND c.is_active AND c.is_verified
  AND c.predicted_empty_at IS NOT NULL
  AND c.predicted_empty_at < sqlc.arg('due_before')::timestamptz
  AND (c.reminder_snoozed_until IS NULL OR c.reminder_snoozed_until <= sqlc.arg('due_date')::date)
  AND NOT EXISTS (SELECT 1 FROM orders o WHERE o.customer_id = c.id AND o.status IN ('pending', 'confirmed', 'on_delivery'))
  AND NOT EXISTS (SELECT 1 FROM reminders r WHERE r.customer_id = c.id AND r.status IN ('queued', 'sent') AND r.due_date >= sqlc.arg('recent_after')::date)
ON CONFLICT (customer_id, due_date) DO NOTHING;

-- name: ListReminders :many
SELECT sqlc.embed(reminders), sqlc.embed(customers)
FROM reminders
JOIN customers ON customers.id = reminders.customer_id
WHERE reminders.depot_id = $1
  AND reminders.status = ANY(sqlc.arg('statuses')::text[])
  AND reminders.due_date >= sqlc.arg('from_date')::date
ORDER BY reminders.status, reminders.predicted_empty_at, reminders.id
LIMIT $2;

-- name: MarkReminderSent :one
UPDATE reminders SET status = 'sent', sent_at = $3, sent_by = $4
WHERE id = $1 AND depot_id = $2 AND status IN ('queued', 'sent')
RETURNING *;

-- name: SkipReminder :one
UPDATE reminders SET status = 'skipped'
WHERE id = $1 AND depot_id = $2 AND status IN ('queued', 'sent')
RETURNING *;

-- name: CountQueuedReminders :one
SELECT count(*) FROM reminders WHERE depot_id = $1 AND status = 'queued' AND due_date >= sqlc.arg('from_date')::date;

-- name: ReminderStats :one
SELECT
  count(*) FILTER (WHERE status IN ('sent', 'ordered'))::bigint AS sent,
  count(*) FILTER (WHERE status = 'ordered')::bigint AS ordered
FROM reminders
WHERE depot_id = $1 AND sent_at >= sqlc.arg('from_at')::timestamptz AND sent_at < sqlc.arg('to_at')::timestamptz;
