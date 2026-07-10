
-- =============================================================================
-- SECTION 22 — TRIPS
-- =============================================================================

-- name: GetTrip :one
SELECT
    t.*,
    ST_AsGeoJSON(t.route_polyline) AS route_geojson
FROM trips t
WHERE t.id = $1;

-- name: GetTripByDO :one
SELECT
    t.*,
    ST_AsGeoJSON(t.route_polyline) AS route_geojson
FROM trips t
WHERE t.do_id = $1;

-- name: GetTripWithDetails :one
SELECT
    t.*,
    v.plate_number,
    u.full_name     AS driver_name,
    u.telegram_user_id AS driver_telegram_id,
    gs.name         AS destination_station_name,
    gs.code         AS destination_station_code,
    rf.name         AS origin_facility_name,
    ST_AsGeoJSON(t.route_polyline) AS route_geojson
FROM trips t
JOIN vehicles             v   ON v.id  = t.vehicle_id
JOIN drivers              d   ON d.id  = t.driver_id
JOIN users                u   ON u.id  = d.user_id
JOIN refinery_facilities  rf  ON rf.id = t.origin_facility_id
LEFT JOIN gas_stations    gs  ON gs.id = t.destination_station_id
WHERE t.id = $1;

-- name: ListActiveTrips :many
-- All in-flight trips across all facilities (for dashboard map feed).
SELECT
    t.id,
    t.status,
    t.vehicle_id,
    t.driver_id,
    t.origin_facility_id,
    t.destination_station_id,
    t.departed_at,
    v.plate_number,
    u.full_name        AS driver_name,
    u.telegram_user_id AS driver_telegram_id,
    gs.name            AS destination_name
FROM trips t
JOIN vehicles          v  ON v.id  = t.vehicle_id
JOIN drivers           d  ON d.id  = t.driver_id
JOIN users             u  ON u.id  = d.user_id
LEFT JOIN gas_stations gs ON gs.id = t.destination_station_id
WHERE t.status IN ('LOADING','LOADED','IN_TRANSIT','ARRIVED','UNLOADING')
ORDER BY t.departed_at ASC NULLS LAST;

-- name: ListActiveTripsByFacilityScope :many
SELECT
    t.id,
    t.status,
    t.vehicle_id,
    t.driver_id,
    t.origin_facility_id,
    t.destination_station_id,
    t.departed_at,
    v.plate_number,
    u.full_name        AS driver_name,
    u.telegram_user_id AS driver_telegram_id,
    gs.name            AS destination_name
FROM trips t
JOIN vehicles          v  ON v.id  = t.vehicle_id
JOIN drivers           d  ON d.id  = t.driver_id
JOIN users             u  ON u.id  = d.user_id
LEFT JOIN gas_stations gs ON gs.id = t.destination_station_id
WHERE t.status IN ('LOADING','LOADED','IN_TRANSIT','ARRIVED','UNLOADING')
  AND t.origin_facility_id = $1
ORDER BY t.departed_at ASC NULLS LAST;

-- name: ListActiveTripsByRefineryScope :many
SELECT
    t.id,
    t.status,
    t.vehicle_id,
    t.driver_id,
    t.origin_facility_id,
    t.destination_station_id,
    t.departed_at,
    v.plate_number,
    u.full_name        AS driver_name,
    u.telegram_user_id AS driver_telegram_id,
    gs.name            AS destination_name
FROM trips t
JOIN refinery_facilities rf ON rf.id = t.origin_facility_id
JOIN vehicles             v  ON v.id  = t.vehicle_id
JOIN drivers              d  ON d.id  = t.driver_id
JOIN users                u  ON u.id  = d.user_id
LEFT JOIN gas_stations    gs ON gs.id = t.destination_station_id
WHERE t.status IN ('LOADING','LOADED','IN_TRANSIT','ARRIVED','UNLOADING')
  AND rf.refinery_id = $1
ORDER BY t.departed_at ASC NULLS LAST;

-- name: ListActiveTripsByStationScope :many
SELECT
    t.id,
    t.status,
    t.vehicle_id,
    t.driver_id,
    t.origin_facility_id,
    t.destination_station_id,
    t.departed_at,
    v.plate_number,
    u.full_name        AS driver_name,
    u.telegram_user_id AS driver_telegram_id,
    gs.name            AS destination_name
FROM trips t
JOIN vehicles          v  ON v.id  = t.vehicle_id
JOIN drivers           d  ON d.id  = t.driver_id
JOIN users             u  ON u.id  = d.user_id
LEFT JOIN gas_stations gs ON gs.id = t.destination_station_id
WHERE t.status IN ('LOADING','LOADED','IN_TRANSIT','ARRIVED','UNLOADING')
  AND t.destination_station_id = $1
