
-- =============================================================================
-- SECTION 18 — STATION TANKS
-- =============================================================================

-- name: GetStationTank :one
SELECT *
FROM station_tanks
WHERE id = $1;

-- name: GetStationTankByFuel :one
SELECT *
FROM station_tanks
WHERE station_id    = $1
  AND fuel_type_code = $2
  AND active = TRUE;

-- name: ListStationTanksByStation :many
SELECT *
FROM station_tanks
WHERE station_id = $1
  AND active = TRUE
ORDER BY fuel_type_code;

-- name: ListStationTanksBelowReorderThreshold :many
-- Phase 3: auto-DO trigger source. Returns all under-threshold active tanks.
SELECT
    st.*,
    gs.name                AS station_name,
    gs.code                AS station_code,
    gs.primary_facility_id,
    gs.region_code,
    ROUND((st.current_volume_l / NULLIF(st.reorder_threshold_l, 0) * 100)::NUMERIC, 1) AS fill_pct
FROM station_tanks st
JOIN gas_stations  gs ON gs.id = st.station_id
WHERE st.active           = TRUE
  AND gs.active           = TRUE
  AND st.current_volume_l <= st.reorder_threshold_l
ORDER BY fill_pct ASC;

-- name: CreateStationTank :one
INSERT INTO station_tanks (
    station_id, tank_code, fuel_type_code,
    capacity_l, current_volume_l, reorder_threshold_l
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: UpdateStationTankVolumeAfterDelivery :one
-- Called by reconciliation engine after delivery is confirmed.
UPDATE station_tanks
SET
    current_volume_l = current_volume_l + $2,
    last_updated_at  = NOW()
WHERE id = $1
  AND active = TRUE
RETURNING *;

-- name: UpdateDipReading :one
UPDATE station_tanks
SET
    last_dip_reading_l  = $2,
    last_dip_at         = NOW(),
    last_updated_at     = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateStationTankReorderThreshold :one
UPDATE station_tanks
SET
    reorder_threshold_l = $2,
    last_updated_at     = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateStationTankVolume :one
-- Manual correction. Caller must write to audit_log.
UPDATE station_tanks
SET
    current_volume_l = $2,
    last_updated_at  = NOW()
WHERE id = $1
RETURNING *;

-- name: DeactivateStationTank :exec
UPDATE station_tanks
SET active = FALSE, last_updated_at = NOW()
WHERE id = $1;
