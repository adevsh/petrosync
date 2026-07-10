
-- =============================================================================
-- SECTION 23 — TRIP EVENTS (APPEND-ONLY)
-- =============================================================================

-- name: GetTripEventByUUID :one
-- Used by idempotency check before inserting from offline queue.
SELECT *
FROM trip_events
WHERE event_uuid = $1;

-- name: ListTripEventsByTrip :many
SELECT *
FROM trip_events
WHERE trip_id = $1
ORDER BY event_timestamp ASC;

-- name: ListTripEventsByTripAndType :many
SELECT *
FROM trip_events
WHERE trip_id    = $1
  AND event_type = $2
ORDER BY event_timestamp ASC;

-- name: GetLatestTripEventByType :one
SELECT *
FROM trip_events
WHERE trip_id    = $1
  AND event_type = $2
ORDER BY event_timestamp DESC
LIMIT 1;

-- name: GetLatestTripEvent :one
SELECT *
FROM trip_events
WHERE trip_id = $1
ORDER BY event_timestamp DESC
LIMIT 1;

-- name: InsertTripEvent :one
-- Append-only. event_uuid checked for duplicate before calling this.
INSERT INTO trip_events (
    trip_id, event_uuid, event_type, event_timestamp,
    actor_user_id, location, payload
) VALUES (
    $1, $2, $3, $4, $5,
    CASE WHEN $6::FLOAT8 IS NOT NULL AND $7::FLOAT8 IS NOT NULL
         THEN ST_SetSRID(ST_MakePoint($6, $7), 4326)
         ELSE NULL
    END,
    $8
)
RETURNING *;
