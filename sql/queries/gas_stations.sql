
-- =============================================================================
-- SECTION 15 — GAS STATIONS
-- =============================================================================

-- name: GetStation :one
SELECT
    gs.*,
    ST_X(gs.location) AS longitude,
    ST_Y(gs.location) AS latitude
FROM gas_stations gs
WHERE gs.id = $1;

-- name: GetStationByCode :one
SELECT
    gs.*,
    ST_X(gs.location) AS longitude,
    ST_Y(gs.location) AS latitude
FROM gas_stations gs
WHERE gs.code = $1;

-- name: ListStationsByRegion :many
SELECT
    gs.*,
    ST_X(gs.location) AS longitude,
    ST_Y(gs.location) AS latitude
FROM gas_stations gs
WHERE gs.region_code = $1
  AND gs.active = TRUE
ORDER BY gs.name;

-- name: ListStationsByFacility :many
SELECT
    gs.*,
    ST_X(gs.location) AS longitude,
    ST_Y(gs.location) AS latitude
FROM gas_stations gs
WHERE gs.primary_facility_id = $1
  AND gs.active = TRUE
ORDER BY gs.name;

-- name: ListStationsServedByFacility :many
-- Returns all stations where the facility is in the supply whitelist.
SELECT
    gs.*,
    ST_X(gs.location) AS longitude,
    ST_Y(gs.location) AS latitude
FROM gas_stations gs
JOIN station_facility_whitelist sfw ON sfw.station_id = gs.id
WHERE sfw.facility_id = $1
  AND gs.active = TRUE
ORDER BY gs.name;

-- name: ListAllActiveStations :many
SELECT
    gs.*,
    ST_X(gs.location) AS longitude,
    ST_Y(gs.location) AS latitude
FROM gas_stations gs
WHERE gs.active = TRUE
ORDER BY gs.region_code, gs.name;

-- name: ListAllActiveStationsByRefineryScope :many
SELECT
    gs.*,
    ST_X(gs.location) AS longitude,
    ST_Y(gs.location) AS latitude
FROM gas_stations gs
JOIN refinery_facilities rf ON rf.id = gs.primary_facility_id
WHERE gs.active = TRUE
  AND rf.refinery_id = $1
ORDER BY gs.region_code, gs.name;

-- name: ListAllActiveStationsByStationScope :many
SELECT
    gs.*,
    ST_X(gs.location) AS longitude,
    ST_Y(gs.location) AS latitude
FROM gas_stations gs
WHERE gs.active = TRUE
  AND gs.id = $1
ORDER BY gs.region_code, gs.name;

-- name: CreateStation :one
INSERT INTO gas_stations (
    code, name, spbu_license_number, region_code, primary_facility_id,
    location, address, operating_hours_start, operating_hours_end,
    contact_name, contact_phone
) VALUES (
    $1, $2, $3, $4, $5,
    ST_SetSRID(ST_MakePoint($6, $7), 4326),
    $8, $9, $10, $11, $12
)
RETURNING *;

-- name: UpdateStation :one
UPDATE gas_stations
SET
    name                  = $2,
    contact_name          = $3,
    contact_phone         = $4,
    operating_hours_start = $5,
    operating_hours_end   = $6,
    address               = $7,
    updated_at            = NOW()
WHERE id = $1
RETURNING *;

-- name: DeactivateStation :exec
UPDATE gas_stations
SET active = FALSE, updated_at = NOW()
WHERE id = $1;
