-- name: ReopenReminderForCancelledOrder :exec
UPDATE reminders SET status = 'sent', order_id = NULL WHERE order_id = $1 AND status = 'ordered';
