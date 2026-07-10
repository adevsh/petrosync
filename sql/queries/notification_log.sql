
-- =============================================================================
-- SECTION 30 — NOTIFICATION LOG (APPEND-ONLY)
-- =============================================================================

-- name: InsertNotification :one
INSERT INTO notification_log (
    trip_id, do_id, recipient_telegram_id, recipient_user_id,
    notification_type, message_text, delivery_status, telegram_message_id, error_message
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING *;

-- name: ListNotificationsByTrip :many
SELECT *
FROM notification_log
WHERE trip_id = $1
ORDER BY sent_at DESC;

-- name: ListNotificationsByRecipient :many
SELECT *
FROM notification_log
WHERE recipient_user_id = $1
ORDER BY sent_at DESC
LIMIT $2;

-- name: CountNotificationsByTypeAndTrip :one
SELECT COUNT(*)::INT AS count
FROM notification_log
WHERE trip_id           = $1
  AND notification_type = $2;
