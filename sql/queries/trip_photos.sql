
-- =============================================================================
-- SECTION 27 — TRIP PHOTOS
-- =============================================================================

-- name: GetTripPhoto :one
SELECT *
FROM trip_photos
WHERE id = $1;

-- name: ListPhotosByTrip :many
SELECT *
FROM trip_photos
WHERE trip_id = $1
ORDER BY taken_at;

-- name: ListPhotosByTripAndEvent :many
SELECT *
FROM trip_photos
WHERE trip_id    = $1
  AND event_type = $2
ORDER BY taken_at;

-- name: ListPhotosByTripAndCompartment :many
SELECT *
FROM trip_photos
WHERE trip_id      = $1
  AND compartment_id = $2
ORDER BY taken_at;

-- name: CreateTripPhoto :one
INSERT INTO trip_photos (
    trip_id, compartment_id, event_type,
    garage_object_key, file_size_bytes, mime_type,
    uploaded_by, taken_at, notes
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING *;

-- name: GetMandatoryPhotoCheckByTrip :one
-- Verifies all mandatory photo events are present for a trip before allowing progression.
-- Returns a bitmask-style row showing which required events have at least one photo.
-- IMPORTANT: has_compartment_sealed_photo returns TRUE if >= 1 sealed photo exists.
-- The service layer must separately verify the sealed photo count matches the trip's
-- compartment count (ListCompartmentsByVehicle) before allowing LOADING → LOADED.
SELECT
    BOOL_OR(event_type = 'WEIGHT_BRIDGE_TARE')  AS has_tare_photo,
    BOOL_OR(event_type = 'WEIGHT_BRIDGE_GROSS') AS has_gross_photo,
    BOOL_OR(event_type = 'COMPARTMENT_SEALED')   AS has_compartment_sealed_photo,
    BOOL_OR(event_type = 'STATION_TANK_BEFORE') AS has_before_photo,
    BOOL_OR(event_type = 'PUMP_METER_READING')  AS has_pump_photo,
    BOOL_OR(event_type = 'STATION_TANK_AFTER')  AS has_after_photo
FROM trip_photos
WHERE trip_id = $1;
