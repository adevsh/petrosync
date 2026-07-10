
-- =============================================================================
-- SECTION 10 — VEHICLE COMPARTMENTS
-- =============================================================================

-- name: GetCompartment :one
SELECT *
FROM vehicle_compartments
WHERE id = $1;

-- name: ListCompartmentsByVehicle :many
SELECT *
FROM vehicle_compartments
WHERE vehicle_id = $1
  AND is_active = TRUE
ORDER BY compartment_number;

-- name: ListAllCompartmentsByVehicle :many
-- Includes inactive compartments (admin view).
SELECT *
FROM vehicle_compartments
WHERE vehicle_id = $1
ORDER BY compartment_number;

-- name: ListCompartmentsByVehicleAndFuel :many
SELECT *
FROM vehicle_compartments
WHERE vehicle_id      = $1
  AND fuel_type_code  = $2
  AND is_active       = TRUE
ORDER BY compartment_number;

-- name: GetTotalCapacityByVehicle :one
SELECT
    vehicle_id,
    COUNT(*)                   AS compartment_count,
    SUM(capacity_l)            AS total_capacity_l
FROM vehicle_compartments
WHERE vehicle_id = $1
  AND is_active = TRUE
GROUP BY vehicle_id;

-- name: CreateCompartment :one
INSERT INTO vehicle_compartments (
    vehicle_id, compartment_number, fuel_type_code, capacity_l
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: UpdateCompartmentFuelType :one
UPDATE vehicle_compartments
SET
    fuel_type_code = $2,
    updated_at     = NOW()
WHERE id = $1
RETURNING *;

-- name: DeactivateCompartment :exec
UPDATE vehicle_compartments
SET is_active = FALSE, updated_at = NOW()
WHERE id = $1;
