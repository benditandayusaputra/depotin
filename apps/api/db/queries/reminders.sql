-- name: ReopenReminderForCancelledOrder :exec
UPDATE reminders SET status = 'sent', order_id = NULL WHERE order_id = $1 AND status = 'ordered';

-- name: GetReminder :one
SELECT * FROM reminders WHERE id = $1 AND depot_id = $2;

-- name: MarkReminderOrdered :execrows
UPDATE reminders SET status = 'ordered', order_id = $2
WHERE id = $1 AND customer_id = $3 AND status = 'sent' AND sent_at >= sqlc.arg('sent_after')::timestamptz;
