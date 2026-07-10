
-- =============================================================================
-- SECTION 26 — GPS EVENTS (APPEND-ONLY, PARTITIONED)
-- =============================================================================

-- name: InsertGPSEvent :one
-- Partition routing is transparent; insert to parent table.
INSERT INTO gps_events (
    trip_id, event_uuid,
    latitude, longitude,
    location,
    speed_kmh, heading_deg, accuracy_m,
    event_timestamp
) VALUES (
    $1, $2,
    $3, $4,
    ST_SetSRID(ST_MakePoint($4, $3), 4326),
    $5, $6, $7,
    $8
)
RETURNING id, trip_id, event_uuid, latitude, longitude,
          speed_kmh, heading_deg, accuracy_m,
          event_timestamp, received_at;

-- name: GetLatestGPSEventByTrip :one
SELECT id, trip_id, latitude, longitude, speed_kmh, heading_deg, event_timestamp
FROM gps_events
WHERE trip_id = $1
ORDER BY event_timestamp DESC
LIMIT 1;

-- name: ListGPSEventsByTripAndTimeRange :many
-- Route reconstruction for a trip segment.
SELECT id, trip_id, latitude, longitude, speed_kmh, heading_deg, event_timestamp
FROM gps_events
WHERE trip_id         = $1
  AND event_timestamp >= $2
  AND event_timestamp <= $3
ORDER BY event_timestamp ASC;

-- name: ListGPSEventsByTrip :many
SELECT id, trip_id, latitude, longitude, speed_kmh, heading_deg, event_timestamp
FROM gps_events
WHERE trip_id = $1
ORDER BY event_timestamp ASC;

-- name: CheckGPSEventUUIDExists :one
SELECT EXISTS (
    SELECT 1 FROM gps_events WHERE event_uuid = $1
) AS exists;
