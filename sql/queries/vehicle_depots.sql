
-- =============================================================================
-- SECTION 6 — VEHICLE DEPOTS
-- =============================================================================

-- name: GetDepot :one
SELECT
    vd.*,
    ST_X(vd.location) AS longitude,
    ST_Y(vd.location) AS latitude
FROM vehicle_depots vd
WHERE vd.id = $1;

-- name: GetDepotByCode :one
SELECT
    vd.*,
    ST_X(vd.location) AS longitude,
    ST_Y(vd.location) AS latitude
FROM vehicle_depots vd
WHERE vd.code = $1;

-- name: ListDepotsByFacility :many
SELECT
    vd.*,
    ST_X(vd.location) AS longitude,
    ST_Y(vd.location) AS latitude
FROM vehicle_depots vd
WHERE vd.primary_facility_id = $1
  AND vd.active = TRUE
ORDER BY vd.name;

-- name: ListAllActiveDepots :many
SELECT
    vd.*,
    rf.code AS facility_code,
    rf.name AS facility_name,
    ST_X(vd.location) AS longitude,
    ST_Y(vd.location) AS latitude
FROM vehicle_depots vd
JOIN refinery_facilities rf ON rf.id = vd.primary_facility_id
WHERE vd.active = TRUE
ORDER BY rf.code, vd.name;

-- name: CreateDepot :one
INSERT INTO vehicle_depots (
    code, name, primary_facility_id, location, default_truck_capacity_l
) VALUES (
    $1, $2, $3,
    ST_SetSRID(ST_MakePoint($4, $5), 4326),
    $6
)
RETURNING *;

-- name: UpdateDepot :one
UPDATE vehicle_depots
SET
    name                     = $2,
    default_truck_capacity_l = $3,
    updated_at               = NOW()
WHERE id = $1
RETURNING *;
