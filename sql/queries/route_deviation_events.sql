
-- =============================================================================
-- SECTION 29 — ROUTE DEVIATION EVENTS
-- =============================================================================

-- name: ListDeviationsByTrip :many
SELECT *
FROM route_deviation_events
WHERE trip_id = $1
ORDER BY detected_at ASC;

-- name: GetOpenDeviationByTrip :one
-- Returns the most recent unresolved deviation (for cooldown state machine).
SELECT *
FROM route_deviation_events
WHERE trip_id    = $1
  AND resolved_at IS NULL
ORDER BY detected_at DESC
LIMIT 1;

-- name: CountTripDeviations :one
-- Determines escalation tier (1 = log, 2 = warn, >2 = Telegram alert).
SELECT COUNT(*)::INT AS deviation_count
FROM route_deviation_events
WHERE trip_id = $1;

-- name: CreateDeviationEvent :one
INSERT INTO route_deviation_events (
    trip_id, detected_at, deviation_meters, occurrence_count
) VALUES (
    $1, NOW(), $2, $3
)
RETURNING *;

-- name: UpdateDeviationDuration :one
-- Called when a deviation is resolved to record total duration.
UPDATE route_deviation_events
SET
    duration_seconds = EXTRACT(EPOCH FROM (NOW() - detected_at))::INT,
    resolved_at      = NOW()
WHERE id = $1
RETURNING *;

-- name: MarkDeviationTelegramNotified :exec
UPDATE route_deviation_events
SET
    telegram_notified    = TRUE,
    telegram_notified_at = NOW()
WHERE id = $1;

-- name: ListUnnotifiedDeviationsAboveThreshold :many
-- Background worker: fetch deviations that have exceeded alert threshold and not yet notified.
SELECT rde.*, t.driver_id, t.vehicle_id
FROM route_deviation_events rde
JOIN trips t ON t.id = rde.trip_id
WHERE rde.telegram_notified = FALSE
  AND rde.resolved_at IS NULL
  AND rde.detected_at <= (NOW() - ($1 || ' minutes')::INTERVAL)
ORDER BY rde.detected_at ASC;
