
-- =============================================================================
-- SECTION 5 — REFINERY FACILITIES
-- =============================================================================

-- name: GetFacility :one
SELECT
    rf.*,
    ST_X(rf.location) AS longitude,
    ST_Y(rf.location) AS latitude
FROM refinery_facilities rf
WHERE rf.id = $1;

-- name: GetFacilityByCode :one
SELECT
    rf.*,
    ST_X(rf.location) AS longitude,
    ST_Y(rf.location) AS latitude
FROM refinery_facilities rf
WHERE rf.code = $1;

-- name: ListFacilitiesByRefinery :many
SELECT
    rf.*,
    ST_X(rf.location) AS longitude,
    ST_Y(rf.location) AS latitude
FROM refinery_facilities rf
WHERE rf.refinery_id = $1
  AND rf.active = TRUE
ORDER BY rf.is_primary DESC, rf.name;

-- name: GetPrimaryFacilityByRefinery :one
SELECT
    rf.*,
    ST_X(rf.location) AS longitude,
    ST_Y(rf.location) AS latitude
FROM refinery_facilities rf
WHERE rf.refinery_id = $1
  AND rf.is_primary = TRUE
  AND rf.active = TRUE;

-- name: ListAllActiveFacilities :many
SELECT
    rf.*,
    r.code  AS refinery_code,
    r.name  AS refinery_name,
    ST_X(rf.location) AS longitude,
    ST_Y(rf.location) AS latitude
FROM refinery_facilities rf
JOIN refineries r ON r.id = rf.refinery_id
WHERE rf.active = TRUE
ORDER BY r.code, rf.is_primary DESC;

-- name: CreateFacility :one
INSERT INTO refinery_facilities (
    code, refinery_id, name, location, is_primary, max_assignment_radius_km, address
) VALUES (
    $1, $2, $3,
    ST_SetSRID(ST_MakePoint($4, $5), 4326),
    $6, $7, $8
)
RETURNING *;

-- name: UpdateFacility :one
UPDATE refinery_facilities
SET
    name                     = $2,
    address                  = $3,
    max_assignment_radius_km = $4,
    updated_at               = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateFacilityLocation :one
UPDATE refinery_facilities
SET
    location   = ST_SetSRID(ST_MakePoint($2, $3), 4326),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeactivateFacility :exec
UPDATE refinery_facilities
SET active = FALSE, updated_at = NOW()
WHERE id = $1;