ORDER BY t.departed_at ASC NULLS LAST;

-- name: ListActiveTripsByDriverUserScope :many
SELECT
    t.id,
    t.status,
    t.vehicle_id,
    t.driver_id,
    t.origin_facility_id,
    t.destination_station_id,
    t.departed_at,
    v.plate_number,
    u.full_name        AS driver_name,
    u.telegram_user_id AS driver_telegram_id,
    gs.name            AS destination_name
FROM trips t
JOIN vehicles          v  ON v.id  = t.vehicle_id
JOIN drivers           d  ON d.id  = t.driver_id
JOIN users             u  ON u.id  = d.user_id
LEFT JOIN gas_stations gs ON gs.id = t.destination_station_id
WHERE t.status IN ('LOADING','LOADED','IN_TRANSIT','ARRIVED','UNLOADING')
  AND u.id = $1
ORDER BY t.departed_at ASC NULLS LAST;

-- name: ListActiveTripsByFacility :many
SELECT t.*
FROM trips t
WHERE t.origin_facility_id = $1
  AND t.status NOT IN ('CLOSED','CANCELLED','RECONCILED')
ORDER BY t.created_at DESC;

-- name: ListTripsByDriver :many
SELECT t.*
FROM trips t
WHERE t.driver_id = $1
ORDER BY t.created_at DESC
LIMIT $2;

-- name: ListTripsByVehicle :many
SELECT t.*
FROM trips t
WHERE t.vehicle_id = $1
ORDER BY t.created_at DESC
LIMIT $2;

-- name: ListTripsByStatus :many
SELECT t.*
FROM trips t
WHERE t.status = $1
ORDER BY t.created_at DESC;

-- name: ListReturnTrips :many
-- All auto-created return-to-facility trips (have parent_trip_id set).
SELECT t.*
FROM trips t
WHERE t.parent_trip_id IS NOT NULL
ORDER BY t.created_at DESC;

-- Real-time dashboard map: latest GPS position per active trip via LATERAL join.
-- name: ListActiveTripsWithLatestGPS :many
SELECT
    t.id,
    t.status,
    t.vehicle_id,
    t.driver_id,
    t.origin_facility_id,
    t.destination_station_id,
    t.departed_at,
    v.plate_number,
    u.full_name         AS driver_name,
    gs.name             AS destination_name,
    ge.latitude         AS last_lat,
    ge.longitude        AS last_lng,
    ge.speed_kmh        AS last_speed_kmh,
    ge.event_timestamp  AS last_gps_at
FROM trips t
JOIN vehicles          v  ON v.id  = t.vehicle_id
JOIN drivers           d  ON d.id  = t.driver_id
JOIN users             u  ON u.id  = d.user_id
LEFT JOIN gas_stations gs ON gs.id = t.destination_station_id
LEFT JOIN LATERAL (
    SELECT latitude, longitude, speed_kmh, event_timestamp
    FROM gps_events
    WHERE trip_id = t.id
    ORDER BY event_timestamp DESC
    LIMIT 1
) ge ON TRUE
WHERE t.status IN ('IN_TRANSIT', 'ARRIVED', 'UNLOADING');

-- name: CreateTrip :one
INSERT INTO trips (
    do_id, vehicle_id, driver_id, status,
    destination_type, origin_facility_id,
    destination_station_id, destination_facility_id,
    parent_trip_id
) VALUES (
    $1, $2, $3, 'CREATED',
    $4, $5, $6, $7, $8
)
RETURNING *;

-- name: UpdateTripStatus :one
UPDATE trips
SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: SetTripDeparted :one
UPDATE trips
SET
    status     = 'IN_TRANSIT',
    departed_at = NOW(),
    updated_at  = NOW()
WHERE id = $1
RETURNING *;

-- name: SetTripArrived :one
UPDATE trips
SET
    status    = 'ARRIVED',
    arrived_at = NOW(),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: SetTripCompleted :one
UPDATE trips
SET
    status       = 'DELIVERED',
    completed_at = NOW(),
    updated_at   = NOW()
WHERE id = $1
RETURNING *;

-- name: AppendTripRoutePoint :exec
-- Extends route polyline with the latest GPS coordinate (called by background worker).
UPDATE trips
SET route_polyline = ST_MakeLine(
        COALESCE(route_polyline, ST_MakePoint($2, $3)::GEOMETRY),
        ST_SetSRID(ST_MakePoint($2, $3), 4326)
    ),
    updated_at = NOW()
WHERE id = $1;
