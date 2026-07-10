
-- =============================================================================
-- SECTION 11 — VEHICLE MAINTENANCE RECORDS
-- =============================================================================

-- name: GetMaintenanceRecord :one
SELECT *
FROM vehicle_maintenance_records
WHERE id = $1;

-- name: ListMaintenanceByVehicle :many
SELECT *
FROM vehicle_maintenance_records
WHERE vehicle_id = $1
ORDER BY started_at DESC;

-- name: ListOpenMaintenanceByVehicle :many
SELECT *
FROM vehicle_maintenance_records
WHERE vehicle_id   = $1
  AND completed_at IS NULL
ORDER BY started_at DESC;

-- name: ListAllOpenMaintenance :many
SELECT
    vmr.*,
    v.plate_number,
    vd.name AS depot_name
FROM vehicle_maintenance_records vmr
JOIN vehicles       v  ON v.id  = vmr.vehicle_id
JOIN vehicle_depots vd ON vd.id = v.current_depot_id
WHERE vmr.completed_at IS NULL
ORDER BY vmr.started_at ASC;

-- name: CreateMaintenanceRecord :one
INSERT INTO vehicle_maintenance_records (
    vehicle_id, recorded_by, maintenance_type,
    description, started_at, estimated_return_at, notes
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: CompleteMaintenanceRecord :one
UPDATE vehicle_maintenance_records
SET
    completed_at = NOW(),
    notes        = COALESCE($2, notes)
WHERE id = $1
RETURNING *;
